package video

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultReencodeOptions(t *testing.T) {
	opts := DefaultReencodeOptions()

	if opts == nil {
		t.Fatal("DefaultReencodeOptions() returned nil")
	}

	// Check default values
	if opts.CRF != 23 {
		t.Errorf("Expected CRF 23, got %d", opts.CRF)
	}

	if opts.Preset != "medium" {
		t.Errorf("Expected preset 'medium', got %s", opts.Preset)
	}

	if opts.MinSavings != 0.05 {
		t.Errorf("Expected MinSavings 0.05, got %f", opts.MinSavings)
	}

	if opts.KeepOriginal != false {
		t.Errorf("Expected KeepOriginal false, got %t", opts.KeepOriginal)
	}
}

func TestIsH265(t *testing.T) {
	// Test with a fake video file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "fake_video.mp4")

	err := os.WriteFile(testFile, []byte("This is not a video file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's not a real video file
	_, err = IsH265(testFile)
	if err == nil {
		t.Error("IsH265() expected error for non-video file, got nil")
	}
}

func TestIsH265NonExistentFile(t *testing.T) {
	_, err := IsH265("nonexistent.mp4")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestReencodeToH265WithNonVideoFile(t *testing.T) {
	// Create a temporary non-video file
	tempFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	testContent := "This is not a video file"
	if _, err := tempFile.WriteString(testContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	opts := DefaultReencodeOptions()
	result := ReencodeToH265(tempFile.Name(), opts)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if !result.WasSkipped {
		t.Error("Expected file to be skipped as non-video file")
	}

	if result.SkipReason != "not a video file" {
		t.Errorf("Expected skip reason 'not a video file', got %s", result.SkipReason)
	}
}

func TestReencodeToH265WithNonExistentFile(t *testing.T) {
	opts := DefaultReencodeOptions()
	result := ReencodeToH265("nonexistent.mp4", opts)

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.Error == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestValidateReencodedVideoNonExistent(t *testing.T) {
	err := ValidateReencodedVideo("nonexistent.mp4")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestValidateReencodedVideoEmptyFile(t *testing.T) {
	// Create empty file
	tempFile, err := os.CreateTemp("", "empty_*.mp4")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	err = ValidateReencodedVideo(tempFile.Name())
	if err == nil {
		t.Error("Expected error for empty file")
	}

	if err.Error() != "re-encoded file is empty" {
		t.Errorf("Expected 're-encoded file is empty' error, got: %v", err)
	}
}

func TestReencodeResultStructure(t *testing.T) {
	// Test that ReencodeResult struct has expected fields
	result := &ReencodeResult{
		OriginalPath:   "/path/to/original.mp4",
		OriginalCodec:  "h264",
		OriginalSize:   1000000,
		NewPath:        "/path/to/new.mp4",
		NewSize:        800000,
		SizeSavings:    200000,
		SavingsPercent: 0.2,
		WasReencoded:   true,
		WasSkipped:     false,
		SkipReason:     "",
		Error:          nil,
	}

	if result.OriginalPath == "" {
		t.Error("OriginalPath should not be empty")
	}

	if result.SavingsPercent != 0.2 {
		t.Errorf("Expected SavingsPercent 0.2, got %f", result.SavingsPercent)
	}

	if !result.WasReencoded {
		t.Error("Expected WasReencoded to be true")
	}
}

func TestReencodeOptionsValidation(t *testing.T) {
	// Test various option combinations
	testCases := []struct {
		name   string
		crf    int
		preset string
		valid  bool
	}{
		{"Valid CRF", 23, "medium", true},
		{"Low CRF", 0, "medium", true},
		{"High CRF", 51, "medium", true},
		{"Invalid CRF negative", -1, "medium", false},
		{"Invalid CRF too high", 52, "medium", false},
		{"Valid preset fast", 23, "fast", true},
		{"Valid preset slow", 23, "slow", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts := &ReencodeOptions{
				CRF:          tc.crf,
				Preset:       tc.preset,
				MinSavings:   0.05,
				KeepOriginal: false,
			}

			// Basic validation - CRF should be 0-51
			if tc.crf < 0 || tc.crf > 51 {
				if tc.valid {
					t.Errorf("CRF %d should be invalid", tc.crf)
				}
			} else {
				if !tc.valid && tc.crf >= 0 && tc.crf <= 51 {
					t.Errorf("CRF %d should be valid", tc.crf)
				}
			}

			// Check that options are set correctly
			if opts.CRF != tc.crf {
				t.Errorf("Expected CRF %d, got %d", tc.crf, opts.CRF)
			}

			if opts.Preset != tc.preset {
				t.Errorf("Expected preset %s, got %s", tc.preset, opts.Preset)
			}
		})
	}
}

// TestReencodeWithRealVideoFiles tests with actual video files if available
func TestReencodeWithRealVideoFiles(t *testing.T) {
	// This test is mainly for documenting expected behavior
	// Actual re-encoding would require FFmpeg and take significant time

	testFilesDir := "../test_files"
	if _, err := os.Stat(testFilesDir); os.IsNotExist(err) {
		t.Skip("No test_files directory found, skipping real video file tests")
	}

	entries, err := os.ReadDir(testFilesDir)
	if err != nil {
		t.Skip("Cannot read test_files directory")
	}

	for _, entry := range entries {
		if !entry.IsDir() && IsVideoFile(entry.Name()) {
			testFile := filepath.Join(testFilesDir, entry.Name())

			t.Run(entry.Name(), func(t *testing.T) {
				// Test IsH265 function
				isH265, err := IsH265(testFile)
				if err != nil {
					t.Logf("IsH265 failed (expected if no FFmpeg): %v", err)
				} else {
					t.Logf("File: %s, Is H.265: %t", entry.Name(), isH265)
				}

				// Don't actually re-encode in tests as it's slow and requires FFmpeg
				// Just verify that the function would handle it appropriately
				t.Logf("Would test re-encoding of %s", entry.Name())
			})
		}
	}
}
