package lcl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestBootstrapPropagatesDiagnosticServiceCreationError(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v0":
			_, _ = w.Write([]byte(`{"minimum_cli_version":"","personal_org":{"slug":"synthetic-org"}}`))
		case "/v0/orgs/synthetic-org/services":
			_, _ = w.Write([]byte(`{"items":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiServer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := *cli.Defaults
	cfg.API.URL = apiServer.URL + "/v0"
	cfg.API.Token = "synthetic-test-token"
	cfg.Lcl.Diagnostic.Addr = "127.0.0.1:0"
	ctx = cli.ContextWithConfig(ctx, &cfg)

	anc, err := api.NewClient(ctx, &cfg)
	require.NoError(t, err)

	drv, tm := uitest.TestTUI(ctx, t)
	defer func() {
		if err := tm.Quit(); err != nil {
			t.Errorf("quit test UI: %v", err)
		}
	}()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				done <- fmt.Errorf("bootstrap panicked: %v", recovered)
			}
		}()
		done <- (Bootstrap{anc: anc}).perform(ctx, drv)
	}()

	uitest.WaitForGoldenContains(t, drv, done,
		"? What lcl.host domain would you like to use for diagnostics?",
	)
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(3 * time.Second):
		t.Fatal("bootstrap did not return after cancellation")
	}
}
