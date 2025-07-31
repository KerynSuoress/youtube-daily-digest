package config

import (
	"fmt"
	"strings"

	"youtube-summarizer/pkg/types"

	"github.com/spf13/viper"
)

// Loader handles configuration loading from files and environment
type Loader struct {
	configPath string
	envPath    string
}

// NewLoader creates a new configuration loader
func NewLoader(configPath, envPath string) *Loader {
	return &Loader{
		configPath: configPath,
		envPath:    envPath,
	}
}

// Load loads configuration from file and environment variables
func (l *Loader) Load() (*types.Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// Set up viper
	viper.SetConfigFile(l.configPath)
	viper.SetConfigType("yaml")

	// Enable environment variable support
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional - we can work with defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into our config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// LoadFromEnvironment loads only environment variables (useful for testing)
func LoadFromEnvironment() (*types.Config, error) {
	config := DefaultConfig()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Manually bind environment variables to config
	bindEnvVars()

	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config from environment: %w", err)
	}

	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// bindEnvVars manually binds environment variables to viper keys
func bindEnvVars() {
	// App configuration
	viper.BindEnv("app.check_frequency", "APP_CHECK_FREQUENCY")
	viper.BindEnv("app.email_frequency", "APP_EMAIL_FREQUENCY")

	// YouTube configuration
	viper.BindEnv("youtube.max_videos_per_channel", "YOUTUBE_MAX_VIDEOS_PER_CHANNEL")

	// Processing configuration
	viper.BindEnv("processing.max_concurrent_videos", "PROCESSING_MAX_CONCURRENT_VIDEOS")
	viper.BindEnv("processing.transcript_timeout", "PROCESSING_TRANSCRIPT_TIMEOUT")

	// Email configuration
	viper.BindEnv("email.smtp_host", "EMAIL_SMTP_HOST")
	viper.BindEnv("email.smtp_port", "EMAIL_SMTP_PORT")
	viper.BindEnv("email.subject_template", "EMAIL_SUBJECT_TEMPLATE")

	// AI configuration
	viper.BindEnv("ai.max_transcript_length", "AI_MAX_TRANSCRIPT_LENGTH")
	viper.BindEnv("ai.summary_prompt", "AI_SUMMARY_PROMPT")
}
