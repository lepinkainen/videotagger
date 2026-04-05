package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// wasProcessedRegex matches files that have already been processed with metadata
var wasProcessedRegex = regexp.MustCompile(`_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^.]*$`)
var hashInBracketsRegex = regexp.MustCompile(`\[([a-fA-F0-9]{8})\]`)

// videoExtensions is the canonical list of supported video file extensions (with leading dot)
var videoExtensions = []string{".mp4", ".m4v", ".webm", ".mov", ".flv", ".mkv", ".avi", ".wmv", ".mpg", ".mpeg", ".divx"}

// VideoExtensionsNoDot returns extensions without leading dots (for use with fd, etc.)
func VideoExtensionsNoDot() []string {
	exts := make([]string, len(videoExtensions))
	for i, ext := range videoExtensions {
		exts[i] = strings.TrimPrefix(ext, ".")
	}
	return exts
}

// IsVideoFile checks if the given file extension is one of known video file extensions
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return slices.Contains(videoExtensions, ext)
}

// IsProcessed checks if a video file has already been processed (has metadata in filename)
func IsProcessed(filename string) bool {
	return wasProcessedRegex.MatchString(filename)
}

// ExtractHashFromFilename extracts the CRC32 hash from a processed filename
// It finds the last bracket section containing an 8-character hex string
func ExtractHashFromFilename(filename string) (string, bool) {
	// First check if file is processed using the existing regex
	if !wasProcessedRegex.MatchString(filename) {
		return "", false
	}

	matches := hashInBracketsRegex.FindAllStringSubmatch(filename, -1)

	if len(matches) == 0 {
		return "", false
	}

	// Return the last hash found
	lastMatch := matches[len(matches)-1]
	return lastMatch[1], true
}

// ExtractMetadataFromFilename extracts resolution, duration, and hash from a processed filename
// Returns resolution (e.g., "1920x1080"), duration in minutes, hash, and whether parsing succeeded
func ExtractMetadataFromFilename(filename string) (resolution string, durationMins int, hash string, ok bool) {
	matches := wasProcessedRegex.FindStringSubmatch(filename)

	if len(matches) < 4 {
		return "", 0, "", false // Not a processed file or no match
	}

	resolution = matches[1] // e.g., "1920x1080"

	// Parse duration (should always be valid integer based on regex)
	var err error
	durationMins, err = strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, "", false
	}

	hash = matches[3] // e.g., "A1B2C3D4"

	return resolution, durationMins, hash, true
}

// ValidateVideoIntegrity checks if a video file is corrupted or invalid
// Returns an error if the file is corrupted or cannot be read
func ValidateVideoIntegrity(filePath string) error {
	// First check if file exists and is readable
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not accessible: %w", err)
	}

	// Use ffprobe to check file integrity without extracting metadata
	// We use a minimal probe to just validate the file structure
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", "--", filePath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check for common corruption indicators
		outputStr := string(output)
		if strings.Contains(outputStr, "moov atom not found") {
			return fmt.Errorf("video file is corrupted (missing metadata): %s", extractFirstLine(outputStr))
		}
		if strings.Contains(outputStr, "Invalid data found") ||
			strings.Contains(outputStr, "corrupt") ||
			strings.Contains(outputStr, "truncated") ||
			strings.Contains(outputStr, "Invalid argument") {
			return fmt.Errorf("video file is corrupted or invalid: %s", extractFirstLine(outputStr))
		}

		// Return generic ffprobe error with output
		return fmt.Errorf("ffprobe error: %w\nOutput: %s", err, extractFirstLine(outputStr))
	}

	return nil
}

// extractFirstLine extracts just the first line from a multi-line string
func extractFirstLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		return strings.TrimSpace(lines[0])
	}
	return "no additional information available"
}
