package cmd

import (
	"fmt"

	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/video"
)

type DuplicatesCmd struct {
	Directory string `arg:"" name:"directory" help:"Directory to scan for duplicates" type:"existingdir" default:"."`
}

func (cmd *DuplicatesCmd) Run() error {
	fmt.Printf("Scanning %s for duplicates...\n", cmd.Directory)

	duplicates, err := video.FindDuplicatesByHash(cmd.Directory)
	if err != nil {
		return fmt.Errorf("failed to find duplicates: %w", err)
	}

	if len(duplicates) == 0 {
		fmt.Printf("%s\n", ui.SuccessStyle.Render("✅ No duplicates found"))
		return nil
	}

	fmt.Printf("%s\n", ui.InfoStyle.Render(fmt.Sprintf("Found %d groups of duplicates:", len(duplicates))))
	for hash, files := range duplicates {
		fmt.Printf("\nHash %s:\n", hash)
		for _, file := range files {
			fmt.Printf("  - %s\n", file)
		}
	}

	return nil
}
