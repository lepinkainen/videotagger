package cmd

import (
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/lepinkainen/videotagger/types"
	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/utils"
	"github.com/lepinkainen/videotagger/video"
)

type ReencodeCmd struct {
	Files        []string `arg:"" name:"files" help:"Video files to re-encode" type:"path"`
	Workers      int      `help:"Number of parallel workers" default:"0"`
	CRF          int      `help:"Constant Rate Factor for quality (0-51, lower=better)" default:"23"`
	Preset       string   `help:"x265 encoding preset" default:"medium" enum:"ultrafast,superfast,veryfast,faster,fast,medium,slow,slower,veryslow,placebo"`
	MinSavings   float64  `help:"Minimum size reduction required (0.0-1.0)" default:"0.20"`
	KeepOriginal bool     `help:"Keep original files as .bak"`
	DryRun       bool     `help:"Show what would be processed without making changes"`
}

func (cmd *ReencodeCmd) Run(appCtx *types.AppContext) error {
	version := types.DefaultVersion
	if appCtx != nil {
		version = appCtx.Version
	}

	// Expand directories to video files
	expandedFiles, err := cmd.ExpandDirectories()
	if err != nil {
		return fmt.Errorf("failed to expand directories: %w", err)
	}
	cmd.Files = expandedFiles

	// Filter out already processed files if not in dry-run mode
	if !cmd.DryRun {
		filtered, skipped := cmd.filterAlreadyH265Files()
		cmd.Files = filtered
		if skipped > 0 {
			fmt.Printf("⏭️  Skipped %d files already encoded with H.265\n", skipped)
		}
	}

	if len(cmd.Files) == 0 {
		fmt.Println("🎯 No files need re-encoding.")
		return nil
	}

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

	// Create re-encode options
	options := &video.ReencodeOptions{
		CRF:          cmd.CRF,
		Preset:       cmd.Preset,
		MinSavings:   cmd.MinSavings,
		KeepOriginal: cmd.KeepOriginal,
	}

	fmt.Println(ui.HeaderStyle.Render(fmt.Sprintf("Video Re-encoder %s", version)))

	if cmd.DryRun {
		fmt.Println(ui.ProcessingStyle.Render("🔍 DRY RUN MODE - No files will be modified"))
		return cmd.runDryRun(options)
	}

	fmt.Println(ui.ProcessingStyle.Render(fmt.Sprintf("🎬 Re-encoding %d files to H.265 with %d workers:", len(cmd.Files), workers)))
	fmt.Printf("⚙️  Settings: CRF=%d, Preset=%s, Min Savings=%.1f%%\n",
		cmd.CRF, cmd.Preset, cmd.MinSavings*100)

	if len(cmd.Files) > 1 && workers > 1 {
		return cmd.runParallel(workers, options)
	}

	// Sequential processing for single file or single worker
	return cmd.runSequential(options)
}

// runDryRun analyzes files without making changes
func (cmd *ReencodeCmd) runDryRun(options *video.ReencodeOptions) error {
	fmt.Printf("📊 Analyzing %d files:\n\n", len(cmd.Files))

	var totalOriginalSize int64
	var estimatedSavings int64
	processableCount := 0

	for _, videoFile := range cmd.Files {
		fmt.Printf("📹 %s\n", videoFile)

		// Get basic file info
		size, err := video.GetFileSize(videoFile)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		codec, err := video.GetVideoCodec(videoFile)
		if err != nil {
			fmt.Printf("   ❌ Error getting codec: %v\n", err)
			continue
		}

		isH265, err := video.IsH265(videoFile)
		if err != nil {
			fmt.Printf("   ❌ Error checking codec: %v\n", err)
			continue
		}

		fmt.Printf("   📏 Size: %.1f MB\n", float64(size)/(1024*1024))
		fmt.Printf("   🎥 Codec: %s\n", codec)

		if isH265 {
			fmt.Printf("   ⏭️  Already H.265, would skip\n")
		} else {
			// Rough estimate: H.265 typically saves 20-50% vs H.264
			estimatedSaving := int64(float64(size) * 0.35) // Conservative 35% estimate
			fmt.Printf("   💾 Estimated saving: ~%.1f MB (35%%)\n", float64(estimatedSaving)/(1024*1024))
			estimatedSavings += estimatedSaving
			processableCount++
		}

		totalOriginalSize += size
		fmt.Println()
	}

	fmt.Printf("📈 Summary:\n")
	fmt.Printf("   Total files: %d\n", len(cmd.Files))
	fmt.Printf("   Would process: %d files\n", processableCount)
	fmt.Printf("   Total size: %.1f MB\n", float64(totalOriginalSize)/(1024*1024))
	fmt.Printf("   Estimated savings: ~%.1f MB\n", float64(estimatedSavings)/(1024*1024))

	if estimatedSavings > 0 {
		savingsPercent := float64(estimatedSavings) / float64(totalOriginalSize) * 100
		fmt.Printf("   Estimated reduction: ~%.1f%%\n", savingsPercent)
	}

	return nil
}

