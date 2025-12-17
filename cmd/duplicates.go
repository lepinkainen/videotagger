package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lepinkainen/videotagger/types"
	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/video"
)

// DuplicatesCmd finds duplicate video files by comparing CRC32 hashes embedded in filenames.
// Files must have been previously tagged with the tag command to include hash information.
type DuplicatesCmd struct {
	Directory string `arg:"" name:"directory" help:"Directory to scan for duplicates" type:"existingdir" default:"."`
	NoTUI     bool   `name:"no-tui" help:"Disable interactive TUI and just list duplicates"`
}

// Run executes the duplicates command and displays results either in an interactive TUI
// or as a simple list depending on the NoTUI flag.
func (cmd *DuplicatesCmd) Run(appCtx *types.AppContext) error {
	version := types.DefaultVersion
	if appCtx != nil {
		version = appCtx.Version
	}
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

	// If no-tui flag is set, just list the duplicates
	if cmd.NoTUI {
		fmt.Printf("\n%s\n", ui.InfoStyle.Render(fmt.Sprintf("Found %d group(s) of duplicates:", len(duplicates))))
		for hash, files := range duplicates {
			fmt.Printf("\n🔸 Hash %s (%d files):\n", hash, len(files))
			for _, file := range files {
				fmt.Printf("  %s\n", file)
			}
		}
		return nil
	}

	// Launch TUI for interactive duplicate management
	model := ui.NewDuplicatesModel(duplicates)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
