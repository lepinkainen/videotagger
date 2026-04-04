package duplicates

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildGroups(t *testing.T) {
	dups := map[string][]string{
		"ABC123": {"file1.mp4", "file2.mp4"},
	}

	groups := BuildGroups(dups)
	if len(groups) != 1 {
		t.Fatalf("Expected 1 group, got %d", len(groups))
	}

	group := groups[0]
	if group.Hash != "ABC123" {
		t.Errorf("Expected hash 'ABC123', got '%s'", group.Hash)
	}

	if len(group.Files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(group.Files))
	}

	if len(group.Selected) != 2 {
		t.Errorf("Expected 2 selection states, got %d", len(group.Selected))
	}

	for i, selected := range group.Selected {
		if selected {
			t.Errorf("Expected file %d to be unselected by default", i)
		}
	}
}

func TestExtractMetadataFromFilename(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		expectedRes      string
		expectedDuration int
	}{
		{
			name:             "Valid processed filename",
			filename:         "video_[1920x1080][45min][A1B2C3D4].mp4",
			expectedRes:      "1920x1080",
			expectedDuration: 45,
		},
		{
			name:             "Valid with different resolution",
			filename:         "/path/to/movie_[3840x2160][120min][DEADBEEF].mkv",
			expectedRes:      "3840x2160",
			expectedDuration: 120,
		},
		{
			name:             "Not a processed file",
			filename:         "regular_video.mp4",
			expectedRes:      "",
			expectedDuration: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, duration := ExtractMetadataFromFilename(tt.filename)
			if res != tt.expectedRes {
				t.Errorf("ExtractMetadataFromFilename(%s) resolution = %s, expected %s", tt.filename, res, tt.expectedRes)
			}
			if duration != tt.expectedDuration {
				t.Errorf("ExtractMetadataFromFilename(%s) duration = %d, expected %d", tt.filename, duration, tt.expectedDuration)
			}
		})
	}
}

func TestRecalculateSelectionStats(t *testing.T) {
	groups := []DuplicateGroup{
		{
			Hash:     "ABC123",
			Files:    []FileMetadata{{Path: "file1.mp4"}, {Path: "file2.mp4"}},
			Selected: []bool{true, false},
		},
		{
			Hash:     "DEF456",
			Files:    []FileMetadata{{Path: "file3.mp4"}, {Path: "file4.mp4"}, {Path: "file5.mp4"}},
			Selected: []bool{true, true, false},
		},
		{
			Hash:     "GHI789",
			Files:    []FileMetadata{{Path: "file6.mp4"}, {Path: "file7.mp4"}},
			Selected: []bool{false, false},
		},
	}

	totalSelected, groupsWithSelections := RecalculateSelectionStats(groups)

	if totalSelected != 3 {
		t.Errorf("Expected totalSelectedCount = 3, got %d", totalSelected)
	}

	if len(groupsWithSelections) != 2 {
		t.Errorf("Expected 2 groups with selections, got %d", len(groupsWithSelections))
	}

	if groupsWithSelections[0] != 1 {
		t.Errorf("Expected group 0 to have 1 selected, got %d", groupsWithSelections[0])
	}

	if groupsWithSelections[1] != 2 {
		t.Errorf("Expected group 1 to have 2 selected, got %d", groupsWithSelections[1])
	}
}

func TestFindKeepIndex_KeepNewest(t *testing.T) {
	now := time.Now().Unix()
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: "old.mp4", ModTime: now - int64(2*time.Hour/time.Second)},
			{Path: "newest.mp4", ModTime: now},
			{Path: "older.mp4", ModTime: now - int64(1*time.Hour/time.Second)},
		},
	}

	keepIndex := FindKeepIndex(group, KeepNewest)
	if keepIndex != 1 {
		t.Errorf("KeepNewest: expected index 1 (newest), got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepOldest(t *testing.T) {
	now := time.Now().Unix()
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: "old.mp4", ModTime: now - int64(2*time.Hour/time.Second)},
			{Path: "newest.mp4", ModTime: now},
			{Path: "oldest.mp4", ModTime: now - int64(3*time.Hour/time.Second)},
		},
	}

	keepIndex := FindKeepIndex(group, KeepOldest)
	if keepIndex != 2 {
		t.Errorf("KeepOldest: expected index 2 (oldest), got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepLargest(t *testing.T) {
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: "small.mp4", Size: 1000},
			{Path: "largest.mp4", Size: 5000},
			{Path: "medium.mp4", Size: 3000},
		},
	}

	keepIndex := FindKeepIndex(group, KeepLargest)
	if keepIndex != 1 {
		t.Errorf("KeepLargest: expected index 1 (largest), got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepSmallest(t *testing.T) {
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: "small.mp4", Size: 1000},
			{Path: "largest.mp4", Size: 5000},
			{Path: "smallest.mp4", Size: 500},
		},
	}

	keepIndex := FindKeepIndex(group, KeepSmallest)
	if keepIndex != 2 {
		t.Errorf("KeepSmallest: expected index 2 (smallest), got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepFirst(t *testing.T) {
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: filepath.Join("path", "zebra.mp4")},
			{Path: filepath.Join("path", "aardvark.mp4")},
			{Path: filepath.Join("path", "middle.mp4")},
		},
	}

	keepIndex := FindKeepIndex(group, KeepFirst)
	if keepIndex != 1 {
		t.Errorf("KeepFirst: expected index 1 (aardvark), got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepLast(t *testing.T) {
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: filepath.Join("path", "zebra.mp4")},
			{Path: filepath.Join("path", "aardvark.mp4")},
			{Path: filepath.Join("path", "middle.mp4")},
		},
	}

	keepIndex := FindKeepIndex(group, KeepLast)
	if keepIndex != 0 {
		t.Errorf("KeepLast: expected index 0 (zebra), got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepFirstPosition(t *testing.T) {
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: "first.mp4"},
			{Path: "second.mp4"},
			{Path: "third.mp4"},
		},
	}

	keepIndex := FindKeepIndex(group, KeepFirstPosition)
	if keepIndex != 0 {
		t.Errorf("KeepFirstPosition: expected index 0, got %d", keepIndex)
	}
}

func TestFindKeepIndex_KeepLastPosition(t *testing.T) {
	group := &DuplicateGroup{
		Files: []FileMetadata{
			{Path: "first.mp4"},
			{Path: "second.mp4"},
			{Path: "third.mp4"},
		},
	}

	keepIndex := FindKeepIndex(group, KeepLastPosition)
	if keepIndex != 2 {
		t.Errorf("KeepLastPosition: expected index 2, got %d", keepIndex)
	}
}

func TestApplyAutoSelectStrategy(t *testing.T) {
	now := time.Now().Unix()
	group := &DuplicateGroup{
		Hash: "ABC123",
		Files: []FileMetadata{
			{Path: "old.mp4", ModTime: now - int64(2*time.Hour/time.Second)},
			{Path: "newest.mp4", ModTime: now},
			{Path: "older.mp4", ModTime: now - int64(1*time.Hour/time.Second)},
		},
		Selected: []bool{false, false, false},
	}

	ApplyAutoSelectStrategy(group, KeepNewest)

	if group.Selected[0] != true {
		t.Error("Expected file 0 to be selected")
	}
	if group.Selected[1] != false {
		t.Error("Expected file 1 (newest) to NOT be selected")
	}
	if group.Selected[2] != true {
		t.Error("Expected file 2 to be selected")
	}
}
