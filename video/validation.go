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
func ExtractHashFromFilename(filename string) (string, bool) {
	matches := wasProcessedRegex.FindStringSubmatch(filename)
	if len(matches) >= 4 {
		return matches[3], true
	}
	return "", false
}
