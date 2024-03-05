package lcl

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/ui"
)

type Command struct {
	Config *cli.Config

	anc *api.Session
}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Command) run(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &models.LclPreamble{})

	var err error
	cmd := &auth.Client{
		Config: c.Config,
		Anc:    c.anc,
		Hint:   &models.LclSignInHint{},
		Source: "lclhost",
	}
	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &models.LclHeader{})
	drv.Activate(ctx, &models.LclHint{})

	userInfo, err := c.anc.UserInfo(ctx)
	if err != nil {
		return err
	}

	orgSlug := userInfo.PersonalOrg.Slug
	realmSlug := "localhost"

	// run audit command
	drv.Activate(ctx, &models.AuditHeader{})
	drv.Activate(ctx, &models.AuditHint{})

	cmdAudit := &Audit{
		Config:    c.Config,
		anc:       c.anc,
		orgSlug:   orgSlug,
		realmSlug: realmSlug,
	}

	lclAuditResult, err := cmdAudit.perform(ctx, drv)
	if err != nil {
		return err
	}

	if lclAuditResult.diagnosticServiceExists && lclAuditResult.trusted {
		drv.Activate(ctx, &models.LclConfigSkip{})
	} else {
		// run config command
		drv.Activate(ctx, &models.LclConfigHeader{})
		drv.Activate(ctx, &models.LclConfigHint{})

		cmdConfig := &LclConfig{
			Config:    c.Config,
			anc:       c.anc,
			orgSlug:   orgSlug,
			realmSlug: realmSlug,
		}

		if err := cmdConfig.perform(ctx, drv); err != nil {
			return err
		}
	}

	// run setup command
	drv.Activate(ctx, &models.SetupHeader{})
	drv.Activate(ctx, &models.SetupHint{})

	cmdSetup := &Setup{
		Config:  c.Config,
		anc:     c.anc,
		orgSlug: orgSlug,
	}

	err = cmdSetup.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func provisionCert(eab *api.Eab, domains []string, acmeURL string) (*tls.Certificate, error) {
	hmacKey, err := base64.URLEncoding.DecodeString(eab.HmacKey)
	if err != nil {
		return nil, err
	}

	mgr := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domains...),
		Client: &acme.Client{
			DirectoryURL: acmeURL,
		},
		ExternalAccountBinding: &acme.ExternalAccountBinding{
			KID: eab.Kid,
			Key: hmacKey,
		},
		RenewBefore: 24 * time.Hour,
	}

	// TODO: switch to using ACME package here, so that extra domains can be sent through for SAN extension
	clientHello := &tls.ClientHelloInfo{
		ServerName:   domains[0],
		CipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
	}

	return mgr.GetCertificate(clientHello)
}
