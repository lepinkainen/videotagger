package ui

// TUI Message Types for worker communication
type WorkerStartedMsg struct {
	WorkerID int
	Filename string
}

type WorkerProgressMsg struct {
	WorkerID int
	Progress float64 // 0.0 to 1.0
	Bytes    int64
	Total    int64
}

type WorkerCompletedMsg struct {
	WorkerID int
	Filename string
	NewName  string
	Success  bool
	Error    error
}

type OverallProgressMsg struct {
	Completed int
	Total     int
}
