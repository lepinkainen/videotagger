package main

import (
	"fmt"
	_ "image/jpeg"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corona10/goimagehash"
	"github.com/lepinkainen/videotagger/video"
)

var Version = "dev"

// TUI Message Types for worker communication
type WorkerStartedMsg struct {
	WorkerID int
	Filename string
}

type WorkerProgressMsg struct {
	WorkerID int
	Progress float64 // 0.0 to 1.0
	Bytes    int64
	Total    int64
}

type WorkerCompletedMsg struct {
	WorkerID int
	Filename string
	NewName  string
	Success  bool
	Error    error
}

type OverallProgressMsg struct {
	Completed int
	Total     int
}

// File log entry for the processed files list
type FileLogEntry struct {
	OriginalName string
	NewName      string
	Status       string // "✓", "❌", "🔄"
	Error        string
}

func (f FileLogEntry) FilterValue() string { return f.OriginalName }
func (f FileLogEntry) Title() string       { return f.OriginalName }
func (f FileLogEntry) Description() string {
	if f.Error != "" {
		return fmt.Sprintf("❌ %s", f.Error)
	}
	if f.NewName != "" {
		return fmt.Sprintf("✓ → %s", f.NewName)
	}
	return "🔄 Processing..."
}

// Worker state tracking
type WorkerState struct {
	ID          int
	CurrentFile string
	Progress    float64
	Status      string // "idle", "processing", "completed", "error"
	Error       error
}

// TUI Model for the application
type TUIModel struct {
	// Application state
	totalFiles     int
	processedFiles int
	workers        map[int]*WorkerState
	fileEntries    []FileLogEntry

	// UI components
	overallProgress progress.Model
	workerProgress  []progress.Model
	fileList        list.Model

	// Layout
	width  int
	height int

	// Control state
	paused   bool
	quitting bool
}

// NewTUIModel creates a new TUI model
func NewTUIModel(numFiles, numWorkers int) TUIModel {
	// Initialize progress bars
	overallProg := progress.New(progress.WithDefaultGradient())
	workerProgs := make([]progress.Model, numWorkers)
	for i := range workerProgs {
		workerProgs[i] = progress.New(progress.WithDefaultGradient())
	}

	// Initialize workers state
	workers := make(map[int]*WorkerState, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = &WorkerState{
			ID:     i,
			Status: "idle",
		}
	}

	// Initialize file list
	fileItems := []list.Item{}
	fileList := list.New(fileItems, list.NewDefaultDelegate(), 0, 0)
	fileList.Title = "Processed Files"

	return TUIModel{
		totalFiles:      numFiles,
		workers:         workers,
		overallProgress: overallProg,
		workerProgress:  workerProgs,
		fileList:        fileList,
	}
}

// Init implements tea.Model
func (m TUIModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "p":
			m.paused = !m.paused
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fileList.SetSize(msg.Width-4, msg.Height/3)

	case WorkerStartedMsg:
		if worker, ok := m.workers[msg.WorkerID]; ok {
			worker.CurrentFile = msg.Filename
			worker.Status = "processing"
		}

	case WorkerProgressMsg:
		if worker, ok := m.workers[msg.WorkerID]; ok {
			worker.Progress = msg.Progress
		}

	case WorkerCompletedMsg:
		if worker, ok := m.workers[msg.WorkerID]; ok {
			worker.Status = "completed"
			worker.CurrentFile = ""
			worker.Progress = 0
		}

		// Add to file log
		entry := FileLogEntry{
			OriginalName: msg.Filename,
			NewName:      msg.NewName,
			Status:       "✓",
		}
		if !msg.Success {
			entry.Status = "❌"
			entry.Error = msg.Error.Error()
		}

		m.fileEntries = append(m.fileEntries, entry)
		items := make([]list.Item, len(m.fileEntries))
		for i, entry := range m.fileEntries {
			items[i] = entry
		}
		m.fileList.SetItems(items)

	case OverallProgressMsg:
		m.processedFiles = msg.Completed
	}

	return m, nil
}

// View implements tea.Model
func (m TUIModel) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	// Header
	header := headerStyle.Render(fmt.Sprintf("VideoTagger %s", Version))

	// Overall progress
	overallPercent := 0.0
	if m.totalFiles > 0 {
		overallPercent = float64(m.processedFiles) / float64(m.totalFiles)
	}
	overallView := fmt.Sprintf("Overall Progress: %s (%d/%d)",
		m.overallProgress.ViewAs(overallPercent),
		m.processedFiles,
		m.totalFiles)

	// Worker status
	workerViews := []string{"Worker Status:"}
	for i, worker := range m.workers {
		status := fmt.Sprintf("Worker %d: ", i+1)
		if worker.Status == "processing" {
			progBar := m.workerProgress[i].ViewAs(worker.Progress)
			status += fmt.Sprintf("%s %s", progBar, worker.CurrentFile)
		} else {
			status += fmt.Sprintf("%-20s %s", worker.Status, worker.CurrentFile)
		}
		workerViews = append(workerViews, status)
	}

	// File list
	fileListView := m.fileList.View()

	// Controls
	controls := "Controls: [q] Quit  [p] Pause/Resume"

	// Combine all sections
	sections := []string{
		header,
		overallView,
		strings.Join(workerViews, "\n"),
		fileListView,
		controls,
	}

	return strings.Join(sections, "\n\n")
}

// Progress bar functionality moved to video package

// Styling functions using lipgloss
var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Background(lipgloss.Color("235")).
			Bold(true).
			Padding(0, 2).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33"))

	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)
)

