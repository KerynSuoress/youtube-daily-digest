package services

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"youtube-summarizer/pkg/types"

	"gopkg.in/gomail.v2"
)

// EmailService implements the types.EmailService interface
type EmailService struct {
	config *types.Config
	logger types.Logger

	// Email credentials
	username string
	password string

	// Template for email content
	emailTemplate *template.Template
}

// NewEmailService creates a new email service
func NewEmailService(
	config *types.Config,
	username, password string,
	logger types.Logger,
) (*EmailService, error) {

	// Create email template
	tmpl, err := template.New("email").Parse(defaultEmailTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse email template: %w", err)
	}

	return &EmailService{
		config:        config,
		logger:        logger,
		username:      username,
		password:      password,
		emailTemplate: tmpl,
	}, nil
}

// EmailData represents the data passed to the email template
type EmailData struct {
	Date       string
	Summaries  []types.Summary
	TotalCount int
}

// SendDigest sends an email digest with the provided summaries
func (es *EmailService) SendDigest(ctx context.Context, summaries []types.Summary) error {
	if len(summaries) == 0 {
		es.logger.Info("No summaries to send, skipping email digest")
		return nil
	}

	es.logger.Info("Preparing to send email digest", "summaryCount", len(summaries))

	// Prepare email data
	emailData := EmailData{
		Date:       time.Now().Format("January 2, 2006"),
		Summaries:  summaries,
		TotalCount: len(summaries),
	}

	// Debug: Log thumbnail URLs being passed to template
	for i, summary := range summaries {
		es.logger.Debug("Email template data", "index", i, "videoTitle", summary.VideoTitle, "thumbnailURL", summary.ThumbnailURL)
	}

	// Generate email content
	subject, body, err := es.generateEmailContent(emailData)
	if err != nil {
		return fmt.Errorf("failed to generate email content: %w", err)
	}

	// Send the email
	if err := es.sendEmail(subject, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	es.logger.Info("Successfully sent email digest", "summaryCount", len(summaries))
	return nil
}

// generateEmailContent creates the subject and body for the digest email
func (es *EmailService) generateEmailContent(data EmailData) (string, string, error) {
	// Generate subject
	subject := strings.ReplaceAll(es.config.Email.SubjectTemplate, "{date}", data.Date)

	// Generate body using template
	var body strings.Builder
	if err := es.emailTemplate.Execute(&body, data); err != nil {
		return "", "", fmt.Errorf("failed to execute email template: %w", err)
	}

	return subject, body.String(), nil
}

// sendEmail sends an email using SMTP
func (es *EmailService) sendEmail(subject, body string) error {
	m := gomail.NewMessage()

	// Set headers
	m.SetHeader("From", es.username)
	m.SetHeader("To", es.username) // Send to self for now
	m.SetHeader("Subject", subject)

	// Set body
	m.SetBody("text/html", body)

	// Create dialer
	d := gomail.NewDialer(
		es.config.Email.SMTPHost,
		es.config.Email.SMTPPort,
		es.username,
		es.password,
	)

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// SendTestEmail sends a test email to verify configuration
func (es *EmailService) SendTestEmail(ctx context.Context) error {
	es.logger.Info("Sending test email")

	// Create test summary
	testSummary := types.Summary{
		ID:           "test-001",
		VideoID:      "dQw4w9WgXcQ",
		VideoTitle:   "Test Video Title",
		ChannelName:  "Test Channel",
		Summary:      "This is a test summary to verify that the email system is working correctly. If you receive this email, your YouTube summarizer email configuration is properly set up.",
		CreatedAt:    time.Now(),
		Status:       "New",
		VideoURL:     "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		PublishedAt:  time.Now().AddDate(0, 0, -1), // Yesterday
		ThumbnailURL: "https://img.youtube.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
		Duration:     "3:33",
		ViewCount:    1234567890,
	}

	return es.SendDigest(ctx, []types.Summary{testSummary})
}

// SetEmailTemplate allows custom email templates
func (es *EmailService) SetEmailTemplate(templateStr string) error {
	tmpl, err := template.New("email").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	es.emailTemplate = tmpl
	es.logger.Info("Updated email template")
	return nil
}

// GetEmailTemplate returns the current email template
func (es *EmailService) GetEmailTemplate() string {
	return defaultEmailTemplate
}

// Default email template with Royal color palette
const defaultEmailTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>YouTube Summary Digest</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            color: #1C1B1F;
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(135deg, #F6F3EB 0%, #FEFFC4 100%);
            min-height: 100vh;
        }
        .container {
            background-color: #F6F3EB;
            border-radius: 16px;
            box-shadow: 0 8px 32px rgba(99, 13, 95, 0.15);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #630D5F 0%, #B37BA4 100%);
            color: #FEFFC4;
            text-align: center;
            padding: 40px 30px;
            margin-bottom: 0;
        }
        .header h1 {
            margin: 0;
            font-size: 2.8em;
            font-weight: 700;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        .header p {
            margin: 15px 0 0 0;
            font-size: 1.2em;
            opacity: 0.9;
            font-weight: 300;
        }
        .stats {
            background: linear-gradient(135deg, #BFA359 0%, #FEFFC4 100%);
            color: #1C1B1F;
            padding: 25px;
            text-align: center;
            font-weight: 600;
            font-size: 1.1em;
            border-bottom: 3px solid #630D5F;
        }
        .content-area {
            padding: 30px;
        }
        .video-card {
            background: linear-gradient(135deg, #FEFFC4 0%, #F6F3EB 100%);
            border: 2px solid #B37BA4;
            border-radius: 16px;
            padding: 0;
            margin-bottom: 30px;
            box-shadow: 0 6px 20px rgba(99, 13, 95, 0.1);
            overflow: hidden;
            transition: transform 0.2s ease;
        }
        .video-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 25px rgba(99, 13, 95, 0.2);
        }
        .video-header {
            display: flex;
            align-items: flex-start;
            padding: 25px;
            gap: 20px;
        }
        .thumbnail-container {
            flex-shrink: 0;
            position: relative;
        }
        .thumbnail {
            width: 180px !important;
            height: 101px !important;
            border-radius: 12px;
            object-fit: cover;
            border: 3px solid #630D5F;
            box-shadow: 0 4px 12px rgba(99, 13, 95, 0.3);
            display: block;
            max-width: 180px;
            max-height: 101px;
        }
        .duration-badge {
            position: absolute;
            bottom: 6px;
            right: 6px;
            background: rgba(28, 27, 31, 0.9);
            color: #FEFFC4;
            padding: 3px 8px;
            border-radius: 6px;
            font-size: 0.8em;
            font-weight: 600;
        }
        .video-info {
            flex: 1;
            min-width: 0;
        }
        .video-title {
            margin: 0 0 12px 0;
            color: #630D5F;
            font-size: 1.4em;
            font-weight: 700;
            line-height: 1.3;
            word-wrap: break-word;
        }
        .video-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 15px;
            margin-bottom: 15px;
            font-size: 0.9em;
            color: #1C1B1F;
        }
        .meta-item {
            display: flex;
            align-items: center;
            gap: 6px;
            background: rgba(179, 123, 164, 0.2);
            padding: 6px 12px;
            border-radius: 20px;
            font-weight: 500;
        }
        .channel-name {
            color: #B37BA4;
            font-weight: 600;
        }
        .summary-content {
            background: rgba(254, 255, 196, 0.3);
            border-left: 5px solid #BFA359;
            padding: 20px;
            margin: 0 25px 25px 25px;
            border-radius: 0 12px 12px 0;
            color: #1C1B1F;
            line-height: 1.7;
            font-size: 1.05em;
        }
        .video-actions {
            padding: 0 25px 25px 25px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            flex-wrap: wrap;
            gap: 15px;
        }
        .published-date {
            color: #B37BA4;
            font-size: 0.9em;
            font-weight: 500;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .watch-button {
            background: linear-gradient(135deg, #630D5F 0%, #B37BA4 100%);
            color: #FEFFC4 !important;
            text-decoration: none !important;
            padding: 12px 24px;
            border-radius: 25px;
            font-size: 1em;
            font-weight: 600;
            display: inline-flex;
            align-items: center;
            gap: 8px;
            transition: all 0.3s ease;
            box-shadow: 0 4px 12px rgba(99, 13, 95, 0.3);
        }
        .watch-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(99, 13, 95, 0.4);
            background: #630D5F;
        }
        .footer {
            text-align: center;
            padding: 30px;
            background: linear-gradient(135deg, #1C1B1F 0%, #630D5F 100%);
            color: #FEFFC4;
            margin-top: 0;
        }
        .footer p {
            margin: 8px 0;
            opacity: 0.9;
        }
        .footer .main-text {
            font-size: 1.1em;
            font-weight: 600;
        }
        .footer .sub-text {
            font-size: 0.95em;
            font-weight: 300;
        }
        @media (max-width: 600px) {
            .video-header {
                flex-direction: column;
                align-items: center;
                text-align: center;
            }
            .thumbnail {
                width: 280px;
                height: 157px;
            }
            .video-actions {
                flex-direction: column;
                align-items: center;
                text-align: center;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>YouTube Daily Digest</h1>
            <p>{{.Date}}</p>
        </div>

        <div class="stats">
            🎬 {{.TotalCount}} video summaries curated for you
        </div>

        <div class="content-area">
            {{range .Summaries}}
            <div class="video-card">
                <div class="video-header" style="display: flex; align-items: flex-start; padding: 25px; gap: 20px;">
                    <div class="thumbnail-container" style="flex-shrink: 0; position: relative;">
                        <img src="{{.ThumbnailURL}}" alt="{{.VideoTitle}} thumbnail" class="thumbnail" 
                             style="width: 180px; height: 101px; border-radius: 12px; object-fit: cover; border: 3px solid #630D5F; display: block; max-width: 180px; max-height: 101px;"
                             onerror="this.style.display='none'; this.nextElementSibling.style.display='block';" />
                        <!-- Fallback for when image fails to load -->
                        <div style="display: none; width: 180px; height: 101px; border-radius: 12px; border: 3px solid #630D5F; background: linear-gradient(135deg, #630D5F, #B37BA4); color: #FEFFC4; align-items: center; justify-content: center; text-align: center; font-size: 12px; font-weight: bold; padding: 10px; box-sizing: border-box;">
                            📺 Video<br/>Thumbnail
                        </div>
                        {{if .Duration}}<div class="duration-badge">{{.Duration}}</div>{{end}}
                    </div>
                    <div class="video-info" style="flex: 1; min-width: 0;">
                        <h3 class="video-title">{{.VideoTitle}}</h3>
                        <div class="video-meta">
                            <div class="meta-item">
                                <span style="margin-right: 5px;">📺</span>
                                <span class="channel-name">{{.ChannelName}}</span>
                            </div>
                            {{if gt .ViewCount 0}}
                            <div class="meta-item">
                                <span>👁</span>
                                <span>{{.ViewCount}} views</span>
                            </div>
                            {{end}}
                        </div>
                    </div>
                </div>
                
                <div class="summary-content">
                    {{.Summary}}
                </div>
                
                <div class="video-actions">
                    <div class="published-date">
                        <span style="margin-right: 5px;">📅</span>
                        <span>Published {{.PublishedAt.Format "Jan 2, 2006"}}</span>
                    </div>
                </div>
                <div class="video-actions">
                    <a href="{{.VideoURL}}" class="watch-button">
                        <span>Watch Video</span>
                    </a>
                </div>
            </div>
            {{end}}
        </div>

        <div class="footer">
            <p class="main-text">Generated for Geronimo Rodriguez</p>
            <p class="sub-text">🤖 Powered by Claude AI • Built with Go • Designed by Keryn Suoress</p>
        </div>
    </div>
</body>
</html>`
