package utils

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestValidateFFmpegDependencies(t *testing.T) {
	// Test when both ffmpeg and ffprobe are available
	ffmpegAvailable := exec.Command("ffmpeg", "-version").Run() == nil
	ffprobeAvailable := exec.Command("ffprobe", "-version").Run() == nil

	if ffmpegAvailable && ffprobeAvailable {
		// Both are available, validation should pass
		err := ValidateFFmpegDependencies()
		if err != nil {
			t.Errorf("Expected validation to pass when both ffmpeg and ffprobe are available, got error: %v", err)
		}
	} else {
		// At least one is missing, validation should fail
		err := ValidateFFmpegDependencies()
		if err == nil {
			t.Error("Expected validation to fail when ffmpeg or ffprobe is missing")
		}

		// Check that error message contains installation instructions
		if !strings.Contains(err.Error(), "Install with:") && !strings.Contains(err.Error(), "Download from") {
			t.Errorf("Expected error message to contain installation instructions, got: %v", err)
		}
	}
}

func TestGetInstallationInstructions(t *testing.T) {
	instructions := getInstallationInstructions()

	// Test that instructions are not empty
	if instructions == "" {
		t.Error("Installation instructions should not be empty")
	}

	// Test platform-specific instructions
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(instructions, "brew install ffmpeg") {
			t.Errorf("Expected macOS instructions to mention brew, got: %s", instructions)
		}
	case "linux":
		if !strings.Contains(instructions, "apt-get install ffmpeg") && !strings.Contains(instructions, "yum install ffmpeg") {
			t.Errorf("Expected Linux instructions to mention package managers, got: %s", instructions)
		}
	case "windows":
		if !strings.Contains(instructions, "ffmpeg.org") && !strings.Contains(instructions, "PATH") {
			t.Errorf("Expected Windows instructions to mention ffmpeg.org and PATH, got: %s", instructions)
		}
	default:
		if !strings.Contains(instructions, "ffmpeg.org") {
			t.Errorf("Expected default instructions to mention ffmpeg.org, got: %s", instructions)
		}
	}
}

func TestValidateFFmpegDependencies_ErrorMessages(t *testing.T) {
	// This test documents the expected error message format
	// We can't easily mock exec.LookPath, so we test with current system state
	err := ValidateFFmpegDependencies()

	if err != nil {
		// If there's an error, it should mention which tool is missing
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "ffmpeg") && !strings.Contains(errorMsg, "ffprobe") {
			t.Errorf("Error message should mention which FFmpeg tool is missing, got: %s", errorMsg)
		}

		// Error message should include installation instructions
		if !strings.Contains(errorMsg, "Install with:") && !strings.Contains(errorMsg, "Download from") {
			t.Errorf("Error message should include installation instructions, got: %s", errorMsg)
		}
	}
}
