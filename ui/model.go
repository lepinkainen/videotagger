package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

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

	// Version for display
	Version string
}

// NewTUIModel creates a new TUI model
func NewTUIModel(numFiles, numWorkers int, version string) TUIModel {
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
		Version:         version,
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
	header := HeaderStyle.Render(fmt.Sprintf("VideoTagger %s", m.Version))

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
