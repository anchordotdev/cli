package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
)

var CmdAuthWhoami = cli.NewCmd[WhoAmI](CmdAuth, "whoami", func(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
})

type WhoAmI struct{}

func (w WhoAmI) UI() cli.UI {
	return cli.UI{
		RunTTY: w.run,
	}
}

func (w *WhoAmI) run(ctx context.Context, tty termenv.File) error {
	cfg := cli.ConfigFromContext(ctx)

	anc, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	res, err := anc.Get("")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected response")
	}

	var userInfo *api.Root
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return err
	}

	fmt.Fprintf(tty, "Hello %s!\n", userInfo.Whoami)
	return nil
}
