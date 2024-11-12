package lcl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cli/browser"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/component"
	componentmodels "github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/detection"
	"github.com/anchordotdev/cli/lcl/models"
	climodels "github.com/anchordotdev/cli/models"
	"github.com/anchordotdev/cli/service"
	servicemodels "github.com/anchordotdev/cli/service/models"
	"github.com/anchordotdev/cli/ui"
)

var CmdLclSetup = cli.NewCmd[Setup](CmdLcl, "setup", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVar(&cfg.Service.Category, "category", cli.Defaults.Service.Category, "Language or software type of the service.")
	cmd.Flags().StringVar(&cfg.Service.CertStyle, "cert-style", cli.Defaults.Service.CertStyle, "Provisioning method for lcl.host certificates.")
	cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization for lcl.host application setup.")
	cmd.Flags().StringVarP(&cfg.Lcl.RealmAPID, "realm", "r", cli.Defaults.Lcl.RealmAPID, "Realm for lcl.host application setup.")
	cmd.Flags().StringVarP(&cfg.Service.APID, "service", "s", cli.Defaults.Service.APID, "Service for lcl.host application setup.")

	// alias
	cmd.Flags().StringVar(&cfg.Service.Category, "language", cli.Defaults.Service.Category, "Language to integrate with Anchor.")
	_ = cmd.Flags().MarkDeprecated("language", "Please use `--category` instead.")
	cmd.Flags().StringVar(&cfg.Service.CertStyle, "method", cli.Defaults.Service.CertStyle, "Provisioning method for lcl.host certificates.")
	_ = cmd.Flags().MarkDeprecated("method", "Please use `--cert-style` instead.")
})

var (
	MethodACME      = "acme"
	MethodAnchor    = "anchor"
	MethodAutomated = "automated"
	MethodManual    = "manual"
	MethodMkcert    = "mkcert"
)

type Setup struct {
	OrgAPID, RealmAPID, ServiceAPID string

	anc       *api.Session
	clipboard cli.Clipboard
}

func (c Setup) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Setup) run(ctx context.Context, drv *ui.Driver) error {
	var err error
	cmd := &auth.Client{
		Anc:    c.anc,
		Source: "lclhost",
	}
	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.SetupHeader)
	drv.Activate(ctx, models.SetupHint)

	err = c.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c *Setup) perform(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	orgAPID, err := c.orgAPID(ctx, cfg, drv)
	if err != nil {
		return err
	}

	realmAPID, err := c.realmAPID(ctx, cfg, drv, orgAPID)
	if err != nil {
		return err
	}

	serviceAPID, err := c.serviceAPID(ctx, cfg, drv, orgAPID, realmAPID)
	if err != nil {
		return err
	}

	if serviceAPID == "" {
		return c.initialSetup(ctx, cfg, drv, orgAPID, realmAPID)
	}
	return c.setupServiceEnv(ctx, drv, orgAPID, realmAPID, serviceAPID)
}

func (c *Setup) initialSetup(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, realmAPID string) error {
	// TODO: select name before category

	category, err := c.serviceCategory(ctx, cfg, drv)
	if err != nil {
		return err
	}

	name, err := c.serviceName(ctx, cfg, drv)
	if err != nil {
		return err
	}

	lclDomain, err := c.serviceDomain(ctx, drv, name)
	if err != nil {
		return err
	}
	if err := checkLoopbackDomain(ctx, drv, lclDomain); err != nil {
		return err
	}

	subdomain := strings.TrimSuffix(lclDomain, ".lcl.host")
	domains := []string{lclDomain, subdomain + ".localhost"}
	port := cfg.LclHostPort()

	drv.Activate(ctx, &models.ProvisionService{
		Name:       name,
		Domains:    domains,
		ServerType: category,
	})

	srv, err := c.anc.GetService(ctx, orgAPID, name)
	if err != nil {
		return err
	}
	if srv == nil {
		if srv, err = c.anc.CreateService(ctx, orgAPID, name, category, port); err != nil {
			return err
		}
	}

	// FIXME: we need to lookup and pass the chain and/or make it non-optional
	chainAPID := "ca"

	atch, err := c.anc.AttachService(ctx, chainAPID, domains, orgAPID, realmAPID, srv.Slug)
	if err != nil {
		return err
	}

	mkcert := &MkCert{
		anc:         c.anc,
		Domains:     domains,
		OrgAPID:     orgAPID,
		RealmAPID:   realmAPID,
		ServiceAPID: srv.Slug,

		ChainAPID: atch.Relationships.Chain.Slug,
		SubCaAPID: atch.Relationships.SubCa.Slug,
	}

	tlsCert, err := mkcert.perform(ctx, cfg, drv)
	if err != nil {
		return err
	}
	drv.Send(models.ServiceProvisionedMsg{})

	certStyle, err := c.certStyle(ctx, cfg, drv)
	if err != nil {
		return err
	}

	switch certStyle {
	case MethodManual, MethodMkcert:
		certStyle = MethodMkcert
		if err := c.manualMethod(ctx, drv, orgAPID, realmAPID, srv.Slug, tlsCert, domains...); err != nil {
			return err
		}
	case MethodACME, MethodAnchor, MethodAutomated:
		certStyle = MethodACME
		setupGuideURL := cfg.SetupGuideURL(orgAPID, srv.Slug)
		lclURL := fmt.Sprintf("https://%s:%d", lclDomain, *srv.LocalhostPort)
		if err := c.automatedMethod(ctx, cfg, drv, setupGuideURL, lclURL); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown method: %s. Please choose either `acme` (recommended) or `mkcert`.", certStyle)
	}

	return c.writeTOML(ctx, cfg, drv, orgAPID, realmAPID, srv.Slug, category, certStyle)
}

