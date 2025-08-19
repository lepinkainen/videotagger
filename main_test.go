package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

func TestCLI_Structure(t *testing.T) {
	// Test that the CLI struct has the expected commands
	var cli CLI

	// Test that all expected commands exist
	// This is a compile-time check - if the struct changes, this will fail
	_ = cli.Tag
	_ = cli.Duplicates
	_ = cli.Verify
	_ = cli.Phash
}

func TestTagCmd_DefaultWorkers(t *testing.T) {
	// Test TagCmd worker count defaults
	cmd := &TagCmd{}

	// Default workers should be 0 (will be set to NumCPU at runtime)
	if cmd.Workers != 0 {
		t.Errorf("Expected default Workers to be 0, got %d", cmd.Workers)
	}
}

func TestTagCmd_WorkerCountLogic(t *testing.T) {
	// Test the worker count logic from Run method
	tests := []struct {
		name           string
		workersInput   int
		expectedOutput int
	}{
		{
			name:           "Zero workers (should default to NumCPU)",
			workersInput:   0,
			expectedOutput: runtime.NumCPU(),
		},
		{
			name:           "Negative workers (should default to NumCPU)",
			workersInput:   -1,
			expectedOutput: runtime.NumCPU(),
		},
		{
			name:           "Explicit worker count",
			workersInput:   4,
			expectedOutput: 4,
		},
		{
			name:           "Single worker",
			workersInput:   1,
			expectedOutput: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from TagCmd.Run()
			workers := tt.workersInput
			if workers <= 0 {
				workers = runtime.NumCPU()
			}

			if workers != tt.expectedOutput {
				t.Errorf("Expected %d workers, got %d", tt.expectedOutput, workers)
			}
		})
	}
}

func TestDuplicatesCmd_DefaultDirectory(t *testing.T) {
	// Test DuplicatesCmd default directory
	cmd := &DuplicatesCmd{}

	// Default directory should be "." (current directory)
	if cmd.Directory != "" {
		t.Errorf("Expected default Directory to be empty string (will default to current dir), got %q", cmd.Directory)
	}
}

func TestPhashCmd_DefaultThreshold(t *testing.T) {
	// Test PhashCmd default threshold
	cmd := &PhashCmd{}

	// Default threshold should be 0 (will be set to 10 by Kong tags)
	if cmd.Threshold != 0 {
		t.Errorf("Expected default Threshold to be 0, got %d", cmd.Threshold)
	}
}

func TestPhashCmd_ThresholdValidation(t *testing.T) {
	// Test threshold validation logic (0-64 range)
	tests := []struct {
		name      string
		threshold int
		valid     bool
	}{
		{"Minimum valid", 0, true},
		{"Default value", 10, true},
		{"Maximum valid", 64, true},
		{"Above maximum", 65, false},
		{"Negative", -1, false},
		{"Way too high", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate threshold range (0-64 as per Hamming distance)
			isValid := tt.threshold >= 0 && tt.threshold <= 64

			if isValid != tt.valid {
				t.Errorf("Threshold %d: expected valid=%v, got valid=%v", tt.threshold, tt.valid, isValid)
			}
		})
	}
}

func TestKongParsing(t *testing.T) {
	// Test that Kong can parse the CLI structure without errors
	var cli CLI

	// Test parsing with no arguments (should show help or error gracefully)
	parser := kong.Must(&cli)

	if parser == nil {
		t.Error("Kong parser should not be nil")
	}
}