// runSequential processes files one by one
func (cmd *ReencodeCmd) runSequential(options *video.ReencodeOptions) error {
	stats := &reencodeStats{}

	for i, videoFile := range cmd.Files {
		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(cmd.Files), videoFile)
		result := video.ReencodeToH265(videoFile, options)
		cmd.handleResult(result, stats)
	}

	cmd.printSummary(stats)
	return nil
}

// runParallel processes files using worker pools
func (cmd *ReencodeCmd) runParallel(workers int, options *video.ReencodeOptions) error {
	jobs := make(chan string, len(cmd.Files))
	results := make(chan *video.ReencodeResult, len(cmd.Files))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for videoFile := range jobs {
				fmt.Printf("Worker %d: Processing %s\n", workerID+1, videoFile)
				result := video.ReencodeToH265(videoFile, options)
				results <- result
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
	close(results)

	// Process results
	stats := &reencodeStats{}
	for result := range results {
		cmd.handleResult(result, stats)
	}

	cmd.printSummary(stats)
	return nil
}

// handleResult processes a re-encoding result and updates statistics
func (cmd *ReencodeCmd) handleResult(result *video.ReencodeResult, stats *reencodeStats) {
	if result.Error != nil {
		fmt.Printf("%s\n", ui.ErrorStyle.Render(fmt.Sprintf("❌ Error: %v", result.Error)))
		stats.ErrorCount++
		return
	}

	if result.WasSkipped {
		fmt.Printf("⏭️  Skipped: %s\n", result.SkipReason)
		stats.SkippedCount++
		return
	}

	if result.WasReencoded {
		sizeMB := float64(result.OriginalSize) / (1024 * 1024)
		newSizeMB := float64(result.NewSize) / (1024 * 1024)
		savingsMB := float64(result.SizeSavings) / (1024 * 1024)

		fmt.Printf("%s\n", ui.SuccessStyle.Render(fmt.Sprintf("✅ %s → %s", result.OriginalCodec, "H.265")))
		fmt.Printf("   📏 %.1f MB → %.1f MB (saved %.1f MB, %.1f%%)\n",
			sizeMB, newSizeMB, savingsMB, result.SavingsPercent*100)

		stats.ProcessedCount++
		stats.TotalOriginalSize += result.OriginalSize
		stats.TotalNewSize += result.NewSize
		stats.TotalSavings += result.SizeSavings
	}
}

// printSummary displays final statistics
func (cmd *ReencodeCmd) printSummary(stats *reencodeStats) {
	fmt.Printf("\n%s\n", ui.HeaderStyle.Render("📊 Re-encoding Summary"))
	fmt.Printf("   Processed: %d files\n", stats.ProcessedCount)
	fmt.Printf("   Skipped: %d files\n", stats.SkippedCount)
	fmt.Printf("   Errors: %d files\n", stats.ErrorCount)

	if stats.ProcessedCount > 0 {
		totalOriginalMB := float64(stats.TotalOriginalSize) / (1024 * 1024)
		totalNewMB := float64(stats.TotalNewSize) / (1024 * 1024)
		totalSavingsMB := float64(stats.TotalSavings) / (1024 * 1024)
		totalSavingsPercent := float64(stats.TotalSavings) / float64(stats.TotalOriginalSize) * 100

		fmt.Printf("   Total space saved: %.1f MB (%.1f%%)\n", totalSavingsMB, totalSavingsPercent)
		fmt.Printf("   Size reduction: %.1f MB → %.1f MB\n", totalOriginalMB, totalNewMB)
	}

	fmt.Printf("\n%s\n", ui.SuccessStyle.Render("🎉 Re-encoding complete!"))
}

// filterAlreadyH265Files removes files that are already H.265 encoded
func (cmd *ReencodeCmd) filterAlreadyH265Files() (filtered []string, skipped int) {

	for _, file := range cmd.Files {
		isH265, err := video.IsH265(file)
		if err == nil && isH265 {
			skipped++
			continue
		}
		filtered = append(filtered, file)
	}

	return
}

// ExpandDirectories expands any directory arguments into lists of video files
func (cmd *ReencodeCmd) ExpandDirectories() ([]string, error) {
	var expandedFiles []string

	for _, path := range cmd.Files {
		// Check if path exists
		fi, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", path, err)
		}

		if fi.IsDir() {
			// Directory: find all video files recursively
			videoFiles, err := video.FindVideoFilesRecursively(path)
			if err != nil {
				return nil, fmt.Errorf("failed to scan directory %s: %w", path, err)
			}
			expandedFiles = append(expandedFiles, videoFiles...)
		} else {
			// Regular file: add as-is
			expandedFiles = append(expandedFiles, path)
		}
	}

	return expandedFiles, nil
}

// reencodeStats tracks statistics during re-encoding
type reencodeStats struct {
	ProcessedCount    int
	SkippedCount      int
	ErrorCount        int
	TotalOriginalSize int64
	TotalNewSize      int64
	TotalSavings      int64
}
