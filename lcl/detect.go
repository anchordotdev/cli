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
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/detection"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/ui"
)

type Detect struct {
	Config *cli.Config

	anc     *api.Session
	orgSlug string
}

func (d Detect) UI() cli.UI {
	return cli.UI{
		RunTUI: d.run,
	}
}

func (d Detect) run(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, new(models.DetectPreamble))

	path, err := os.Getwd()
	if err != nil {
		return err
	}
	dirFS := os.DirFS(path).(detection.FS)

	detectors := detection.DefaultDetectors
	if d.Config.Lcl.Detect.Language != "" {
		if langDetector, ok := detection.DetectorsByFlag[d.Config.Lcl.Detect.Language]; !ok {
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
	drv.Activate(ctx, &models.DetectCategory{
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
	drv.Activate(ctx, &models.DetectName{
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
	drv.Activate(ctx, &models.DomainInput{
		InputCh:    inputc,
		Default:    serviceName,
		TLD:        "lcl.host",
		SkipHeader: true,
	})

	var serviceSubdomain string
	select {
	case serviceSubdomain = <-inputc:
	case <-ctx.Done():
		return ctx.Err()
	}

	domains := []string{serviceSubdomain + ".lcl.host", serviceSubdomain + ".localhost"}

	cmdProvision := &Provision{
		Config:    d.Config,
		Domains:   domains,
		orgSlug:   d.orgSlug,
		realmSlug: "localhost",
	}

	service, _, _, tlsCert, err := cmdProvision.run(ctx, drv, d.anc, serviceName, serviceCategory)
	if err != nil {
		return err
	}

	cmdCert := cert.Provision{
		Cert:   tlsCert,
		Config: d.Config,
	}

	if err := cmdCert.RunTUI(ctx, drv, domains...); err != nil {
		return err
	}

	setupGuideURL := d.Config.AnchorURL + "/" + url.QueryEscape(d.orgSlug) + "/services/" + url.QueryEscape(service.Slug) + "/guide"
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
