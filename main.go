package main

import (
	"github.com/alecthomas/kong"
	"github.com/lepinkainen/videotagger/cmd"
)

var Version = "dev"

type CLI struct {
	Tag        *cmd.TagCmd        `cmd:"" help:"Tag video files with metadata and hash"`
	Duplicates *cmd.DuplicatesCmd `cmd:"" help:"Find duplicate files by hash"`
	Verify     *cmd.VerifyCmd     `cmd:"" help:"Verify file hash integrity"`
	Phash      *cmd.PhashCmd      `cmd:"" help:"Find perceptually similar videos"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
