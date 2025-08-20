package video

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/progress"
)

// validateVideoFile performs all file validation checks and returns structured results
func validateVideoFile(videoFile string) (*FileValidationResult, error) {
	fi, err := os.Stat(videoFile)
	if err != nil {
		return nil, err
	}

	result := &FileValidationResult{
		FileInfo:    fi,
		IsDirectory: fi.IsDir(),
		IsVideoFile: IsVideoFile(videoFile),
		IsProcessed: IsProcessed(videoFile),
	}

	return result, nil
}

// extractVideoMetadata extracts resolution and duration from a video file
func extractVideoMetadata(videoFile string) (*VideoMetadata, error) {
	resolution, err := GetVideoResolution(videoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get resolution: %w", err)
	}

	durationMins, err := GetVideoDuration(videoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get duration: %w", err)
	}

	return &VideoMetadata{
		Resolution:   resolution,
		DurationMins: durationMins,
	}, nil
}

// calculateFileHash calculates the CRC32 hash of a file with optional progress tracking
func calculateFileHash(videoFile string, progressWriter io.Writer) (uint32, error) {
	f, err := os.Open(videoFile)
	if err != nil {
		return 0, fmt.Errorf("failed to open file for hash calculation: %w", err)
	}
	defer f.Close()

	h := crc32.NewIEEE()
	var writers []io.Writer
	writers = append(writers, h)
	if progressWriter != nil {
		writers = append(writers, progressWriter)
	}

	if _, err := io.Copy(io.MultiWriter(writers...), f); err != nil {
		return 0, fmt.Errorf("failed to calculate hash: %w", err)
	}

	return h.Sum32(), nil
}

// generateTaggedFilename creates the new filename with metadata tags
func generateTaggedFilename(originalPath string, metadata *VideoMetadata, crc uint32) string {
	ext := filepath.Ext(originalPath)
	baseName := originalPath[0 : len(originalPath)-len(ext)]
	return fmt.Sprintf("%s_[%s][%.0fmin][%08X]%s", baseName, metadata.Resolution, metadata.DurationMins, crc, ext)
}

// renameVideoFile performs the actual file rename operation
func renameVideoFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// ProcessVideoFile handles the processing of a single video file
func ProcessVideoFile(videoFile string) {
	// Validate the file
	validationResult, err := validateVideoFile(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error processing %s: %v", videoFile, err)))
		return
	}

	// Directory, skip
	if validationResult.IsDirectory {
		fmt.Printf("%s is a directory.\n", videoFile)
		return
	}

	// Not a video file, skip
	if !validationResult.IsVideoFile {
		fmt.Printf("%s is not a video file, skipping\n", videoFile)
		return
	}

	// Already processed, skip
	if validationResult.IsProcessed {
		return
	}

	fileSize := validationResult.FileInfo.Size()

	// Create a custom progress bar with lipgloss styling
	prog := progress.New(progress.WithDefaultGradient())
	fmt.Printf("%s\n", processingStyle.Render(fmt.Sprintf("📊 Processing: %s", videoFile)))

	// Create a progress writer
	progressWriter := &progressWriter{
		total: fileSize,
		prog:  prog,
		done:  make(chan bool),
	}
	go progressWriter.render()

	// Extract video metadata
	metadata, err := extractVideoMetadata(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error: %v", err)))
		return
	}

	// Calculate file hash with progress tracking
	crc, err := calculateFileHash(videoFile, progressWriter)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error: %v", err)))
		return
	}

	// Generate new filename
	newFilename := generateTaggedFilename(videoFile, metadata, crc)

	progressWriter.done <- true

	// Rename the file
	if err := renameVideoFile(videoFile, newFilename); err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error renaming file: %v", err)))
	} else {
		fmt.Printf("%s\n", successStyle.Render(fmt.Sprintf("✅ %s", filepath.Base(newFilename))))
	}
}
