package main

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/schollz/progressbar/v3"
)

var Version = "dev"

// this matches the latter part as closely as possible, grabbing the three segments as groups
// examplevideo_[1280x720][33min][996A868B].wmv
var wasProcessedRegex = regexp.MustCompile(`_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^\.]*$`)

// isVideoFile checks if the given file extension is one of knovideo file extensions.
func isVideoFile(path string) bool {
	var desiredExtensions = []string{".mp4", ".webm", ".mov", ".flv", ".mkv", ".avi", ".wmv", ".mpg"}

	ext := filepath.Ext(path)
	ext = strings.ToLower(ext) // handle cases where extension is upper case

	for _, v := range desiredExtensions {
		if v == ext {
			return true
		}
	}
	return false
}

// getVideoResolution extracts the video resolution using ffprobe
func getVideoResolution(videoFile string) (string, error) {
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

// getVideoDuration extracts the video duration using ffprobe
func getVideoDuration(videoFile string) (float64, error) {
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

func main() {
	fmt.Printf("Video Tagger %s\n", Version)

	if len(os.Args) < 2 {
		fmt.Printf("Version: %s\nUsage: %s <video-file(s)>\n", Version, filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	fmt.Printf("Processing %d files:\n", len(os.Args)-1)

	for _, videoFile := range os.Args[1:] {
		//fmt.Printf("\nStarting to process: %s\n", videoFile)

		fi, err := os.Stat(videoFile)
		if err != nil {
			fmt.Printf("❌ Error processing %s: %v\n", videoFile, err)
			continue
		}

		// Directory, skip
		if fi.IsDir() {
			fmt.Printf("%s is a directory.\n", videoFile)
			continue
		}

		// Not a video file, skip
		if !isVideoFile(videoFile) {
			fmt.Printf("%s is not a video file, skipping\n", videoFile)
			continue
		}

		if wasProcessedRegex.MatchString(videoFile) {
			//fmt.Printf("%s already processed.\n", videoFile)
			continue
		}

		fileSize := fi.Size()
		bar := progressbar.DefaultBytes(fileSize)
		bar.Describe(videoFile)

		resolution, err := getVideoResolution(videoFile)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		durationMins, err := getVideoDuration(videoFile)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		// Open the file to calculate CRC32
		f, err := os.Open(videoFile)
		if err != nil {
			fmt.Printf("❌ Error opening file for CRC calculation: %v\n", err)
			continue
		}
		defer f.Close()

		h := crc32.NewIEEE()
		if _, err := io.Copy(io.MultiWriter(h, bar), f); err != nil {
			fmt.Printf("❌ Error calculating CRC: %v\n", err)
			continue
		}
		crc := h.Sum32()

		ext := filepath.Ext(videoFile)
		newFilename := fmt.Sprintf("%s_[%s][%.0fmin][%08X]%s", videoFile[0:len(videoFile)-len(ext)], resolution, durationMins, crc, ext)

		bar.Finish()

		// Rename the file
		if err := os.Rename(videoFile, newFilename); err != nil {
			fmt.Printf("❌ Error renaming file: %v\n", err)
		} else {
			fmt.Printf("✅ %s\n", newFilename)
		}
	}
	fmt.Printf("\n✅ Processing complete.\n")
}
