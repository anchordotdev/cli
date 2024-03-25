package version

import (
	"context"
	"fmt"

	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
)

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
