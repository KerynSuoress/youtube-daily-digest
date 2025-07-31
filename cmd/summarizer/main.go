package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"youtube-summarizer/internal/clients"
	"youtube-summarizer/internal/config"
	"youtube-summarizer/internal/logger"
	"youtube-summarizer/internal/services"
	"youtube-summarizer/internal/storage"
	"youtube-summarizer/pkg/types"
)

func main() {
	// Parse command line flags
	var (
		configPath  = flag.String("config", "configs/config.yaml", "Path to configuration file")
		envPath     = flag.String("env", ".env", "Path to environment file")
		excelPath   = flag.String("excel", "youtube-data.xlsx", "Path to Excel data file")
		testEmail   = flag.Bool("test-email", false, "Send test email and exit")
		development = flag.Bool("dev", false, "Run in development mode")
		showHelp    = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	// Initialize logger
	appLogger, err := logger.New(*development)
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer appLogger.Sync()

	appLogger.Info("Starting YouTube Summarizer", "version", "1.0.0", "development", *development)

	// Load environment variables
	if err := godotenv.Load(*envPath); err != nil {
		appLogger.Warn("Failed to load .env file (continuing with environment variables)", "error", err)
	}

	// Load configuration
	configLoader := config.NewLoader(*configPath, *envPath)
	cfg, err := configLoader.Load()
	if err != nil {
		appLogger.Error("Failed to load configuration", err)
		os.Exit(1)
	}

	appLogger.Info("Configuration loaded successfully")

	// Initialize application
	app, err := initializeApp(cfg, *excelPath, appLogger)
	if err != nil {
		appLogger.Error("Failed to initialize application", err)
		os.Exit(1)
	}

	// Handle test email mode
	if *testEmail {
		appLogger.Info("Running in test email mode")
		if err := app.emailService.SendTestEmail(context.Background()); err != nil {
			appLogger.Error("Failed to send test email", err)
			os.Exit(1)
		}
		appLogger.Info("Test email sent successfully")
		return
	}

	// Run the application
	if err := runApp(app, appLogger); err != nil {
		appLogger.Error("Application error", err)
		os.Exit(1)
	}
}

// App holds all application dependencies
type App struct {
	storage      *storage.ExcelStorage
	processor    *services.VideoProcessor
	emailService *services.EmailService
	config       *types.Config
	logger       types.Logger
}

// initializeApp sets up all dependencies and services
func initializeApp(cfg *types.Config, excelPath string, appLogger *logger.Logger) (*App, error) {
	// Get required environment variables
	youtubeAPIKey := os.Getenv("YOUTUBE_API_KEY")
	if youtubeAPIKey == "" {
		return nil, fmt.Errorf("YOUTUBE_API_KEY environment variable is required")
	}

	claudeAPIKey := os.Getenv("CLAUDE_API_KEY")
	if claudeAPIKey == "" {
		return nil, fmt.Errorf("CLAUDE_API_KEY environment variable is required")
	}

	rapidAPIKey := os.Getenv("RAPID_API_KEY")
	if rapidAPIKey == "" {
		appLogger.Warn("RAPID_API_KEY not found, transcript functionality may be limited")
	}

	emailUsername := os.Getenv("EMAIL_USERNAME")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	if emailUsername == "" || emailPassword == "" {
		appLogger.Warn("Email credentials not found, email functionality will be disabled")
	}

	// Initialize storage
	excelStorage := storage.NewExcelStorage(excelPath, appLogger)
	if err := excelStorage.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize Excel storage: %w", err)
	}

	// Initialize API clients
	youtubeClient := clients.NewYouTubeClient(youtubeAPIKey, appLogger)
	claudeClient := clients.NewClaudeClient(claudeAPIKey, appLogger)

	var transcriptClient types.TranscriptClient
	if rapidAPIKey != "" {
		transcriptClient = clients.NewTranscriptClient(rapidAPIKey, appLogger)
	} else {
		// Use mock transcript client if no API key
		transcriptClient = clients.NewMockTranscriptClient(appLogger)
		appLogger.Info("Using mock transcript client (no RapidAPI key provided)")
	}

	// Initialize services
	processor := services.NewVideoProcessor(
		excelStorage,
		youtubeClient,
		transcriptClient,
		claudeClient,
		cfg,
		appLogger,
	)

	var emailService *services.EmailService
	if emailUsername != "" && emailPassword != "" {
		var err error
		emailService, err = services.NewEmailService(cfg, emailUsername, emailPassword, appLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize email service: %w", err)
		}
	} else {
		appLogger.Warn("Email service disabled due to missing credentials")
	}

	return &App{
		storage:      excelStorage,
		processor:    processor,
		emailService: emailService,
		config:       cfg,
		logger:       appLogger,
	}, nil
}

