package ui

// TUI Message Types for duplicate file management
type DuplicateGroupSelectedMsg struct {
	GroupIndex int
}

type FileSelectedMsg struct {
	FileIndex int
	Selected  bool
}

type DeleteSelectedMsg struct {
	Confirmed bool
}

type DeletionCompleteMsg struct {
	FilePath string
	Success  bool
	Error    error
}

type AllFilesSelectedMsg struct{}

type ClearSelectionsMsg struct{}

type NextGroupMsg struct{}

type PrevGroupMsg struct{}

type SkipGroupMsg struct{}
