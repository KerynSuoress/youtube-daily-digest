package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"youtube-summarizer/pkg/types"
)

// ClaudeClient implements the types.AIClient interface using Claude API
type ClaudeClient struct {
	httpClient *HTTPClient
	apiKey     string
	baseURL    string
	model      string
	logger     types.Logger
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey string, logger types.Logger) *ClaudeClient {
	return &ClaudeClient{
		httpClient: NewHTTPClient(60 * time.Second), // Longer timeout for AI requests
		apiKey:     apiKey,
		baseURL:    "https://api.anthropic.com/v1",
		model:      "claude-sonnet-4-20250514", // Latest Claude model from official docs
		logger:     logger,
	}
}

// ClaudeRequest represents the request structure for Claude API
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	Content []ClaudeContent `json:"content"`
	Model   string          `json:"model"`
	Usage   ClaudeUsage     `json:"usage"`
}

// ClaudeContent represents content in the response
type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeUsage represents token usage information
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeError represents an error response from Claude API
type ClaudeError struct {
	Error ClaudeErrorDetail `json:"error"`
}

// ClaudeErrorDetail represents the error details
type ClaudeErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Summarize generates a summary of the video transcript using Claude
func (cc *ClaudeClient) Summarize(ctx context.Context, transcript, title string) (string, error) {
	// Truncate transcript if it's too long
	maxLength := 50000 // Conservative limit for Claude input
	if len(transcript) > maxLength {
		transcript = transcript[:maxLength] + "... [transcript truncated]"
		cc.logger.Debug("Truncated long transcript", "originalLength", len(transcript), "maxLength", maxLength)
	}

	// Create the prompt
	prompt := fmt.Sprintf(`Video Title: "%s"

Summarize the key takeaways from the following video transcript into a concise paragraph. Focus on the main points and actionable advice:

%s`, title, transcript)

	// Prepare the request
	request := ClaudeRequest{
		Model:     cc.model,
		MaxTokens: 1000, // Reasonable limit for summary
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Claude request: %w", err)
	}

	cc.logger.Debug("Sending request to Claude API", "videoTitle", title, "transcriptLength", len(transcript))

	// Make the API request
	req, err := http.NewRequestWithContext(ctx, "POST", cc.baseURL+"/messages", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create Claude API request: %w", err)
	}

	// Set headers according to official Anthropic API docs
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cc.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := cc.httpClient.DoWithContext(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to call Claude API: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		var claudeError ClaudeError
		if err := json.NewDecoder(resp.Body).Decode(&claudeError); err == nil {
			return "", fmt.Errorf("Claude API error (%d): %s", resp.StatusCode, claudeError.Error.Message)
		}
		return "", fmt.Errorf("Claude API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var claudeResponse ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResponse); err != nil {
		return "", fmt.Errorf("failed to decode Claude API response: %w", err)
	}

	// Extract the summary from the response
	if len(claudeResponse.Content) == 0 {
		return "", fmt.Errorf("Claude API returned empty content")
	}

	summary := strings.TrimSpace(claudeResponse.Content[0].Text)
	if summary == "" {
		return "", fmt.Errorf("Claude API returned empty summary")
	}

	cc.logger.Info("Generated summary using Claude",
		"videoTitle", title,
		"inputTokens", claudeResponse.Usage.InputTokens,
		"outputTokens", claudeResponse.Usage.OutputTokens,
		"summaryLength", len(summary))

	return summary, nil
}

// SetModel allows changing the Claude model used for summarization
func (cc *ClaudeClient) SetModel(model string) {
	cc.model = model
	cc.logger.Debug("Changed Claude model", "model", model)
}

// GetModel returns the current Claude model being used
func (cc *ClaudeClient) GetModel() string {
	return cc.model
}
