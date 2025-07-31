package storage

import (
	"context"
	"fmt"
	"time"

	"youtube-summarizer/pkg/types"

	"github.com/xuri/excelize/v2"
)

// ExcelStorage implements the types.Storage interface using Excel files
type ExcelStorage struct {
	filePath string
	logger   types.Logger
}

// NewExcelStorage creates a new Excel storage instance
func NewExcelStorage(filePath string, logger types.Logger) *ExcelStorage {
	return &ExcelStorage{
		filePath: filePath,
		logger:   logger,
	}
}

// Initialize creates the Excel file with proper structure if it doesn't exist
func (es *ExcelStorage) Initialize() error {
	// Try to open existing file
	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		// File doesn't exist, create new one
		es.logger.Info("Creating new Excel file", "path", es.filePath)
		file = excelize.NewFile()
	}
	defer file.Close()

	// Ensure all required sheets exist with headers
	if err := es.ensureSheet(file, ChannelsSheet, ChannelHeaders()); err != nil {
		return fmt.Errorf("failed to ensure channels sheet: %w", err)
	}

	if err := es.ensureSheet(file, ProcessedVideosSheet, ProcessedVideoHeaders()); err != nil {
		return fmt.Errorf("failed to ensure processed videos sheet: %w", err)
	}

	if err := es.ensureSheet(file, SummariesSheet, SummaryHeaders()); err != nil {
		return fmt.Errorf("failed to ensure summaries sheet: %w", err)
	}

	// Delete the default "Sheet1" if it exists and is empty
	if sheetList := file.GetSheetList(); len(sheetList) > 3 {
		for _, sheetName := range sheetList {
			if sheetName == "Sheet1" {
				file.DeleteSheet(sheetName)
				break
			}
		}
	}

	if err := file.SaveAs(es.filePath); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	es.logger.Info("Excel storage initialized successfully", "path", es.filePath)
	return nil
}

// ensureSheet creates a sheet with headers if it doesn't exist
func (es *ExcelStorage) ensureSheet(file *excelize.File, sheetName string, headers []string) error {
	// Check if sheet exists
	sheetList := file.GetSheetList()
	exists := false
	for _, name := range sheetList {
		if name == sheetName {
			exists = true
			break
		}
	}

	// Create sheet if it doesn't exist
	if !exists {
		if _, err := file.NewSheet(sheetName); err != nil {
			return fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
		}
	}

	// Add headers if the sheet is empty
	cellValue, err := file.GetCellValue(sheetName, "A1")
	if err != nil || cellValue == "" {
		for i, header := range headers {
			cell := fmt.Sprintf("%c1", 'A'+i)
			if err := file.SetCellValue(sheetName, cell, header); err != nil {
				return fmt.Errorf("failed to set header %s: %w", header, err)
			}
		}
	}

	return nil
}

// GetChannels retrieves all channels from Excel
func (es *ExcelStorage) GetChannels(ctx context.Context) ([]types.Channel, error) {
	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	rows, err := file.GetRows(ChannelsSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from channels sheet: %w", err)
	}

	var channels []types.Channel
	// Skip header row (index 0)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 { // At least ID and Name required
			continue
		}

		channel := types.Channel{
			ID:   row[0],
			Name: row[1],
		}
		if len(row) > 2 {
			channel.Username = row[2]
		}

		channels = append(channels, channel)
	}

	es.logger.Debug("Retrieved channels from Excel", "count", len(channels))
	return channels, nil
}

