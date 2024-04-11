package lcl

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestProvision(t *testing.T) {
	if !srv.IsProxy() {
		t.Skip("provision unsupported in non-mock mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.AnchorURL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Lcl.Service = "hi-ankydotdev"
	cfg.Lcl.Subdomain = "hi-ankydotdev"
	cfg.Trust.MockMode = true
	cfg.Trust.NoSudo = true

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}

	anc, err := api.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("diagnostic", func(t *testing.T) {
		drv, tm := uitest.TestTUI(ctx, t)

		cmd := &Provision{
			Domains: []string{"subdomain.lcl.host", "subdomain.localhost"},
		}

		errc := make(chan error, 1)
		go func() {
			_, _, err := cmd.run(ctx, drv, anc, "test-service", "diagnostic", nil)
			errc <- err
		}()

		t.Skip("pending ability to stub out the detection.Perform results")

		teatest.WaitFor(
			t, tm.Output(),
			func(bts []byte) bool {
				if len(errc) > 0 {
					if err := <-errc; err != nil {
						t.Fatal(<-errc)
					}
				}

				return bytes.Contains(bts, []byte("- Created test-service resources for lcl.host diagnostic server on Anchor.dev."))
			},
			teatest.WithCheckInterval(time.Millisecond*100),
			teatest.WithDuration(time.Second*3),
		)
	})
}
