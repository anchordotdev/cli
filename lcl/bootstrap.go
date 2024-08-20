package lcl

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/cli/browser"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/diagnostic"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	trustmodels "github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
)

var CmdBootstrap = cli.NewCmd[Bootstrap](CmdLcl, "bootstrap", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Lcl.Diagnostic.Addr, "addr", "a", cli.Defaults.Lcl.Diagnostic.Addr, "Address for local diagnostic web server.")
})

var loopbackAddrs = []string{"127.0.0.1", "::1"}

type Bootstrap struct {
	anc *api.Session

	auditInfo *truststore.AuditInfo
}

func (c Bootstrap) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c Bootstrap) runTUI(ctx context.Context, drv *ui.Driver) error {
	if cli.CalledAsFromContext(ctx) == "config" {
		drv.Activate(ctx, ui.Section{
			Name: "LclConfigDeprecation",
			Model: ui.MessageLines{
				ui.Warning(fmt.Sprintf("%s is deprecated, use %s",
					ui.Whisper("`anchor lcl config`"),
					ui.Whisper("`anchor lcl bootstrap`"),
				)),
			},
		})
	}

	var err error
	cmd := &auth.Client{
		Anc:    c.anc,
		Source: "lclhost",
	}
	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.BootstrapHeader)
	drv.Activate(ctx, models.BootstrapHint)

	err = c.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c Bootstrap) perform(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	orgAPID, err := c.personalOrgAPID(ctx)
	if err != nil {
		return err
	}

	realmAPID, err := c.localhostRealmAPID()
	if err != nil {
		return err
	}

	_, diagPort, err := net.SplitHostPort(cfg.Lcl.Diagnostic.Addr)
	if err != nil {
		return err
	}

	diagService, err := c.getDiagnosticService(ctx, drv, orgAPID)
	if err != nil {
		return err
	}
	if diagService == nil {
		drv.Send(models.BootstrapDiagnosticNotFoundMsg{})

		if diagService, err = c.createDiagnosticService(ctx, drv, orgAPID, diagPort); err != nil {
			return nil
		}
	} else {
		drv.Send(models.BootstrapDiagnosticFoundMsg{})

		domains := []string{diagService.Slug + ".lcl.host", diagService.Slug + ".localhost"}

		drv.Activate(ctx, &models.ProvisionService{
			Name:       diagService.Slug,
			Domains:    domains,
			ServerType: "diagnostic",
		})
	}

	srvDiag, err := c.diagnosticServer(ctx, cfg, drv, realmAPID, diagService)
	if err != nil {
		return err
	}

	requestc := srvDiag.RequestChan()
	if err := srvDiag.Start(ctx); err != nil {
		return err
	}
	defer srvDiag.Close()

	drv.Send(models.ServiceProvisionedMsg{})

	auditInfo := c.auditInfo
	if auditInfo == nil {
		if auditInfo, err = c.performAudit(ctx, drv, orgAPID, realmAPID); err != nil {
			return err
		}
	}

	// If no certificates are missing, skip http and go directly to https
	if len(auditInfo.Missing) != 0 {
		okHTTPS, err := c.checkHTTP(ctx, cfg, drv, diagService, diagPort, requestc)
		if err != nil {
			return err
		}
		if okHTTPS {
			// TODO: doesn't seem possible if srvDiag.EnableTLS() is called after
			drv.Activate(ctx, new(models.BootstrapSuccess))
			return nil
		}

		drv.Activate(ctx, trustmodels.TrustHeader)

		cmdTrust := &Trust{
			anc:       c.anc,
			auditInfo: auditInfo,
		}

		if err = cmdTrust.perform(ctx, drv); err != nil {
			return err
		}
	}

	srvDiag.EnableTLS()

	return c.checkHTTPS(ctx, cfg, drv, diagService, diagPort, requestc)
}

func (c Bootstrap) personalOrgAPID(ctx context.Context) (string, error) {
	userInfo, err := c.anc.UserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.PersonalOrg.Slug, nil
}

func (c Bootstrap) localhostRealmAPID() (string, error) {
	return "localhost", nil
}

func (c Bootstrap) createDiagnosticService(ctx context.Context, drv *ui.Driver, orgAPID, diagPort string) (*api.Service, error) {
	serviceName, err := c.diagnosticServiceName(ctx, drv, "hi-"+orgAPID)
	if err != nil {
		return nil, err
	}

	domains := []string{serviceName + ".lcl.host", serviceName + ".localhost"}

	if err := checkLoopbackDomain(ctx, drv, domains[0]); err != nil {
		return nil, err
	}

	drv.Activate(ctx, &models.ProvisionService{
		Name:       serviceName,
		Domains:    domains,
		ServerType: "diagnostic",
	})

	srv, err := c.anc.GetService(ctx, orgAPID, serviceName)
	if err != nil {
		return nil, err
	}
	if srv != nil {
		return srv, nil
	}

	localhostPort, err := strconv.Atoi(diagPort)
	if err != nil {
		return nil, err
	}

	return c.anc.CreateService(ctx, orgAPID, serviceName, "diagnostic", &localhostPort)
}

