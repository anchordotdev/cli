package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/keyring"
)

var (
	ErrSigninFailed = errors.New("sign in failed")
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
	if len(s.Config.API.Token) == 0 {
		fmt.Fprintf(tty, "To complete sign in, please:\n  1. Visit https://anchor.dev/settings and add a new Personal Access Token (PAT).\n  2. Copy the key from the new token and paste it below when prompted.\n")

		if _, err := fmt.Fprintf(tty, "Personal Access Token (PAT): "); err != nil {
			return err
		}

		line, err := term.ReadPassword(int(tty.Fd()))
		if err != nil {
			return err
		}
		pat := strings.TrimSpace(string(line))
		if !strings.HasPrefix(pat, "ap0_") || len(pat) != 64 {
			return fmt.Errorf("invalid PAT key")
		}

		s.Config.API.Token = pat

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
	req.SetBasicAuth(s.Config.API.Token, "")

	res, err := anc.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusForbidden {
		return ErrSigninFailed
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code: %d", res.StatusCode)
	}

	var userInfo *api.Root
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return fmt.Errorf("decoding userInfo failed: %w", err)
	}

	kr := keyring.Keyring{Config: s.Config}
	if err := kr.Set(keyring.APIToken, s.Config.API.Token); err != nil {
		return err
	}

	fmt.Fprintf(tty, "Success, hello %s!\n", userInfo.Whoami)
	return nil
}
