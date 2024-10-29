package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/cmdtest"
	_ "github.com/anchordotdev/cli/testflags"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/benburkert/dns"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestCmdServiceProbe(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, cmdServiceProbe, "service", "probe", "--help")
	})
}

func TestProbe(t *testing.T) {
	if !srv.IsProxy() {
		t.Skip("service probe errors unsupported in non-proxy mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiToken, err := srv.GeneratePAT("anky@anchor.dev")
	if err != nil {
		t.Fatal(err)
	}

	cfg := cmdtest.Config(ctx)
	cfg.API.Token = apiToken
	cfg.API.URL = srv.URL
	cfg.Dashboard.URL = "http://anchor.lcl.host:" + srv.RailsPort
	cfg.Service.Probe.Timeout = 3 * time.Second

	anc, err := api.NewClient(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	attachments, err := anc.GetServiceAttachments(ctx, "ankydotdev", "ankydotdev")
	if err != nil {
		t.Fatal(err)
	}
	if want, got := 1, len(attachments); want != got {
		t.Fatalf("want %d ankydotdev service attachments, got %d", want, got)
	}

	eab, err := anc.CreateEAB(ctx, "ca", "ankydotdev", "localhost", "ankydotdev", *attachments[0].Relationships.SubCa.Apid)
	if err != nil {
		t.Fatal(err)
	}

	acmeURL := cfg.AcmeURL("ankydotdev", "localhost", attachments[0].Relationships.Chain.Apid)

	tlsCert, err := api.ProvisionCert(eab, []string{"ankydotdev.lcl.host"}, acmeURL)
	if err != nil {
		t.Fatal(err)
	}

	srvHTTPS := httptest.NewUnstartedServer(http.NotFoundHandler())
	srvHTTPS.TLS = &tls.Config{
		Certificates: []tls.Certificate{*tlsCert},
	}
	srvHTTPS.StartTLS()
	defer srvHTTPS.Close()

	cfg.Test.NetDialer = cli.DialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
		return new(net.Dialer).DialContext(ctx, network, srvHTTPS.Listener.Addr().String())
	})

	ctx = cli.ContextWithConfig(ctx, cfg)

	cmd := &Probe{
		anc: anc,
	}

	drv, tm := uitest.TestTUI(ctx, t)
	drv.Replace(fmt.Sprintf("[::1]:%d", attachments[0].Port), "[::1]:<service-port>")
	drv.Replace(fmt.Sprintf("127.0.0.1:%d", attachments[0].Port), "127.0.0.1:<service-port>")

	errc := make(chan error, 1)
	go func() {
		errc <- cmd.UI().RunTUI(ctx, drv)
		errc <- tm.Quit()
	}()

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*2))
	uitest.TestGolden(t, drv.Golden())
}

func TestProbeErrors(t *testing.T) {
	if srv.IsProxy() {
		t.Skip("service probe errors unsupported in proxy mode")
	}

	apiToken, err := srv.GeneratePAT("anky@anchor.dev")
	if err != nil {
		t.Fatal(err)
	}

	setup := func(ctx context.Context) (*cli.Config, *api.Session, error) {
		cfg := cmdtest.Config(ctx)
		cfg.API.Token = apiToken
		cfg.API.URL = srv.URL
		cfg.Service.Probe.Timeout = 2 * time.Second

		anc, err := api.NewClient(ctx, cfg)
		return cfg, anc, err
	}

	t.Run("dns-failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dnsZone := &dns.Zone{
			Origin: ".",
			TTL:    1 * time.Minute,
			RRs:    dns.RRSet{},
		}

		dnsMux := new(dns.ResolveMux)
		dnsMux.Handle(dns.TypeANY, dnsZone.Origin, dnsZone)

		dnsClient := &dns.Client{
			Resolver: dnsMux,
		}

		resolver := &net.Resolver{
			PreferGo: true,
			Dial:     dnsClient.Dial,
		}

		cfg, anc, err := setup(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Test.NetResolver = resolver

		ctx = cli.ContextWithConfig(ctx, cfg)

		cmd := &Probe{
			anc: anc,
		}

		drv, tm := uitest.TestTUI(ctx, t)

		if _, err = resolver.LookupHost(ctx, "service.lcl.host"); err == nil {
			t.Fatal("expected DNS lookup error, got none")
		}
		drv.Replace(err.Error(), "<dns-lookup-error>")

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*2))
		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("tcp-failure-connection-timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		dialer := cli.DialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, context.DeadlineExceeded
		})

		cfg, anc, err := setup(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Test.NetDialer = dialer
		cfg.Service.Probe.Timeout = 100 * time.Millisecond

		ctx = cli.ContextWithConfig(ctx, cfg)

		cmd := &Probe{
			anc: anc,
		}

		drv, tm := uitest.TestTUI(ctx, t)
		drv.Replace("[::1]:44344, ", "")
		drv.Replace("127.0.0.1:44344", "<tcp-addresses>")

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*2))
		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("tcp-failure-connection-failure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ln, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}

		ln.Close() // close right so dial fails

		dialer := cli.DialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			_, port, err := net.SplitHostPort(ln.Addr().String())
			if err != nil {
				return nil, err
			}

			return new(net.Dialer).DialContext(ctx, network, host+":"+port)
		})

		cfg, anc, err := setup(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Test.NetDialer = dialer
		cfg.Service.Probe.Timeout = 100 * time.Millisecond

		ctx = cli.ContextWithConfig(ctx, cfg)

		cmd := &Probe{
			anc: anc,
		}

		drv, tm := uitest.TestTUI(ctx, t)
		drv.Replace("[::1]:44344, ", "")
		drv.Replace("127.0.0.1:44344", "<tcp-addresses>")

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*2))
		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("tls-failure-non-tls-server", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		srvHTTP := httptest.NewServer(http.NotFoundHandler())

		dialer := cli.DialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			_, port, err := net.SplitHostPort(srvHTTP.Listener.Addr().String())
			if err != nil {
				return nil, err
			}

			return new(net.Dialer).DialContext(ctx, network, host+":"+port)
		})

		cfg, anc, err := setup(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Test.NetDialer = dialer

		ctx = cli.ContextWithConfig(ctx, cfg)

		cmd := &Probe{
			anc: anc,
		}

		drv, tm := uitest.TestTUI(ctx, t)
		drv.Replace("[::1]:44344, ", "")
		drv.Replace("127.0.0.1:44344", "<tcp-addresses>")

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*2))
		uitest.TestGolden(t, drv.Golden())
	})

	t.Run("tls-failure-unknown-cert", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		srvHTTP := httptest.NewTLSServer(http.NotFoundHandler())

		dialer := cli.DialFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			_, port, err := net.SplitHostPort(srvHTTP.Listener.Addr().String())
			if err != nil {
				return nil, err
			}

			return new(net.Dialer).DialContext(ctx, network, host+":"+port)
		})

		cfg, anc, err := setup(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cfg.Test.NetDialer = dialer

		ctx = cli.ContextWithConfig(ctx, cfg)

		cmd := &Probe{
			anc: anc,
		}

		drv, tm := uitest.TestTUI(ctx, t)
		drv.Replace("[::1]:44344, ", "")
		drv.Replace("127.0.0.1:44344", "<tcp-addresses>")

		errc := make(chan error, 1)
		go func() {
			errc <- cmd.UI().RunTUI(ctx, drv)
			errc <- tm.Quit()
		}()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*2))
		uitest.TestGolden(t, drv.Golden())
	})
}
