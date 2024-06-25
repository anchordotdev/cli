package lcl

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/ui/uitest"
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

	anc, err := api.NewClient(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("diagnostic", func(t *testing.T) {
		drv, _ := uitest.TestTUI(ctx, t)

		cmd := &Provision{
			Domains: []string{"subdomain.lcl.host", "subdomain.localhost"},
		}

		errc := make(chan error, 1)
		go func() {
			_, _, err := cmd.run(ctx, drv, anc, "test-service", "diagnostic", nil)
			errc <- err
		}()

		t.Skip("pending ability to stub out the detection.Perform results")

		uitest.WaitForGoldenContains(t, drv, errc,
			"- Created test-service resources for lcl.host diagnostic server on Anchor.dev.",
		)
	})
}
