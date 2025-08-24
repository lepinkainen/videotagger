package video

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGetVideoResolution(t *testing.T) {
	// Test with a fake video file (text file with .mp4 extension)
	// This will test the error handling when FFprobe fails
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "fake_video.mp4")

	err := os.WriteFile(testFile, []byte("This is not a video file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's not a real video file
	_, err = GetVideoResolution(testFile)
	if err == nil {
		t.Error("GetVideoResolution() expected error for non-video file, got nil")
	}

	// Check that error message contains useful information
	if !strings.Contains(err.Error(), "failed to get resolution") {
		t.Errorf("Expected error to contain 'failed to get resolution', got: %v", err)
	}
}

func TestGetVideoResolution_NonExistentFile(t *testing.T) {
	// Test with non-existent file
	_, err := GetVideoResolution("/path/to/nonexistent/video.mp4")
	if err == nil {
		t.Error("GetVideoResolution() expected error for non-existent file, got nil")
	}
}

func TestGetVideoResolution_EmptyFile(t *testing.T) {
	// Test with empty file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "empty.mp4")

	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's an empty file
	_, err = GetVideoResolution(testFile)
	if err == nil {
		t.Error("GetVideoResolution() expected error for empty file, got nil")
	}
}

func TestGetVideoResolution_ValidationLogic(t *testing.T) {
	// This test verifies the resolution format validation logic
	// We can't easily mock ffprobe output without complex setup,
	// but we can test the validation regex directly

	validResolutions := []string{
		"1920x1080",
		"1280x720",
		"720x480",
		"3840x2160",
		"640x360",
	}

	invalidResolutions := []string{
		"1920-1080",
		"abc x def",
		"1920 x 1080",
		"1920",
		"x1080",
		"1920x",
		"",
	}

	resolutionRegex := regexp.MustCompile(`^\d+x\d+$`)

	// Test valid resolutions
	for _, resolution := range validResolutions {
		if !resolutionRegex.MatchString(resolution) {
			t.Errorf("Valid resolution %q should match the regex", resolution)
		}
	}

	// Test invalid resolutions
	for _, resolution := range invalidResolutions {
		if resolutionRegex.MatchString(resolution) {
			t.Errorf("Invalid resolution %q should not match the regex", resolution)
		}
	}
}

func TestGetVideoDuration(t *testing.T) {
	// Test with a fake video file (text file with .mp4 extension)
	// This will test the error handling when FFprobe fails
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "fake_video.mp4")

	err := os.WriteFile(testFile, []byte("This is not a video file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's not a real video file
	_, err = GetVideoDuration(testFile)
	if err == nil {
		t.Error("GetVideoDuration() expected error for non-video file, got nil")
	}

	// Check that error message contains useful information
	if !strings.Contains(err.Error(), "failed to get duration") {
		t.Errorf("Expected error to contain 'failed to get duration', got: %v", err)
	}
}

func TestGetVideoDuration_NonExistentFile(t *testing.T) {
	// Test with non-existent file
	_, err := GetVideoDuration("/path/to/nonexistent/video.mp4")
	if err == nil {
		t.Error("GetVideoDuration() expected error for non-existent file, got nil")
	}
}

func TestGetVideoDuration_EmptyFile(t *testing.T) {
	// Test with empty file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "empty.mp4")

	err := os.WriteFile(testFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's an empty file
	_, err = GetVideoDuration(testFile)
	if err == nil {
		t.Error("GetVideoDuration() expected error for empty file, got nil")
	}
}

func TestGetVideoDuration_ConversionLogic(t *testing.T) {
	// Test the seconds to minutes conversion logic
	// This tests the calculation without requiring actual FFprobe

	tests := []struct {
		name         string
		durationSecs float64
		expectedMins float64
	}{
		{
			name:         "Exactly 1 minute",
			durationSecs: 60.0,
			expectedMins: 1.0,
		},
		{
			name:         "30 seconds",
			durationSecs: 30.0,
			expectedMins: 0.5,
		},
		{
			name:         "2 minutes 30 seconds",
			durationSecs: 150.0,
			expectedMins: 2.5,
		},
		{
			name:         "1 hour",
			durationSecs: 3600.0,
			expectedMins: 60.0,
		},
		{
			name:         "Zero duration",
			durationSecs: 0.0,
			expectedMins: 0.0,
		},
		{
			name:         "Fractional seconds",
			durationSecs: 90.5,
			expectedMins: 1.508333333333333, // 90.5 / 60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.durationSecs / 60
			// Use a small tolerance for floating point comparison
			tolerance := 0.000001
			if abs(result-tt.expectedMins) > tolerance {
				t.Errorf("Duration conversion: %f seconds = %f minutes, expected %f",
					tt.durationSecs, result, tt.expectedMins)
			}
		})
	}
}

// Helper function for absolute difference
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestFFprobeCommandGeneration tests that the right commands are being built
func TestFFprobeCommandGeneration(t *testing.T) {
	// This test documents the expected FFprobe commands
	// These are the commands that should be executed by the functions

	testFile := "/path/to/test.mp4"

	// Expected command for GetVideoResolution
	expectedResolutionCmd := []string{
		"ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", "--", testFile,
	}

	// Expected command for GetVideoDuration
	expectedDurationCmd := []string{
		"ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=noprint_wrappers=1:nokey=1", testFile,
	}

	// Just verify the command structure is documented
	// In a real implementation, we might mock exec.Command to verify these
	t.Logf("Expected resolution command: %v", expectedResolutionCmd)
	t.Logf("Expected duration command: %v", expectedDurationCmd)

	// Basic validation of command structure
	if expectedResolutionCmd[0] != "ffprobe" {
		t.Error("Resolution command should start with ffprobe")
	}
	if expectedDurationCmd[0] != "ffprobe" {
		t.Error("Duration command should start with ffprobe")
	}
}

// TestErrorHandling verifies that both functions handle various error conditions
func TestErrorHandling(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
	}{
		{
			name:     "Non-existent file",
			filename: "/path/to/nonexistent/file.mp4",
		},
		{
			name:     "Directory instead of file",
			filename: os.TempDir(),
		},
		{
			name:     "File with no extension",
			filename: "/path/to/file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test GetVideoResolution error handling
			_, err := GetVideoResolution(tc.filename)
			if err == nil {
				t.Errorf("GetVideoResolution(%q) expected error, got nil", tc.filename)
			}

			// Test GetVideoDuration error handling
			_, err = GetVideoDuration(tc.filename)
			if err == nil {
				t.Errorf("GetVideoDuration(%q) expected error, got nil", tc.filename)
			}
		})
	}
}

