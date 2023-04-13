package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
)

type WhoAmI struct {
	Config *cli.Config
}

func (w WhoAmI) TUI() cli.TUI {
	return cli.TUI{
		Run: w.run,
	}
}

func (w *WhoAmI) run(ctx context.Context, tty termenv.File) error {
	anc, err := api.Client(w.Config)
	if err == api.ErrSignedOut {
		fmt.Fprintf(tty, "Sign-in required!\n")
		return nil
	}
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
