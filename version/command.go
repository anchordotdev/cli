package version

import (
	"context"
	"fmt"

	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
)

var CmdVersion = cli.NewCmd[Command](cli.CmdRoot, "version", func(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
})

type Command struct{}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTTY: c.run,
	}
}

func (c Command) run(ctx context.Context, tty termenv.File) error {
	_, err := fmt.Fprintln(tty, String())
	return err
}
