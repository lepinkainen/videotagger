package main

import (
	"fmt"
	"hash/crc32"
	"image"
	_ "image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/corona10/goimagehash"
	"github.com/schollz/progressbar/v3"
)

var Version = "dev"

type CLI struct {
	Tag        TagCmd        `cmd:"" help:"Tag video files with metadata and hash"`
	Duplicates DuplicatesCmd `cmd:"" help:"Find duplicate files by hash"`
	Verify     VerifyCmd     `cmd:"" help:"Verify file hash integrity"`
	Phash      PhashCmd      `cmd:"" help:"Find perceptually similar videos"`
}

type TagCmd struct {
	Files   []string `arg:"" name:"files" help:"Video files to process" type:"existingfile"`
	Workers int      `help:"Number of parallel workers" default:"0"`
}

type DuplicatesCmd struct {
	Directory string `arg:"" name:"directory" help:"Directory to scan for duplicates" type:"existingdir" default:"."`
}

type VerifyCmd struct {
	Files []string `arg:"" name:"files" help:"Video files to verify" type:"existingfile"`
}

type PhashCmd struct {
	Files     []string `arg:"" name:"files" help:"Video files to compare" type:"existingfile"`
	Threshold int      `help:"Hamming distance threshold for similarity (0-64)" default:"10"`
}

// this matches the latter part as closely as possible, grabbing the three segments as groups
// examplevideo_[1280x720][33min][996A868B].wmv
var wasProcessedRegex = regexp.MustCompile(`_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^\.]*$`)

// extractHashFromFilename extracts the CRC32 hash from a processed filename
func extractHashFromFilename(filename string) (string, bool) {
	matches := wasProcessedRegex.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return "", false
	}
	return matches[3], true // Return the hash (3rd capture group)
}

// findDuplicatesByHash scans a directory for video files and groups them by hash
func findDuplicatesByHash(directory string) (map[string][]string, error) {
	hashToFiles := make(map[string][]string)

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !isVideoFile(path) {
			return nil
		}

		if hash, ok := extractHashFromFilename(filepath.Base(path)); ok {
			hashToFiles[hash] = append(hashToFiles[hash], path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Filter out hashes with only one file (not duplicates)
	duplicates := make(map[string][]string)
	for hash, files := range hashToFiles {
		if len(files) > 1 {
			duplicates[hash] = files
		}
	}

	return duplicates, nil
}

// calculateCRC32 calculates the CRC32 hash of a file
func calculateCRC32(filename string) (uint32, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}

	return h.Sum32(), nil
}

// calculateVideoPerceptualHash extracts a frame from video and calculates perceptual hash
func calculateVideoPerceptualHash(videoFile string) (*goimagehash.ImageHash, error) {
	// Create temporary file for extracted frame
	tempFrame := filepath.Join(os.TempDir(), fmt.Sprintf("frame_%d.jpg", os.Getpid()))
	defer os.Remove(tempFrame)

	// Extract frame at 30% through the video
	cmd := exec.Command("ffmpeg", "-i", videoFile, "-ss", "00:00:30", "-vframes", "1", "-f", "image2", "-y", tempFrame)
	err := cmd.Run()
	if err != nil {
		// Try extracting at 10 seconds if percentage fails
		cmd = exec.Command("ffmpeg", "-i", videoFile, "-ss", "10", "-vframes", "1", "-f", "image2", "-y", tempFrame)
		if err = cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to extract frame: %w", err)
		}
	}

	// Calculate perceptual hash of extracted frame
	file, err := os.Open(tempFrame)
	if err != nil {
		return nil, fmt.Errorf("failed to open extracted frame: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate perceptual hash: %w", err)
	}

	return hash, nil
}

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

func (cmd *TagCmd) Run() error {
	fmt.Printf("Video Tagger %s\n", Version)
	fmt.Printf("Processing %d files:\n", len(cmd.Files))

	// Set default worker count to number of CPUs
	workers := cmd.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// If only one file or one worker, process sequentially
	if len(cmd.Files) == 1 || workers == 1 {
		for _, videoFile := range cmd.Files {
			processVideoFile(videoFile)
		}
	} else {
		// Process files in parallel
		jobs := make(chan string, len(cmd.Files))
		var wg sync.WaitGroup

		// Start workers
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for videoFile := range jobs {
					processVideoFile(videoFile)
				}
			}()
		}

		// Send jobs
		for _, videoFile := range cmd.Files {
			jobs <- videoFile
		}
		close(jobs)

		// Wait for completion
		wg.Wait()
	}

	fmt.Printf("\n✅ Processing complete.\n")
	return nil
}

