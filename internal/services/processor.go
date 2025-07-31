package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"youtube-summarizer/pkg/types"
)

// VideoProcessor implements the types.VideoProcessor interface
type VideoProcessor struct {
	storage          types.Storage
	youtubeClient    types.YouTubeClient
	transcriptClient types.TranscriptClient
	aiClient         types.AIClient
	config           *types.Config
	logger           types.Logger
}

// NewVideoProcessor creates a new video processor
func NewVideoProcessor(
	storage types.Storage,
	youtubeClient types.YouTubeClient,
	transcriptClient types.TranscriptClient,
	aiClient types.AIClient,
	config *types.Config,
	logger types.Logger,
) *VideoProcessor {
	return &VideoProcessor{
		storage:          storage,
		youtubeClient:    youtubeClient,
		transcriptClient: transcriptClient,
		aiClient:         aiClient,
		config:           config,
		logger:           logger,
	}
}

// ProcessNewVideos processes new videos from all configured channels
func (vp *VideoProcessor) ProcessNewVideos(ctx context.Context) error {
	vp.logger.Info("Starting video processing cycle")

	// Get all channels to monitor
	channels, err := vp.storage.GetChannels(ctx)
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	if len(channels) == 0 {
		vp.logger.Info("No channels configured for monitoring")
		return nil
	}

	vp.logger.Info("Processing channels", "count", len(channels))

	// Process each channel concurrently with a semaphore to limit concurrency
	semaphore := make(chan struct{}, vp.config.Processing.MaxConcurrentVideos)
	var wg sync.WaitGroup
	errorsChan := make(chan error, len(channels))

	for _, channel := range channels {
		wg.Add(1)
		go func(ch types.Channel) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := vp.processChannel(ctx, ch); err != nil {
				vp.logger.Error("Failed to process channel", err, "channelID", ch.ID, "channelName", ch.Name)
				errorsChan <- fmt.Errorf("channel %s (%s): %w", ch.Name, ch.ID, err)
			}
		}(channel)
	}

	// Wait for all channels to be processed
	wg.Wait()
	close(errorsChan)

	// Collect errors
	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		vp.logger.Warn("Some channels failed to process", "errorCount", len(errors))
		// Don't fail the entire process if some channels fail
		for _, err := range errors {
			vp.logger.Error("Channel processing error", err)
		}
	}

	vp.logger.Info("Completed video processing cycle")
	return nil
}

// processChannel processes videos from a single channel
func (vp *VideoProcessor) processChannel(ctx context.Context, channel types.Channel) error {
	vp.logger.Debug("Processing channel", "channelID", channel.ID, "channelName", channel.Name)

	// Get recent videos from the channel
	videos, err := vp.youtubeClient.GetChannelVideos(ctx, channel.ID, vp.config.YouTube.MaxVideosPerChannel)
	if err != nil {
		return fmt.Errorf("failed to get channel videos: %w", err)
	}

	vp.logger.Debug("Retrieved videos from channel", "channelID", channel.ID, "count", len(videos))

	// Process each video with rate limiting
	processedCount := 0
	for i, video := range videos {
		// Add delay between videos to respect API limits (except for first video)
		if i > 0 {
			vp.logger.Debug("Rate limiting: waiting 2 seconds before next video")
			time.Sleep(2 * time.Second)
		}

		// Check if video is already processed
		processed, err := vp.storage.IsVideoProcessed(ctx, video.ID)
		if err != nil {
			vp.logger.Error("Failed to check if video is processed", err, "videoID", video.ID)
			continue
		}

		if processed {
			vp.logger.Debug("Video already processed, skipping", "videoID", video.ID)
			continue
		}

		// Process the video
		if err := vp.processVideo(ctx, video); err != nil {
			vp.logger.Error("Failed to process video", err, "videoID", video.ID, "title", video.Title)
			continue
		}

		processedCount++
	}

	vp.logger.Info("Completed channel processing",
		"channelID", channel.ID,
		"channelName", channel.Name,
		"totalVideos", len(videos),
		"processedVideos", processedCount)

	return nil
}

// getTranscriptAndThumbnail gets transcript and best thumbnail URL from the API
func (vp *VideoProcessor) getTranscriptAndThumbnail(ctx context.Context, videoID string) (string, string, error) {
	// Use the new method that returns both transcript and thumbnail
	data, err := vp.transcriptClient.GetTranscriptWithThumbnail(ctx, videoID)
	if err != nil {
		return "", "", err
	}

	return data.Transcript, data.ThumbnailURL, nil
}