// SaveSummary saves a summary to Excel
func (es *ExcelStorage) SaveSummary(ctx context.Context, summary types.Summary) error {
	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		return fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer func() {
		if saveErr := file.SaveAs(es.filePath); saveErr != nil {
			es.logger.Error("Failed to save Excel file", saveErr)
		}
		file.Close()
	}()

	// Find the next empty row
	rows, err := file.GetRows(SummariesSheet)
	if err != nil {
		return fmt.Errorf("failed to get rows from summaries sheet: %w", err)
	}

	nextRow := len(rows) + 1
	excelSummary := FromSummary(summary)

	// Write summary data
	data := []interface{}{
		excelSummary.ID,
		excelSummary.VideoID,
		excelSummary.VideoTitle,
		excelSummary.ChannelName,
		excelSummary.Summary,
		excelSummary.CreatedAt,
		excelSummary.Status,
		excelSummary.VideoURL,
	}

	for i, value := range data {
		cell := fmt.Sprintf("%c%d", 'A'+i, nextRow)
		if err := file.SetCellValue(SummariesSheet, cell, value); err != nil {
			return fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	es.logger.Debug("Saved summary to Excel", "summaryID", summary.ID, "videoID", summary.VideoID)
	return nil
}

// GetPendingSummaries retrieves summaries with "New" status
func (es *ExcelStorage) GetPendingSummaries(ctx context.Context) ([]types.Summary, error) {
	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	rows, err := file.GetRows(SummariesSheet)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from summaries sheet: %w", err)
	}

	var summaries []types.Summary
	// Skip header row (index 0)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 7 { // Minimum required columns
			continue
		}

		// Check if status is "New"
		status := ""
		if len(row) > 6 {
			status = row[6]
		}
		if status != "New" {
			continue
		}

		excelSummary := ExcelSummary{
			ID:          row[0],
			VideoID:     row[1],
			VideoTitle:  row[2],
			ChannelName: row[3],
			Summary:     row[4],
			CreatedAt:   row[5],
			Status:      status,
		}

		// Read additional columns (VideoURL, PublishedAt, ThumbnailURL, Duration, ViewCount)
		if len(row) > 7 {
			excelSummary.VideoURL = row[7]
		}
		if len(row) > 8 {
			excelSummary.PublishedAt = row[8]
		}
		if len(row) > 9 {
			excelSummary.ThumbnailURL = row[9]
		}
		if len(row) > 10 {
			excelSummary.Duration = row[10]
		}
		if len(row) > 11 {
			excelSummary.ViewCount = row[11]
		}

		summary, err := excelSummary.ToSummary()
		if err != nil {
			es.logger.Warn("Failed to parse summary date", "error", err, "summaryID", excelSummary.ID)
			continue
		}

		summaries = append(summaries, summary)
	}

	es.logger.Debug("Retrieved pending summaries", "count", len(summaries))
	return summaries, nil
}

// MarkSummariesProcessed updates the status of summaries to "Processed"
func (es *ExcelStorage) MarkSummariesProcessed(ctx context.Context, summaryIDs []string) error {
	if len(summaryIDs) == 0 {
		return nil
	}

	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		return fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer func() {
		if saveErr := file.SaveAs(es.filePath); saveErr != nil {
			es.logger.Error("Failed to save Excel file", saveErr)
		}
		file.Close()
	}()

	rows, err := file.GetRows(SummariesSheet)
	if err != nil {
		return fmt.Errorf("failed to get rows from summaries sheet: %w", err)
	}

	// Create a map for faster lookup
	idMap := make(map[string]bool)
	for _, id := range summaryIDs {
		idMap[id] = true
	}

	updatedCount := 0
	// Skip header row (index 0)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 1 {
			continue
		}

		summaryID := row[0]
		if idMap[summaryID] {
			// Update status to "Processed"
			statusCell := fmt.Sprintf("G%d", i+1) // Column G is status (0-based index 6)
			if err := file.SetCellValue(SummariesSheet, statusCell, "Processed"); err != nil {
				es.logger.Error("Failed to update summary status", err, "summaryID", summaryID)
				continue
			}
			updatedCount++
		}
	}

	es.logger.Debug("Marked summaries as processed", "count", updatedCount)
	return nil
}

// IsVideoProcessed checks if a video has already been processed
func (es *ExcelStorage) IsVideoProcessed(ctx context.Context, videoID string) (bool, error) {
	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer file.Close()

	rows, err := file.GetRows(ProcessedVideosSheet)
	if err != nil {
		return false, fmt.Errorf("failed to get rows from processed videos sheet: %w", err)
	}

	// Skip header row (index 0)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 1 {
			continue
		}
		if row[0] == videoID {
			return true, nil
		}
	}

	return false, nil
}

// MarkVideoProcessed adds a video to the processed videos list
func (es *ExcelStorage) MarkVideoProcessed(ctx context.Context, videoID string) error {
	// First check if already processed
	processed, err := es.IsVideoProcessed(ctx, videoID)
	if err != nil {
		return err
	}
	if processed {
		return nil // Already processed
	}

	file, err := excelize.OpenFile(es.filePath)
	if err != nil {
		return fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer func() {
		if saveErr := file.SaveAs(es.filePath); saveErr != nil {
			es.logger.Error("Failed to save Excel file", saveErr)
		}
		file.Close()
	}()

	// Find the next empty row
	rows, err := file.GetRows(ProcessedVideosSheet)
	if err != nil {
		return fmt.Errorf("failed to get rows from processed videos sheet: %w", err)
	}

	nextRow := len(rows) + 1

	// Write processed video data
	data := []interface{}{
		videoID,
		"", // ChannelID - will be populated when we have video details
		"", // Title - will be populated when we have video details
		time.Now().Format("2006-01-02 15:04:05"),
	}

	for i, value := range data {
		cell := fmt.Sprintf("%c%d", 'A'+i, nextRow)
		if err := file.SetCellValue(ProcessedVideosSheet, cell, value); err != nil {
			return fmt.Errorf("failed to set cell %s: %w", cell, err)
		}
	}

	es.logger.Debug("Marked video as processed", "videoID", videoID)
	return nil
}
