package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/keyring"
)

type SignIn struct {
	Config *cli.Config
}

func (s SignIn) TUI() cli.TUI {
	return cli.TUI{
		Run: s.run,
	}
}

func (s *SignIn) run(ctx context.Context, tty termenv.File) error {
	apiToken := s.Config.API.Token
	if len(apiToken) == 0 {
		if _, err := fmt.Fprintf(tty, "API Token: "); err != nil {
			return err
		}

		line, err := term.ReadPassword(int(tty.Fd()))
		if err != nil {
			return err
		}
		apiToken = strings.TrimSpace(string(line))

		if _, err := fmt.Fprintln(tty); err != nil {
			return err
		}
	}

	anc, err := api.Client(s.Config)
	if err != nil && err != api.ErrSignedOut {
		return err
	}

	req, err := http.NewRequest("GET", "", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiToken, "")

	res, err := anc.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code: %d", res.StatusCode)
	}

	var userInfo *api.Root
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return fmt.Errorf("decoding userInfo failed: %w", err)
	}

	kr := keyring.Keyring{Config: s.Config}
	if err := kr.Set(keyring.APIToken, apiToken); err != nil {
		return err
	}

	fmt.Fprintf(tty, "Success, hello %s!\n", userInfo.Whoami)
	return nil
}