// runApp runs the main application loop
func runApp(app *App, appLogger *logger.Logger) error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run initial processing
	appLogger.Info("Starting initial video processing")
	if err := app.processor.ProcessNewVideos(ctx); err != nil {
		appLogger.Error("Failed to process videos", err)
		return err
	}

	// Send email digest if there are pending summaries
	if app.emailService != nil {
		summaries, err := app.processor.ProcessPendingSummariesForEmail(ctx)
		if err != nil {
			appLogger.Error("Failed to get summaries for email", err)
		} else if len(summaries) > 0 {
			if err := app.emailService.SendDigest(ctx, summaries); err != nil {
				appLogger.Error("Failed to send email digest", err)
			} else {
				// Mark summaries as processed
				summaryIDs := make([]string, len(summaries))
				for i, summary := range summaries {
					summaryIDs[i] = summary.ID
				}
				if err := app.storage.MarkSummariesProcessed(ctx, summaryIDs); err != nil {
					appLogger.Error("Failed to mark summaries as processed", err)
				}
			}
		}
	}

	// Set up ticker for periodic processing
	var ticker *time.Ticker
	switch app.config.App.CheckFrequency {
	case "hourly":
		ticker = time.NewTicker(1 * time.Hour)
	case "daily":
		ticker = time.NewTicker(24 * time.Hour)
	case "weekly":
		ticker = time.NewTicker(7 * 24 * time.Hour)
	default:
		ticker = time.NewTicker(24 * time.Hour) // Default to daily
	}
	defer ticker.Stop()

	appLogger.Info("YouTube Summarizer started successfully", "checkFrequency", app.config.App.CheckFrequency)

	// Main application loop
	for {
		select {
		case <-ctx.Done():
			appLogger.Info("Context cancelled, shutting down")
			return nil

		case sig := <-sigChan:
			appLogger.Info("Received shutdown signal", "signal", sig)
			cancel()

		case <-ticker.C:
			appLogger.Info("Running periodic video processing")

			// Process videos
			if err := app.processor.ProcessNewVideos(ctx); err != nil {
				appLogger.Error("Periodic video processing failed", err)
				continue
			}

			// Send email digest if configured and it's time
			if app.emailService != nil && app.shouldSendEmail() {
				summaries, err := app.processor.ProcessPendingSummariesForEmail(ctx)
				if err != nil {
					appLogger.Error("Failed to get summaries for email", err)
					continue
				}

				if len(summaries) > 0 {
					if err := app.emailService.SendDigest(ctx, summaries); err != nil {
						appLogger.Error("Failed to send email digest", err)
					} else {
						// Mark summaries as processed
						summaryIDs := make([]string, len(summaries))
						for i, summary := range summaries {
							summaryIDs[i] = summary.ID
						}
						if err := app.storage.MarkSummariesProcessed(ctx, summaryIDs); err != nil {
							appLogger.Error("Failed to mark summaries as processed", err)
						}
					}
				}
			}
		}
	}
}

// shouldSendEmail determines if it's time to send an email digest
func (app *App) shouldSendEmail() bool {
	// Simple implementation - always send when there are summaries
	// In a more sophisticated version, this would check the last email sent time
	// and the configured email frequency
	return true
}

// printHelp prints usage information
func printHelp() {
	fmt.Printf(`YouTube Summarizer - Enterprise Go Application

USAGE:
    %s [OPTIONS]

OPTIONS:
    -config string    Path to configuration file (default: "configs/config.yaml")
    -env string       Path to environment file (default: ".env")
    -excel string     Path to Excel data file (default: "youtube-data.xlsx")
    -test-email       Send test email and exit
    -dev              Run in development mode with verbose logging
    -help             Show this help message

ENVIRONMENT VARIABLES:
    YOUTUBE_API_KEY    YouTube Data API v3 key (required)
    CLAUDE_API_KEY     Claude API key for summarization (required)
    RAPID_API_KEY      RapidAPI key for transcript fetching (optional)
    EMAIL_USERNAME     Email username for SMTP (optional)
    EMAIL_PASSWORD     Email password for SMTP (optional)

EXAMPLES:
    # Run with default configuration
    %s

    # Run in development mode
    %s -dev

    # Test email configuration
    %s -test-email

    # Use custom paths
    %s -config ./my-config.yaml -excel ./my-data.xlsx

DOCUMENTATION:
    For detailed setup instructions, see README.md
`, filepath.Base(os.Args[0]), filepath.Base(os.Args[0]), filepath.Base(os.Args[0]), filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
}
