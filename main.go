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

var debug = false

// this matches the latter part as closely as possible, grabbing the three segments as groups
// examplevideo_[1280x720][33min][996A868B].wmv
var wasProcessedRegex = regexp.MustCompile(`_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^\.]*$`)

// isVideoFile checks if the given file extension is one of knovideo file extensions.
func isVideoFile(path string) bool {
	var desiredExtensions = []string{".mp4", ".webm", ".mov", ".flv", ".mkv", ".avi", ".wmv", ".mpg"}

	ext := filepath.Ext(path)
	for _, v := range desiredExtensions {
		if v == ext {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a file to process.")
		os.Exit(1)
	}
	videoFile := os.Args[1]

	fi, err := os.Stat(videoFile)
	if err != nil {
		fmt.Println("An error occurred: ", err)
		os.Exit(1)
	}
	fileSize := fi.Size()

	// Directory, skip
	if fi.IsDir() {
		fmt.Printf(videoFile + " is a directory.\n")
		os.Exit(1)
	}

	// Not a video file, skip
	if !isVideoFile(videoFile) {
		//fmt.Printf(videoFile + " is not a video file.\n")
		os.Exit(1)
	}

	if wasProcessedRegex.MatchString(videoFile) {
		fmt.Printf(videoFile + " already processed.\n")
		os.Exit(1)
	}

	bar := progressbar.DefaultBytes(fileSize)
	bar.Describe(videoFile)

	// Get resolution
	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", "--", videoFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("An error occurred: ", err)
	}

	// in some cases the command prints <resolution>\n\n<resolution>, fix it here
	outputParts := strings.SplitN(string(output), "\n", 2)
	firstPart := outputParts[0]

	// Extract the resolution and build a new filename
	resolution := strings.TrimSpace(string(firstPart))
	// sometimes the system will return a resolution like 1280x720x - remove the trailing x
	resolution = strings.TrimSuffix(resolution, "x")

	ext := filepath.Ext(videoFile)

	// Get duration
	cmd = exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoFile)
	output, err = cmd.Output()
	if err != nil {
		fmt.Println("An error occurred when getting duration: ", err)
	}

	durationSecs, _ := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	durationMins := durationSecs / 60

	// Open the file to calculate CRC32
	f, err := os.Open(videoFile)
	if err != nil {
		fmt.Println("Error opening file for CRC calculation: ", err)
		os.Exit(1)
	}
	defer f.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(io.MultiWriter(h, bar), f); err != nil {
		fmt.Println("Error calculating CRC: ", err)
		os.Exit(1)
	}
	crc := h.Sum32()

	//newFilename := fmt.Sprintf("%s_[%s]%s", videoFile[0:len(videoFile)-len(ext)], resolution, ext)
	newFilename := fmt.Sprintf("%s_[%s][%.0fmin][%08X]%s", videoFile[0:len(videoFile)-len(ext)], resolution, durationMins, crc, ext)

	bar.Finish()

	// Rename the file
	if err := os.Rename(videoFile, newFilename); err != nil {
		fmt.Println("Error renaming file: ", err)
	} else {
		fmt.Printf("-> %s\n", newFilename)
	}
}
