package video

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// progressWriter wraps progress bar for io.Writer interface
type progressWriter struct {
	total   int64
	current int64
	prog    progress.Model
	done    chan bool
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.current += int64(n)
	return n, nil
}

func (pw *progressWriter) render() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pw.done:
			// Show 100% progress before clearing
			fmt.Printf("\r%s\n", pw.prog.ViewAs(1.0))
			return
		case <-ticker.C:
			if pw.current > 0 {
				percent := float64(pw.current) / float64(pw.total)
				fmt.Printf("\r%s", pw.prog.ViewAs(percent))
			}
		}
	}
}

// Styling definitions
var (
	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)
)
