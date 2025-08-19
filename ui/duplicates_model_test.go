package ui

import (
	"testing"
)

func TestNewDuplicatesModel(t *testing.T) {
	duplicates := map[string][]string{
		"ABC123": {"file1.mp4", "file2.mp4"},
		"DEF456": {"file3.mp4", "file4.mp4", "file5.mp4"},
	}

	model := NewDuplicatesModel(duplicates)

	if len(model.groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(model.groups))
	}

	if model.currentGroup != 0 {
		t.Errorf("Expected currentGroup to be 0, got %d", model.currentGroup)
	}

	if model.currentFile != 0 {
		t.Errorf("Expected currentFile to be 0, got %d", model.currentFile)
	}
}

func TestNewDuplicatesModelEmptyInput(t *testing.T) {
	duplicates := map[string][]string{}

	model := NewDuplicatesModel(duplicates)

	if len(model.groups) != 0 {
		t.Errorf("Expected 0 groups for empty input, got %d", len(model.groups))
	}
}

func TestDuplicateGroupStructure(t *testing.T) {
	duplicates := map[string][]string{
		"ABC123": {"file1.mp4", "file2.mp4"},
	}

	model := NewDuplicatesModel(duplicates)

	if len(model.groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(model.groups))
	}

	group := model.groups[0]
	if group.Hash != "ABC123" {
		t.Errorf("Expected hash 'ABC123', got '%s'", group.Hash)
	}

	if len(group.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(group.Files))
	}

	if len(group.Selected) != 2 {
		t.Errorf("Expected 2 selection states, got %d", len(group.Selected))
	}

	// Ensure no files are selected by default
	for i, selected := range group.Selected {
		if selected {
			t.Errorf("Expected file %d to be unselected by default", i)
		}
	}
}