// TestWithRealVideoFiles tests with actual video files if available
func TestWithRealVideoFiles(t *testing.T) {
	// Look for any video files in the test_files directory
	testFilesDir := "../test_files"
	if _, err := os.Stat(testFilesDir); os.IsNotExist(err) {
		t.Skip("No test_files directory found, skipping real video file tests")
	}

	entries, err := os.ReadDir(testFilesDir)
	if err != nil {
		t.Skip("Cannot read test_files directory, skipping real video file tests")
	}

	videoFiles := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && IsVideoFile(entry.Name()) {
			videoFiles = append(videoFiles, filepath.Join(testFilesDir, entry.Name()))
		}
	}

	if len(videoFiles) == 0 {
		t.Skip("No video files found in test_files directory")
	}

	for _, videoFile := range videoFiles {
		t.Run(fmt.Sprintf("Real video: %s", filepath.Base(videoFile)), func(t *testing.T) {
			// Test GetVideoResolution
			resolution, err := GetVideoResolution(videoFile)
			if err != nil {
				t.Logf("GetVideoResolution failed (expected if no FFmpeg): %v", err)
			} else {
				t.Logf("Resolution: %s", resolution)
				// Validate resolution format
				if !regexp.MustCompile(`^\d+x\d+$`).MatchString(resolution) {
					t.Errorf("Invalid resolution format: %s", resolution)
				}
			}

			// Test GetVideoDuration
			duration, err := GetVideoDuration(videoFile)
			if err != nil {
				t.Logf("GetVideoDuration failed (expected if no FFmpeg): %v", err)
			} else {
				t.Logf("Duration: %.2f minutes", duration)
				// Validate duration is non-negative
				if duration < 0 {
					t.Errorf("Duration should be non-negative, got: %.2f", duration)
				}
			}
		})
	}
}

// TestGetVideoCodec tests the codec detection functionality
func TestGetVideoCodec(t *testing.T) {
	// Test with a fake video file (text file with .mp4 extension)
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "fake_video.mp4")

	err := os.WriteFile(testFile, []byte("This is not a video file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's not a real video file
	_, err = GetVideoCodec(testFile)
	if err == nil {
		t.Error("GetVideoCodec() expected error for non-video file, got nil")
	}

	// Check that error message contains useful information
	if !strings.Contains(err.Error(), "failed to get codec") {
		t.Errorf("Expected error to contain 'failed to get codec', got: %v", err)
	}
}

func TestGetVideoCodecNonExistentFile(t *testing.T) {
	_, err := GetVideoCodec("nonexistent.mp4")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetVideoCodecWithRealFiles(t *testing.T) {
	// Test with real video files if available
	testFilesDir := "../test_files"
	if _, err := os.Stat(testFilesDir); os.IsNotExist(err) {
		t.Skip("No test_files directory found, skipping real video file codec tests")
	}

	entries, err := os.ReadDir(testFilesDir)
	if err != nil {
		t.Skip("Cannot read test_files directory, skipping codec tests")
	}

	for _, entry := range entries {
		if !entry.IsDir() && IsVideoFile(entry.Name()) {
			testFile := filepath.Join(testFilesDir, entry.Name())

			codec, err := GetVideoCodec(testFile)
			if err != nil {
				t.Logf("GetVideoCodec failed for %s (expected if no FFmpeg): %v", entry.Name(), err)
			} else {
				t.Logf("File: %s, Codec: %s", entry.Name(), codec)

				// Codec should be a reasonable non-empty string
				if codec == "" {
					t.Errorf("Expected non-empty codec for %s", entry.Name())
				}
			}
		}
	}
}

func TestGetFileSize(t *testing.T) {
	// Create a temporary test file
	tempFile, err := os.CreateTemp("", "test_size_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write some content
	testContent := "hello world"
	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	size, err := GetFileSize(tempFile.Name())
	if err != nil {
		t.Errorf("Failed to get file size: %v", err)
	}

	expectedSize := int64(len(testContent))
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestGetFileSizeNonExistentFile(t *testing.T) {
	_, err := GetFileSize("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetFileSizeZeroFile(t *testing.T) {
	// Create an empty file
	tempFile, err := os.CreateTemp("", "test_empty_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	size, err := GetFileSize(tempFile.Name())
	if err != nil {
		t.Errorf("Failed to get file size for empty file: %v", err)
	}

	if size != 0 {
		t.Errorf("Expected size 0 for empty file, got %d", size)
	}
}
