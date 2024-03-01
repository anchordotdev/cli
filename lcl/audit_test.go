package lcl

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestAudit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}

	t.Run("basics", func(t *testing.T) {
		t.Skip("pending fixing golden file based timing issues on CI")
		// FIXME: it looks like it's running fast enough that view doesn't get called before transitioning away from initial state

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