// processVideoFile handles the processing of a single video file
func processVideoFile(videoFile string) {
	fi, err := os.Stat(videoFile)
	if err != nil {
		fmt.Printf("❌ Error processing %s: %v\n", videoFile, err)
		return
	}

	// Directory, skip
	if fi.IsDir() {
		fmt.Printf("%s is a directory.\n", videoFile)
		return
	}

	// Not a video file, skip
	if !isVideoFile(videoFile) {
		fmt.Printf("%s is not a video file, skipping\n", videoFile)
		return
	}

	if wasProcessedRegex.MatchString(videoFile) {
		return
	}

	fileSize := fi.Size()
	bar := progressbar.DefaultBytes(fileSize)
	bar.Describe(videoFile)

	resolution, err := getVideoResolution(videoFile)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	durationMins, err := getVideoDuration(videoFile)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	// Open the file to calculate CRC32
	f, err := os.Open(videoFile)
	if err != nil {
		fmt.Printf("❌ Error opening file for CRC calculation: %v\n", err)
		return
	}
	defer f.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(io.MultiWriter(h, bar), f); err != nil {
		fmt.Printf("❌ Error calculating CRC: %v\n", err)
		return
	}
	crc := h.Sum32()

	ext := filepath.Ext(videoFile)
	newFilename := fmt.Sprintf("%s_[%s][%.0fmin][%08X]%s", videoFile[0:len(videoFile)-len(ext)], resolution, durationMins, crc, ext)

	_ = bar.Finish()

	// Rename the file
	if err := os.Rename(videoFile, newFilename); err != nil {
		fmt.Printf("❌ Error renaming file: %v\n", err)
	} else {
		fmt.Printf("✅ %s\n", newFilename)
	}
}

func (cmd *DuplicatesCmd) Run() error {
	fmt.Printf("Scanning %s for duplicates...\n", cmd.Directory)

	duplicates, err := findDuplicatesByHash(cmd.Directory)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		fmt.Println("✅ No duplicates found")
		return nil
	}

	fmt.Printf("Found %d groups of duplicates:\n", len(duplicates))
	for hash, files := range duplicates {
		fmt.Printf("\nHash %s:\n", hash)
		for _, file := range files {
			fmt.Printf("  - %s\n", file)
		}
	}

	return nil
}

func (cmd *PhashCmd) Run() error {
	if len(cmd.Files) < 2 {
		fmt.Println("❌ Need at least 2 files to compare")
		return nil
	}

	fmt.Printf("Calculating perceptual hashes for %d files...\n", len(cmd.Files))

	type FileHash struct {
		File string
		Hash *goimagehash.ImageHash
	}

	var fileHashes []FileHash

	for _, videoFile := range cmd.Files {
		if !isVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		hash, err := calculateVideoPerceptualHash(videoFile)
		if err != nil {
			fmt.Printf("❌ Error calculating perceptual hash for %s: %v\n", videoFile, err)
			continue
		}

		fileHashes = append(fileHashes, FileHash{File: videoFile, Hash: hash})
		fmt.Printf("✅ Processed %s\n", videoFile)
	}

	fmt.Printf("\nComparing %d files for similarity (threshold: %d):\n", len(fileHashes), cmd.Threshold)

	found := false
	for i := 0; i < len(fileHashes); i++ {
		for j := i + 1; j < len(fileHashes); j++ {
			distance, err := fileHashes[i].Hash.Distance(fileHashes[j].Hash)
			if err != nil {
				fmt.Printf("❌ Error comparing %s and %s: %v\n", fileHashes[i].File, fileHashes[j].File, err)
				continue
			}

			if distance <= cmd.Threshold {
				fmt.Printf("🎯 Similar (distance %d): %s ↔ %s\n", distance, fileHashes[i].File, fileHashes[j].File)
				found = true
			}
		}
	}

	if !found {
		fmt.Println("✅ No similar files found within threshold")
	}

	return nil
}

func (cmd *VerifyCmd) Run() error {
	fmt.Printf("Verifying %d files...\n", len(cmd.Files))

	var verified, failed int

	for _, videoFile := range cmd.Files {
		if !isVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		expectedHash, ok := extractHashFromFilename(filepath.Base(videoFile))
		if !ok {
			fmt.Printf("⚠️  %s has not been processed (no hash in filename)\n", videoFile)
			continue
		}

		actualHash, err := calculateCRC32(videoFile)
		if err != nil {
			fmt.Printf("❌ Error calculating hash for %s: %v\n", videoFile, err)
			failed++
			continue
		}

		if strings.EqualFold(expectedHash, fmt.Sprintf("%08X", actualHash)) {
			fmt.Printf("✅ %s\n", videoFile)
			verified++
		} else {
			fmt.Printf("❌ %s (expected: %s, got: %08X)\n", videoFile, expectedHash, actualHash)
			failed++
		}
	}

	fmt.Printf("\n✅ Verified: %d, ❌ Failed: %d\n", verified, failed)
	return nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
