package video

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestIsFdAvailable(t *testing.T) {
	// Test isFdAvailable function
	result := isFdAvailable()

	// Check if fd is actually in PATH
	_, err := exec.LookPath("fd")
	expected := err == nil

	if result != expected {
		t.Errorf("isFdAvailable() = %v, expected %v", result, expected)
	}
}

func TestFindTaggedFilesWithWalkDir(t *testing.T) {
	// Test the fallback method explicitly
	testDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"processed1_[1920x1080][45min][ABCD1234].mp4",
		"processed2_[1280x720][30min][ABCD5678].avi",
		"unprocessed.mp4", // Should be ignored
		"document.txt",    // Should be ignored
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	files, err := findTaggedFilesWithWalkDir(testDir)
	if err != nil {
		t.Fatalf("findTaggedFilesWithWalkDir() error = %v", err)
	}

	// Should find only the processed video files
	expectedCount := 2
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(files), files)
	}

	// Verify the correct files were found
	foundProcessed1 := false
	foundProcessed2 := false
	for _, file := range files {
		basename := filepath.Base(file)
		if strings.Contains(basename, "processed1_[1920x1080][45min][ABCD1234].mp4") {
			foundProcessed1 = true
		}
		if strings.Contains(basename, "processed2_[1280x720][30min][ABCD5678].avi") {
			foundProcessed2 = true
		}
	}

	if !foundProcessed1 || !foundProcessed2 {
		t.Errorf("Expected to find both processed files, got: %v", files)
	}
}

func TestFindTaggedFilesWithFd(t *testing.T) {
	// Test fd method if available
	if !isFdAvailable() {
		t.Skip("fd not available, skipping fd-specific test")
	}

	testDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"processed1_[1920x1080][45min][ABCD1234].mp4",
		"processed2_[1280x720][30min][ABCD5678].avi",
		"unprocessed.mp4", // Should be ignored
		"document.txt",    // Should be ignored
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	files, err := findTaggedFilesWithFd(testDir)
	if err != nil {
		t.Fatalf("findTaggedFilesWithFd() error = %v", err)
	}

	// Should find only the processed video files
	expectedCount := 2
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(files), files)
	}

	// Verify the correct files were found
	foundProcessed1 := false
	foundProcessed2 := false
	for _, file := range files {
		basename := filepath.Base(file)
		if strings.Contains(basename, "processed1_[1920x1080][45min][ABCD1234].mp4") {
			foundProcessed1 = true
		}
		if strings.Contains(basename, "processed2_[1280x720][30min][ABCD5678].avi") {
			foundProcessed2 = true
		}
	}

	if !foundProcessed1 || !foundProcessed2 {
		t.Errorf("Expected to find both processed files, got: %v", files)
	}
}

