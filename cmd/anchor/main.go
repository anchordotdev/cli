package main

import (
	"context"
	"log"
	"os"

	"github.com/anchordotdev/cli"
	_ "github.com/anchordotdev/cli/auth"
	_ "github.com/anchordotdev/cli/lcl"
	_ "github.com/anchordotdev/cli/trust"
	versionpkg "github.com/anchordotdev/cli/version"
	"github.com/muesli/termenv"
)

var (
	// Version info set by GoReleaser via ldflags

	version, commit, date string
)

func init() {
	if version != "" {
		cli.Version.Commit = commit
		cli.Version.Date = date
		cli.Version.Version = version
	}
	cli.CmdRoot.PersistentPreRunE = versionpkg.VersionCheck
}

func main() {
	// enable ANSI processing for Windows, see: https://github.com/muesli/termenv#platform-support
	restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		err := restoreConsole()
		if err != nil {
			log.Fatal(err)
		}
	}()

	ctx, cancel := context.WithCancel(cli.CmdRoot.Context())
	defer cancel()

	if err := cli.CmdRoot.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
