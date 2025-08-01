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
			MaxVideosOnFirstRun: 10,
		},
		YouTube: types.YouTubeConfig{
			MaxVideosPerChannel: 5,
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
	if c.App.MaxVideosOnFirstRun <= 0 {
		return fmt.Errorf("app.max_videos_on_first_run must be greater than 0")
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
