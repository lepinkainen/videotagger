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
	"time"

	"github.com/alecthomas/kong"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/corona10/goimagehash"
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
		processVideoFile(videoFile)
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
				processVideoFile(videoFile)
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

// processVideoFile handles the processing of a single video file
func processVideoFile(videoFile string) {
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
	if !isVideoFile(videoFile) {
		fmt.Printf("%s is not a video file, skipping\n", videoFile)
		return
	}

	if wasProcessedRegex.MatchString(videoFile) {
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

	resolution, err := getVideoResolution(videoFile)
	if err != nil {
		fmt.Printf("%s\n", errorStyle.Render(fmt.Sprintf("❌ Error: %v", err)))
		return
	}

	durationMins, err := getVideoDuration(videoFile)
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

func (cmd *DuplicatesCmd) Run() error {
	fmt.Printf("Scanning %s for duplicates...\n", cmd.Directory)

	duplicates, err := findDuplicatesByHash(cmd.Directory)
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
		if !isVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		hash, err := calculateVideoPerceptualHash(videoFile)
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