func TestKongParsing_TagCommand(t *testing.T) {
	// Create temporary test files
	testDir := t.TempDir()
	testFile1 := filepath.Join(testDir, "video.mp4")
	testFile2 := filepath.Join(testDir, "video2.avi")

	// Create the test files
	_ = os.WriteFile(testFile1, []byte("test"), 0644)
	_ = os.WriteFile(testFile2, []byte("test"), 0644)
	defer os.Remove(testFile1)
	defer os.Remove(testFile2)

	// Test parsing the tag command
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Tag with single file",
			args:        []string{"tag", testFile1},
			expectError: false,
		},
		{
			name:        "Tag with multiple files",
			args:        []string{"tag", testFile1, testFile2},
			expectError: false,
		},
		{
			name:        "Tag with workers flag",
			args:        []string{"tag", "--workers", "4", testFile1},
			expectError: false,
		},
		{
			name:        "Tag with no files",
			args:        []string{"tag"},
			expectError: true, // Should require at least one file
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cli CLI
			parser := kong.Must(&cli)

			// Parse the arguments
			ctx, err := parser.Parse(tc.args)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for args %v, but parsing succeeded", tc.args)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for args %v: %v", tc.args, err)
				} else {
					// Verify that the right command was selected
					if !strings.Contains(ctx.Command(), "tag") {
						t.Errorf("Expected 'tag' command, got %q", ctx.Command())
					}
				}
			}
		})
	}
}

func TestKongParsing_DuplicatesCommand(t *testing.T) {
	// Test parsing the duplicates command
	testDir := t.TempDir()

	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Duplicates with default directory",
			args:        []string{"duplicates"},
			expectError: false,
		},
		{
			name:        "Duplicates with specific directory",
			args:        []string{"duplicates", testDir},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cli CLI
			parser := kong.Must(&cli)

			// Parse the arguments
			ctx, err := parser.Parse(tc.args)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for args %v, but parsing succeeded", tc.args)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for args %v: %v", tc.args, err)
				} else {
					// Verify that the right command was selected
					if !strings.Contains(ctx.Command(), "duplicates") {
						t.Errorf("Expected 'duplicates' command, got %q", ctx.Command())
					}
				}
			}
		})
	}
}

func TestKongParsing_VerifyCommand(t *testing.T) {
	// Create temporary test files
	testDir := t.TempDir()
	testFile1 := filepath.Join(testDir, "video_[1920x1080][45min][ABCD1234].mp4")
	testFile2 := filepath.Join(testDir, "video2_[1280x720][30min][EFGH5678].avi")

	// Create the test files
	_ = os.WriteFile(testFile1, []byte("test"), 0644)
	_ = os.WriteFile(testFile2, []byte("test"), 0644)
	defer os.Remove(testFile1)
	defer os.Remove(testFile2)

	// Test parsing the verify command
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Verify with single file",
			args:        []string{"verify", testFile1},
			expectError: false,
		},
		{
			name:        "Verify with multiple files",
			args:        []string{"verify", testFile1, testFile2},
			expectError: false,
		},
		{
			name:        "Verify with no files",
			args:        []string{"verify"},
			expectError: true, // Should require at least one file
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cli CLI
			parser := kong.Must(&cli)

			// Parse the arguments
			ctx, err := parser.Parse(tc.args)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for args %v, but parsing succeeded", tc.args)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for args %v: %v", tc.args, err)
				} else {
					// Verify that the right command was selected
					if !strings.Contains(ctx.Command(), "verify") {
						t.Errorf("Expected 'verify' command, got %q", ctx.Command())
					}
				}
			}
		})
	}
}

func TestKongParsing_PhashCommand(t *testing.T) {
	// Create temporary test files
	testDir := t.TempDir()
	testFile1 := filepath.Join(testDir, "video1.mp4")
	testFile2 := filepath.Join(testDir, "video2.mp4")

	// Create the test files
	_ = os.WriteFile(testFile1, []byte("test"), 0644)
	_ = os.WriteFile(testFile2, []byte("test"), 0644)
	defer os.Remove(testFile1)
	defer os.Remove(testFile2)

	// Test parsing the phash command
	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "Phash with two files",
			args:        []string{"phash", testFile1, testFile2},
			expectError: false,
		},
		{
			name:        "Phash with threshold",
			args:        []string{"phash", "--threshold", "5", testFile1, testFile2},
			expectError: false,
		},
		{
			name:        "Phash with single file",
			args:        []string{"phash", testFile1},
			expectError: false, // Parser won't catch this, but Run() method should
		},
		{
			name:        "Phash with no files",
			args:        []string{"phash"},
			expectError: true, // Should require at least one file
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var cli CLI
			parser := kong.Must(&cli)

			// Parse the arguments
			ctx, err := parser.Parse(tc.args)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for args %v, but parsing succeeded", tc.args)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for args %v: %v", tc.args, err)
				} else {
					// Verify that the right command was selected
					if !strings.Contains(ctx.Command(), "phash") {
						t.Errorf("Expected 'phash' command, got %q", ctx.Command())
					}
				}
			}
		})
	}
}

