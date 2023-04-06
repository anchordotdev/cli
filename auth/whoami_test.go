package auth

import (
	"context"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
)

func TestWhoAmI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("signed-out", func(t *testing.T) {
		keyring.MockInit()

		cfg := new(cli.Config)
		cfg.API.URL = srv.URL

		cmd := &WhoAmI{
			Config: cfg,
		}

		buf, err := apitest.RunTUI(ctx, cmd.TUI())
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "Sign-in required!\n", buf.String(); want != got {
			t.Errorf("want output %q, got %q", want, got)
		}
	})

	t.Run("signed-in", func(t *testing.T) {
		keyring.MockInit()

		cfg := new(cli.Config)
		cfg.API.URL = srv.URL
		cfg.API.Token = "test-token"

		cmd := &WhoAmI{
			Config: cfg,
		}

		buf, err := apitest.RunTUI(ctx, cmd.TUI())
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "Hello username!\n", buf.String(); want != got {
			t.Errorf("want output %q, got %q", want, got)
		}
	})
}
