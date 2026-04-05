package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lepinkainen/videotagger/duplicates"
	"github.com/lepinkainen/videotagger/utils"
	"github.com/lepinkainen/videotagger/video"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppState mirrors the current duplicates state for the UI.
type AppState struct {
	Groups               []duplicates.DuplicateGroup `json:"groups"`
	TotalSelectedCount   int                         `json:"totalSelectedCount"`
	GroupsWithSelections map[int]int                 `json:"groupsWithSelections"`
}

// Preview describes a preview payload for a video.
type Preview struct {
	Type  string `json:"type"`
	Data  string `json:"data"`
	Error string `json:"error,omitempty"`
}

// App hosts the Wails backend for the duplicates UI.
type App struct {
	ctx                  context.Context
	groups               []duplicates.DuplicateGroup
	totalSelectedCount   int
	groupsWithSelections map[int]int
	previewCache         map[string]string
	previewMu            sync.Mutex
}

func NewApp() *App {
	return &App{
		groups:               nil,
		groupsWithSelections: make(map[int]int),
		previewCache:         make(map[string]string),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// SelectDirectory opens a native folder picker.
func (a *App) SelectDirectory() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("app context not ready")
	}

	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select a folder to scan",
	})
}

// ConfirmDeleteSelected asks the user to confirm file deletion using a native dialog.
func (a *App) ConfirmDeleteSelected(count int) (bool, error) {
	if a.ctx == nil {
		return false, fmt.Errorf("app context not ready")
	}
	if count <= 0 {
		return false, nil
	}

	result, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Confirm deletion",
		Message:       fmt.Sprintf("Delete %d selected file(s)? This cannot be undone.", count),
		Buttons:       []string{"Cancel", "Delete"},
		DefaultButton: "Cancel",
		CancelButton:  "Cancel",
	})
	if err != nil {
		return false, err
	}

	return result == "Delete", nil
}

// ScanDirectory finds duplicates in the provided directory.
func (a *App) ScanDirectory(directory string) (AppState, error) {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return AppState{}, fmt.Errorf("directory is required")
	}

	duplicatesMap, err := video.FindDuplicatesByHash(directory)
	if err != nil {
		return AppState{}, err
	}

	a.groups = duplicates.BuildGroups(duplicatesMap)
	a.previewMu.Lock()
	a.previewCache = make(map[string]string)
	a.previewMu.Unlock()
	return a.recalculate()
}

// GetState returns the current in-memory state.
func (a *App) GetState() AppState {
	return AppState{
		Groups:               a.groups,
		TotalSelectedCount:   a.totalSelectedCount,
		GroupsWithSelections: a.groupsWithSelections,
	}
}

// ToggleSelection flips the selected state for a file.
func (a *App) ToggleSelection(groupIndex, fileIndex int) (AppState, error) {
	if err := a.validateFileIndex(groupIndex, fileIndex); err != nil {
		return AppState{}, err
	}

	a.groups[groupIndex].Selected[fileIndex] = !a.groups[groupIndex].Selected[fileIndex]
	return a.recalculate()
}

// SelectAllInGroup selects all files in a group.
func (a *App) SelectAllInGroup(groupIndex int) (AppState, error) {
	if err := a.validateGroupIndex(groupIndex); err != nil {
		return AppState{}, err
	}

	for i := range a.groups[groupIndex].Selected {
		a.groups[groupIndex].Selected[i] = true
	}

	return a.recalculate()
}

// ClearSelectionInGroup clears selections in a group.
func (a *App) ClearSelectionInGroup(groupIndex int) (AppState, error) {
	if err := a.validateGroupIndex(groupIndex); err != nil {
		return AppState{}, err
	}

	for i := range a.groups[groupIndex].Selected {
		a.groups[groupIndex].Selected[i] = false
	}

	return a.recalculate()
}

