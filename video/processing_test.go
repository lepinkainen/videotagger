package video

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test pure functions

func TestValidateVideoFile(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T) string
		wantErr       bool
		wantDirectory bool
		wantVideoFile bool
		wantProcessed bool
	}{
		{
			name: "valid unprocessed video file",
			setup: func(t *testing.T) string {
				testDir := t.TempDir()
				testFile := filepath.Join(testDir, "test.mp4")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return testFile
			},
			wantErr:       false,
			wantDirectory: false,
			wantVideoFile: true,
			wantProcessed: false,
		},
		{
			name: "processed video file",
			setup: func(t *testing.T) string {
				testDir := t.TempDir()
				testFile := filepath.Join(testDir, "test_[1920x1080][45min][ABCD1234].mp4")
				err := os.WriteFile(testFile, []byte("test content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return testFile
			},
			wantErr:       false,
			wantDirectory: false,
			wantVideoFile: true,
			wantProcessed: true,
		},
		{
			name: "directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr:       false,
			wantDirectory: true,
			wantVideoFile: false,
			wantProcessed: false,
		},
		{
			name: "non-video file",
			setup: func(t *testing.T) string {
				testDir := t.TempDir()
				testFile := filepath.Join(testDir, "document.txt")
				err := os.WriteFile(testFile, []byte("text content"), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return testFile
			},
			wantErr:       false,
			wantDirectory: false,
			wantVideoFile: false,
			wantProcessed: false,
		},
		{
			name: "non-existent file",
			setup: func(t *testing.T) string {
				return "/path/to/nonexistent/file.mp4"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setup(t)
			result, err := validateVideoFile(filePath)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.IsDirectory != tt.wantDirectory {
				t.Errorf("IsDirectory = %v, want %v", result.IsDirectory, tt.wantDirectory)
			}

			if result.IsVideoFile != tt.wantVideoFile {
				t.Errorf("IsVideoFile = %v, want %v", result.IsVideoFile, tt.wantVideoFile)
			}

			if result.IsProcessed != tt.wantProcessed {
				t.Errorf("IsProcessed = %v, want %v", result.IsProcessed, tt.wantProcessed)
			}
		})
	}
}

func TestGenerateTaggedFilename(t *testing.T) {
	tests := []struct {
		name         string
		originalPath string
		metadata     *VideoMetadata
		crc          uint32
		want         string
	}{
		{
			name:         "basic mp4 file",
			originalPath: "/path/to/video.mp4",
			metadata:     &VideoMetadata{Resolution: "1920x1080", DurationMins: 45.5},
			crc:          0xDEADBEEF,
			want:         "/path/to/video_[1920x1080][46min][DEADBEEF].mp4",
		},
		{
			name:         "avi file with different resolution",
			originalPath: "test.avi",
			metadata:     &VideoMetadata{Resolution: "1280x720", DurationMins: 30.2},
			crc:          0x12345678,
			want:         "test_[1280x720][30min][12345678].avi",
		},
		{
			name:         "file with no extension",
			originalPath: "video",
			metadata:     &VideoMetadata{Resolution: "720x480", DurationMins: 15.0},
			crc:          0xABCDEF00,
			want:         "video_[720x480][15min][ABCDEF00]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateTaggedFilename(tt.originalPath, tt.metadata, tt.crc)
			if got != tt.want {
				t.Errorf("generateTaggedFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateFileHash(t *testing.T) {
	// Create a test file with known content
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.mp4")
	testContent := []byte("test video content for hash calculation")
	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test without progress writer
	hash1, err := calculateFileHash(testFile, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test with progress writer
	var progressBuffer bytes.Buffer
	hash2, err := calculateFileHash(testFile, &progressBuffer)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Both calls should return the same hash
	if hash1 != hash2 {
		t.Errorf("Hash mismatch: %08X vs %08X", hash1, hash2)
	}

	// Progress buffer should contain the file content
	if progressBuffer.Len() != len(testContent) {
		t.Errorf("Progress buffer length = %d, want %d", progressBuffer.Len(), len(testContent))
	}

	// Test non-existent file
	_, err = calculateFileHash("/nonexistent/file.mp4", nil)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestRenameVideoFile(t *testing.T) {
	testDir := t.TempDir()
	oldPath := filepath.Join(testDir, "original.mp4")
	newPath := filepath.Join(testDir, "renamed.mp4")

	// Create original file
	err := os.WriteFile(oldPath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test successful rename
	err = renameVideoFile(oldPath, newPath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify old file doesn't exist
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Original file should not exist after rename")
	}

	// Verify new file exists
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("New file should exist after rename: %v", err)
	}

	// Test rename non-existent file
	err = renameVideoFile("/nonexistent/file.mp4", "/some/path.mp4")
	if err == nil {
		t.Error("Expected error when renaming non-existent file")
	}
}

// Integration tests for ProcessVideoFile

func TestProcessVideoFile_DirectoryInput(t *testing.T) {
	// Test that ProcessVideoFile handles directory input correctly
	testDir := t.TempDir()

	// Redirect stdout to capture output
	// ProcessVideoFile prints directly to stdout, so we can't easily capture it
	// For now, we'll just ensure it doesn't panic
	ProcessVideoFile(testDir)

	// The function should return gracefully without processing directories
	// Since it prints to stdout, we can't easily assert the output without complex setup
}

func TestProcessVideoFile_NonVideoFile(t *testing.T) {
	// Test that ProcessVideoFile skips non-video files
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "document.txt")

	err := os.WriteFile(testFile, []byte("This is a text document"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should skip the file gracefully
	ProcessVideoFile(testFile)

	// The function should return without processing non-video files
}

func TestProcessVideoFile_NonExistentFile(t *testing.T) {
	// Test that ProcessVideoFile handles non-existent files
	nonExistentFile := "/path/to/nonexistent/video.mp4"

	// This should handle the error gracefully
	ProcessVideoFile(nonExistentFile)

	// The function should return after printing an error message
}

func TestProcessVideoFile_AlreadyProcessed(t *testing.T) {
	// Test that ProcessVideoFile skips already processed files
	testDir := t.TempDir()

	// Create a file that looks like it's already been processed
	processedFile := filepath.Join(testDir, "video_[1920x1080][45min][ABCD1234].mp4")
	err := os.WriteFile(processedFile, []byte("fake video content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(processedFile)

	// This should skip the file because it's already processed
	ProcessVideoFile(processedFile)

	// Verify the file wasn't renamed (since it was already processed)
	if _, err := os.Stat(processedFile); os.IsNotExist(err) {
		t.Error("Already processed file should not be modified")
	}
}

func TestProcessVideoFile_UnprocessedVideoFile(t *testing.T) {
	// Test ProcessVideoFile with an unprocessed video file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test_video.mp4")

	// Create a fake video file
	err := os.WriteFile(testFile, []byte("fake video content for testing"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This will attempt to process the file, but will likely fail because:
	// 1. It's not a real video (FFmpeg will fail)
	// 2. We don't have FFmpeg installed (in CI environments)
	ProcessVideoFile(testFile)

	// The file should still exist (processing failed, so no rename occurred)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		// If the file doesn't exist, it might have been renamed or deleted
		// Check if there's a renamed version
		entries, _ := os.ReadDir(testDir)
		renamed := false
		for _, entry := range entries {
			if strings.Contains(entry.Name(), "test_video") && strings.Contains(entry.Name(), "[") {
				renamed = true
				defer os.Remove(filepath.Join(testDir, entry.Name()))
				break
			}
		}
		if !renamed {
			t.Error("Test file disappeared without being renamed")
		}
	}
}
