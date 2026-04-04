package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lepinkainen/videotagger/duplicates"
)

// formatFileSize converts bytes to human-readable format
func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// DuplicatesModel represents the TUI model for duplicate file management
type DuplicatesModel struct {
	// Data
	groups       []duplicates.DuplicateGroup
	currentGroup int
	currentFile  int

	// Global selection tracking
	totalSelectedCount   int
	groupsWithSelections map[int]int // groupIndex -> count of selected files

	// UI state
	width  int
	height int

	// Interaction state
	confirmingDeletion bool
	pendingDeletion    []string // files pending deletion
	showHelp           bool

	// Control state
	quitting bool
}

// NewDuplicatesModel creates a new duplicates TUI model
func NewDuplicatesModel(duplicatePaths map[string][]string) DuplicatesModel {
	groups := duplicates.BuildGroups(duplicatePaths)

	return DuplicatesModel{
		groups:               groups,
		currentGroup:         0,
		currentFile:          0,
		showHelp:             true,
		totalSelectedCount:   0,
		groupsWithSelections: make(map[int]int),
	}
}

// Init implements tea.Model
func (m DuplicatesModel) Init() tea.Cmd {
	return nil
}

// recalculateSelectionStats recalculates the total selected count and groups with selections
func (m *DuplicatesModel) recalculateSelectionStats() {
	m.totalSelectedCount, m.groupsWithSelections = duplicates.RecalculateSelectionStats(m.groups)
}

// applyAutoSelectStrategy applies an auto-selection strategy to the current group
// Selects all files EXCEPT the one to keep based on the strategy
func (m *DuplicatesModel) applyAutoSelectStrategy(strategy duplicates.AutoSelectStrategy) {
	if len(m.groups) == 0 {
		return
	}

	group := &m.groups[m.currentGroup]
	duplicates.ApplyAutoSelectStrategy(group, strategy)

	m.recalculateSelectionStats()
}

// Update implements tea.Model
func (m DuplicatesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmingDeletion {
			return m.handleConfirmationInput(msg)
		}
		return m.handleNormalInput(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case DeletionCompleteMsg:
		m.handleDeletionComplete(msg)
	}

	return m, nil
}

func (m DuplicatesModel) handleNormalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.groups) == 0 {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "h", "?":
		m.showHelp = !m.showHelp

	case "up", "k":
		if m.currentFile > 0 {
			m.currentFile--
		}

	case "down", "j":
		if m.currentFile < len(m.groups[m.currentGroup].Files)-1 {
			m.currentFile++
		}

	case "left", "p":
		if m.currentGroup > 0 {
			m.currentGroup--
			m.currentFile = 0
		}

	case "right", "n":
		if m.currentGroup < len(m.groups)-1 {
			m.currentGroup++
			m.currentFile = 0
		}

	case " ": // spacebar to toggle selection
		group := &m.groups[m.currentGroup]
		group.Selected[m.currentFile] = !group.Selected[m.currentFile]
		m.recalculateSelectionStats()

	case "a": // select all files in current group
		group := &m.groups[m.currentGroup]
		for i := range group.Selected {
			group.Selected[i] = true
		}
		m.recalculateSelectionStats()

	case "c": // clear all selections in current group
		group := &m.groups[m.currentGroup]
		for i := range group.Selected {
			group.Selected[i] = false
		}
		m.recalculateSelectionStats()

	case "s": // skip current group
		if m.currentGroup < len(m.groups)-1 {
			m.currentGroup++
			m.currentFile = 0
		} else {
			// If this was the last group, quit
			m.quitting = true
			return m, tea.Quit
		}

	case "enter":
		return m.handleDeleteCommand()

	// Auto-selection strategies (1-8)
	case "1":
		m.applyAutoSelectStrategy(duplicates.KeepNewest)
	case "2":
		m.applyAutoSelectStrategy(duplicates.KeepOldest)
	case "3":
		m.applyAutoSelectStrategy(duplicates.KeepLargest)
	case "4":
		m.applyAutoSelectStrategy(duplicates.KeepSmallest)
	case "5":
		m.applyAutoSelectStrategy(duplicates.KeepFirst)
	case "6":
		m.applyAutoSelectStrategy(duplicates.KeepLast)
	case "7":
		m.applyAutoSelectStrategy(duplicates.KeepFirstPosition)
	case "8":
		m.applyAutoSelectStrategy(duplicates.KeepLastPosition)
	}

	return m, nil
}

func (m DuplicatesModel) handleConfirmationInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmingDeletion = false
		return m, m.executeDeleteCommand()

	case "n", "N", "ctrl+c", "esc":
		m.confirmingDeletion = false
		m.pendingDeletion = nil
	}

	return m, nil
}

