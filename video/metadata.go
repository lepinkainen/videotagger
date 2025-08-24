package video

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// GetVideoResolution extracts the video resolution using ffprobe
func GetVideoResolution(videoFile string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", "--", videoFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Get the actual error message from ffprobe
		return "", fmt.Errorf("failed to get resolution: %w\nffprobe output: %s", err, string(output))
	}

	// Fix cases where command prints multiple resolutions
	outputParts := strings.SplitN(string(output), "\n", 2)
	resolution := strings.TrimSpace(outputParts[0])
	resolution = strings.TrimSuffix(resolution, "x")

	// Validate resolution format
	if !regexp.MustCompile(`^\d+x\d+$`).MatchString(resolution) {
		return "", fmt.Errorf("invalid resolution format: %s", resolution)
	}

	return resolution, nil
}

// GetVideoDuration extracts the video duration using ffprobe and returns it in minutes
func GetVideoDuration(videoFile string) (float64, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoFile)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get duration: %w", err)
	}

	durationSecs, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return durationSecs / 60, nil
}

// GetVideoCodec extracts the video codec using ffprobe
func GetVideoCodec(videoFile string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=codec_name", "-of", "default=noprint_wrappers=1:nokey=1", videoFile)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get codec: %w", err)
	}

	codec := strings.TrimSpace(string(output))
	if codec == "" {
		return "", fmt.Errorf("could not detect video codec")
	}

	return codec, nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filePath string) (int64, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file size: %w", err)
	}
	return fi.Size(), nil
}
