package lcl

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"

	"github.com/cli/browser"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/diagnostic"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/ui"
)

type LclConfig struct {
	Config *cli.Config

	anc                *api.Session
	orgSlug, realmSlug string
}

func (c LclConfig) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c LclConfig) runTUI(ctx context.Context, drv *ui.Driver) error {
	var err error
	cmd := &auth.Client{
		Config: c.Config,
		Anc:    c.anc,
		Source: "lclhost",
	}
	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &models.LclConfigHeader{})
	drv.Activate(ctx, &models.LclConfigHint{})

	err = c.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c LclConfig) perform(ctx context.Context, drv *ui.Driver) error {
	if c.orgSlug == "" {
		userInfo, err := c.anc.UserInfo(ctx)
		if err != nil {
			return nil
		}
		c.orgSlug = userInfo.PersonalOrg.Slug
	}

	if c.realmSlug == "" {
		c.realmSlug = "localhost"
	}

	_, diagPort, err := net.SplitHostPort(c.Config.Lcl.DiagnosticAddr)
	if err != nil {
		return err
	}

	inputc := make(chan string)
	drv.Activate(ctx, &models.DomainInput{
		InputCh: inputc,
		Default: "hi-" + c.orgSlug,
		TLD:     "lcl.host",
	})

	var serviceName string
	select {
	case serviceName = <-inputc:
	case <-ctx.Done():
		return ctx.Err()
	}

	domains := []string{serviceName + ".lcl.host", serviceName + ".localhost"}

	cmdProvision := &Provision{
		Config:    c.Config,
		Domains:   domains,
		orgSlug:   c.orgSlug,
		realmSlug: c.realmSlug,
	}

	_, _, _, cert, err := cmdProvision.run(ctx, drv, c.anc, serviceName, "diagnostic")
	if err != nil {
		return err
	}

	domain := cert.Leaf.Subject.CommonName

	var requestedScheme string

	// FIXME: ? spinner while booting server, transitioning to server booted message
	srvDiag := &diagnostic.Server{
		Addr:       c.Config.Lcl.DiagnosticAddr,
		LclHostURL: c.Config.Lcl.LclHostURL,
		GetCertificate: func(cii *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return cert, nil
		},
	}

	if err := srvDiag.Start(ctx); err != nil {
		return err
	}
	requestc := srvDiag.RequestChan()

	auditInfo, err := trust.PerformAudit(ctx, c.Config, c.anc, c.orgSlug, c.realmSlug)
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

		drv.Activate(ctx, &models.LclConfig{
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

		if !c.Config.Trust.MockMode {
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
			drv.Activate(ctx, new(models.LclConfigSuccess))
			return nil
		}

		cmdTrust := &trust.Command{
			Config:    c.Config,
			Anc:       c.anc,
			OrgSlug:   c.orgSlug,
			RealmSlug: c.realmSlug,
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

	drv.Activate(ctx, &models.LclConfig{
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

	if !c.Config.Trust.MockMode {
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

	drv.Activate(ctx, new(models.LclConfigSuccess))

	return nil
}
