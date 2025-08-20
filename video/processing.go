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

// processVideoFileCore handles the core logic of processing a video file without side effects
func processVideoFileCore(videoFile string) *ProcessingResult {
	result := &ProcessingResult{
		OriginalPath: videoFile,
	}

	// Validate the file
	validationResult, err := validateVideoFile(videoFile)
	if err != nil {
		result.Error = err
		return result
	}

	// Directory, skip
	if validationResult.IsDirectory {
		result.WasSkipped = true
		result.SkipReason = "is a directory"
		return result
	}

	// Not a video file, skip
	if !validationResult.IsVideoFile {
		result.WasSkipped = true
		result.SkipReason = "is not a video file"
		return result
	}

	// Already processed, skip
	if validationResult.IsProcessed {
		result.WasSkipped = true
		result.SkipReason = "already processed"
		return result
	}

	// Extract video metadata
	metadata, err := extractVideoMetadata(videoFile)
	if err != nil {
		result.Error = err
		return result
	}
	result.Metadata = metadata

	// Calculate file hash without progress tracking for pure function
	crc, err := calculateFileHash(videoFile, nil)
	if err != nil {
		result.Error = err
		return result
	}
	result.CRC32 = crc

	// Generate new filename
	newFilename := generateTaggedFilename(videoFile, metadata, crc)
	result.NewPath = newFilename

	// Attempt to rename the file
	if err := renameVideoFile(videoFile, newFilename); err != nil {
		result.Error = err
		return result
	}

	result.WasRenamed = true
	return result
}

// ProcessVideoFile handles the processing of a single video file with console output
func ProcessVideoFile(videoFile string) {
	result := processVideoFileCore(videoFile)

	// Handle the result with appropriate console output
	if result.Error != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error processing %s: %v", videoFile, result.Error)))
		return
	}

	if result.WasSkipped {
		switch result.SkipReason {
		case "is a directory":
			fmt.Printf("%s is a directory.\n", videoFile)
		case "is not a video file":
			fmt.Printf("%s is not a video file, skipping\n", videoFile)
		case "already processed":
			// Silently skip already processed files
		}
		return
	}

	// For successful processing, show progress and results
	fileInfo, _ := os.Stat(videoFile)
	fileSize := fileInfo.Size()

	// Create a custom progress bar with lipgloss styling
	prog := progress.New(progress.WithDefaultGradient())
	fmt.Printf("%s\n", processingStyle.Render(fmt.Sprintf("📊 Processing: %s", videoFile)))

	// Create a progress writer for visual feedback
	progressWriter := &progressWriter{
		total: fileSize,
		prog:  prog,
		done:  make(chan bool),
	}
	go progressWriter.render()

	// Recalculate hash with progress tracking for UI
	_, _ = calculateFileHash(videoFile, progressWriter)
	progressWriter.done <- true

	if result.WasRenamed {
		fmt.Printf("%s\n", successStyle.Render(fmt.Sprintf("✅ %s", filepath.Base(result.NewPath))))
	}
}
