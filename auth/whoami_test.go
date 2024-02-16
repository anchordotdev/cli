package auth

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/api/apitest"
)

func TestWhoAmI(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("signed-out", func(t *testing.T) {
		cfg := new(cli.Config)
		cfg.API.URL = srv.URL
		cfg.Keyring.MockMode = true

		cmd := &WhoAmI{
			Config: cfg,
		}

		_, err := apitest.RunTTY(ctx, cmd.UI())
		if want, got := api.ErrSignedOut, err; want != got {
			t.Fatalf("want signin failure error %q, got %q", want, got)
		}
	})

	t.Run("signed-in", func(t *testing.T) {
		cfg := new(cli.Config)
		cfg.API.URL = srv.URL
		cfg.Keyring.MockMode = true

		var err error
		if cfg.API.Token, err = srv.GeneratePAT("example@example.com"); err != nil {
			t.Fatal(err)
		}

		cmd := &WhoAmI{
			Config: cfg,
		}

		buf, err := apitest.RunTTY(ctx, cmd.UI())
		if err != nil {
			t.Fatal(err)
		}

		if want, got := "Hello example@example.com!\n", buf.String(); want != got {
			t.Errorf("want output %q, got %q", want, got)
		}
	})
}
