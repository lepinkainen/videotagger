package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lepinkainen/videotagger/ui"
	"github.com/lepinkainen/videotagger/video"
)

type VerifyCmd struct {
	Files []string `arg:"" name:"files" help:"Video files to verify" type:"existingfile"`
}

func (cmd *VerifyCmd) Run() error {
	fmt.Printf("%s\n", ui.InfoStyle.Render(fmt.Sprintf("Verifying %d files...", len(cmd.Files))))

	var verified, failed int

	for _, videoFile := range cmd.Files {
		if !video.IsVideoFile(videoFile) {
			fmt.Printf("⚠️  %s is not a video file, skipping\n", videoFile)
			continue
		}

		expectedHash, ok := video.ExtractHashFromFilename(filepath.Base(videoFile))
		if !ok {
			fmt.Printf("⚠️  %s has not been processed (no hash in filename)\n", videoFile)
			continue
		}

		actualHash, err := video.CalculateCRC32(videoFile)
		if err != nil {
			fmt.Printf("%s\n", ui.ErrorStyle.Render(fmt.Sprintf("❌ Error calculating hash for %s: %v", videoFile, err)))
			failed++
			continue
		}

		if strings.EqualFold(expectedHash, fmt.Sprintf("%08X", actualHash)) {
			fmt.Printf("%s\n", ui.SuccessStyle.Render(fmt.Sprintf("✅ %s", videoFile)))
			verified++
		} else {
			fmt.Printf("%s\n", ui.ErrorStyle.Render(fmt.Sprintf("❌ %s (expected: %s, got: %08X)", videoFile, expectedHash, actualHash)))
			failed++
		}
	}

	fmt.Printf("\n%s\n", ui.InfoStyle.Render(fmt.Sprintf("✅ Verified: %d, ❌ Failed: %d", verified, failed)))
	return nil
}