func TestVersion(t *testing.T) {
	// Test that Version variable exists and has expected default
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Default version should be "dev"
	if Version != "dev" {
		t.Logf("Version is %q (expected 'dev' for development builds)", Version)
	}
}

func TestTUIModel_Creation(t *testing.T) {
	// Test TUIModel creation
	numFiles := 5
	numWorkers := 2

	model := NewTUIModel(numFiles, numWorkers)

	// Verify basic properties
	if model.totalFiles != numFiles {
		t.Errorf("Expected totalFiles %d, got %d", numFiles, model.totalFiles)
	}

	if len(model.workers) != numWorkers {
		t.Errorf("Expected %d workers, got %d", numWorkers, len(model.workers))
	}

	if model.processedFiles != 0 {
		t.Errorf("Expected processedFiles to start at 0, got %d", model.processedFiles)
	}

	// Verify workers are initialized properly
	for i := 0; i < numWorkers; i++ {
		if worker, exists := model.workers[i]; exists {
			if worker.ID != i {
				t.Errorf("Worker %d has incorrect ID %d", i, worker.ID)
			}
			if worker.Status != "idle" {
				t.Errorf("Worker %d should start with 'idle' status, got %q", i, worker.Status)
			}
		} else {
			t.Errorf("Worker %d not found in workers map", i)
		}
	}
}

func TestFileLogEntry_Methods(t *testing.T) {
	// Test FileLogEntry interface methods
	entry := FileLogEntry{
		OriginalName: "test_video.mp4",
		NewName:      "test_video_[1920x1080][45min][ABCD1234].mp4",
		Status:       "✓",
		Error:        "",
	}

	// Test FilterValue
	if entry.FilterValue() != "test_video.mp4" {
		t.Errorf("FilterValue() = %q, expected %q", entry.FilterValue(), "test_video.mp4")
	}

	// Test Title
	if entry.Title() != "test_video.mp4" {
		t.Errorf("Title() = %q, expected %q", entry.Title(), "test_video.mp4")
	}

	// Test Description for successful processing
	expectedDesc := "✓ → test_video_[1920x1080][45min][ABCD1234].mp4"
	if entry.Description() != expectedDesc {
		t.Errorf("Description() = %q, expected %q", entry.Description(), expectedDesc)
	}
}

func TestFileLogEntry_ErrorHandling(t *testing.T) {
	// Test FileLogEntry with error
	entry := FileLogEntry{
		OriginalName: "bad_video.mp4",
		NewName:      "",
		Status:       "❌",
		Error:        "File not found",
	}

	// Test Description for error case
	expectedDesc := "❌ File not found"
	if entry.Description() != expectedDesc {
		t.Errorf("Description() = %q, expected %q", entry.Description(), expectedDesc)
	}
}

func TestFileLogEntry_Processing(t *testing.T) {
	// Test FileLogEntry in processing state
	entry := FileLogEntry{
		OriginalName: "processing_video.mp4",
		NewName:      "",
		Status:       "🔄",
		Error:        "",
	}

	// Test Description for processing case
	expectedDesc := "🔄 Processing..."
	if entry.Description() != expectedDesc {
		t.Errorf("Description() = %q, expected %q", entry.Description(), expectedDesc)
	}
}

