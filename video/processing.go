package video

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// progressWriter wraps progress bar for io.Writer interface
type progressWriter struct {
	total   int64
	current int64
	prog    progress.Model
	done    chan bool
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.current += int64(n)
	return n, nil
}

func (pw *progressWriter) render() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pw.done:
			// Final render at 100%
			fmt.Printf("\r%s", pw.prog.ViewAs(1.0))
			return
		case <-ticker.C:
			if pw.current > 0 {
				percent := float64(pw.current) / float64(pw.total)
				fmt.Printf("\r%s", pw.prog.ViewAs(percent))
			}
		}
	}
}

// Styling definitions
var (
	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
)

// ProcessVideoFile handles the processing of a single video file
func ProcessVideoFile(videoFile string) {
	fi, err := os.Stat(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error processing %s: %v", videoFile, err)))
		return
	}

	// Directory, skip
	if fi.IsDir() {
		fmt.Printf("%s is a directory.\n", videoFile)
		return
	}

	// Not a video file, skip
	if !IsVideoFile(videoFile) {
		fmt.Printf("%s is not a video file, skipping\n", videoFile)
		return
	}

	if IsProcessed(videoFile) {
		return
	}

	fileSize := fi.Size()

	// Create a custom progress bar with lipgloss styling
	prog := progress.New(progress.WithDefaultGradient())
	fmt.Printf("%s\n", processingStyle.Render(fmt.Sprintf("📊 Processing: %s", videoFile)))

	// Create a progress writer
	progressWriter := &progressWriter{
		total: fileSize,
		prog:  prog,
		done:  make(chan bool),
	}
	go progressWriter.render()

	resolution, err := GetVideoResolution(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error: %v", err)))
		return
	}

	durationMins, err := GetVideoDuration(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error: %v", err)))
		return
	}

	// Open the file to calculate CRC32
	f, err := os.Open(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error opening file for CRC calculation: %v", err)))
		return
	}
	defer f.Close()

	h := crc32.NewIEEE()
	if _, err := io.Copy(io.MultiWriter(h, progressWriter), f); err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error calculating CRC: %v", err)))
		return
	}
	crc := h.Sum32()

	ext := filepath.Ext(videoFile)
	newFilename := fmt.Sprintf("%s_[%s][%.0fmin][%08X]%s", videoFile[0:len(videoFile)-len(ext)], resolution, durationMins, crc, ext)

	progressWriter.done <- true
	fmt.Printf("\n")

	// Rename the file
	if err := os.Rename(videoFile, newFilename); err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error renaming file: %v", err)))
	} else {
		fmt.Printf("%s\n", successStyle.Render(fmt.Sprintf("✅ %s", newFilename)))
	}
}

// FindDuplicatesByHash scans a directory for video files and groups them by CRC32 hash
func FindDuplicatesByHash(directory string) (map[string][]string, error) {
	hashToFiles := make(map[string][]string)

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !IsVideoFile(path) {
			return nil
		}

		if hash, ok := ExtractHashFromFilename(filepath.Base(path)); ok {
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
