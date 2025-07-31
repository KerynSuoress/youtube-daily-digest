package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"youtube-summarizer/pkg/types"
)

// YouTubeClient implements the types.YouTubeClient interface
type YouTubeClient struct {
	httpClient *HTTPClient
	apiKey     string
	baseURL    string
	logger     types.Logger
}

// NewYouTubeClient creates a new YouTube API client
func NewYouTubeClient(apiKey string, logger types.Logger) *YouTubeClient {
	return &YouTubeClient{
		httpClient: NewHTTPClient(30 * time.Second),
		apiKey:     apiKey,
		baseURL:    "https://www.googleapis.com/youtube/v3",
		logger:     logger,
	}
}

// YouTubeAPIResponse represents the API response structure
type YouTubeAPIResponse struct {
	Items []YouTubeVideoItem `json:"items"`
}

// YouTubeVideoItem represents a video item from the API
type YouTubeVideoItem struct {
	ID             YouTubeVideoID         `json:"id"`
	Snippet        YouTubeVideoSnippet    `json:"snippet"`
	Statistics     YouTubeVideoStatistics `json:"statistics,omitempty"`
	ContentDetails YouTubeContentDetails  `json:"contentDetails,omitempty"`
}

// YouTubeVideoID represents video ID structure
type YouTubeVideoID struct {
	VideoID string `json:"videoId,omitempty"`
	Kind    string `json:"kind"`
}

// YouTubeVideoSnippet represents video snippet information
type YouTubeVideoSnippet struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ChannelID    string    `json:"channelId"`
	ChannelTitle string    `json:"channelTitle"`
	PublishedAt  time.Time `json:"publishedAt"`
}

// YouTubeVideoStatistics represents video statistics
type YouTubeVideoStatistics struct {
	ViewCount string `json:"viewCount"`
}

// YouTubeContentDetails represents video content details
type YouTubeContentDetails struct {
	Duration string `json:"duration"`
}

// GetChannelVideos retrieves recent videos from a YouTube channel
func (yc *YouTubeClient) GetChannelVideos(ctx context.Context, channelID string, maxResults int) ([]types.Video, error) {
	// Build the API URL
	apiURL := fmt.Sprintf("%s/search", yc.baseURL)
	params := url.Values{}
	params.Add("key", yc.apiKey)
	params.Add("channelId", channelID)
	params.Add("part", "snippet")
	params.Add("order", "date")
	params.Add("type", "video")
	params.Add("maxResults", strconv.Itoa(maxResults))

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	yc.logger.Debug("Fetching channel videos", "channelID", channelID, "maxResults", maxResults)

	// Make the API request
	resp, err := yc.httpClient.Get(ctx, fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel videos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var apiResponse YouTubeAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode YouTube API response: %w", err)
	}

	// Convert to our video format
	var videos []types.Video
	for _, item := range apiResponse.Items {
		videoID := item.ID.VideoID
		if videoID == "" {
			continue
		}

		video := types.Video{
			ID:          videoID,
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
			ChannelID:   item.Snippet.ChannelID,
			ChannelName: item.Snippet.ChannelTitle,
			PublishedAt: item.Snippet.PublishedAt,
			URL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		}

		videos = append(videos, video)
	}

	yc.logger.Info("Retrieved channel videos", "channelID", channelID, "count", len(videos))
	return videos, nil
}

// GetVideoDetails retrieves detailed information about a specific video
func (yc *YouTubeClient) GetVideoDetails(ctx context.Context, videoID string) (*types.Video, error) {
	// Build the API URL
	apiURL := fmt.Sprintf("%s/videos", yc.baseURL)
	params := url.Values{}
	params.Add("key", yc.apiKey)
	params.Add("id", videoID)
	params.Add("part", "snippet,statistics,contentDetails")

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	yc.logger.Debug("Fetching video details", "videoID", videoID)

	// Make the API request
	resp, err := yc.httpClient.Get(ctx, fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch video details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var apiResponse YouTubeAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode YouTube API response: %w", err)
	}

	if len(apiResponse.Items) == 0 {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	item := apiResponse.Items[0]

	// Parse view count
	var viewCount int64
	if item.Statistics.ViewCount != "" {
		if count, err := strconv.ParseInt(item.Statistics.ViewCount, 10, 64); err == nil {
			viewCount = count
		}
	}

	video := &types.Video{
		ID:          videoID,
		Title:       item.Snippet.Title,
		Description: item.Snippet.Description,
		ChannelID:   item.Snippet.ChannelID,
		ChannelName: item.Snippet.ChannelTitle,
		PublishedAt: item.Snippet.PublishedAt,
		Duration:    item.ContentDetails.Duration,
		ViewCount:   viewCount,
		URL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
	}

	yc.logger.Debug("Retrieved video details", "videoID", videoID, "title", video.Title)
	return video, nil
}