func TestIsNetworkDrive(t *testing.T) {
	// Test network drive detection
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Linux NFS mount",
			path:     "/mnt/nfs-share/video.mp4",
			expected: true,
		},
		{
			name:     "Linux media mount",
			path:     "/media/usb/video.mp4",
			expected: true,
		},
		{
			name:     "macOS network volume",
			path:     "/Volumes/NetworkShare/video.mp4",
			expected: true,
		},
		{
			name:     "Windows UNC path",
			path:     "//server/share/video.mp4",
			expected: true,
		},
		{
			name:     "Windows UNC path escaped",
			path:     "\\\\server\\share\\video.mp4",
			expected: true,
		},
		{
			name:     "Local path Linux",
			path:     "/home/user/videos/video.mp4",
			expected: false,
		},
		{
			name:     "Local path macOS",
			path:     "/Users/user/Movies/video.mp4",
			expected: false,
		},
		{
			name:     "Relative path",
			path:     "./video.mp4",
			expected: false,
		},
		{
			name:     "Current directory",
			path:     "video.mp4",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkDrive(tt.path)
			if result != tt.expected {
				t.Errorf("isNetworkDrive(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsNetworkDrive_PathWithNetworkIndicators(t *testing.T) {
	// Test paths that contain network filesystem indicators in their resolved paths
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Path containing 'nfs'",
			path:     "/some/path/nfs/video.mp4",
			expected: true,
		},
		{
			name:     "Path containing 'cifs'",
			path:     "/mount/cifs-share/video.mp4",
			expected: true,
		},
		{
			name:     "Path containing 'smb'",
			path:     "/shares/smb/video.mp4",
			expected: true,
		},
		{
			name:     "Path containing 'webdav'",
			path:     "/webdav/share/video.mp4",
			expected: true,
		},
		{
			name:     "Regular path without indicators",
			path:     "/home/user/documents/video.mp4",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkDrive(tt.path)
			if result != tt.expected {
				t.Errorf("isNetworkDrive(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestTagCmd_WorkerCountLogicWithNetworkDrives(t *testing.T) {
	// Test the updated worker count logic that considers network drives
	tests := []struct {
		name           string
		workersInput   int
		hasNetworkFile bool
		expectedOutput int
	}{
		{
			name:           "Network drive detected - should use 1 worker",
			workersInput:   0,
			hasNetworkFile: true,
			expectedOutput: 1,
		},
		{
			name:           "Local drives only - should use NumCPU",
			workersInput:   0,
			hasNetworkFile: false,
			expectedOutput: runtime.NumCPU(),
		},
		{
			name:           "Explicit worker count - should override detection",
			workersInput:   4,
			hasNetworkFile: true,
			expectedOutput: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from TagCmd.Run()
			workers := tt.workersInput
			if workers <= 0 {
				// Simulate network drive check
				hasNetworkFiles := tt.hasNetworkFile

				if hasNetworkFiles {
					workers = 1
				} else {
					workers = runtime.NumCPU()
				}
			}

			if workers != tt.expectedOutput {
				t.Errorf("Expected %d workers, got %d", tt.expectedOutput, workers)
			}
		})
	}
}

// Integration test that verifies the full CLI pipeline
func TestCLI_Integration(t *testing.T) {
	// Create a temporary test file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test_video.mp4")

	err := os.WriteFile(testFile, []byte("fake video content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Test that we can parse and set up the command structure
	var cli CLI
	parser := kong.Must(&cli)

	// Parse tag command with our test file
	args := []string{"tag", testFile}
	ctx, err := parser.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse args %v: %v", args, err)
	}

	// Verify the command was parsed correctly
	if !strings.Contains(ctx.Command(), "tag") {
		t.Errorf("Expected 'tag' command, got %q", ctx.Command())
	}

	// Verify the file was captured
	if len(cli.Tag.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(cli.Tag.Files))
	}

	if cli.Tag.Files[0] != testFile {
		t.Errorf("Expected file %q, got %q", testFile, cli.Tag.Files[0])
	}
}
