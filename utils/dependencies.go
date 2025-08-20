package utils

import (
	"fmt"
	"os/exec"
	"runtime"
)

// ValidateFFmpegDependencies checks if ffmpeg and ffprobe are available in PATH
func ValidateFFmpegDependencies() error {
	// Check for ffprobe
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return fmt.Errorf("ffprobe not found in PATH. %s", getInstallationInstructions())
	}

	// Check for ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH. %s", getInstallationInstructions())
	}

	return nil
}

// getInstallationInstructions returns platform-specific installation instructions
func getInstallationInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install with: brew install ffmpeg"
	case "linux":
		return "Install with: apt-get install ffmpeg (Ubuntu/Debian) or yum install ffmpeg (CentOS/RHEL)"
	case "windows":
		return "Download from https://ffmpeg.org/download.html and add to PATH"
	default:
		return "Download from https://ffmpeg.org/download.html"
	}
}
