package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"youtube-summarizer/pkg/types"
)

// TranscriptClient implements the types.TranscriptClient interface
type TranscriptClient struct {
	httpClient  *HTTPClient
	rapidAPIKey string
	baseURL     string
	logger      types.Logger
}

// NewTranscriptClient creates a new transcript client using RapidAPI
func NewTranscriptClient(rapidAPIKey string, logger types.Logger) *TranscriptClient {
	return &TranscriptClient{
		httpClient:  NewHTTPClient(45 * time.Second), // Longer timeout for transcript fetching
		rapidAPIKey: rapidAPIKey,
		baseURL:     "https://youtube-transcriptor.p.rapidapi.com",
		logger:      logger,
	}
}

// TranscriptResponse represents the actual API response format
type TranscriptResponse struct {
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	AvailableLangs  []string          `json:"availableLangs"`
	LengthInSeconds string            `json:"lengthInSeconds"`
	Thumbnails      []Thumbnail       `json:"thumbnails"`
	Transcription   []TranscriptEntry `json:"transcription"`
}

// Thumbnail represents video thumbnail info
type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// TranscriptEntry represents a single transcript entry
type TranscriptEntry struct {
	Subtitle string  `json:"subtitle"`
	Start    float64 `json:"start"`
	Dur      float64 `json:"dur"`
}

// Alternative transcript client for fallback
type AlternativeTranscriptClient struct {
	httpClient *HTTPClient
	logger     types.Logger
}

// NewAlternativeTranscriptClient creates a fallback transcript client
func NewAlternativeTranscriptClient(logger types.Logger) *AlternativeTranscriptClient {
	return &AlternativeTranscriptClient{
		httpClient: NewHTTPClient(30 * time.Second),
		logger:     logger,
	}
}

// GetTranscript fetches the transcript for a YouTube video
func (tc *TranscriptClient) GetTranscript(ctx context.Context, videoID string) (string, error) {
	data, err := tc.GetTranscriptWithThumbnail(ctx, videoID)
	if err != nil {
		return "", err
	}
	return data.Transcript, nil
}

// GetTranscriptWithThumbnail fetches both transcript and thumbnail for a YouTube video
func (tc *TranscriptClient) GetTranscriptWithThumbnail(ctx context.Context, videoID string) (*types.TranscriptData, error) {
	// First try RapidAPI
	data, err := tc.getRapidAPITranscriptWithThumbnail(ctx, videoID)
	if err != nil {
		tc.logger.Warn("RapidAPI transcript failed, trying alternative", "videoID", videoID, "error", err)

		// Fallback to alternative method
		altClient := NewAlternativeTranscriptClient(tc.logger)
		return altClient.getAlternativeTranscriptWithThumbnail(ctx, videoID)
	}

	return data, nil
}

