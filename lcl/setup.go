package lcl

import (
	"context"
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

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/detection"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/ui"
)

type Setup struct {
	Config *cli.Config

	anc     *api.Session
	orgSlug string
}

func (c Setup) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c Setup) run(ctx context.Context, drv *ui.Driver) error {
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

	drv.Activate(ctx, &models.SetupHeader{})
	drv.Activate(ctx, &models.SetupHint{})

	err = c.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c Setup) perform(ctx context.Context, drv *ui.Driver) error {
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
	if c.Config.Lcl.Setup.Language != "" {
		if langDetector, ok := detection.DetectorsByFlag[c.Config.Lcl.Setup.Language]; !ok {
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
		Done:    "Entered %s domain for local application development",
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
		Config:    c.Config,
		Domains:   domains,
		orgSlug:   c.orgSlug,
		realmSlug: realmSlug,
	}

	service, tlsCert, err := cmdProvision.run(ctx, drv, c.anc, serviceName, serviceCategory, nil)
	if err != nil {
		return err
	}

	cmdCert := cert.Provision{
		Cert:   tlsCert,
		Config: c.Config,
	}

	if err := cmdCert.RunTUI(ctx, drv, domains...); err != nil {
		return err
	}

	setupGuideURL := c.Config.AnchorURL + "/" + url.QueryEscape(c.orgSlug) + "/services/" + url.QueryEscape(service.Slug) + "/guide"
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

	if !c.Config.Trust.MockMode {
		if err := browser.OpenURL(setupGuideURL); err != nil {
			return err
		}
	}

	return nil
}

var (
	parameterizeUnwantedRegex           = regexp.MustCompile(`[^a-z0-9\-_]+`)
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
