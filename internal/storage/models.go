package storage

import (
	"strconv"
	"time"
	"youtube-summarizer/pkg/types"
)

const (
	// Excel sheet names
	ChannelsSheet        = "Channels"
	ProcessedVideosSheet = "ProcessedVideos"
	SummariesSheet       = "Summaries"
)

// ExcelChannel represents a channel record in Excel
type ExcelChannel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username,omitempty"`
	Added    string `json:"added"` // Date added as string
}

// ExcelProcessedVideo represents a processed video record in Excel
type ExcelProcessedVideo struct {
	VideoID     string `json:"video_id"`
	ChannelID   string `json:"channel_id"`
	Title       string `json:"title"`
	ProcessedAt string `json:"processed_at"` // Date as string
}

// ExcelSummary represents a summary record in Excel
type ExcelSummary struct {
	ID           string `json:"id"`
	VideoID      string `json:"video_id"`
	VideoTitle   string `json:"video_title"`
	ChannelName  string `json:"channel_name"`
	Summary      string `json:"summary"`
	CreatedAt    string `json:"created_at"` // Date as string
	Status       string `json:"status"`     // New, Processed
	VideoURL     string `json:"video_url"`
	PublishedAt  string `json:"published_at"`
	ThumbnailURL string `json:"thumbnail_url"`
	Duration     string `json:"duration"`
	ViewCount    string `json:"view_count"` // String for Excel compatibility
}

// ToChannel converts ExcelChannel to types.Channel
func (ec *ExcelChannel) ToChannel() types.Channel {
	return types.Channel{
		ID:       ec.ID,
		Name:     ec.Name,
		Username: ec.Username,
	}
}

// FromChannel converts types.Channel to ExcelChannel
func FromChannel(c types.Channel) ExcelChannel {
	return ExcelChannel{
		ID:       c.ID,
		Name:     c.Name,
		Username: c.Username,
		Added:    time.Now().Format("2006-01-02"),
	}
}

// ToSummary converts ExcelSummary to types.Summary
func (es *ExcelSummary) ToSummary() (types.Summary, error) {
	createdAt, err := time.Parse("2006-01-02 15:04:05", es.CreatedAt)
	if err != nil {
		// Try alternative format
		createdAt, err = time.Parse("2006-01-02", es.CreatedAt)
		if err != nil {
			return types.Summary{}, err
		}
	}

	publishedAt, err := time.Parse("2006-01-02 15:04:05", es.PublishedAt)
	if err != nil {
		// Try alternative format
		publishedAt, err = time.Parse("2006-01-02", es.PublishedAt)
		if err != nil {
			publishedAt = createdAt // Fallback to created date
		}
	}

	viewCount := int64(0)
	if es.ViewCount != "" {
		if count, err := strconv.ParseInt(es.ViewCount, 10, 64); err == nil {
			viewCount = count
		}
	}

	return types.Summary{
		ID:           es.ID,
		VideoID:      es.VideoID,
		VideoTitle:   es.VideoTitle,
		ChannelName:  es.ChannelName,
		Summary:      es.Summary,
		CreatedAt:    createdAt,
		Status:       es.Status,
		VideoURL:     es.VideoURL,
		PublishedAt:  publishedAt,
		ThumbnailURL: es.ThumbnailURL,
		Duration:     es.Duration,
		ViewCount:    viewCount,
	}, nil
}

// FromSummary converts types.Summary to ExcelSummary
func FromSummary(s types.Summary) ExcelSummary {
	return ExcelSummary{
		ID:           s.ID,
		VideoID:      s.VideoID,
		VideoTitle:   s.VideoTitle,
		ChannelName:  s.ChannelName,
		Summary:      s.Summary,
		CreatedAt:    s.CreatedAt.Format("2006-01-02 15:04:05"),
		Status:       s.Status,
		VideoURL:     s.VideoURL,
		PublishedAt:  s.PublishedAt.Format("2006-01-02 15:04:05"),
		ThumbnailURL: s.ThumbnailURL,
		Duration:     s.Duration,
		ViewCount:    strconv.FormatInt(s.ViewCount, 10),
	}
}

// ChannelHeaders returns the Excel column headers for channels
func ChannelHeaders() []string {
	return []string{"ID", "Name", "Username", "Added"}
}

// ProcessedVideoHeaders returns the Excel column headers for processed videos
func ProcessedVideoHeaders() []string {
	return []string{"VideoID", "ChannelID", "Title", "ProcessedAt"}
}

// SummaryHeaders returns the Excel column headers for summaries
func SummaryHeaders() []string {
	return []string{"ID", "VideoID", "VideoTitle", "ChannelName", "Summary", "CreatedAt", "Status", "VideoURL", "PublishedAt", "ThumbnailURL", "Duration", "ViewCount"}
}