// processVideo processes a single video (transcript + summary)
func (vp *VideoProcessor) processVideo(ctx context.Context, video types.Video) error {
	vp.logger.Debug("Processing video", "videoID", video.ID, "title", video.Title)

	// Create a timeout context for this video
	videoCtx, cancel := context.WithTimeout(ctx, vp.config.Processing.TranscriptTimeout)
	defer cancel()

	// Get the transcript, with fallback to video description
	transcript, thumbnailURL, err := vp.getTranscriptAndThumbnail(videoCtx, video.ID)
	if err != nil {
		vp.logger.Warn("Transcript failed, using video description as fallback", "videoID", video.ID, "error", err)
		// Use video title and description as fallback
		transcript = fmt.Sprintf("Video Title: %s\n\nVideo Description: %s", video.Title, video.Description)
		if len(transcript) < 50 { // Very short description
			transcript = fmt.Sprintf("Video Title: %s\n\nThis video discusses topics related to the title. Please watch the video for detailed content.", video.Title)
		}
		// Use default YouTube thumbnail as fallback
		thumbnailURL = fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", video.ID)
	}

	// Truncate transcript if it's too long
	if len(transcript) > vp.config.AI.MaxTranscriptLength {
		transcript = transcript[:vp.config.AI.MaxTranscriptLength] + "... [truncated]"
		vp.logger.Debug("Truncated long transcript", "videoID", video.ID, "maxLength", vp.config.AI.MaxTranscriptLength)
	}

	// Generate summary using AI
	summary, err := vp.aiClient.Summarize(ctx, transcript, video.Title)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Create summary record
	summaryRecord := types.Summary{
		ID:           vp.generateSummaryID(),
		VideoID:      video.ID,
		VideoTitle:   video.Title,
		ChannelName:  video.ChannelName,
		Summary:      summary,
		CreatedAt:    time.Now(),
		Status:       "New",
		VideoURL:     video.URL,
		PublishedAt:  video.PublishedAt,
		ThumbnailURL: thumbnailURL,
		Duration:     video.Duration,
		ViewCount:    video.ViewCount,
	}

	// Save the summary
	if err := vp.storage.SaveSummary(ctx, summaryRecord); err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	// Mark video as processed
	if err := vp.storage.MarkVideoProcessed(ctx, video.ID); err != nil {
		return fmt.Errorf("failed to mark video as processed: %w", err)
	}

	vp.logger.Info("Successfully processed video",
		"videoID", video.ID,
		"title", video.Title,
		"summaryLength", len(summary))

	return nil
}

// GetProcessedVideos retrieves all processed videos
func (vp *VideoProcessor) GetProcessedVideos(ctx context.Context) ([]types.Video, error) {
	// This would require additional storage methods to track processed videos with full details
	// For now, we'll return an empty slice as the storage interface focuses on summaries
	vp.logger.Debug("GetProcessedVideos called - feature not fully implemented")
	return []types.Video{}, nil
}

// UpdateConfig updates the processor configuration
func (vp *VideoProcessor) UpdateConfig(config types.Config) error {
	vp.config = &config
	vp.logger.Info("Updated processor configuration")
	return nil
}

// generateSummaryID generates a unique ID for a summary
func (vp *VideoProcessor) generateSummaryID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("sum_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("sum_%s", hex.EncodeToString(bytes))
}

// GetSummaryStats returns basic statistics about processed summaries
func (vp *VideoProcessor) GetSummaryStats(ctx context.Context) (map[string]interface{}, error) {
	pendingSummaries, err := vp.storage.GetPendingSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending summaries: %w", err)
	}

	stats := map[string]interface{}{
		"pending_summaries": len(pendingSummaries),
		"last_check":        time.Now().Format("2006-01-02 15:04:05"),
	}

	return stats, nil
}

// ProcessPendingSummariesForEmail processes summaries that are ready to be sent via email
func (vp *VideoProcessor) ProcessPendingSummariesForEmail(ctx context.Context) ([]types.Summary, error) {
	summaries, err := vp.storage.GetPendingSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending summaries: %w", err)
	}

	vp.logger.Info("Retrieved pending summaries for email", "count", len(summaries))
	return summaries, nil
}
