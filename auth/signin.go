package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/browser"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"

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
	output := termenv.DefaultOutput()
	cp := output.ColorProfile()

	anc, err := api.Client(s.Config)
	if err != nil && err != api.ErrSignedOut {
		return err
	}

	codesRes, err := anc.Post("/auth/cli/codes", "application/json", nil)
	if err != nil {
		return err
	}
	if codesRes.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code: %d", codesRes.StatusCode)
	}

	var codes *api.AuthCliCodesResponse
	if err = json.NewDecoder(codesRes.Body).Decode(&codes); err != nil {
		return err
	}

	fmt.Fprintln(tty)
	if isatty.IsTerminal(tty.Fd()) {
		fmt.Fprintln(tty,
			output.String("!").Foreground(cp.Color("#ff6000")),
			"First copy your user code:",
			output.String(codes.UserCode).Background(cp.Color("#7000ff")).Bold(),
		)
		fmt.Fprintln(tty,
			"Then",
			output.String("Press Enter").Bold(),
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
			output.String("!").Foreground(cp.Color("#ff6000")),
			"Open",
			output.String(codes.VerificationUri).Faint().Underline(),
			"in a browser and enter your user code:",
			output.String(codes.UserCode).Bold(),
		)
	}

	var looper = func() error {
		for {
			body := new(bytes.Buffer)
			req := api.CreateCliTokenJSONRequestBody{
				DeviceCode: codes.DeviceCode,
			}
			if err = json.NewEncoder(body).Encode(req); err != nil {
				return err
			}
			tokensRes, err := anc.Post("/auth/cli/pat_tokens", "application/json", body)
			if err != nil {
				return err
			}

			switch tokensRes.StatusCode {
			case http.StatusOK:
				var patTokens *api.AuthCliPatTokensResponse
				if err = json.NewDecoder(tokensRes.Body).Decode(&patTokens); err != nil {
					return err
				}
				s.Config.API.Token = patTokens.PatToken
				return nil
			case http.StatusBadRequest:
				var errorsRes *api.Error
				if err = json.NewDecoder(tokensRes.Body).Decode(&errorsRes); err != nil {
					return err
				}
				switch errorsRes.Type {
				case "urn:anchordev:api:cli-auth:authorization-pending":
					time.Sleep(time.Duration(codes.Interval) * time.Second)
				case "urn:anchordev:api:cli-auth:expired-device-code":
					return fmt.Errorf("Your authorization request has expired, please try again.")
				case "urn:anchordev:api:cli-auth:incorrect-device-code":
					return fmt.Errorf("Your authorization request was not found, please try again.")
				default:
					return fmt.Errorf("unexpected error: %s", errorsRes.Detail)
				}
			default:
				return fmt.Errorf("unexpected response code: %d", tokensRes.StatusCode)
			}
		}
	}
	if err := looper(); err != nil {
		return err
	}

	anc, err = api.Client(s.Config)
	if err != nil {
		return err
	}

	res, err := anc.Get("")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code: %d", res.StatusCode)
	}

	var userInfo *api.Root
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return err
	}

	kr := keyring.Keyring{Config: s.Config}
	if err := kr.Set(keyring.APIToken, s.Config.API.Token); err != nil {
		return err
	}

	fmt.Fprintln(tty)
	fmt.Fprintf(tty,
		"Success, hello %s!",
		output.String(userInfo.Whoami).Bold(),
	)
	fmt.Fprintln(tty)

	return nil
}
