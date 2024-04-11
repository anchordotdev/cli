package lcl

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestLclMkcert(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Lcl.Service = "hi-lcl-mkcert"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("lcl_mkcert@anchor.dev"); err != nil {
		t.Fatal(err)
	}
  ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		t.Skip("pending better support for building needed models before running")

		if !srv.IsProxy() {
			t.Skip("mkcert unsupported in proxy mode")
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cmd := MkCert{
			domains:         []string{"hi-lcl-mkcert.lcl.host", "hi-lcl-mkcert.localhost"},
			subCaSubjectUID: "ABCD:EF12:23456",
		}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