func (c Bootstrap) diagnosticServer(ctx context.Context, cfg *cli.Config, drv *ui.Driver, realmAPID string, srv *api.Service) (*diagnostic.Server, error) {
	chainAPID := "ca" // FIXME: we need to lookup and pass the chain and/or make it non-optional
	domains := []string{srv.Slug + ".lcl.host", srv.Slug + ".localhost"}

	atch, err := c.anc.AttachService(ctx, chainAPID, domains, srv.Relationships.Organization.Slug, realmAPID, srv.Slug)
	if err != nil {
		return nil, err
	}

	mkcert := &MkCert{
		anc:         c.anc,
		domains:     domains,
		OrgAPID:     srv.Relationships.Organization.Slug,
		RealmAPID:   realmAPID,
		ServiceAPID: srv.Slug,

		ChainAPID: atch.Relationships.Chain.Slug,
		SubCaAPID: atch.Relationships.SubCa.Slug,
	}

	tlsCert, err := mkcert.perform(ctx, drv)
	if err != nil {
		return nil, err
	}

	return &diagnostic.Server{
		Addr:       cfg.Lcl.Diagnostic.Addr,
		LclHostURL: cfg.Lcl.LclHostURL,
		GetCertificate: func(cii *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return tlsCert, nil
		},
	}, nil
}

func (c Bootstrap) performAudit(ctx context.Context, drv *ui.Driver, orgAPID, realmAPID string) (*truststore.AuditInfo, error) {
	stores, err := trust.LoadStores(ctx, drv)
	if err != nil {
		return nil, err
	}

	cas, err := trust.FetchExpectedCAs(ctx, c.anc, orgAPID, realmAPID)
	if err != nil {
		return nil, err
	}

	return trust.PerformAudit(ctx, stores, cas)
}

func (c Bootstrap) checkHTTP(ctx context.Context, cfg *cli.Config, drv *ui.Driver, srv *api.Service, diagPort string, requestc <-chan string) (bool, error) {
	domain := srv.Slug + ".lcl.host" // TODO: look this up properly

	httpURL, err := url.Parse("http://" + domain + ":" + diagPort)
	if err != nil {
		return false, err
	}
	httpConfirmCh := make(chan struct{})

	drv.Activate(ctx, &models.Bootstrap{
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
		return false, ctx.Err()
	}

	var browserless bool
	if !cfg.Trust.MockMode {
		if err := browser.OpenURL(httpURL.String()); err != nil {
			browserless = true
			drv.Activate(ctx, models.Browserless)
		}
	}

	var requestedScheme string
	if !browserless {
		select {
		case requestedScheme = <-requestc:
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}

	if requestedScheme == "https" {
		drv.Activate(ctx, new(models.BootstrapSuccess))
		return true, nil
	}

	return false, nil
}

func (c Bootstrap) checkHTTPS(ctx context.Context, cfg *cli.Config, drv *ui.Driver, srv *api.Service, diagPort string, requestc <-chan string) error {
	domain := srv.Slug + ".lcl.host" // TODO: look this up properly

	httpsURL, err := url.Parse("https://" + domain + ":" + diagPort)
	if err != nil {
		return err
	}
	httpsConfirmCh := make(chan struct{})

	drv.Activate(ctx, &models.Bootstrap{
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

	var browserless bool
	if !cfg.Trust.MockMode {
		if err := browser.OpenURL(httpsURL.String()); err != nil {
			browserless = true
			drv.Activate(ctx, models.Browserless)
		}
	}

	var requestedScheme string
	if !browserless {
		for requestedScheme != "https" {
			select {
			case requestedScheme = <-requestc:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	drv.Activate(ctx, &models.BootstrapSuccess{
		URL: httpsURL,
	})

	return nil
}

func (c Bootstrap) diagnosticServiceName(ctx context.Context, drv *ui.Driver, defaultSubdomain string) (string, error) {
	inputc := make(chan string)
	drv.Activate(ctx, &models.DomainInput{
		InputCh: inputc,
		Default: defaultSubdomain,
		TLD:     "lcl.host",
		Prompt:  "What lcl.host domain would you like to use for diagnostics?",
		Done:    "Entered %s domain for lcl.host diagnostic certificate.",
	})

	select {
	case lclDomain := <-inputc:
		return strings.TrimSuffix(lclDomain, ".lcl.host"), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c Bootstrap) getDiagnosticService(ctx context.Context, drv *ui.Driver, orgAPID string) (*api.Service, error) {
	drv.Activate(ctx, &models.BootstrapDiagnostic{})

	var diagnosticService *api.Service
	services, err := c.anc.GetOrgServices(ctx, orgAPID)
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		if service.ServerType == "diagnostic" {
			diagnosticService = &service
		}
	}

	if diagnosticService == nil {
		drv.Send(models.BootstrapDiagnosticNotFoundMsg{})
	} else {
		drv.Send(models.BootstrapDiagnosticFoundMsg{})
	}

	return diagnosticService, nil
}
