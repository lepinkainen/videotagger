package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/video"
)

type DuplicatesCmd struct {
	Directory string `arg:"" name:"directory" help:"Directory to scan for duplicates" type:"existingdir" default:"."`
}

func (cmd *DuplicatesCmd) Run() error {
	version := "dev" // TODO: Pass version from main package
	fmt.Println(ui.HeaderStyle.Render(fmt.Sprintf("Video Tagger %s", version)))
	fmt.Printf("Scanning %s for duplicates...\n", cmd.Directory)

	duplicates, err := video.FindDuplicatesByHash(cmd.Directory)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		fmt.Printf("%s\n", ui.SuccessStyle.Render("✅ No duplicates found"))
		return nil
	}

	// Launch TUI for interactive duplicate management
	model := ui.NewDuplicatesModel(duplicates)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
