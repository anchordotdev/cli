package lcl

import (
	"context"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestAudit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no pty support on windows")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}

	t.Run("basics", func(t *testing.T) {
		if srv.IsProxy() {
			t.Skip("lcl audit unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := Audit{
			Config: cfg,
		}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
