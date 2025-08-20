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

// ProcessingResult represents the result of processing a video file
type ProcessingResult struct {
	OriginalPath string
	NewPath      string
	WasSkipped   bool
	SkipReason   string
	Error        error
	Metadata     *VideoMetadata
	CRC32        uint32
	WasRenamed   bool
}
