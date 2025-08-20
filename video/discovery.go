package video

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindVideoFilesRecursively scans a directory for unprocessed video files
func FindVideoFilesRecursively(directory string) ([]string, error) {
	var files []string
	var err error

	// Use fd if available for better performance, otherwise fall back to filepath.WalkDir
	if isFdAvailable() {
		files, err = findUnprocessedFilesWithFd(directory)
		if err != nil {
			// If fd fails, fall back to the standard method
			files, err = findUnprocessedFilesWithWalkDir(directory)
		}
	} else {
		files, err = findUnprocessedFilesWithWalkDir(directory)
	}

	return files, err
}

// FindDuplicatesByHash scans a directory for video files and groups them by CRC32 hash
func FindDuplicatesByHash(directory string) (map[string][]string, error) {
	hashToFiles := make(map[string][]string)

	var files []string
	var err error

	// Use fd if available for better performance, otherwise fall back to filepath.WalkDir
	if isFdAvailable() {
		files, err = findTaggedFilesWithFd(directory)
		if err != nil {
			// If fd fails, fall back to the standard method
			files, err = findTaggedFilesWithWalkDir(directory)
		}
	} else {
		files, err = findTaggedFilesWithWalkDir(directory)
	}

	if err != nil {
		return nil, err
	}

	// Extract hashes from the found files
	for _, path := range files {
		if hash, ok := ExtractHashFromFilename(filepath.Base(path)); ok {
			hashToFiles[hash] = append(hashToFiles[hash], path)
		}
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

// isFdAvailable checks if the 'fd' command is available in PATH
func isFdAvailable() bool {
	_, err := exec.LookPath("fd")
	return err == nil
}

// findTaggedFilesWithWalkDir uses filepath.WalkDir to find tagged video files (fallback method)
func findTaggedFilesWithWalkDir(directory string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !IsVideoFile(path) {
			return nil
		}

		if IsProcessed(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// findUnprocessedFilesWithWalkDir uses filepath.WalkDir to find unprocessed video files
func findUnprocessedFilesWithWalkDir(directory string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !IsVideoFile(path) {
			return nil
		}

		if !IsProcessed(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// findUnprocessedFilesWithFd uses the 'fd' command to efficiently find unprocessed video files
func findUnprocessedFilesWithFd(directory string) ([]string, error) {
	// Find all video files and filter out processed ones
	videoExts := []string{"mp4", "webm", "mov", "flv", "mkv", "avi", "wmv", "mpg"}
	extPattern := "\\." + strings.Join(videoExts, "|\\.")

	cmd := exec.Command("fd", extPattern, "--type", "f", "--case-sensitive", "false", directory)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" && IsVideoFile(line) && !IsProcessed(line) {
			files = append(files, line)
		}
	}

	return files, nil
}

// findTaggedFilesWithFd uses the 'fd' command to efficiently find tagged video files
func findTaggedFilesWithFd(directory string) ([]string, error) {
	// Pattern matches tagged files: _[resolution][duration][hash].ext
	pattern := `_\[.*\]\[.*min\]\[[a-fA-F0-9]{8}\]\.`

	cmd := exec.Command("fd", pattern, "--type", "f", directory)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" && IsVideoFile(line) {
			files = append(files, line)
		}
	}

	return files, nil
}
