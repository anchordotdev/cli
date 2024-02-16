package lcl

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/ui"
)

type Command struct {
	Config *cli.Config
}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Command) run(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &models.LclPreamble{})

	anc, err := api.NewClient(c.Config)
	if errors.Is(err, api.ErrSignedOut) {
		if err := c.runSignIn(ctx, drv); err != nil {
			return err
		}
		if anc, err = api.NewClient(c.Config); err != nil {
			return err
		}
	}

	userInfo, err := anc.UserInfo(ctx)
	if errors.Is(err, api.ErrSignedOut) {
		if err := c.runSignIn(ctx, drv); err != nil {
			return err
		}
		if anc, err = api.NewClient(c.Config); err != nil {
			return err
		}
		if userInfo, err = anc.UserInfo(ctx); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	orgSlug := userInfo.PersonalOrg.Slug
	realmSlug := "localhost"

	drv.Activate(ctx, &models.LclScan{})

	auditInfo, err := trust.PerformAudit(ctx, c.Config, anc, orgSlug, realmSlug)
	if err != nil {
		return err
	}

	var diagnosticService *api.Service
	services, err := anc.GetOrgServices(ctx, orgSlug)
	if err != nil {
		return err
	}
	for _, service := range services {
		if service.ServerType == "diagnostic" {
			diagnosticService = &service
		}
	}

	drv.Send(models.ScanFinishedMsg{})

	if diagnosticService == nil || len(auditInfo.Missing) != 0 {
		inputc := make(chan string)
		drv.Activate(ctx, &models.DomainInput{
			InputCh: inputc,
			Default: "hi-" + orgSlug,
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
			orgSlug:   orgSlug,
			realmSlug: realmSlug,
		}

		_, _, _, cert, err := cmdProvision.run(ctx, drv, anc, serviceName, "diagnostic")
		if err != nil {
			return err
		}

		cmdDiagnostic := &Diagnostic{
			Config:    c.Config,
			anc:       anc,
			orgSlug:   orgSlug,
			realmSlug: realmSlug,
		}

		if err := cmdDiagnostic.runTUI(ctx, drv, cert); err != nil {
			return err
		}
	}

	// run detect command

	cmdDetect := &Detect{
		Config:  c.Config,
		anc:     anc,
		orgSlug: orgSlug,
	}

	if err := cmdDetect.run(ctx, drv); err != nil {
		return err
	}

	return nil
}

func (c *Command) runSignIn(ctx context.Context, drv *ui.Driver) error {
	cmdSignIn := &auth.SignIn{
		Config:   c.Config,
		Preamble: ui.StepHint("You need to signin first, so we can track resources for you."),
		Source:   "lclhost",
	}
	return cmdSignIn.RunTUI(ctx, drv)
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

func createEAB(anc *http.Client, chainParam string, orgParam string, realmParam string, serviceParam string, subCaParam string) (*api.Eab, error) {
	eabBody := new(bytes.Buffer)
	eabReq := api.CreateEabTokenJSONRequestBody{}
	eabReq.Relationships.Chain.Slug = chainParam
	eabReq.Relationships.Organization.Slug = orgParam
	eabReq.Relationships.Realm.Slug = realmParam
	eabReq.Relationships.Service.Slug = &serviceParam
	eabReq.Relationships.SubCa.Slug = subCaParam

	if err := json.NewEncoder(eabBody).Encode(eabReq); err != nil {
		return nil, err
	}
	eabRes, err := anc.Post("/acme/eab-tokens", "application/json", eabBody)
	if err != nil {
		return nil, err
	}
	if eabRes.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response")
	}
	var eab api.Eab
	if err := json.NewDecoder(eabRes.Body).Decode(&eab); err != nil {
		return nil, err
	}
	return &eab, nil
}