func (m DuplicatesModel) handleDeleteCommand() (tea.Model, tea.Cmd) {
	selectedFiles := duplicates.CollectSelectedFiles(m.groups)

	if len(selectedFiles) == 0 {
		return m, nil // No files selected anywhere
	}

	m.pendingDeletion = selectedFiles
	m.confirmingDeletion = true
	return m, nil
}

func (m DuplicatesModel) executeDeleteCommand() tea.Cmd {
	return func() tea.Msg {
		failedPath, err := duplicates.DeleteFiles(m.pendingDeletion)
		if err != nil {
			return DeletionCompleteMsg{
				FilePath: failedPath,
				Success:  false,
				Error:    err,
			}
		}
		return DeletionCompleteMsg{
			FilePath: "", // Empty means all successful
			Success:  true,
			Error:    nil,
		}
	}
}

func (m *DuplicatesModel) handleDeletionComplete(msg DeletionCompleteMsg) {
	if msg.Success && msg.FilePath == "" {
		m.groups = duplicates.ApplyDeletion(m.groups, m.pendingDeletion)

		// Handle case where all groups were removed
		if len(m.groups) == 0 {
			m.quitting = true
		} else {
			// Ensure currentGroup is valid
			if m.currentGroup >= len(m.groups) {
				m.currentGroup = len(m.groups) - 1
			}

			// Ensure currentFile is valid for the current group
			if m.currentFile >= len(m.groups[m.currentGroup].Files) {
				m.currentFile = max(len(m.groups[m.currentGroup].Files)-1, 0)
			}

			m.recalculateSelectionStats()
		}
	}

	m.pendingDeletion = nil
}

// View implements tea.Model
func (m DuplicatesModel) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if len(m.groups) == 0 {
		return m.renderNoGroups()
	}

	if m.confirmingDeletion {
		return m.renderConfirmationDialog()
	}

	return m.renderMainView()
}

func (m DuplicatesModel) renderNoGroups() string {
	style := SuccessStyle.MarginTop(2).MarginLeft(2)
	return style.Render("✅ All duplicates have been processed!\n\nPress 'q' to quit.")
}

func (m DuplicatesModel) renderConfirmationDialog() string {
	var content strings.Builder

	content.WriteString(HeaderStyle.Render("⚠️  Confirm Deletion"))
	content.WriteString("\n\n")
	fmt.Fprintf(&content, "Are you sure you want to delete %d file(s)?\n\n", len(m.pendingDeletion))

	for _, file := range m.pendingDeletion {
		fmt.Fprintf(&content, "  • %s\n", file)
	}

	content.WriteString("\n")
	content.WriteString(ErrorStyle.Render("This action cannot be undone!"))
	content.WriteString("\n\n")
	content.WriteString("Press 'y' to confirm, 'n' to cancel")

	return content.String()
}

func (m DuplicatesModel) renderMainView() string {
	var content strings.Builder

	// Header with global selection stats
	header := fmt.Sprintf("VideoTagger - Duplicate File Manager (Group %d of %d) | %d files selected across %d groups",
		m.currentGroup+1, len(m.groups), m.totalSelectedCount, len(m.groupsWithSelections))
	content.WriteString(HeaderStyle.Render(header))
	content.WriteString("\n\n")

	// Group info
	group := m.groups[m.currentGroup]
	groupInfo := fmt.Sprintf("Hash: %s (%d files)", group.Hash, len(group.Files))
	content.WriteString(InfoStyle.Render(groupInfo))
	content.WriteString("\n\n")

	// File list
	content.WriteString(m.renderFileList(group))
	content.WriteString("\n")

	// Help
	if m.showHelp {
		content.WriteString(m.renderHelp())
	} else {
		content.WriteString("Press 'h' for help")
	}

	return content.String()
}

