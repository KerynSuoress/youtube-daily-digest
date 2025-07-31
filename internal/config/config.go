package config

import (
	"fmt"
	"time"

	"youtube-summarizer/pkg/types"
)

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *types.Config {
	return &types.Config{
		App: types.AppConfig{
			CheckFrequency: "daily",
			EmailFrequency: "weekly",
		},
		YouTube: types.YouTubeConfig{
			MaxVideosPerChannel: 1,
		},
		Processing: types.ProcessingConfig{
			MaxConcurrentVideos: 3,
			TranscriptTimeout:   30 * time.Second,
		},
		Email: types.EmailConfig{
			SMTPHost:        "smtp.gmail.com",
			SMTPPort:        587,
			SubjectTemplate: "YouTube Summary - {date}",
		},
		AI: types.AIConfig{
			MaxTranscriptLength: 15000,
			SummaryPrompt: `Video Title: "{title}". Summarize the key takeaways from the following video transcript into a concise paragraph. Focus on the main points and actionable advice:

{transcript}`,
		},
	}
}

// Validate checks if the configuration is valid
func Validate(c *types.Config) error {
	if c.App.CheckFrequency == "" {
		return fmt.Errorf("app.check_frequency cannot be empty")
	}

	if c.App.CheckFrequency != "daily" && c.App.CheckFrequency != "weekly" && c.App.CheckFrequency != "hourly" {
		return fmt.Errorf("app.check_frequency must be 'daily', 'weekly', or 'hourly'")
	}

	if c.App.EmailFrequency == "" {
		return fmt.Errorf("app.email_frequency cannot be empty")
	}

	if c.App.EmailFrequency != "daily" && c.App.EmailFrequency != "weekly" {
		return fmt.Errorf("app.email_frequency must be 'daily' or 'weekly'")
	}

	if c.YouTube.MaxVideosPerChannel <= 0 {
		return fmt.Errorf("youtube.max_videos_per_channel must be greater than 0")
	}

	if c.Processing.MaxConcurrentVideos <= 0 {
		return fmt.Errorf("processing.max_concurrent_videos must be greater than 0")
	}

	if c.Processing.TranscriptTimeout <= 0 {
		return fmt.Errorf("processing.transcript_timeout must be greater than 0")
	}

	if c.Email.SMTPHost == "" {
		return fmt.Errorf("email.smtp_host cannot be empty")
	}

	if c.Email.SMTPPort <= 0 {
		return fmt.Errorf("email.smtp_port must be greater than 0")
	}

	if c.AI.MaxTranscriptLength <= 0 {
		return fmt.Errorf("ai.max_transcript_length must be greater than 0")
	}

	if c.AI.SummaryPrompt == "" {
		return fmt.Errorf("ai.summary_prompt cannot be empty")
	}

	return nil
}