// getRapidAPITranscriptWithThumbnail uses RapidAPI to fetch transcript and thumbnail
func (tc *TranscriptClient) getRapidAPITranscriptWithThumbnail(ctx context.Context, videoID string) (*types.TranscriptData, error) {
	// Build the URL exactly like the RapidAPI example
	url := fmt.Sprintf("https://youtube-transcriptor.p.rapidapi.com/transcript?video_id=%s&lang=en", videoID)

	tc.logger.Debug("Fetching transcript from RapidAPI", "videoID", videoID)

	// Create request exactly like the RapidAPI example
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcript request: %w", err)
	}

	// Set headers exactly like the RapidAPI example
	req.Header.Add("x-rapidapi-key", tc.rapidAPIKey)
	req.Header.Add("x-rapidapi-host", "youtube-transcriptor.p.rapidapi.com")
	req.Header.Add("Accept", "application/json")

	// Make the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transcript: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transcript API returned status %d", res.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Log the raw response
	tc.logger.Debug("Raw API response", "videoID", videoID, "body", string(body))

	// Parse the JSON response - it's an array with one object containing transcription
	var responseArray []TranscriptResponse
	if err := json.Unmarshal(body, &responseArray); err != nil {
		// Log the error with the response body for debugging
		tc.logger.Error("Failed to parse JSON response", err, "videoID", videoID, "responseBody", string(body))
		return nil, fmt.Errorf("failed to decode transcript response: %w", err)
	}

	if len(responseArray) == 0 {
		return nil, fmt.Errorf("empty response array for video %s", videoID)
	}

	// Get the transcript entries from the transcription field
	transcriptEntries := responseArray[0].Transcription

	tc.logger.Debug("Extracted transcript entries", "videoID", videoID, "entryCount", len(transcriptEntries))

	// Combine all transcript entries
	var transcriptText strings.Builder
	for _, entry := range transcriptEntries {
		if transcriptText.Len() > 0 {
			transcriptText.WriteString(" ")
		}
		transcriptText.WriteString(strings.TrimSpace(entry.Subtitle))
	}

	transcript := transcriptText.String()
	if transcript == "" {
		return nil, fmt.Errorf("empty transcript received for video %s", videoID)
	}

	// Use reliable YouTube thumbnail URLs that work in email clients
	// These are simple, direct URLs without query parameters that email clients handle better
	thumbnailURL := fmt.Sprintf("https://img.youtube.com/vi/%s/hqdefault.jpg", videoID)

	tc.logger.Debug("Using standard YouTube thumbnail", "videoID", videoID, "thumbnailURL", thumbnailURL)

	// Alternative: if we want to try API thumbnails, prefer simple JPG URLs without query params
	if len(responseArray[0].Thumbnails) > 0 {
		for _, thumb := range responseArray[0].Thumbnails {
			// Prefer JPG URLs without complex query parameters for email compatibility
			if strings.Contains(thumb.URL, ".jpg") && !strings.Contains(thumb.URL, "?") {
				thumbnailURL = thumb.URL
				tc.logger.Debug("Using simple API thumbnail", "videoID", videoID, "thumbnailURL", thumbnailURL)
				break
			}
		}
	}

	tc.logger.Info("Retrieved transcript from RapidAPI",
		"videoID", videoID,
		"length", len(transcript),
		"segments", len(transcriptEntries),
		"thumbnailURL", thumbnailURL)

	return &types.TranscriptData{
		Transcript:   transcript,
		ThumbnailURL: thumbnailURL,
	}, nil
}

// getAlternativeTranscriptWithThumbnail uses a fallback method to get transcripts
func (atc *AlternativeTranscriptClient) getAlternativeTranscriptWithThumbnail(ctx context.Context, videoID string) (*types.TranscriptData, error) {
	// This is a placeholder for alternative transcript fetching methods
	// In a real implementation, you might use:
	// 1. YouTube's official captions API (if available)
	// 2. Another third-party service
	// 3. A local transcript extraction tool

	atc.logger.Warn("Alternative transcript method not implemented", "videoID", videoID)
	return nil, fmt.Errorf("alternative transcript method not available for video %s", videoID)
}

// YouTube Direct Caption API (placeholder for future implementation)
func (atc *AlternativeTranscriptClient) getYouTubeCaptions(ctx context.Context, videoID string) (string, error) {
	// This would use YouTube's caption API if we had access
	// For now, return an error
	return "", fmt.Errorf("YouTube direct caption API not implemented")
}

// MockTranscriptClient for testing purposes
type MockTranscriptClient struct {
	logger types.Logger
}

// NewMockTranscriptClient creates a mock transcript client for testing
func NewMockTranscriptClient(logger types.Logger) *MockTranscriptClient {
	return &MockTranscriptClient{logger: logger}
}

// GetTranscript returns a mock transcript for testing
func (mtc *MockTranscriptClient) GetTranscript(ctx context.Context, videoID string) (string, error) {
	data, err := mtc.GetTranscriptWithThumbnail(ctx, videoID)
	if err != nil {
		return "", err
	}
	return data.Transcript, nil
}

// GetTranscriptWithThumbnail returns mock transcript and thumbnail for testing
func (mtc *MockTranscriptClient) GetTranscriptWithThumbnail(ctx context.Context, videoID string) (*types.TranscriptData, error) {
	mtc.logger.Debug("Using mock transcript with thumbnail", "videoID", videoID)

	transcript := fmt.Sprintf("This is a mock transcript for video %s. "+
		"In this video, we discuss various topics including technology, innovation, and best practices. "+
		"The main points covered are: 1) Understanding the fundamentals, 2) Implementing solutions effectively, "+
		"3) Best practices for optimization, and 4) Future considerations. "+
		"This mock transcript is used for testing purposes and demonstrates how the summarization system works "+
		"with actual transcript content.", videoID)

	// Generate mock thumbnail URL
	thumbnailURL := fmt.Sprintf("https://img.youtube.com/vi/%s/maxresdefault.jpg", videoID)

	return &types.TranscriptData{
		Transcript:   transcript,
		ThumbnailURL: thumbnailURL,
	}, nil
}
