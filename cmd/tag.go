package cmd

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/utils"
	"github.com/lepinkainen/videotagger/video"
)

type TagCmd struct {
	Files   []string `arg:"" name:"files" help:"Video files to process" type:"path"`
	Workers int      `help:"Number of parallel workers" default:"0"`
}

func (cmd *TagCmd) Run() error {
	version := "dev" // TODO: Pass version from main package
	// Set default worker count based on drive type
	workers := cmd.Workers
	if workers <= 0 {
		// Check if any files are on network drives
		hasNetworkFiles := false
		for _, file := range cmd.Files {
			if utils.IsNetworkDrive(file) {
				hasNetworkFiles = true
				break
			}
		}

		if hasNetworkFiles {
			workers = 1 // Use single worker for network drives
			fmt.Printf("⚠️  Network drive detected, using 1 worker for optimal performance\n")
		} else {
			workers = runtime.NumCPU() // Use all CPUs for local drives
		}
	}

	// Use TUI for multiple files with multiple workers
	if len(cmd.Files) > 1 && workers > 1 {
		return cmd.runWithTUI(workers, version)
	}

	// Fall back to simple mode for single file or single worker
	fmt.Println(ui.HeaderStyle.Render(fmt.Sprintf("Video Tagger %s", version)))
	fmt.Println(ui.ProcessingStyle.Render(fmt.Sprintf("Processing %d files:", len(cmd.Files))))

	for _, videoFile := range cmd.Files {
		video.ProcessVideoFile(videoFile)
	}

	fmt.Printf("\n%s\n", ui.SuccessStyle.Render("✅ Processing complete."))
	return nil
}

// runWithTUI runs the tag command with TUI interface
func (cmd *TagCmd) runWithTUI(workers int, version string) error {
	// For now, fall back to simple mode while we develop the TUI
	// TODO: Implement full TUI integration
	fmt.Println(ui.HeaderStyle.Render(fmt.Sprintf("Video Tagger %s (TUI Mode)", version)))
	fmt.Println(ui.ProcessingStyle.Render(fmt.Sprintf("Processing %d files with %d workers:", len(cmd.Files), workers)))

	// Process files in parallel (without TUI for now)
	jobs := make(chan string, len(cmd.Files))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for videoFile := range jobs {
				fmt.Printf("Worker %d: Processing %s\n", workerID+1, videoFile)
				video.ProcessVideoFile(videoFile)
			}
		}(i)
	}

	// Send jobs
	for _, videoFile := range cmd.Files {
		jobs <- videoFile
	}
	close(jobs)

	// Wait for completion
	wg.Wait()

	fmt.Printf("\n%s\n", ui.SuccessStyle.Render("✅ Processing complete."))
	return nil
}

// TODO: Complete TUI implementation in future phase
