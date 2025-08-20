package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/lepinkainen/videotagger/cmd"
	"github.com/lepinkainen/videotagger/types"
	"github.com/lepinkainen/videotagger/utils"
)

var Version = "dev"

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Println(Version)
	return nil
}

type CLI struct {
	Tag        *cmd.TagCmd        `cmd:"" help:"Tag video files with metadata and hash"`
	Duplicates *cmd.DuplicatesCmd `cmd:"" help:"Find duplicate files by hash"`
	Verify     *cmd.VerifyCmd     `cmd:"" help:"Verify file hash integrity"`
	Phash      *cmd.PhashCmd      `cmd:"" help:"Find perceptually similar videos"`
	Version    *VersionCmd        `cmd:"" help:"Show version information"`
}

func main() {
	var cli CLI
	appCtx := &types.AppContext{
		Version: Version,
	}
	ctx := kong.Parse(&cli, kong.Bind(appCtx))

	// Validate FFmpeg dependencies before running any command
	// Skip validation for version command as it doesn't require FFmpeg
	if ctx.Command() != "version" {
		if err := utils.ValidateFFmpegDependencies(); err != nil {
			ctx.FatalIfErrorf(err)
		}
	}

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
