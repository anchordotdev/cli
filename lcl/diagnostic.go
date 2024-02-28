package lcl

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"

	"github.com/cli/browser"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/diagnostic"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/ui"
)

type Diagnostic struct {
	Config *cli.Config

	anc                *api.Session
	orgSlug, realmSlug string
}

func (d *Diagnostic) runTUI(ctx context.Context, drv *ui.Driver, cert *tls.Certificate) error {
	_, diagPort, err := net.SplitHostPort(d.Config.Lcl.DiagnosticAddr)
	if err != nil {
		return err
	}

	domain := cert.Leaf.Subject.CommonName

	var requestedScheme string

	// FIXME: ? spinner while booting server, transitioning to server booted message
	srvDiag := &diagnostic.Server{
		Addr:       d.Config.Lcl.DiagnosticAddr,
		LclHostURL: d.Config.Lcl.LclHostURL,
		GetCertificate: func(cii *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cert, nil
		},
	}

	if err := srvDiag.Start(ctx); err != nil {
		return err
	}
	requestc := srvDiag.RequestChan()

	auditInfo, err := trust.PerformAudit(ctx, d.Config, d.anc, d.orgSlug, d.realmSlug)
	if err != nil {
		return err
	}

	// If no certificates are missing, skip http and go directly to https
	if len(auditInfo.Missing) != 0 {
		httpURL, err := url.Parse("http://" + domain + ":" + diagPort)
		if err != nil {
			return err
		}
		httpConfirmCh := make(chan struct{})

		drv.Activate(ctx, &models.Diagnostic{
			ConfirmCh: httpConfirmCh,

			Domain:     domain,
			Port:       diagPort,
			Scheme:     "http",
			ShowHeader: true,
		})

		drv.Send(models.OpenURLMsg(httpURL.String()))

		select {
		case <-httpConfirmCh:
		case <-ctx.Done():
			return ctx.Err()
		}

		if !d.Config.Trust.MockMode {
			if err := browser.OpenURL(httpURL.String()); err != nil {
				return err
			}
		}

		select {
		case requestedScheme = <-requestc:
		case <-ctx.Done():
			return ctx.Err()
		}

		if requestedScheme == "https" {
			// TODO: skip to "detect"
			drv.Activate(ctx, new(models.DiagnosticSuccess))
			return nil
		}

		cmdTrust := &trust.Command{
			Config:    d.Config,
			Anc:       d.anc,
			OrgSlug:   d.orgSlug,
			RealmSlug: d.realmSlug,
		}

		if err := cmdTrust.UI().RunTUI(ctx, drv); err != nil {
			return err
		}
	}

	httpsURL, err := url.Parse("https://" + domain + ":" + diagPort)
	if err != nil {
		return err
	}
	httpsConfirmCh := make(chan struct{})

	drv.Activate(ctx, &models.Diagnostic{
		ConfirmCh: httpsConfirmCh,

		Domain:     domain,
		Port:       diagPort,
		Scheme:     "https",
		ShowHeader: true,
	})

	drv.Send(models.OpenURLMsg(httpsURL.String()))

	select {
	case <-httpsConfirmCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	if !d.Config.Trust.MockMode {
		if err := browser.OpenURL(httpsURL.String()); err != nil {
			return err
		}
	}

	for requestedScheme != "https" {
		select {
		case requestedScheme = <-requestc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	drv.Activate(ctx, new(models.DiagnosticSuccess))

	return nil
}
