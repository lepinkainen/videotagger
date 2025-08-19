package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DuplicateGroup represents a group of duplicate files with the same hash
type DuplicateGroup struct {
	Hash         string
	Files        []string
	Selected     []bool   // which files are selected for deletion
	DeletedFiles []string // files that have been successfully deleted
}

// DuplicatesModel represents the TUI model for duplicate file management
type DuplicatesModel struct {
	// Data
	groups       []DuplicateGroup
	currentGroup int
	currentFile  int

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
func NewDuplicatesModel(duplicates map[string][]string) DuplicatesModel {
	var groups []DuplicateGroup

	for hash, files := range duplicates {
		group := DuplicateGroup{
			Hash:         hash,
			Files:        files,
			Selected:     make([]bool, len(files)),
			DeletedFiles: make([]string, 0),
		}
		groups = append(groups, group)
	}

	return DuplicatesModel{
		groups:       groups,
		currentGroup: 0,
		currentFile:  0,
		showHelp:     true,
	}
}

// Init implements tea.Model
func (m DuplicatesModel) Init() tea.Cmd {
	return nil
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

	case "a": // select all files in current group
		group := &m.groups[m.currentGroup]
		for i := range group.Selected {
			group.Selected[i] = true
		}

	case "c": // clear all selections in current group
		group := &m.groups[m.currentGroup]
		for i := range group.Selected {
			group.Selected[i] = false
		}

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
	group := &m.groups[m.currentGroup]
	var selectedFiles []string

	for i, selected := range group.Selected {
		if selected {
			selectedFiles = append(selectedFiles, group.Files[i])
		}
	}

	if len(selectedFiles) == 0 {
		return m, nil // No files selected, do nothing
	}

	m.pendingDeletion = selectedFiles
	m.confirmingDeletion = true
	return m, nil
}

func (m DuplicatesModel) executeDeleteCommand() tea.Cmd {
	return func() tea.Msg {
		for _, filePath := range m.pendingDeletion {
			err := os.Remove(filePath)
			if err != nil {
				return DeletionCompleteMsg{
					FilePath: filePath,
					Success:  false,
					Error:    err,
				}
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
		// All files deleted successfully
		group := &m.groups[m.currentGroup]

		// Remove deleted files from the group
		var remainingFiles []string
		var remainingSelected []bool

		for _, file := range group.Files {
			deleted := false
			for _, deletedFile := range m.pendingDeletion {
				if file == deletedFile {
					deleted = true
					group.DeletedFiles = append(group.DeletedFiles, file)
					break
				}
			}
			if !deleted {
				remainingFiles = append(remainingFiles, file)
				remainingSelected = append(remainingSelected, false) // reset selections
			}
		}

		group.Files = remainingFiles
		group.Selected = remainingSelected

		// If only one file remains, remove this group
		if len(group.Files) <= 1 {
			m.groups = append(m.groups[:m.currentGroup], m.groups[m.currentGroup+1:]...)
			if m.currentGroup >= len(m.groups) {
				if len(m.groups) == 0 {
					m.quitting = true
				} else {
					m.currentGroup = len(m.groups) - 1
				}
			}
		}

		// Reset file selection
		if len(m.groups) > 0 && m.currentFile >= len(m.groups[m.currentGroup].Files) {
			m.currentFile = len(m.groups[m.currentGroup].Files) - 1
			if m.currentFile < 0 {
				m.currentFile = 0
			}
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
	content.WriteString(fmt.Sprintf("Are you sure you want to delete %d file(s)?\n\n", len(m.pendingDeletion)))

	for _, file := range m.pendingDeletion {
		content.WriteString(fmt.Sprintf("  • %s\n", file))
	}

	content.WriteString("\n")
	content.WriteString(ErrorStyle.Render("This action cannot be undone!"))
	content.WriteString("\n\n")
	content.WriteString("Press 'y' to confirm, 'n' to cancel")

	return content.String()
}

func (m DuplicatesModel) renderMainView() string {
	var content strings.Builder

	// Header
	header := fmt.Sprintf("VideoTagger - Duplicate File Manager (Group %d of %d)",
		m.currentGroup+1, len(m.groups))
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

func (m DuplicatesModel) renderFileList(group DuplicateGroup) string {
	var content strings.Builder

	for i, file := range group.Files {
		var line strings.Builder

		// Selection indicator
		if group.Selected[i] {
			line.WriteString("[✓] ")
		} else {
			line.WriteString("[ ] ")
		}

		// File path
		fileName := filepath.Base(file)
		fullPath := file

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

		line.WriteString(fmt.Sprintf(" (%s)", fullPath))
		content.WriteString(line.String())
		content.WriteString("\n")
	}

	return content.String()
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
		"Actions:",
		"  Enter        Delete selected files (with confirmation)",
		"  s            Skip current group",
		"  h/?          Toggle this help",
		"  q            Quit",
		"",
	}

	return strings.Join(help, "\n")
}