type CLI struct {
	Tag        TagCmd        `cmd:"" help:"Tag video files with metadata and hash"`
	Duplicates DuplicatesCmd `cmd:"" help:"Find duplicate files by hash"`
	Verify     VerifyCmd     `cmd:"" help:"Verify file hash integrity"`
	Phash      PhashCmd      `cmd:"" help:"Find perceptually similar videos"`
}

type TagCmd struct {
	Files   []string `arg:"" name:"files" help:"Video files to process" type:"path"`
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

// Video processing functions moved to video package

// Moved to video package

// Moved to video package

// All video processing functions moved to video package

func (cmd *TagCmd) Run() error {
	// Set default worker count to number of CPUs
	workers := cmd.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Use TUI for multiple files with multiple workers
	if len(cmd.Files) > 1 && workers > 1 {
		return cmd.runWithTUI(workers)
	}

	// Fall back to simple mode for single file or single worker
	fmt.Println(headerStyle.Render(fmt.Sprintf("Video Tagger %s", Version)))
	fmt.Println(processingStyle.Render(fmt.Sprintf("Processing %d files:", len(cmd.Files))))

	for _, videoFile := range cmd.Files {
		video.ProcessVideoFile(videoFile)
	}

	fmt.Printf("\n%s\n", successStyle.Render("✅ Processing complete."))
	return nil
}

// runWithTUI runs the tag command with TUI interface
func (cmd *TagCmd) runWithTUI(workers int) error {
	// For now, fall back to simple mode while we develop the TUI
	// TODO: Implement full TUI integration
	fmt.Println(headerStyle.Render(fmt.Sprintf("Video Tagger %s (TUI Mode)", Version)))
	fmt.Println(processingStyle.Render(fmt.Sprintf("Processing %d files with %d workers:", len(cmd.Files), workers)))

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

	fmt.Printf("\n%s\n", successStyle.Render("✅ Processing complete."))
	return nil
}

// TODO: Complete TUI implementation in future phase

// TODO: Complete full TUI implementation in future iterations

// processVideoFile moved to video.ProcessVideoFile

func (cmd *DuplicatesCmd) Run() error {
	fmt.Printf("Scanning %s for duplicates...\n", cmd.Directory)

	duplicates, err := video.FindDuplicatesByHash(cmd.Directory)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		fmt.Printf("%s\n", successStyle.Render("✅ No duplicates found"))
		return nil
	}

	fmt.Printf("%s\n", infoStyle.Render(fmt.Sprintf("Found %d groups of duplicates:", len(duplicates))))
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
		fmt.Printf("%s\n", errorStyle.Render("❌ Need at least 2 files to compare"))
		return nil
	}

	fmt.Printf("%s\n", infoStyle.Render(fmt.Sprintf("Calculating perceptual hashes for %d files...", len(cmd.Files))))

	type FileHash struct {
		File string
		Hash *goimagehash.ImageHash
	}

	var fileHashes []FileHash

	for _, videoFile := range cmd.Files {
		if !video.IsVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		hash, err := video.CalculateVideoPerceptualHash(videoFile)
		if err != nil {
			fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error calculating perceptual hash for %s: %v", videoFile, err)))
			continue
		}

		fileHashes = append(fileHashes, FileHash{File: videoFile, Hash: hash})
		fmt.Printf("%s\n", successStyle.Render(fmt.Sprintf("✅ Processed %s", videoFile)))
	}

	fmt.Printf("\n%s\n", infoStyle.Render(fmt.Sprintf("Comparing %d files for similarity (threshold: %d):", len(fileHashes), cmd.Threshold)))

	found := false
	for i := 0; i < len(fileHashes); i++ {
		for j := i + 1; j < len(fileHashes); j++ {
			distance, err := fileHashes[i].Hash.Distance(fileHashes[j].Hash)
			if err != nil {
				fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error comparing %s and %s: %v", fileHashes[i].File, fileHashes[j].File, err)))
				continue
			}

			if distance <= cmd.Threshold {
				fmt.Printf("🎯 Similar (distance %d): %s ↔ %s\n", distance, fileHashes[i].File, fileHashes[j].File)
				found = true
			}
		}
	}

	if !found {
		fmt.Printf("%s\n", successStyle.Render("✅ No similar files found within threshold"))
	}

	return nil
}

func (cmd *VerifyCmd) Run() error {
	fmt.Printf("%s\n", infoStyle.Render(fmt.Sprintf("Verifying %d files...", len(cmd.Files))))

	var verified, failed int

	for _, videoFile := range cmd.Files {
		if !video.IsVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		expectedHash, ok := video.ExtractHashFromFilename(filepath.Base(videoFile))
		if !ok {
			fmt.Printf("⚠️  %s has not been processed (no hash in filename)\n", videoFile)
			continue
		}

		actualHash, err := video.CalculateCRC32(videoFile)
		if err != nil {
			fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error calculating hash for %s: %v", videoFile, err)))
			failed++
			continue
		}

		if strings.EqualFold(expectedHash, fmt.Sprintf("%08X", actualHash)) {
			fmt.Printf("%s\n", successStyle.Render(fmt.Sprintf("✅ %s", videoFile)))
			verified++
		} else {
			fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ %s (expected: %s, got: %08X)", videoFile, expectedHash, actualHash)))
			failed++
		}
	}

	fmt.Printf("\n%s\n", infoStyle.Render(fmt.Sprintf("✅ Verified: %d, ❌ Failed: %d", verified, failed)))
	return nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
