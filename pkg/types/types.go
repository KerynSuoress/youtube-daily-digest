package types

import (
	"context"
	"time"
)

// Channel represents a YouTube channel to monitor
type Channel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
}

// Video represents a YouTube video
type Video struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ChannelID   string    `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	PublishedAt time.Time `json:"published_at"`
	Duration    string    `json:"duration"`
	ViewCount   int64     `json:"view_count"`
	URL         string    `json:"url"`
}

// Summary represents a video summary
type Summary struct {
	ID           string    `json:"id"`
	VideoID      string    `json:"video_id"`
	VideoTitle   string    `json:"video_title"`
	ChannelName  string    `json:"channel_name"`
	Summary      string    `json:"summary"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"` // New, Processed
	VideoURL     string    `json:"video_url"`
	PublishedAt  time.Time `json:"published_at"`
	ThumbnailURL string    `json:"thumbnail_url"`
	Duration     string    `json:"duration"`
	ViewCount    int64     `json:"view_count"`
}

// TranscriptData contains transcript and thumbnail information
type TranscriptData struct {
	Transcript   string
	ThumbnailURL string
}

// Config represents the application configuration
type Config struct {
	App        AppConfig        `yaml:"app"`
	YouTube    YouTubeConfig    `yaml:"youtube"`
	Processing ProcessingConfig `yaml:"processing"`
	Email      EmailConfig      `yaml:"email"`
	AI         AIConfig         `yaml:"ai"`
}

type AppConfig struct {
	CheckFrequency string `yaml:"check_frequency"` // daily, weekly, hourly
	EmailFrequency string `yaml:"email_frequency"` // daily, weekly
}

type YouTubeConfig struct {
	MaxVideosPerChannel int `yaml:"max_videos_per_channel"`
}

type ProcessingConfig struct {
	MaxConcurrentVideos int           `yaml:"max_concurrent_videos"`
	TranscriptTimeout   time.Duration `yaml:"transcript_timeout"`
}

type EmailConfig struct {
	SMTPHost        string `yaml:"smtp_host"`
	SMTPPort        int    `yaml:"smtp_port"`
	SubjectTemplate string `yaml:"subject_template"`
}

type AIConfig struct {
	MaxTranscriptLength int    `yaml:"max_transcript_length"`
	SummaryPrompt       string `yaml:"summary_prompt"`
}

// Core interfaces for future UI expansion

// VideoProcessor handles the main business logic
type VideoProcessor interface {
	ProcessNewVideos(ctx context.Context) error
	GetProcessedVideos(ctx context.Context) ([]Video, error)
	UpdateConfig(config Config) error
}

// Storage handles data persistence
type Storage interface {
	GetChannels(ctx context.Context) ([]Channel, error)
	SaveSummary(ctx context.Context, summary Summary) error
	GetPendingSummaries(ctx context.Context) ([]Summary, error)
	MarkSummariesProcessed(ctx context.Context, summaryIDs []string) error
	IsVideoProcessed(ctx context.Context, videoID string) (bool, error)
	MarkVideoProcessed(ctx context.Context, videoID string) error
}

// AIClient handles AI summarization
type AIClient interface {
	Summarize(ctx context.Context, transcript, title string) (string, error)
}

// YouTubeClient handles YouTube API interactions
type YouTubeClient interface {
	GetChannelVideos(ctx context.Context, channelID string, maxResults int) ([]Video, error)
	GetVideoDetails(ctx context.Context, videoID string) (*Video, error)
}

// TranscriptClient handles transcript fetching
type TranscriptClient interface {
	GetTranscript(ctx context.Context, videoID string) (string, error)
	GetTranscriptWithThumbnail(ctx context.Context, videoID string) (*TranscriptData, error)
}

// EmailService handles email delivery
type EmailService interface {
	SendDigest(ctx context.Context, summaries []Summary) error
}

// Logger provides structured logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, err error, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}
