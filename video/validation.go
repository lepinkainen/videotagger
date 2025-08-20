package video

import (
	"path/filepath"
	"regexp"
	"strings"
)

// wasProcessedRegex matches files that have already been processed with metadata
var wasProcessedRegex = regexp.MustCompile(`_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^\.]*$`)

// IsVideoFile checks if the given file extension is one of known video file extensions
func IsVideoFile(path string) bool {
	var desiredExtensions = []string{".mp4", ".webm", ".mov", ".flv", ".mkv", ".avi", ".wmv", ".mpg"}

	ext := filepath.Ext(path)
	ext = strings.ToLower(ext) // handle cases where extension is upper case

	for _, v := range desiredExtensions {
		if v == ext {
			return true
		}
	}
	return false
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

	// Find all 8-character hex strings in brackets: [ABCD1234]
	hashRegex := regexp.MustCompile(`\[([a-fA-F0-9]{8})\]`)
	matches := hashRegex.FindAllStringSubmatch(filename, -1)

	if len(matches) == 0 {
		return "", false
	}

	// Return the last hash found
	lastMatch := matches[len(matches)-1]
	return lastMatch[1], true
}
