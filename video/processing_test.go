package video

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestFindDuplicatesByHash(t *testing.T) {
	// Test FindDuplicatesByHash with a directory containing known duplicates
	testDir := "../test_files"

	// Check if test_files directory exists
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("test_files directory not found, skipping duplicate detection test")
	}

	duplicates, err := FindDuplicatesByHash(testDir)
	if err != nil {
		t.Fatalf("FindDuplicatesByHash() error = %v", err)
	}

	// Based on the existing test files, we should find duplicates
	// Enjoy_[1078x1578][0min][C1300DCA].mp4 and Dupe of Enjoy_[1078x1578][0min][C1300DCA].mp4
	// should have the same hash: C1300DCA
	expectedHash := "C1300DCA"
	if duplicateFiles, exists := duplicates[expectedHash]; exists {
		if len(duplicateFiles) < 2 {
			t.Errorf("Expected at least 2 files with hash %s, got %d", expectedHash, len(duplicateFiles))
		}

		// Verify the files have the expected names
		foundEnjoy := false
		foundDupe := false
		for _, file := range duplicateFiles {
			basename := filepath.Base(file)
			if strings.Contains(basename, "Enjoy_[1078x1578][0min][C1300DCA]") {
				foundEnjoy = true
			}
			if strings.Contains(basename, "Dupe of Enjoy_[1078x1578][0min][C1300DCA]") {
				foundDupe = true
			}
		}

		if !foundEnjoy || !foundDupe {
			t.Errorf("Expected to find both original and duplicate files, got: %v", duplicateFiles)
		}
	} else {
		t.Errorf("Expected to find duplicates with hash %s", expectedHash)
	}
}

func TestFindDuplicatesByHash_EmptyDirectory(t *testing.T) {
	// Test FindDuplicatesByHash with empty directory
	testDir := t.TempDir()

	duplicates, err := FindDuplicatesByHash(testDir)
	if err != nil {
		t.Fatalf("FindDuplicatesByHash() error = %v", err)
	}

	if len(duplicates) != 0 {
		t.Errorf("Expected no duplicates in empty directory, got %d", len(duplicates))
	}
}

func TestFindDuplicatesByHash_NonExistentDirectory(t *testing.T) {
	// Test FindDuplicatesByHash with non-existent directory
	nonExistentDir := "/path/to/nonexistent/directory"

	_, err := FindDuplicatesByHash(nonExistentDir)
	if err == nil {
		t.Error("FindDuplicatesByHash() expected error for non-existent directory, got nil")
	}
}

func TestFindDuplicatesByHash_DirectoryWithUnprocessedFiles(t *testing.T) {
	// Test FindDuplicatesByHash with directory containing unprocessed files
	testDir := t.TempDir()

	// Create some unprocessed video files
	testFiles := []string{"video1.mp4", "video2.avi", "document.txt"}
	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	duplicates, err := FindDuplicatesByHash(testDir)
	if err != nil {
		t.Fatalf("FindDuplicatesByHash() error = %v", err)
	}

	// Should find no duplicates because files are not processed (no hash in filename)
	if len(duplicates) != 0 {
		t.Errorf("Expected no duplicates for unprocessed files, got %d", len(duplicates))
	}
}

func TestFindDuplicatesByHash_ProcessedFilesNoDuplicates(t *testing.T) {
	// Test FindDuplicatesByHash with processed files that have unique hashes
	testDir := t.TempDir()

	// Create processed files with unique hashes
	testFiles := []string{
		"video1_[1920x1080][45min][AAAAAAAA].mp4",
		"video2_[1280x720][30min][BBBBBBBB].avi",
		"video3_[720x480][15min][CCCCCCCC].mkv",
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	duplicates, err := FindDuplicatesByHash(testDir)
	if err != nil {
		t.Fatalf("FindDuplicatesByHash() error = %v", err)
	}

	// Should find no duplicates because all hashes are unique
	if len(duplicates) != 0 {
		t.Errorf("Expected no duplicates for unique hashes, got %d", len(duplicates))
	}
}

func TestFindDuplicatesByHash_ProcessedFilesWithDuplicates(t *testing.T) {
	// Test FindDuplicatesByHash with processed files that have duplicate hashes
	testDir := t.TempDir()

	// Create processed files where some have the same hash
	testFiles := []string{
		"video1_[1920x1080][45min][DEADBEEF].mp4",
		"copy_of_video1_[1920x1080][45min][DEADBEEF].mp4",
		"video2_[1280x720][30min][CAFEBABE].avi",
		"different_video_[720x480][15min][DEADBEEF].mkv", // Same hash as video1
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	duplicates, err := FindDuplicatesByHash(testDir)
	if err != nil {
		t.Fatalf("FindDuplicatesByHash() error = %v", err)
	}

	// Should find duplicates for hash DEADBEEF
	expectedHash := "DEADBEEF"
	if duplicateFiles, exists := duplicates[expectedHash]; exists {
		if len(duplicateFiles) != 3 {
			t.Errorf("Expected 3 files with hash %s, got %d: %v", expectedHash, len(duplicateFiles), duplicateFiles)
		}
	} else {
		t.Errorf("Expected to find duplicates with hash %s", expectedHash)
	}

	// Should not find CAFEBABE in duplicates (only one file)
	if duplicateFiles, exists := duplicates["CAFEBABE"]; exists {
		t.Errorf("Expected no duplicates for unique hash CAFEBABE, got %v", duplicateFiles)
	}
}

func TestFindDuplicatesByHash_MixedProcessedUnprocessed(t *testing.T) {
	// Test FindDuplicatesByHash with mix of processed and unprocessed files
	testDir := t.TempDir()

	// Create mix of processed and unprocessed files
	testFiles := []string{
		"processed1_[1920x1080][45min][ABCD1234].mp4",
		"processed2_[1920x1080][45min][ABCD1234].mp4", // Duplicate hash
		"unprocessed.mp4", // No hash in filename
		"document.txt",    // Not a video file
		"processed3_[1280x720][30min][EFGH5678].avi", // Unique hash
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	duplicates, err := FindDuplicatesByHash(testDir)
	if err != nil {
		t.Fatalf("FindDuplicatesByHash() error = %v", err)
	}

	// Should only find duplicates for ABCD1234
	if len(duplicates) != 1 {
		t.Errorf("Expected 1 group of duplicates, got %d", len(duplicates))
	}

	expectedHash := "ABCD1234"
	if duplicateFiles, exists := duplicates[expectedHash]; exists {
		if len(duplicateFiles) != 2 {
			t.Errorf("Expected 2 files with hash %s, got %d: %v", expectedHash, len(duplicateFiles), duplicateFiles)
		}
	} else {
		t.Errorf("Expected to find duplicates with hash %s", expectedHash)
	}
}
