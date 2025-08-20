package video

import "os"

// VideoMetadata contains the extracted metadata for a video file
type VideoMetadata struct {
	Resolution   string
	DurationMins float64
}

// FileValidationResult contains the result of file validation
type FileValidationResult struct {
	FileInfo    os.FileInfo
	IsDirectory bool
	IsVideoFile bool
	IsProcessed bool
}
