package trust

import (
	"context"
	"os"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
)

type Clean struct {
	Config *cli.Config
}

func (c Clean) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c *Clean) runTUI(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &models.CleanPreflight{
		CertStates:  c.Config.Trust.Clean.States,
		TrustStores: c.Config.Trust.Stores,
	})

	anc, err := api.NewClient(c.Config)
	if err != nil {
		return err
	}

	userInfo, err := anc.UserInfo(ctx)
	if err != nil {
		return err
	}
	drv.Send(models.HandleMsg(userInfo.PersonalOrg.Slug))

	org, realm, err := fetchOrgAndRealm(ctx, c.Config, anc)
	if err != nil {
		return err
	}

	expectedCAs, err := fetchExpectedCAs(ctx, anc, org, realm)
	if err != nil {
		return err
	}
	drv.Send(models.ExpectedCAsMsg(expectedCAs))

	stores, sudoMgr, err := loadStores(c.Config)
	if err != nil {
		return err
	}

	sudoMgr.AroundSudo = func(sudo func()) {
		unpausec := drv.Pause()
		defer close(unpausec)

		sudo()
	}

	audit := &truststore.Audit{
		Expected: expectedCAs,
		Stores:   stores,
		SelectFn: checkAnchorCert,
	}

	info, err := audit.Perform()
	if err != nil {
		return err
	}

	targetCAs := info.AllCAs(c.Config.Trust.Clean.States...)
	drv.Send(models.TargetCAsMsg(targetCAs))

	tmpDir, err := os.MkdirTemp("", "anchor-trust-clean")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, ca := range targetCAs {
		if ca.FilePath == "" {
			if err := writeCAFile(ca, tmpDir); err != nil {
				return err
			}
		}

		confirmc := make(chan struct{})

		drv.Activate(ctx, &models.CleanCA{
			CA:        ca,
			ConfirmCh: confirmc,
		})

		select {
		case <-confirmc:
		case <-ctx.Done():
			return ctx.Err()
		}

		for _, store := range stores {
			if !info.IsPresent(ca, store) {
				continue
			}

			drv.Send(models.CleaningStoreMsg{Store: store})

			if _, err := store.UninstallCA(ca); err != nil {
				return err
			}

			drv.Send(models.CleanedStoreMsg{Store: store})
		}
	}

	drv.Activate(ctx, &models.CleanEpilogue{
		Count: len(targetCAs),
	})

	return nil
}
