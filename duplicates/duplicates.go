package duplicates

import (
	"os"
	"path/filepath"

	"github.com/lepinkainen/videotagger/video"
)

// FileMetadata represents metadata for a single file.
type FileMetadata struct {
	Path         string `json:"path"`
	Size         int64  `json:"size"`
	ModTime      int64  `json:"modTime"`
	Resolution   string `json:"resolution"`
	DurationMins int    `json:"durationMins"`
}

// DuplicateGroup represents a group of duplicate files with the same hash.
type DuplicateGroup struct {
	Hash         string         `json:"hash"`
	Files        []FileMetadata `json:"files"`
	Selected     []bool         `json:"selected"`
	DeletedFiles []FileMetadata `json:"deletedFiles"`
}

// AutoSelectStrategy defines strategies for auto-selecting files to delete.
type AutoSelectStrategy int

const (
	KeepNewest        AutoSelectStrategy = iota // Keep newest file by mod time.
	KeepOldest                                  // Keep oldest file by mod time.
	KeepLargest                                 // Keep largest file by size.
	KeepSmallest                                // Keep smallest file by size.
	KeepFirst                                   // Keep first file alphabetically.
	KeepLast                                    // Keep last file alphabetically.
	KeepFirstPosition                           // Keep first file in list.
	KeepLastPosition                            // Keep last file in list.
)

// BuildGroups converts duplicate path groups into enriched duplicate groups.
func BuildGroups(duplicates map[string][]string) []DuplicateGroup {
	groups := make([]DuplicateGroup, 0, len(duplicates))

	for hash, filePaths := range duplicates {
		fileMetadata := make([]FileMetadata, 0, len(filePaths))
		for _, path := range filePaths {
			metadata := FileMetadata{
				Path: path,
			}

			metadata.Resolution, metadata.DurationMins = ExtractMetadataFromFilename(path)

			if stat, err := os.Stat(path); err == nil {
				metadata.Size = stat.Size()
				metadata.ModTime = stat.ModTime().Unix()
			}

			fileMetadata = append(fileMetadata, metadata)
		}

		group := DuplicateGroup{
			Hash:         hash,
			Files:        fileMetadata,
			Selected:     make([]bool, len(fileMetadata)),
			DeletedFiles: make([]FileMetadata, 0),
		}
		groups = append(groups, group)
	}

	return groups
}

// ExtractMetadataFromFilename extracts resolution and duration from a processed filename.
func ExtractMetadataFromFilename(path string) (resolution string, durationMins int) {
	filename := filepath.Base(path)
	resolution, durationMins, _, ok := video.ExtractMetadataFromFilename(filename)
	if !ok {
		return "", 0
	}

	return resolution, durationMins
}

// RecalculateSelectionStats computes global selection stats for all groups.
func RecalculateSelectionStats(groups []DuplicateGroup) (totalSelected int, groupsWithSelections map[int]int) {
	totalSelected = 0
	groupsWithSelections = make(map[int]int)

	for groupIndex, group := range groups {
		selectedInGroup := 0
		for _, selected := range group.Selected {
			if selected {
				selectedInGroup++
				totalSelected++
			}
		}
		if selectedInGroup > 0 {
			groupsWithSelections[groupIndex] = selectedInGroup
		}
	}

	return totalSelected, groupsWithSelections
}

// ApplyAutoSelectStrategy applies an auto-selection strategy to a group.
func ApplyAutoSelectStrategy(group *DuplicateGroup, strategy AutoSelectStrategy) {
	if len(group.Files) == 0 {
		return
	}

	for i := range group.Selected {
		group.Selected[i] = false
	}

	keepIndex := FindKeepIndex(group, strategy)
	for i := range group.Selected {
		if i != keepIndex {
			group.Selected[i] = true
		}
	}
}

// FindKeepIndex determines which file to keep based on the strategy.
func FindKeepIndex(group *DuplicateGroup, strategy AutoSelectStrategy) int {
	if len(group.Files) == 0 {
		return 0
	}

	switch strategy {
	case KeepNewest:
		newestIndex := 0
		for i, file := range group.Files {
			if file.ModTime > group.Files[newestIndex].ModTime {
				newestIndex = i
			}
		}
		return newestIndex

	case KeepOldest:
		oldestIndex := 0
		for i, file := range group.Files {
			if file.ModTime < group.Files[oldestIndex].ModTime {
				oldestIndex = i
			}
		}
		return oldestIndex

	case KeepLargest:
		largestIndex := 0
		for i, file := range group.Files {
			if file.Size > group.Files[largestIndex].Size {
				largestIndex = i
			}
		}
		return largestIndex

	case KeepSmallest:
		smallestIndex := 0
		for i, file := range group.Files {
			if file.Size < group.Files[smallestIndex].Size {
				smallestIndex = i
			}
		}
		return smallestIndex

	case KeepFirst:
		firstIndex := 0
		for i, file := range group.Files {
			if filepath.Base(file.Path) < filepath.Base(group.Files[firstIndex].Path) {
				firstIndex = i
			}
		}
		return firstIndex

	case KeepLast:
		lastIndex := 0
		for i, file := range group.Files {
			if filepath.Base(file.Path) > filepath.Base(group.Files[lastIndex].Path) {
				lastIndex = i
			}
		}
		return lastIndex

	case KeepFirstPosition:
		return 0

	case KeepLastPosition:
		return len(group.Files) - 1

	default:
		return 0
	}
}

// CollectSelectedFiles returns the file paths selected across all groups.
func CollectSelectedFiles(groups []DuplicateGroup) []string {
	var selectedFiles []string
	for _, group := range groups {
		for i, selected := range group.Selected {
			if selected && i < len(group.Files) {
				selectedFiles = append(selectedFiles, group.Files[i].Path)
			}
		}
	}

	return selectedFiles
}

// DeleteFiles removes the selected files, stopping on the first failure.
func DeleteFiles(paths []string) (string, error) {
	for _, filePath := range paths {
		if err := os.Remove(filePath); err != nil {
			return filePath, err
		}
	}

	return "", nil
}

// ApplyDeletion removes deleted files from groups and drops groups with <= 1 file.
func ApplyDeletion(groups []DuplicateGroup, deletedPaths []string) []DuplicateGroup {
	if len(deletedPaths) == 0 {
		return groups
	}

	deletedSet := make(map[string]struct{}, len(deletedPaths))
	for _, path := range deletedPaths {
		deletedSet[path] = struct{}{}
	}

	updated := make([]DuplicateGroup, 0, len(groups))
	for _, group := range groups {
		var remainingFiles []FileMetadata
		var remainingSelected []bool

		for i, file := range group.Files {
			if _, deleted := deletedSet[file.Path]; deleted {
				group.DeletedFiles = append(group.DeletedFiles, file)
				continue
			}

			remainingFiles = append(remainingFiles, file)
			if i < len(group.Selected) {
				remainingSelected = append(remainingSelected, group.Selected[i])
			} else {
				remainingSelected = append(remainingSelected, false)
			}
		}

		group.Files = remainingFiles
		group.Selected = remainingSelected

		if len(group.Files) > 1 {
			updated = append(updated, group)
		}
	}

	return updated
}