func TestFindDuplicatesByHash_CompareMethodsConsistency(t *testing.T) {
	// Test that both fd and walkdir methods produce consistent results
	testDir := t.TempDir()

	// Create test files with duplicates
	testFiles := []string{
		"video1_[1920x1080][45min][DEADBEEF].mp4",
		"video2_[1920x1080][45min][DEADBEEF].mp4", // Duplicate
		"video3_[1280x720][30min][CAFEBABE].avi",  // Unique
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	// Test walkdir method
	walkDirFiles, err := findTaggedFilesWithWalkDir(testDir)
	if err != nil {
		t.Fatalf("findTaggedFilesWithWalkDir() error = %v", err)
	}

	// Test fd method if available
	if isFdAvailable() {
		fdFiles, err := findTaggedFilesWithFd(testDir)
		if err != nil {
			t.Fatalf("findTaggedFilesWithFd() error = %v", err)
		}

		// Both methods should find the same number of files
		if len(walkDirFiles) != len(fdFiles) {
			t.Errorf("Method inconsistency: walkdir found %d files, fd found %d files", len(walkDirFiles), len(fdFiles))
		}

		// Convert to maps for easier comparison
		walkDirMap := make(map[string]bool)
		for _, file := range walkDirFiles {
			walkDirMap[filepath.Base(file)] = true
		}

		fdMap := make(map[string]bool)
		for _, file := range fdFiles {
			fdMap[filepath.Base(file)] = true
		}

		// Check that both methods found the same files
		for filename := range walkDirMap {
			if !fdMap[filename] {
				t.Errorf("fd method missed file found by walkdir: %s", filename)
			}
		}

		for filename := range fdMap {
			if !walkDirMap[filename] {
				t.Errorf("walkdir method missed file found by fd: %s", filename)
			}
		}
	}
}

func TestFindVideoFilesRecursively(t *testing.T) {
	// Create a temporary directory structure with video files
	testDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"video1.mp4",
		"video2.avi",
		"subfolder/video3.mkv",
		"subfolder/nested/video4.mov",
		"already_processed_[1920x1080][45min][12345678].mp4",
		"document.txt", // Non-video file
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(testDir, file)
		dir := filepath.Dir(fullPath)

		// Create directory if needed
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create empty file
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	// Test FindVideoFilesRecursively
	files, err := FindVideoFilesRecursively(testDir)
	if err != nil {
		t.Fatalf("FindVideoFilesRecursively() error = %v", err)
	}

	// Should find only unprocessed video files
	expectedFiles := []string{
		filepath.Join(testDir, "video1.mp4"),
		filepath.Join(testDir, "video2.avi"),
		filepath.Join(testDir, "subfolder/video3.mkv"),
		filepath.Join(testDir, "subfolder/nested/video4.mov"),
	}

	if len(files) != len(expectedFiles) {
		t.Errorf("Expected %d files, got %d", len(expectedFiles), len(files))
		t.Logf("Found files: %v", files)
		t.Logf("Expected files: %v", expectedFiles)
	}

	// Convert to maps for easier comparison
	foundMap := make(map[string]bool)
	for _, file := range files {
		foundMap[file] = true
	}

	for _, expected := range expectedFiles {
		if !foundMap[expected] {
			t.Errorf("Expected file not found: %s", expected)
		}
	}
}

func TestFindUnprocessedFilesWithWalkDir(t *testing.T) {
	// Test the unprocessed files walkdir method
	testDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"unprocessed1.mp4",
		"unprocessed2.avi",
		"processed_[1920x1080][45min][ABCD1234].mp4",
		"document.txt", // Should be ignored
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	files, err := findUnprocessedFilesWithWalkDir(testDir)
	if err != nil {
		t.Fatalf("findUnprocessedFilesWithWalkDir() error = %v", err)
	}

	// Should find only the unprocessed video files
	expectedCount := 2
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(files), files)
	}

	// Verify the correct files were found
	foundUnprocessed1 := false
	foundUnprocessed2 := false
	for _, file := range files {
		basename := filepath.Base(file)
		if basename == "unprocessed1.mp4" {
			foundUnprocessed1 = true
		}
		if basename == "unprocessed2.avi" {
			foundUnprocessed2 = true
		}
	}

	if !foundUnprocessed1 || !foundUnprocessed2 {
		t.Errorf("Expected to find both unprocessed files, got: %v", files)
	}
}

func TestFindUnprocessedFilesWithFd(t *testing.T) {
	// Test fd method for unprocessed files if available
	if !isFdAvailable() {
		t.Skip("fd not available, skipping fd-specific test")
	}

	testDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"unprocessed1.mp4",
		"unprocessed2.avi",
		"processed_[1920x1080][45min][ABCD1234].mp4",
		"document.txt", // Should be ignored
	}

	for _, filename := range testFiles {
		testFile := filepath.Join(testDir, filename)
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		defer os.Remove(testFile)
	}

	files, err := findUnprocessedFilesWithFd(testDir)
	if err != nil {
		t.Fatalf("findUnprocessedFilesWithFd() error = %v", err)
	}

	// Should find only the unprocessed video files
	expectedCount := 2
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, got %d: %v", expectedCount, len(files), files)
	}

	// Verify the correct files were found
	foundUnprocessed1 := false
	foundUnprocessed2 := false
	for _, file := range files {
		basename := filepath.Base(file)
		if basename == "unprocessed1.mp4" {
			foundUnprocessed1 = true
		}
		if basename == "unprocessed2.avi" {
			foundUnprocessed2 = true
		}
	}

	if !foundUnprocessed1 || !foundUnprocessed2 {
		t.Errorf("Expected to find both unprocessed files, got: %v", files)
	}
}
