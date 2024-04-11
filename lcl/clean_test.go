package lcl

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestClean(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
  ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("lcl clean unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := LclClean{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
