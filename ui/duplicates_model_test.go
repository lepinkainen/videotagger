package ui

import "testing"

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

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{2147483648, "2.0 GB"},
	}

	for _, tt := range tests {
		result := formatFileSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatFileSize(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}
