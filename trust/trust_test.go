package trust

import (
	"context"
	"flag"
	"os"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/ui/uitest"
)

var srv = &apitest.Server{
	Host:    "api.anchor.lcl.host",
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
	cfg.NonInteractive = true
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		if !srv.IsProxy() {
			t.Skip("trust unsupported in mock mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := Command{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
