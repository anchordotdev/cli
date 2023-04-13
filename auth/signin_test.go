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
}
