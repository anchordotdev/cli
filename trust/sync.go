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

type Sync struct {
	Config *cli.Config

	Anc                *api.Session
	OrgSlug, RealmSlug string
}

func (s Sync) UI() cli.UI {
	return cli.UI{
		RunTUI: s.runTUI,
	}
}

func (s *Sync) runTUI(ctx context.Context, drv *ui.Driver) error {
	confirmc := make(chan struct{})
	drv.Activate(ctx, &models.SyncPreflight{
		NonInteractive: s.Config.NonInteractive,
		ConfirmCh:      confirmc,
	})

	cas, err := fetchExpectedCAs(ctx, s.Anc, s.OrgSlug, s.RealmSlug)
	if err != nil {
		return err
	}

	stores, sudoMgr, err := loadStores(s.Config)
	if err != nil {
		return err
	}

	// TODO: handle nosudo

	sudoMgr.AroundSudo = func(sudo func()) {
		unpausec := drv.Pause()
		defer close(unpausec)

		sudo()
	}

	audit := &truststore.Audit{
		Expected: cas,
		Stores:   stores,
		SelectFn: checkAnchorCert,
	}

	info, err := audit.Perform()
	if err != nil {
		return err
	}
	drv.Send(models.AuditInfoMsg(info))

	if len(info.Missing) == 0 {
		drv.Send(models.PreflightFinishedMsg{})

		return nil
	}

	if !s.Config.NonInteractive {
		select {
		case <-confirmc:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	tmpDir, err := os.MkdirTemp("", "anchor-trust-sync")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, ca := range info.Missing {
		if err := writeCAFile(ca, tmpDir); err != nil {
			return err
		}

		drv.Activate(ctx, &models.SyncInstallCA{
			CA: ca,
		})

		for _, store := range stores {
			if info.IsPresent(ca, store) {
				continue
			}
			drv.Send(models.SyncInstallingCAMsg{Store: store})

			if ok, err := store.InstallCA(ca); err != nil {
				return err
			} else if !ok {
				panic("impossible")
			}
			drv.Send(models.SyncInstalledCAMsg{Store: store})
		}
	}

	return nil
}
