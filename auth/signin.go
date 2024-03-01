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
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth/models"
	"github.com/anchordotdev/cli/keyring"
	"github.com/anchordotdev/cli/ui"
)

var (
	ErrSigninFailed = errors.New("sign in failed")
)

type SignIn struct {
	Config *cli.Config

	Source string

	Hint tea.Model
}

func (s SignIn) UI() cli.UI {
	return cli.UI{
		RunTTY: s.runTTY,
		RunTUI: s.RunTUI,
	}
}

func (s *SignIn) runTTY(ctx context.Context, tty termenv.File) error {
	output := termenv.DefaultOutput()
	cp := output.ColorProfile()

	fmt.Fprintln(tty,
		output.String("# Run `anchor auth signin`").Bold(),
	)

	anc, err := api.NewClient(s.Config)
	if err != nil && err != api.ErrSignedOut {
		return err
	}

	codes, err := anc.GenerateUserFlowCodes(ctx, s.Source)
	if err != nil {
		return err
	}

	if isatty.IsTerminal(tty.Fd()) {
		if err := clipboard.WriteAll(codes.UserCode); err != nil {
			fmt.Fprintln(tty,
				" ",
				output.String("!").Background(cp.Color("#7000ff")),
				output.String("Copy").Foreground(cp.Color("#ff6000")).Bold(),
				"your user code:",
				output.String(codes.UserCode).Background(cp.Color("#7000ff")).Bold(),
			)
		} else {
			fmt.Fprintln(tty,
				" ",
				output.String("!").Background(cp.Color("#7000ff")),
				"Copied your user code",
				output.String(codes.UserCode).Background(cp.Color("#7000ff")).Bold(),
				"to your clipboard.",
			)
		}
		fmt.Fprintln(tty,
			" ",
			output.String("| When prompted you will paste this code in your browser to connect your CLI.").Faint(),
		)
		fmt.Fprintln(tty,
			" ",
			output.String("!").Background(cp.Color("#7000ff")),
			output.String("Press Enter").Foreground(cp.Color("#ff6000")).Bold(),
			"to open",
			output.String(codes.VerificationUri).Faint().Underline(),
			"in your browser...",
		)
		fmt.Scanln()

		if err = browser.OpenURL(codes.VerificationUri); err != nil {
			return err
		}
	} else {
		fmt.Fprintln(tty,
			" ",
			output.String("!").Background(cp.Color("#7000ff")),
			output.String("Open").Foreground(cp.Color("#ff6000")).Bold(),
			output.String(codes.VerificationUri).Faint().Underline(),
			"in a browser and enter your user code:",
			output.String(codes.UserCode).Bold(),
		)
	}

	var patToken string
	for patToken == "" {
		if patToken, err = anc.CreatePATToken(ctx, codes.DeviceCode); err != nil {
			return err
		}

		if patToken == "" {
			time.Sleep(time.Duration(codes.Interval) * time.Second)
		}
	}
	s.Config.API.Token = patToken

	userInfo, err := fetchUserInfo(s.Config)
	if err != nil {
		return err
	}

	kr := keyring.Keyring{Config: s.Config}
	if err := kr.Set(keyring.APIToken, s.Config.API.Token); err != nil {
		return err
	}

	fmt.Fprintln(tty)
	fmt.Fprintf(tty,
		"  - Signed in as %s!",
		output.String(userInfo.Whoami).Bold(),
	)
	fmt.Fprintln(tty)

	return nil
}

func (s *SignIn) RunTUI(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &models.SignInHeader{})

	if s.Hint == nil {
		s.Hint = &models.SignInHint{}
	}
	drv.Activate(ctx, s.Hint)

	anc, err := api.NewClient(s.Config)
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

	if !s.Config.NonInteractive {
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
	s.Config.API.Token = patToken

	userInfo, err := fetchUserInfo(s.Config)
	if err != nil {
		return err
	}

	kr := keyring.Keyring{Config: s.Config}
	if err := kr.Set(keyring.APIToken, s.Config.API.Token); err != nil {
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
