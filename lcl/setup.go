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
	"github.com/anchordotdev/cli/detection"
	"github.com/anchordotdev/cli/lcl/models"
	climodels "github.com/anchordotdev/cli/models"
	"github.com/anchordotdev/cli/ui"
)

var CmdLclSetup = cli.NewCmd[Setup](CmdLcl, "setup", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVar(&cfg.Lcl.Setup.Language, "language", "", "Language to integrate with Anchor.")
	cmd.Flags().StringVar(&cfg.Lcl.Setup.Method, "method", "", "Integration method for certificates.")
})

var (
	MethodAutomatic = "automatic"
	MethodManual    = "manual"
)

type Setup struct {
	anc     *api.Session
	orgSlug string
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

	drv.Activate(ctx, &models.SetupScan{})

	if c.orgSlug == "" {
		userInfo, err := c.anc.UserInfo(ctx)
		if err != nil {
			return err
		}
		c.orgSlug = userInfo.PersonalOrg.Slug
	}

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
	realmSlug := "localhost"

	cmdProvision := &Provision{
		Domains:   domains,
		orgSlug:   c.orgSlug,
		realmSlug: realmSlug,
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
	case MethodManual:
		if err := c.manualMethod(ctx, drv, tlsCert, domains...); err != nil {
			return err
		}
	case MethodAutomatic:
		setupGuideURL := cfg.AnchorURL + "/" + url.QueryEscape(c.orgSlug) + "/services/" + url.QueryEscape(service.Slug) + "/guide"
		lclURL := fmt.Sprintf("https://%s:%d", lclDomain, *service.LocalhostPort)
		if err := c.automaticMethod(ctx, drv, setupGuideURL, lclURL); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown method: %s. Please choose either `anchor` (recommended) or `mkcert`.", setupMethod)
	}

	return nil
}

func (c *Setup) manualMethod(ctx context.Context, drv *ui.Driver, tlsCert *tls.Certificate, domains ...string) error {
	cmdCert := cert.Provision{
		Cert: tlsCert,
	}

	if err := cmdCert.RunTUI(ctx, drv, domains...); err != nil {
		return err
	}

	return nil
}

func (c *Setup) automaticMethod(ctx context.Context, drv *ui.Driver, setupGuideURL string, LclURL string) error {
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
	value = parameterizeUnwantedRegex.ReplaceAllString(value, "-")
	value = parameterizeDuplicateSeparatorRegex.ReplaceAllString(value, "-")
	value = parameterizeLeadingTrailingRegex.ReplaceAllString(value, "")
	value = strings.ToLower(value) // fwiw: not part of rails parameterize

	return value
}
