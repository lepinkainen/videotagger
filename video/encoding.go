package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReencodeOptions holds configuration for video re-encoding
type ReencodeOptions struct {
	CRF          int     // Constant Rate Factor (0-51, 23 is default)
	Preset       string  // x265 preset (ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo)
	MinSavings   float64 // Minimum size reduction percentage required (0.0-1.0)
	KeepOriginal bool    // Whether to keep original file as .bak
}

// DefaultReencodeOptions returns sensible defaults for H.265 encoding
func DefaultReencodeOptions() *ReencodeOptions {
	return &ReencodeOptions{
		CRF:          23,       // Good quality/size balance
		Preset:       "medium", // Good speed/compression balance
		MinSavings:   0.05,     // Require at least 5% savings
		KeepOriginal: false,    // Don't keep originals by default
	}
}

// ReencodeResult holds the results of a re-encoding operation
type ReencodeResult struct {
	OriginalPath   string
	OriginalCodec  string
	OriginalSize   int64
	NewPath        string
	NewSize        int64
	SizeSavings    int64
	SavingsPercent float64
	WasReencoded   bool
	WasSkipped     bool
	SkipReason     string
	Error          error
}

// IsH265 checks if a video file is already encoded with H.265/HEVC
func IsH265(videoFile string) (bool, error) {
	codec, err := GetVideoCodec(videoFile)
	if err != nil {
		return false, err
	}

	// Check for H.265/HEVC codec names
	codec = strings.ToLower(codec)
	return codec == "hevc" || codec == "h265", nil
}

// ReencodeToH265 re-encodes a video file to H.265 with size comparison
func ReencodeToH265(videoFile string, options *ReencodeOptions) *ReencodeResult {
	result := &ReencodeResult{
		OriginalPath: videoFile,
	}

	// Validate input file
	if !IsVideoFile(videoFile) {
		result.WasSkipped = true
		result.SkipReason = "not a video file"
		return result
	}

	// Get original file info
	originalSize, err := GetFileSize(videoFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get original file size: %w", err)
		return result
	}
	result.OriginalSize = originalSize

	// Check if already H.265
	isH265, err := IsH265(videoFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to check video codec: %w", err)
		return result
	}
	if isH265 {
		result.WasSkipped = true
		result.SkipReason = "already H.265/HEVC encoded"
		return result
	}

	// Get original codec for reporting
	originalCodec, err := GetVideoCodec(videoFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get original codec: %w", err)
		return result
	}
	result.OriginalCodec = originalCodec

	// Create temporary output file
	ext := filepath.Ext(videoFile)
	tempFile := strings.TrimSuffix(videoFile, ext) + "_temp_h265" + ext
	defer func() {
		// Clean up temp file if it exists
		_ = os.Remove(tempFile)
	}()

	// Build FFmpeg command for H.265 encoding
	cmd := exec.Command("ffmpeg",
		"-i", videoFile,
		"-c:v", "libx265",
		"-crf", fmt.Sprintf("%d", options.CRF),
		"-preset", options.Preset,
		"-c:a", "copy", // Copy audio without re-encoding
		"-y", // Overwrite output file
		tempFile,
	)

	// Run the encoding
	if runErr := cmd.Run(); runErr != nil {
		result.Error = fmt.Errorf("failed to re-encode video: %w", runErr)
		return result
	}

	// Check if temp file was created successfully
	newSize, err := GetFileSize(tempFile)
	if err != nil {
		result.Error = fmt.Errorf("failed to get new file size: %w", err)
		return result
	}
	result.NewSize = newSize

	// Calculate savings
	result.SizeSavings = originalSize - newSize
	result.SavingsPercent = float64(result.SizeSavings) / float64(originalSize)

	// Check if savings meet minimum threshold
	if result.SavingsPercent < options.MinSavings {
		result.WasSkipped = true
		result.SkipReason = fmt.Sprintf("insufficient savings (%.1f%%, minimum %.1f%%)",
			result.SavingsPercent*100, options.MinSavings*100)
		return result
	}

	// If we should keep original, rename it first
	if options.KeepOriginal {
		backupFile := videoFile + ".bak"
		if err := os.Rename(videoFile, backupFile); err != nil {
			result.Error = fmt.Errorf("failed to backup original file: %w", err)
			return result
		}
	}

	// Replace original with re-encoded version
	if err := os.Rename(tempFile, videoFile); err != nil {
		// If we backed up the original, try to restore it
		if options.KeepOriginal {
			_ = os.Rename(videoFile+".bak", videoFile)
		}
		result.Error = fmt.Errorf("failed to replace original file: %w", err)
		return result
	}

	result.WasReencoded = true
	result.NewPath = videoFile
	return result
}

// ValidateReencodedVideo performs basic validation on a re-encoded video
func ValidateReencodedVideo(videoFile string) error {
	// Check if file exists and has reasonable size
	size, err := GetFileSize(videoFile)
	if err != nil {
		return fmt.Errorf("re-encoded file not accessible: %w", err)
	}

	if size == 0 {
		return fmt.Errorf("re-encoded file is empty")
	}

	// Try to get basic video properties to ensure it's valid
	_, err = GetVideoResolution(videoFile)
	if err != nil {
		return fmt.Errorf("re-encoded file appears corrupted: %w", err)
	}

	return nil
}