func (m DuplicatesModel) renderFileList(group duplicates.DuplicateGroup) string {
	var content strings.Builder

	// Extract paths from metadata for optimizePaths
	paths := make([]string, len(group.Files))
	for i, file := range group.Files {
		paths[i] = file.Path
	}

	// Calculate optimized paths for display
	optimizedPaths := optimizePaths(paths)

	for i, file := range group.Files {
		// Line 1: Selection indicator and filename
		var line strings.Builder

		// Selection indicator
		if group.Selected[i] {
			line.WriteString("[✓] ")
		} else {
			line.WriteString("[ ] ")
		}

		// File path
		fileName := filepath.Base(file.Path)
		displayPath := optimizedPaths[i]

		// Highlight current file
		if i == m.currentFile {
			if group.Selected[i] {
				line.WriteString(SuccessStyle.Reverse(true).Render(fileName))
			} else {
				line.WriteString(lipgloss.NewStyle().Reverse(true).Render(fileName))
			}
		} else {
			if group.Selected[i] {
				line.WriteString(SuccessStyle.Render(fileName))
			} else {
				line.WriteString(fileName)
			}
		}

		fmt.Fprintf(&line, " (%s)", displayPath)
		content.WriteString(line.String())
		content.WriteString("\n")

		// Line 2: Metadata (resolution, duration, size, mod time)
		var metadata strings.Builder
		metadata.WriteString("    └─ ")

		// Resolution
		if file.Resolution != "" {
			metadata.WriteString(file.Resolution)
		} else {
			metadata.WriteString("???")
		}

		metadata.WriteString(" | ")

		// Duration
		if file.DurationMins > 0 {
			fmt.Fprintf(&metadata, "%dmin", file.DurationMins)
		} else {
			metadata.WriteString("?min")
		}

		metadata.WriteString(" | ")

		// File size
		metadata.WriteString(formatFileSize(file.Size))

		metadata.WriteString(" | ")

		// Modification time (date and time)
		if file.ModTime != 0 {
			metadata.WriteString(time.Unix(file.ModTime, 0).Format("2006-01-02 15:04"))
		} else {
			metadata.WriteString("unknown date")
		}

		// Render metadata line with faint style
		content.WriteString(InfoStyle.Faint(true).Render(metadata.String()))
		content.WriteString("\n")
	}

	return content.String()
}

// optimizePaths finds the common path prefix and returns optimized display paths
// that show only the meaningful differences, keeping the topmost directory for context
func optimizePaths(paths []string) []string {
	if len(paths) <= 1 {
		return paths
	}

	// Split all paths into components
	pathComponents := make([][]string, len(paths))
	for i, path := range paths {
		pathComponents[i] = strings.Split(filepath.Clean(path), string(filepath.Separator))
	}

	// Find the common prefix length (excluding the root if it's empty)
	commonPrefixLength := 0
	if len(pathComponents[0]) > 0 {
		maxLength := len(pathComponents[0])
		for _, components := range pathComponents[1:] {
			if len(components) < maxLength {
				maxLength = len(components)
			}
		}

		// Find common prefix
		for i := range maxLength {
			first := pathComponents[0][i]
			allMatch := true
			for j := 1; j < len(pathComponents); j++ {
				if pathComponents[j][i] != first {
					allMatch = false
					break
				}
			}
			if allMatch {
				commonPrefixLength = i + 1
			} else {
				break
			}
		}
	}

	// Generate optimized paths
	result := make([]string, len(paths))
	for i, components := range pathComponents {
		// Keep at least one directory for context, but remove common prefix
		startIndex := commonPrefixLength
		if startIndex > 0 && len(components) > startIndex {
			startIndex = commonPrefixLength - 1 // Keep one level of context
		}
		if startIndex < 0 {
			startIndex = 0
		}

		// Build the optimized path
		if startIndex < len(components) {
			optimizedComponents := components[startIndex:]
			result[i] = filepath.Join(optimizedComponents...)

			// Add leading separator if we removed some components
			if startIndex > 0 {
				result[i] = "..." + string(filepath.Separator) + result[i]
			}
		} else {
			result[i] = paths[i] // Fallback to original path
		}
	}

	return result
}

func (m DuplicatesModel) renderHelp() string {
	help := []string{
		"",
		"Navigation:",
		"  ↑/↓ or j/k   Navigate files in current group",
		"  ←/→ or p/n   Previous/Next duplicate group",
		"",
		"Selection:",
		"  Space        Toggle file selection",
		"  a            Select all files in group",
		"  c            Clear all selections in group",
		"",
		"Auto-Select Strategies (select all EXCEPT the one to keep):",
		"  1            Keep newest file (by modification time)",
		"  2            Keep oldest file (by modification time)",
		"  3            Keep largest file (by size)",
		"  4            Keep smallest file (by size)",
		"  5            Keep first file (alphabetically)",
		"  6            Keep last file (alphabetically)",
		"  7            Keep first file (in list)",
		"  8            Keep last file (in list)",
		"",
		"Actions:",
		"  Enter        Delete all selected files from all groups (with confirmation)",
		"  s            Skip current group",
		"  h/?          Toggle this help",
		"  q            Quit",
		"",
		fmt.Sprintf("Currently: %d files selected across %d groups", m.totalSelectedCount, len(m.groupsWithSelections)),
		"",
	}

	return strings.Join(help, "\n")
}
