package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/cli/browser"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth/models"
	"github.com/anchordotdev/cli/keyring"
	"github.com/anchordotdev/cli/ui"
)

var (
	CmdAuthSignin = cli.NewCmd[SignIn](CmdAuth, "signin", func(cmd *cobra.Command) {})

	ErrSigninFailed = errors.New("sign in failed")
)

type SignIn struct {
	Source string

	Hint tea.Model
}

func (s SignIn) UI() cli.UI {
	return cli.UI{
		RunTUI: s.RunTUI,
	}
}

func (s *SignIn) RunTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, &models.SignInHeader{})

	if s.Hint == nil {
		s.Hint = &models.SignInHint{}
	}
	drv.Activate(ctx, s.Hint)

	anc, err := api.NewClient(cfg)
	if err != nil && err != api.ErrSignedOut {
		return err
	}

	codes, err := anc.GenerateUserFlowCodes(ctx, s.Source)
	if err != nil {
		return err
	}

	// TODO: skipping TTY check since this is TUI mode, but is it needed?
	clipboardErr := clipboard.WriteAll(codes.UserCode)

	confirmc := make(chan struct{})
	drv.Activate(ctx, &models.SignInPrompt{
		ConfirmCh:       confirmc,
		InClipboard:     (clipboardErr == nil),
		UserCode:        codes.UserCode,
		VerificationURL: codes.VerificationUri,
	})

	if !cfg.NonInteractive {
		select {
		case <-confirmc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := browser.OpenURL(codes.VerificationUri); err != nil {
		return err
	}

	drv.Activate(ctx, new(models.SignInChecker))

	var patToken string
	for patToken == "" {
		if patToken, err = anc.CreatePATToken(ctx, codes.DeviceCode); err != nil {
			return err
		}

		if patToken == "" {
			time.Sleep(time.Duration(codes.Interval) * time.Second)
		}
	}
	cfg.API.Token = patToken

	userInfo, err := fetchUserInfo(cfg)
	if err != nil {
		return err
	}

	kr := keyring.Keyring{Config: cfg}
	if err := kr.Set(keyring.APIToken, cfg.API.Token); err != nil {
		return err
	}

	drv.Send(models.UserSignInMsg(userInfo.Whoami))

	return nil
}

func fetchUserInfo(cfg *cli.Config) (*api.Root, error) {
	anc, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	res, err := anc.Get("")
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response code: %d", res.StatusCode)
	}

	var userInfo *api.Root
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}
