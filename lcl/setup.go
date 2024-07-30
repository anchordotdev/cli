package lcl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cli/browser"
	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/component"
	"github.com/anchordotdev/cli/detection"
	"github.com/anchordotdev/cli/lcl/models"
	climodels "github.com/anchordotdev/cli/models"
	"github.com/anchordotdev/cli/service"
	servicemodels "github.com/anchordotdev/cli/service/models"
	"github.com/anchordotdev/cli/trust"
	trustmodels "github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/ui"
)

var CmdLclSetup = cli.NewCmd[Setup](CmdLcl, "setup", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVar(&cfg.Lcl.Setup.Language, "language", "", "Language to integrate with Anchor.")
	cmd.Flags().StringVar(&cfg.Lcl.Setup.Method, "method", "", "Integration method for certificates.")
	cmd.Flags().StringVarP(&cfg.Lcl.Org, "org", "o", "", "Organization for lcl.host application setup.")
	cmd.Flags().StringVarP(&cfg.Lcl.Realm, "realm", "r", "", "Realm for lcl.host application setup.")
	cmd.Flags().StringVarP(&cfg.Lcl.Service, "service", "s", "", "Service for lcl.host application setup.")
})

var (
	MethodAnchor    = "anchor"
	MethodAutomated = "automated"
	MethodManual    = "manual"
	MethodMkcert    = "mkcert"
)

type Setup struct {
	OrgAPID, RealmAPID, ServiceAPID string

	anc *api.Session
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
		return c.createNewService(ctx, drv, orgAPID, realmAPID)
	}

	return c.setupService(ctx, drv, orgAPID, realmAPID, serviceAPID)
}

func (c *Setup) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if c.OrgAPID != "" {
		return c.OrgAPID, nil
	}
	if cfg.Lcl.Org != "" {
		return cfg.Lcl.Org, nil
	}

	selector := &component.Selector[api.Organization]{
		Prompt: "Which organization's lcl.host local development environment do you want to setup?",
		Flag:   "--org",

		Fetcher: &component.Fetcher[api.Organization]{
			FetchFn: func() ([]api.Organization, error) { return c.anc.GetOrgs(ctx) },
		},
	}

	org, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	return org.Apid, nil
}

func (c *Setup) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
	if c.RealmAPID != "" {
		return c.RealmAPID, nil
	}
	if cfg.Lcl.Realm != "" {
		return cfg.Lcl.Realm, nil
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
	if cfg.Service.Env.Service != "" {
		return cfg.Service.Env.Service, nil
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
	if service == nil {
		return "", nil
	}
	return service.Slug, nil
}

func (c *Setup) createNewService(ctx context.Context, drv *ui.Driver, orgAPID, realmAPID string) error {
	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, &models.SetupScan{})

	path, err := os.Getwd()
	if err != nil {
		return err
	}
	dirFS := os.DirFS(path).(detection.FS)

	detectors := detection.DefaultDetectors
	if cfg.Lcl.Setup.Language != "" {
		if langDetector, ok := detection.DetectorsByFlag[cfg.Lcl.Setup.Language]; !ok {
			return errors.New("invalid language specified")
		} else {
			detectors = []detection.Detector{langDetector}
		}
	}

	results, err := detection.Perform(detectors, dirFS)
	if err != nil {
		return err
	}
	drv.Send(results)

	choicec := make(chan string)
	drv.Activate(ctx, &models.SetupCategory{
		ChoiceCh: choicec,
		Results:  results,
	})

	var serviceCategory string
	select {
	case serviceCategory = <-choicec:
	case <-ctx.Done():
		return ctx.Err()
	}

	inputc := make(chan string)
	drv.Activate(ctx, &models.SetupName{
		InputCh: inputc,
		Default: filepath.Base(path), // TODO: use detected name recommendation
	})

	var serviceName string
	select {
	case serviceName = <-inputc:
	case <-ctx.Done():
		return ctx.Err()
	}

	defaultDomain := parameterize(serviceName)

	inputc = make(chan string)
	drv.Activate(ctx, &models.DomainInput{
		InputCh: inputc,
		Default: defaultDomain,
		TLD:     "lcl.host",
		Prompt:  "What lcl.host domain would you like to use for local application development?",
		Done:    "Entered %s domain for local application development.",
	})

	var lclDomain string
	select {
	case lclDomain = <-inputc:
	case <-ctx.Done():
		return ctx.Err()
	}

	drv.Activate(ctx, &models.DomainResolver{
		Domain: lclDomain,
	})

	addrs, err := new(net.Resolver).LookupHost(ctx, lclDomain)
	if err != nil {
		drv.Send(models.DomainStatusMsg(false))
		return err
	}

	for _, addr := range addrs {
		if !slices.Contains(loopbackAddrs, addr) {
			drv.Send(models.DomainStatusMsg(false))

			return fmt.Errorf("%s domain resolved to non-loopback interface address: %s", lclDomain, addr)
		}
	}
	drv.Send(models.DomainStatusMsg(true))

	subdomain := strings.TrimSuffix(lclDomain, ".lcl.host")
	domains := []string{lclDomain, subdomain + ".localhost"}

	cmdProvision := &Provision{
		Domains:   domains,
		orgSlug:   orgAPID,
		realmSlug: realmAPID,
	}

	service, tlsCert, err := cmdProvision.run(ctx, drv, c.anc, serviceName, serviceCategory, nil)
	if err != nil {
		return err
	}

	setupMethod := cfg.Lcl.Setup.Method
	if setupMethod == "" {
		choicec = make(chan string)
		drv.Activate(ctx, &models.SetupMethod{
			ChoiceCh: choicec,
		})

		select {
		case setupMethod = <-choicec:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	switch setupMethod {
	case MethodManual, MethodMkcert:
		return c.manualMethod(ctx, drv, orgAPID, realmAPID, service.Slug, tlsCert, domains...)
	case MethodAnchor, MethodAutomated:
		setupGuideURL := cfg.AnchorURL + "/" + url.QueryEscape(orgAPID) + "/services/" + url.QueryEscape(service.Slug) + "/guide"
		lclURL := fmt.Sprintf("https://%s:%d", lclDomain, *service.LocalhostPort)
		return c.automatedMethod(ctx, drv, setupGuideURL, lclURL)
	default:
		return fmt.Errorf("Unknown method: %s. Please choose either `anchor` (recommended) or `mkcert`.", setupMethod)
	}
}

func (c *Setup) setupService(ctx context.Context, drv *ui.Driver, orgAPID, realmAPID, serviceAPID string) error {
	drv.Activate(ctx, trustmodels.TrustHeader)
	drv.Activate(ctx, trustmodels.TrustHint)

	cmdTrust := &trust.Command{
		Anc:       c.anc,
		OrgSlug:   orgAPID,
		RealmSlug: realmAPID,
	}

	if err := cmdTrust.Perform(ctx, drv); err != nil {
		return err
	}

	drv.Activate(ctx, servicemodels.ServiceEnvHeader)
	drv.Activate(ctx, servicemodels.ServiceEnvHint)

	cmdServiceEnv := &service.Env{
		Anc:         c.anc,
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

func (c *Setup) automatedMethod(ctx context.Context, drv *ui.Driver, setupGuideURL string, LclURL string) error {
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

	cfg := cli.ConfigFromContext(ctx)
	if !cfg.Trust.MockMode {
		if err := browser.OpenURL(setupGuideURL); err != nil {
			drv.Activate(ctx, &climodels.Browserless{Url: setupGuideURL})
		}
	}

	drv.Activate(ctx, &models.SetupGuideHint{LclUrl: LclURL})

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
