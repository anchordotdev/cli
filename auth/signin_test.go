package auth

import (
	"context"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
)

func TestSignIn(t *testing.T) {
	keyring.MockInit()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.API.Token = "test-token"

	cmd := &SignIn{
		Config: cfg,
	}

	buf, err := apitest.RunTUI(ctx, cmd.TUI())
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "Success, hello username!\n", buf.String(); want != got {
		t.Errorf("want output %q, got %q", want, got)
	}
}
