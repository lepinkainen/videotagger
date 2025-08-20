package video

import (
	"testing"
)

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Valid video files
		{"MP4 lowercase", "test.mp4", true},
		{"MP4 uppercase", "test.MP4", true},
		{"WebM", "test.webm", true},
		{"MOV", "test.mov", true},
		{"FLV", "test.flv", true},
		{"MKV", "test.mkv", true},
		{"AVI", "test.avi", true},
		{"WMV", "test.wmv", true},
		{"MPG", "test.mpg", true},

		// With full path
		{"Full path MP4", "/path/to/video.mp4", true},
		{"Relative path", "./videos/test.mov", true},

		// Invalid files
		{"No extension", "test", false},
		{"Text file", "test.txt", false},
		{"Image file", "test.jpg", false},
		{"Audio file", "test.mp3", false},
		{"Document", "test.pdf", false},
		{"Empty string", "", false},

		// Edge cases
		{"Multiple dots", "test.video.mp4", true},
		{"Hidden file", ".hidden.mp4", true},
		{"Space in name", "test file.mp4", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVideoFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsVideoFile(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsProcessed(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		// Processed files
		{"Standard processed", "video_[1920x1080][45min][A1B2C3D4].mp4", true},
		{"Short duration", "test_[720x480][2min][12345678].avi", true},
		{"Long duration", "movie_[3840x2160][120min][ABCDEF12].mkv", true},
		{"Lowercase hash", "file_[1280x720][30min][abcd1234].webm", true},
		{"Mixed case hash", "vid_[640x360][5min][AbCd1234].mov", true},

		// Non-processed files
		{"No metadata", "video.mp4", false},
		{"Partial metadata", "video_[1920x1080].mp4", false},
		{"Wrong format", "video_1920x1080_45min_A1B2C3D4.mp4", false},
		{"Missing resolution", "video_[45min][A1B2C3D4].mp4", false},
		{"Missing duration", "video_[1920x1080][A1B2C3D4].mp4", false},
		{"Missing hash", "video_[1920x1080][45min].mp4", false},
		{"Invalid hash length", "video_[1920x1080][45min][ABC].mp4", false},
		{"Non-hex hash", "video_[1920x1080][45min][GGGGGGGG].mp4", false},
		{"Empty string", "", false},

		// Edge cases with complex filenames
		{"Complex name processed", "My Movie - Director's Cut_[1920x1080][180min][DEADBEEF].mkv", true},
		{"Numbers in name", "Video123_[1280x720][15min][12AB34CD].mp4", true},
		{"Spaces processed", "My Video File_[1920x1080][60min][ABCDEF12].avi", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsProcessed(tt.filename)
			if result != tt.expected {
				t.Errorf("IsProcessed(%q) = %v, expected %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestExtractHashFromFilename(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		expectedHash string
		expectedOk   bool
	}{
		// Valid processed files
		{"Standard format", "video_[1920x1080][45min][A1B2C3D4].mp4", "A1B2C3D4", true},
		{"Lowercase hash", "test_[720x480][2min][abcd1234].avi", "abcd1234", true},
		{"Mixed case", "movie_[3840x2160][120min][AbCd1234].mkv", "AbCd1234", true},
		{"All digits", "file_[1280x720][30min][12345678].webm", "12345678", true},
		{"All letters", "vid_[640x360][5min][ABCDEFAB].mov", "ABCDEFAB", true},

		// Edge cases with valid format
		{"Complex filename", "My Movie - Director's Cut_[1920x1080][180min][DEADBEEF].mkv", "DEADBEEF", true},
		{"Numbers in name", "Video123_[1280x720][15min][12AB34CD].mp4", "12AB34CD", true},
		{"Spaces in name", "My Video File_[1920x1080][60min][FEDCBA98].avi", "FEDCBA98", true},

		// Test cases for extracting hash from last bracket section (GitHub issue #2)
		{"Movie with year bracket", "Movie[2023]_[1920x1080][45min][ABCD1234].mp4", "ABCD1234", true},
		{"Series with episode bracket", "Series[S01E01]_[1920x1080][45min][DEADBEEF].mkv", "DEADBEEF", true},
		{"Multiple extra brackets", "Film[Director][2024][Remastered]_[1920x1080][120min][12345678].mp4", "12345678", true},
		{"Hash not in 3rd position", "Show[S02E03][HD]_[3840x2160][30min][FEDCBA98].webm", "FEDCBA98", true},

		// Invalid formats
		{"No metadata", "video.mp4", "", false},
		{"Partial metadata", "video_[1920x1080].mp4", "", false},
		{"Wrong brackets", "video_(1920x1080)_(45min)_(A1B2C3D4).mp4", "", false},
		{"Missing resolution", "video_[45min][A1B2C3D4].mp4", "", false},
		{"Missing duration", "video_[1920x1080][A1B2C3D4].mp4", "", false},
		{"Missing hash", "video_[1920x1080][45min].mp4", "", false},
		{"Short hash", "video_[1920x1080][45min][ABC].mp4", "", false},
		{"Long hash", "video_[1920x1080][45min][ABCDEF123].mp4", "", false},
		{"Non-hex hash", "video_[1920x1080][45min][GGGGGGGG].mp4", "", false},
		{"Empty string", "", "", false},

		// Invalid resolution formats
		{"Invalid resolution 1", "video_[1920-1080][45min][A1B2C3D4].mp4", "", false},
		{"Invalid resolution 2", "video_[abc x def][45min][A1B2C3D4].mp4", "", false},
		{"Missing resolution x", "video_[1920 1080][45min][A1B2C3D4].mp4", "", false},

		// Invalid duration formats
		{"Invalid duration 1", "video_[1920x1080][45sec][A1B2C3D4].mp4", "", false},
		{"Invalid duration 2", "video_[1920x1080][abc min][A1B2C3D4].mp4", "", false},
		{"Missing min suffix", "video_[1920x1080][45][A1B2C3D4].mp4", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, ok := ExtractHashFromFilename(tt.filename)
			if ok != tt.expectedOk {
				t.Errorf("ExtractHashFromFilename(%q) ok = %v, expected %v", tt.filename, ok, tt.expectedOk)
			}
			if hash != tt.expectedHash {
				t.Errorf("ExtractHashFromFilename(%q) hash = %q, expected %q", tt.filename, hash, tt.expectedHash)
			}
		})
	}
}