func (c *Setup) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if c.OrgAPID != "" {
		return c.OrgAPID, nil
	}

	if cfg.Org.APID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Org.APID },
			Flag:          "--org",
			Singular:      "organization",
		})
		return cfg.Org.APID, nil
	}

	selector := &component.Selector[api.Organization]{
		Prompt: "Which organization's lcl.host local development environment do you want to setup?",
		Flag:   "--org",

		Creatable: true,

		Fetcher: &component.Fetcher[api.Organization]{
			FetchFn: func() ([]api.Organization, error) { return c.anc.GetOrgs(ctx) },
		},
	}

	org, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	if org == nil || (*org == api.Organization{}) {
		orgName, err := c.orgName(ctx, cfg, drv)
		if err != nil {
			return "", err
		}

		if org, err = c.anc.CreateOrg(ctx, orgName); err != nil {
			return "", err
		}
		// FIXME: provide nicer output about using newly created value, and hint flag?
		return org.Apid, nil
	}
	return org.Apid, nil

}

func (c *Setup) orgName(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if cfg.Org.Name != "" {
		return cfg.Org.Name, nil
	}

	inputc := make(chan string)
	drv.Activate(ctx, &models.SetupOrgName{
		InputCh: inputc,
	})

	select {
	case orgName := <-inputc:
		return orgName, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Setup) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
	if c.RealmAPID != "" {
		return c.RealmAPID, nil
	}

	if cfg.Lcl.RealmAPID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Lcl.RealmAPID },
			Flag:          "--realm",
			Singular:      "realm",
		})
		return cfg.Lcl.RealmAPID, nil
	}

	selector := &component.Selector[api.Realm]{
		Prompt: fmt.Sprintf("Which %s realm's lcl.host local development environment do you want to setup?", ui.Emphasize(orgAPID)),
		Flag:   "--realm",

		Fetcher: &component.Fetcher[api.Realm]{
			FetchFn: func() ([]api.Realm, error) { return c.anc.GetOrgRealms(ctx, orgAPID) },
		},
	}

	realm, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	return realm.Apid, nil
}

func (c *Setup) serviceAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, realmAPID string) (string, error) {
	if c.ServiceAPID != "" {
		return c.ServiceAPID, nil
	}

	if cfg.Service.APID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Service.APID },
			Flag:          "--service",
			Singular:      "service",
		})
		return cfg.Service.APID, nil
	}

	selector := &component.Selector[api.Service]{
		Prompt: fmt.Sprintf("Which %s/%s service's lcl.host local development environment do you want to setup?", ui.Emphasize(orgAPID), ui.Emphasize(realmAPID)),
		Flag:   "--service",

		Creatable: true,

		Fetcher: &component.Fetcher[api.Service]{
			FetchFn: func() ([]api.Service, error) { return c.anc.GetOrgServices(ctx, orgAPID, api.NonDiagnosticServices) },
		},
	}

	service, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	if service == nil || (*service == api.Service{}) {
		return "", nil
	}
	return service.Slug, nil
}

