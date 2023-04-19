package auth

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
)

func TestSignIn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Keyring.MockMode = true

	t.Run("valid-token", func(t *testing.T) {
		var err error
		if cfg.API.Token, err = srv.GeneratePAT("example@example.com"); err != nil {
			t.Fatal(err)
		}

		cmd := &SignIn{
			Config: cfg,
		}

		buf, err := apitest.RunTUI(ctx, cmd.TUI())
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "Success, hello example@example.com!\n", buf.String(); want != got {
			t.Errorf("want output %q, got %q", want, got)
		}
	})

	t.Run("invalid-token", func(t *testing.T) {
		if !srv.IsProxy() {
			t.Skip("server doesn't authenticate in mock mode")
			return
		}

		cfg.API.Token = "bad-pat-token"

		cmd := &SignIn{
			Config: cfg,
		}

		_, err := apitest.RunTUI(ctx, cmd.TUI())
		if want, got := ErrSigninFailed, err; want != got {
			t.Fatalf("want signin failure error %q, got %q", want, got)
		}
	})
}
