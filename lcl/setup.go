package lcl

import (
	"context"
	"errors"
	"net/url"
	"os"
	"path/filepath"

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

	drv.Activate(ctx, new(models.SetupHeader))
	drv.Activate(ctx, new(models.SetupHint))

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
			return nil
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

	inputc = make(chan string)
	drv.Activate(ctx, &models.SetupDomain{
		InputCh: inputc,
		Default: serviceName,
		TLD:     "lcl.host",
	})

	var serviceSubdomain string
	select {
	case serviceSubdomain = <-inputc:
	case <-ctx.Done():
		return ctx.Err()
	}

	domains := []string{serviceSubdomain + ".lcl.host", serviceSubdomain + ".localhost"}
	realmSlug := "localhost"

	cmdProvision := &Provision{
		Config:    c.Config,
		Domains:   domains,
		orgSlug:   c.orgSlug,
		realmSlug: realmSlug,
	}

	service, tlsCert, err := cmdProvision.run(ctx, drv, c.anc, serviceName, serviceCategory)
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

	if err := browser.OpenURL(setupGuideURL); err != nil {
		return err
	}

	return nil
}
