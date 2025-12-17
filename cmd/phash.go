package cmd

import (
	"fmt"

	"github.com/corona10/goimagehash"
	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/video"
)

// PhashCmd finds perceptually similar videos using frame-based perceptual hashing.
// This command compares video frames extracted from each file to identify videos
// that appear similar even if they differ in encoding or resolution.
type PhashCmd struct {
	Files     []string `arg:"" name:"files" help:"Video files to compare" type:"existingfile"`
	Threshold int      `help:"Hamming distance threshold for similarity (0-64)" default:"10"`
}

// Run executes the perceptual hash comparison command, comparing all pairs of videos
// and reporting any that fall within the similarity threshold (lower distance = more similar).
func (cmd *PhashCmd) Run() error {
	if len(cmd.Files) < 2 {
		fmt.Printf("%s\n", ui.ErrorStyle.Render("❌ Need at least 2 files to compare"))
		return nil
	}

	fmt.Printf("%s\n", ui.InfoStyle.Render(fmt.Sprintf("Calculating perceptual hashes for %d files...", len(cmd.Files))))

	type FileHash struct {
		File string
		Hash *goimagehash.ImageHash
	}

	var fileHashes []FileHash

	for _, videoFile := range cmd.Files {
		if !video.IsVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		hash, err := video.CalculateVideoPerceptualHash(videoFile)
		if err != nil {
			fmt.Printf("%s\n", ui.ErrorStyle.Render(fmt.Sprintf("❌ Error calculating perceptual hash for %s: %v", videoFile, err)))
			continue
		}

		fileHashes = append(fileHashes, FileHash{File: videoFile, Hash: hash})
		fmt.Printf("%s\n", ui.SuccessStyle.Render(fmt.Sprintf("✅ Processed %s", videoFile)))
	}

	fmt.Printf("\n%s\n", ui.InfoStyle.Render(fmt.Sprintf("Comparing %d files for similarity (threshold: %d):", len(fileHashes), cmd.Threshold)))

	found := false
	for i := 0; i < len(fileHashes); i++ {
		for j := i + 1; j < len(fileHashes); j++ {
			distance, err := fileHashes[i].Hash.Distance(fileHashes[j].Hash)
			if err != nil {
				fmt.Printf("%s\n", ui.ErrorStyle.Render(fmt.Sprintf("❌ Error comparing %s and %s: %v", fileHashes[i].File, fileHashes[j].File, err)))
				continue
			}

			if distance <= cmd.Threshold {
				fmt.Printf("🎯 Similar (distance %d): %s ↔ %s\n", distance, fileHashes[i].File, fileHashes[j].File)
				found = true
			}
		}
	}

	if !found {
		fmt.Printf("%s\n", ui.SuccessStyle.Render("✅ No similar files found within threshold"))
	}

	return nil
}