// ApplyAutoSelect applies an auto-selection strategy to the group.
func (a *App) ApplyAutoSelect(groupIndex, strategy int) (AppState, error) {
	if err := a.validateGroupIndex(groupIndex); err != nil {
		return AppState{}, err
	}

	if strategy < int(duplicates.KeepNewest) || strategy > int(duplicates.KeepLastPosition) {
		return AppState{}, fmt.Errorf("unknown auto-select strategy: %d", strategy)
	}

	duplicates.ApplyAutoSelectStrategy(&a.groups[groupIndex], duplicates.AutoSelectStrategy(strategy))
	return a.recalculate()
}

// DeleteSelected removes all selected files across groups.
func (a *App) DeleteSelected() (AppState, error) {
	selected := duplicates.CollectSelectedFiles(a.groups)
	if len(selected) == 0 {
		return a.recalculate()
	}

	failedPath, err := duplicates.DeleteFiles(selected)
	if err != nil {
		return AppState{}, fmt.Errorf("failed to delete %s: %w", failedPath, err)
	}

	a.groups = duplicates.ApplyDeletion(a.groups, selected)
	a.previewMu.Lock()
	for _, path := range selected {
		delete(a.previewCache, path)
	}
	a.previewMu.Unlock()

	return a.recalculate()
}

// GetPreview returns a single-frame preview or a video fallback URL.
func (a *App) GetPreview(path string) (Preview, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Preview{}, fmt.Errorf("path is required")
	}

	a.previewMu.Lock()
	if cached, ok := a.previewCache[path]; ok {
		a.previewMu.Unlock()
		return Preview{Type: "image", Data: cached}, nil
	}
	a.previewMu.Unlock()

	if _, err := os.Stat(path); err != nil {
		return Preview{}, err
	}

	if err := utils.ValidateFFmpegDependencies(); err != nil {
		return Preview{Type: "video", Data: fileURL(path), Error: err.Error()}, nil
	}

	previewBytes, err := extractPreviewFrame(path)
	if err != nil {
		return Preview{Type: "video", Data: fileURL(path), Error: err.Error()}, nil
	}

	encoded := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(previewBytes)
	a.previewMu.Lock()
	a.previewCache[path] = encoded
	a.previewMu.Unlock()

	return Preview{Type: "image", Data: encoded}, nil
}

func (a *App) validateGroupIndex(groupIndex int) error {
	if groupIndex < 0 || groupIndex >= len(a.groups) {
		return fmt.Errorf("group index out of range")
	}
	return nil
}

func (a *App) validateFileIndex(groupIndex, fileIndex int) error {
	if err := a.validateGroupIndex(groupIndex); err != nil {
		return err
	}

	group := a.groups[groupIndex]
	if fileIndex < 0 || fileIndex >= len(group.Files) {
		return fmt.Errorf("file index out of range")
	}

	return nil
}

func (a *App) recalculate() (AppState, error) {
	a.totalSelectedCount, a.groupsWithSelections = duplicates.RecalculateSelectionStats(a.groups)
	return AppState{
		Groups:               a.groups,
		TotalSelectedCount:   a.totalSelectedCount,
		GroupsWithSelections: a.groupsWithSelections,
	}, nil
}

func extractPreviewFrame(videoFile string) ([]byte, error) {
	attempts := []string{"00:00:30", "00:00:10", "00:00:01"}
	var lastErr error

	for _, timestamp := range attempts {
		previewPath := filepath.Join(os.TempDir(), fmt.Sprintf("videotagger_preview_%d.jpg", time.Now().UnixNano()))
		cmd := exec.Command(
			"ffmpeg",
			"-hide_banner",
			"-loglevel",
			"error",
			"-ss",
			timestamp,
			"-i",
			videoFile,
			"-vframes",
			"1",
			"-f",
			"image2",
			"-y",
			previewPath,
		)

		if err := cmd.Run(); err != nil {
			lastErr = err
			_ = os.Remove(previewPath)
			continue
		}

		previewBytes, err := os.ReadFile(previewPath)
		_ = os.Remove(previewPath)
		if err != nil {
			lastErr = err
			continue
		}

		return previewBytes, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("failed to extract preview")
	}
	return nil, lastErr
}

func fileURL(path string) string {
	urlValue := url.URL{Scheme: "file", Path: path}
	return urlValue.String()
}