func (c *Setup) serviceName(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if cfg.Service.Name != "" {
		return cfg.Service.Name, nil
	}

	var defaultName string
	if path, err := os.Getwd(); err == nil {
		defaultName = filepath.Base(path) // TODO: use detected name recommendation
	}

	inputc := make(chan string)
	drv.Activate(ctx, &models.SetupServiceName{
		InputCh: inputc,
		Default: defaultName,
	})

	select {
	case serviceName := <-inputc:
		return serviceName, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Setup) serviceCategory(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if cfg.Service.Category != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Service.Category },
			Flag:          "--category",
			Singular:      "service category",
		})
		return cfg.Service.Category, nil
	}

	drv.Activate(ctx, &models.SetupScan{})

	path, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dirFS := os.DirFS(path).(detection.FS)

	detectors := detection.DefaultDetectors
	if cfg.Service.Category != "" {
		if langDetector, ok := detection.DetectorsByFlag[cfg.Service.Category]; !ok {
			return "", errors.New("invalid language specified")
		} else {
			detectors = []detection.Detector{langDetector}
		}
	}

	results, err := detection.Perform(detectors, dirFS)
	if err != nil {
		return "", err
	}
	drv.Send(results)

	choicec := make(chan string)
	drv.Activate(ctx, &models.SetupCategory{
		ChoiceCh: choicec,
		Results:  results,
	})

	select {
	case category := <-choicec:
		return category, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Setup) serviceDomain(ctx context.Context, drv *ui.Driver, name string) (string, error) {
	// TODO: check cfg.Lcl.Domain or something

	defaultDomain := parameterize(name)

	inputc := make(chan string)
	drv.Activate(ctx, &models.DomainInput{
		InputCh: inputc,
		Default: defaultDomain,
		TLD:     "lcl.host",
		Prompt:  "What lcl.host domain would you like to use for local application development?",
		Done:    "Entered %s domain for local application development.",
	})

	select {
	case domain := <-inputc:
		return domain, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Setup) certStyle(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if cfg.Service.CertStyle != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Service.CertStyle },
			Flag:          "--cert-style",
			Singular:      "certificate style",
		})
		return cfg.Service.CertStyle, nil
	}

	choicec := make(chan string)
	drv.Activate(ctx, &models.SetupMethod{
		ChoiceCh: choicec,
	})

	select {
	case certStyle := <-choicec:
		return certStyle, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *Setup) setupServiceEnv(ctx context.Context, drv *ui.Driver, orgAPID, realmAPID, serviceAPID string) error {
	drv.Activate(ctx, servicemodels.ServiceEnvHeader)
	drv.Activate(ctx, servicemodels.ServiceEnvHint)

	cmdServiceEnv := &service.Env{
		Anc:       c.anc,
		Clipboard: c.clipboard,

		OrgAPID:     orgAPID,
		RealmAPID:   realmAPID,
		ServiceAPID: serviceAPID,
	}

	err := cmdServiceEnv.Perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c *Setup) manualMethod(ctx context.Context, drv *ui.Driver, orgAPID string, realmAPID string, serviceAPID string, tlsCert *tls.Certificate, domains ...string) error {
	cmdCertProvision := cert.Provision{
		Cert:        tlsCert,
		Domains:     domains,
		OrgAPID:     orgAPID,
		RealmAPID:   realmAPID,
		ServiceAPID: serviceAPID,
	}

	return cmdCertProvision.Perform(ctx, drv)
}

func (c *Setup) automatedMethod(ctx context.Context, cfg *cli.Config, drv *ui.Driver, setupGuideURL string, LclURL string) error {
	setupGuideConfirmCh := make(chan struct{})

	drv.Activate(ctx, &models.SetupGuidePrompt{
		ConfirmCh: setupGuideConfirmCh,
	})

	drv.Send(models.OpenSetupGuideMsg(setupGuideURL))

	select {
	case <-setupGuideConfirmCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	if !cfg.Trust.MockMode {
		if err := browser.OpenURL(setupGuideURL); err != nil {
			drv.Activate(ctx, &climodels.Browserless{Url: setupGuideURL})
		}
	}

	drv.Activate(ctx, &models.SetupGuideHint{LclUrl: LclURL})

	return nil
}

func (c *Setup) writeTOML(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, realmAPID, serviceAPID, category, certStyle string) error {
	cfg = cfg.Copy()

	cfg.Org.APID = orgAPID
	cfg.Lcl.RealmAPID = realmAPID
	cfg.Service.APID = serviceAPID
	cfg.Service.Category = category
	cfg.Service.CertStyle = certStyle

	if err := cfg.WriteTOML(); err != nil {
		return err
	}

	drv.Activate(ctx, models.SetupAnchorTOML)

	return nil
}

var (
	// unlike ActiveSupport parameterize, we also drop underscore as it is invalid in subdomains
	parameterizeUnwantedRegex           = regexp.MustCompile(`[^a-z0-9\-]+`)
	parameterizeDuplicateSeparatorRegex = regexp.MustCompile(`-{2,}`)
	parameterizeLeadingTrailingRegex    = regexp.MustCompile(`^-|-$`)
)

// based on: https://apidock.com/rails/ActiveSupport/Inflector/parameterize
func parameterize(value string) string {
	value = strings.ToLower(value) // fwiw: not part of rails parameterize
	value = parameterizeUnwantedRegex.ReplaceAllString(value, "-")
	value = parameterizeDuplicateSeparatorRegex.ReplaceAllString(value, "-")
	value = parameterizeLeadingTrailingRegex.ReplaceAllString(value, "")

	return value
}
