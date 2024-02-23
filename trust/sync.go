package trust

import (
	"context"
	"errors"
	"os"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
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
	anc := s.Anc
	if anc == nil {
		var err error
		if anc, err = s.apiClient(ctx, drv); err != nil {
			return err
		}
	}

	orgSlug, realmSlug := s.OrgSlug, s.RealmSlug
	if orgSlug == "" || realmSlug == "" {
		if orgSlug != "" || realmSlug != "" {
			panic("sync: OrgSlug & RealmSlug must be initialized together")
		}

		var err error
		if orgSlug, realmSlug, err = fetchOrgAndRealm(ctx, s.Config, anc); err != nil {
			return err
		}
	}

	confirmc := make(chan struct{})
	drv.Activate(ctx, &models.SyncPreflight{
		NonInteractive: s.Config.NonInteractive,
		ConfirmCh:      confirmc,
	})

	cas, err := fetchExpectedCAs(ctx, anc, orgSlug, realmSlug)
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

	// FIXME: this write is required for the InstallCAs to work, feels like a leaky abstraction
	for _, ca := range info.Missing {
		if err := writeCAFile(ca, tmpDir); err != nil {
			return err
		}
	}

	for _, store := range stores {
		drv.Activate(ctx, &models.SyncUpdateStore{
			Store: store,
		})
		for _, ca := range info.Missing {
			if info.IsPresent(ca, store) {
				continue
			}
			drv.Send(models.SyncStoreInstallingCAMsg{CA: *ca})
			if ok, err := store.InstallCA(ca); err != nil {
				return err
			} else if !ok {
				panic("impossible")
			}
			drv.Send(models.SyncStoreInstalledCAMsg{CA: *ca})
		}
	}

	return nil
}

func (s *Sync) apiClient(ctx context.Context, drv *ui.Driver) (*api.Session, error) {
	anc, err := api.NewClient(s.Config)
	if errors.Is(err, api.ErrSignedOut) {
		if err := s.runSignIn(ctx, drv); err != nil {
			return nil, err
		}
		if anc, err = api.NewClient(s.Config); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return anc, nil
}

func (s *Sync) runSignIn(ctx context.Context, drv *ui.Driver) error {
	cmdSignIn := &auth.SignIn{
		Config:   s.Config,
		Preamble: ui.StepHint("You need to signin first, so we can track the CAs to sync."),
	}
	return cmdSignIn.RunTUI(ctx, drv)
}
