package trust

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
)

var srv = &apitest.Server{
	RootDir: "../..",
}

func TestMain(m *testing.M) {
	flag.Parse()

	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}

	defer os.Exit(m.Run())

	srv.Close()
}

func TestTrust(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.API.Token = "test-token"
	cfg.Trust.NoSudo = true

	cmd := &Command{
		Config: cfg,
	}

	buf, err := apitest.RunTUI(ctx, cmd.TUI())
	if err != nil {
		t.Fatal(err)
	}

	msg := "\"oas-examples - AnchorCA\" Ed25519 cert (61c111892d03dbaa77505062) installed in the mock store\n\"oas-examples - AnchorCA\" ECDSA cert (61c111892d03fec352f95595) installed in the mock store\n\"oas-examples - AnchorCA\" RSA cert (61c111892d03674512b31880) installed in the mock store\n"
	if want, got := msg, buf.String(); want != got {
		t.Errorf("want output %q, got %q", want, got)
	}
}
