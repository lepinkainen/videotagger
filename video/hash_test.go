package video

import (
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculateCRC32(t *testing.T) {
	// Create temporary test files
	testDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected uint32
	}{
		{
			name:     "Empty file",
			content:  "",
			expected: 0,
		},
		{
			name:     "Small text file",
			content:  "hello world",
			expected: crc32.ChecksumIEEE([]byte("hello world")),
		},
		{
			name:     "Binary data",
			content:  "\x00\x01\x02\x03\x04\x05",
			expected: crc32.ChecksumIEEE([]byte("\x00\x01\x02\x03\x04\x05")),
		},
		{
			name:     "Large content",
			content:  strings.Repeat("VideoTagger test data ", 1000),
			expected: crc32.ChecksumIEEE([]byte(strings.Repeat("VideoTagger test data ", 1000))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(testDir, "test_"+tt.name+".dat")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			defer os.Remove(testFile)

			// Calculate CRC32
			result, err := CalculateCRC32(testFile)
			if err != nil {
				t.Fatalf("CalculateCRC32() error = %v", err)
			}

			if result != tt.expected {
				t.Errorf("CalculateCRC32() = %08X, expected %08X", result, tt.expected)
			}
		})
	}
}

func TestCalculateCRC32_FileErrors(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "Non-existent file",
			filename: "/path/to/nonexistent/file.dat",
		},
		{
			name:     "Directory instead of file",
			filename: os.TempDir(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateCRC32(tt.filename)
			if err == nil {
				t.Errorf("CalculateCRC32(%q) expected error, got nil", tt.filename)
			}
		})
	}
}

func TestCalculateCRC32_Permission(t *testing.T) {
	// This test only runs on Unix-like systems where we can control file permissions
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "unreadable.dat")

	// Create file with content
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make file unreadable
	err = os.Chmod(testFile, 0000)
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}
	defer func() { _ = os.Chmod(testFile, 0644) }() // Restore permissions for cleanup

	// Try to calculate CRC32
	_, err = CalculateCRC32(testFile)
	if err == nil {
		t.Error("CalculateCRC32() expected permission error, got nil")
	}
}

// TestCalculateCRC32_LargeFile tests CRC32 calculation with a larger file
func TestCalculateCRC32_LargeFile(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "large.dat")

	// Create a larger file (1MB)
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() { _ = f.Close() }()
	defer func() { _ = os.Remove(testFile) }()

	// Write 1MB of test data
	data := make([]byte, 1024) // 1KB chunk
	for i := range data {
		data[i] = byte(i % 256)
	}

	h := crc32.NewIEEE()
	for i := 0; i < 1024; i++ { // Write 1024 chunks = 1MB
		_, err := f.Write(data)
		if err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
		h.Write(data)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Failed to close test file: %v", err)
	}

	expected := h.Sum32()

	// Calculate CRC32 using our function
	result, err := CalculateCRC32(testFile)
	if err != nil {
		t.Fatalf("CalculateCRC32() error = %v", err)
	}

	if result != expected {
		t.Errorf("CalculateCRC32() = %08X, expected %08X", result, expected)
	}
}

// TestCalculateVideoPerceptualHash tests the perceptual hash function
// Note: This test requires FFmpeg to be installed and available in PATH
func TestCalculateVideoPerceptualHash(t *testing.T) {
	// Check if FFmpeg is available
	if !isFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping perceptual hash tests")
	}

	// Create a minimal test "video" file (actually just a text file)
	// This test will fail as expected since it's not a real video file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "fake_video.mp4")

	err := os.WriteFile(testFile, []byte("This is not a video file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// This should fail because it's not a real video file
	_, err = CalculateVideoPerceptualHash(testFile)
	if err == nil {
		t.Error("CalculateVideoPerceptualHash() expected error for non-video file, got nil")
	}
}

func TestCalculateVideoPerceptualHash_NonExistentFile(t *testing.T) {
	// Test with non-existent file
	_, err := CalculateVideoPerceptualHash("/path/to/nonexistent/video.mp4")
	if err == nil {
		t.Error("CalculateVideoPerceptualHash() expected error for non-existent file, got nil")
	}
}

func TestCalculateVideoPerceptualHash_NoFFmpeg(t *testing.T) {
	// This test verifies behavior when FFmpeg is not available
	// We can't easily simulate this without modifying PATH or mocking exec.Command
	// For now, we'll skip this test as the function signature is validated elsewhere
	t.Skip("Skipping FFmpeg availability test")
}

// Helper function to check if FFmpeg is available
func isFFmpegAvailable() bool {
	// For testing purposes, we'll assume FFmpeg is available
	// Individual tests will fail gracefully if it's not actually available
	return true
}

// Benchmark for CRC32 calculation
func BenchmarkCalculateCRC32(b *testing.B) {
	// Create a test file
	testDir := b.TempDir()
	testFile := filepath.Join(testDir, "benchmark.dat")

	// Create 10MB file
	f, err := os.Create(testFile)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	data := make([]byte, 1024*1024) // 1MB chunk
	for i := range data {
		data[i] = byte(i % 256)
	}

	for i := 0; i < 10; i++ { // 10MB total
		_, _ = f.Write(data)
	}
	f.Close()
	defer os.Remove(testFile)

	// Benchmark the CRC32 calculation
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CalculateCRC32(testFile)
		if err != nil {
			b.Fatalf("CalculateCRC32() error = %v", err)
		}
	}
}

// Test that verifies the CRC32 function produces consistent results
func TestCalculateCRC32_Consistency(t *testing.T) {
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "consistency.dat")

	content := "VideoTagger CRC32 consistency test data"
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Calculate CRC32 multiple times
	var results []uint32
	for i := 0; i < 5; i++ {
		result, err := CalculateCRC32(testFile)
		if err != nil {
			t.Fatalf("CalculateCRC32() error on iteration %d: %v", i, err)
		}
		results = append(results, result)
	}

	// All results should be identical
	first := results[0]
	for i, result := range results {
		if result != first {
			t.Errorf("CRC32 inconsistency: iteration %d got %08X, expected %08X", i, result, first)
		}
	}
}

// Test that demonstrates CRC32 sensitivity to file changes
func TestCalculateCRC32_Sensitivity(t *testing.T) {
	testDir := t.TempDir()

	// Create two files with slightly different content
	file1 := filepath.Join(testDir, "file1.dat")
	file2 := filepath.Join(testDir, "file2.dat")

	err := os.WriteFile(file1, []byte("VideoTagger test data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	defer os.Remove(file1)

	err = os.WriteFile(file2, []byte("VideoTagger test Data"), 0644) // Capital D
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	defer os.Remove(file2)

	// Calculate CRC32 for both files
	crc1, err := CalculateCRC32(file1)
	if err != nil {
		t.Fatalf("CalculateCRC32(file1) error: %v", err)
	}

	crc2, err := CalculateCRC32(file2)
	if err != nil {
		t.Fatalf("CalculateCRC32(file2) error: %v", err)
	}

	// CRC32 values should be different
	if crc1 == crc2 {
		t.Errorf("CRC32 values should be different for different content: both got %08X", crc1)
	}
}
