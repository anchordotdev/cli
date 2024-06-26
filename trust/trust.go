package trust

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/component"
	"github.com/anchordotdev/cli/ext509"
	"github.com/anchordotdev/cli/ext509/oid"
	"github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	truststoremodels "github.com/anchordotdev/cli/truststore/models"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdTrust = cli.NewCmd[Command](cli.CmdRoot, "trust", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Trust.Org, "org", "o", "", "Organization to trust.")
	cmd.Flags().BoolVar(&cfg.Trust.NoSudo, "no-sudo", false, "Disable sudo prompts.")
	cmd.Flags().StringVarP(&cfg.Trust.Realm, "realm", "r", "", "Realm to trust.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", []string{"homebrew", "nss", "system"}, "Trust stores to update.")

	cmd.MarkFlagsRequiredTogether("org", "realm")
})

type Command struct {
	Anc                *api.Session
	OrgSlug, RealmSlug string
}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Command) run(ctx context.Context, drv *ui.Driver) error {
	var err error
	clientCmd := &auth.Client{
		Anc: c.Anc,
	}
	c.Anc, err = clientCmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.TrustHeader)
	drv.Activate(ctx, models.TrustHint)

	err = c.Perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c *Command) Perform(ctx context.Context, drv *ui.Driver) error {
	var err error
	cfg := cli.ConfigFromContext(ctx)

	if isVMOrContainer(cfg) {
		drv.Activate(ctx, &models.VMHint{})
	}

	if c.OrgSlug == "" {
		c.OrgSlug = cfg.Trust.Org
	}
	if c.OrgSlug == "" {
		choicec, err := component.OrgSelector(ctx, drv, c.Anc,
			"Which organization do you want to trust?",
		)
		if err != nil {
			return err
		}
		select {
		case org := <-choicec:
			c.OrgSlug = org.Slug
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if c.RealmSlug == "" {
		c.RealmSlug = cfg.Trust.Realm
	}
	if c.RealmSlug == "" {
		choicec, err := component.RealmSelector(ctx, drv, c.Anc, c.OrgSlug,
			fmt.Sprintf("Which %s realm do you want to trust?", ui.Emphasize(c.OrgSlug)),
		)
		if err != nil {
			return err
		}
		select {
		case realm := <-choicec:
			c.RealmSlug = realm.Slug
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	drv.Activate(ctx, &truststoremodels.TrustStoreAudit{})

	cas, err := fetchExpectedCAs(ctx, c.Anc, c.OrgSlug, c.RealmSlug)
	if err != nil {
		return err
	}

	stores, sudoMgr, err := loadStores(cfg)
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
	auditInfo, err := audit.Perform()
	if err != nil {
		return err
	}
	drv.Send(truststoremodels.AuditInfoMsg(auditInfo))

	if len(auditInfo.Missing) == 0 {
		return nil
	}

	confirmCh := make(chan struct{})
	drv.Activate(ctx, &models.TrustUpdateConfirm{
		ConfirmCh: confirmCh,
	})

	select {
	case <-confirmCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	tmpDir, err := os.MkdirTemp("", "anchor-trust")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// FIXME: this write is required for the InstallCAs to work, feels like a leaky abstraction
	for _, ca := range auditInfo.Missing {
		if err := writeCAFile(ca, tmpDir); err != nil {
			return err
		}
	}

	for _, store := range stores {
		drv.Activate(ctx, &models.TrustUpdateStore{
			Config: cfg,
			Store:  store,
		})
		for _, ca := range auditInfo.Missing {
			if auditInfo.IsPresent(ca, store) {
				continue
			}
			drv.Send(models.TrustStoreInstallingCAMsg{CA: *ca})
			if ok, err := store.InstallCA(ca); err != nil {
				return err
			} else if !ok {
				panic("impossible")
			}
			drv.Send(models.TrustStoreInstalledCAMsg{CA: *ca})
		}
	}

	return nil
}

func fetchOrgAndRealm(ctx context.Context, anc *api.Session) (string, string, error) {
	cfg := cli.ConfigFromContext(ctx)

	org, realm := cfg.Trust.Org, cfg.Trust.Realm
	if (org == "") != (realm == "") {
		return "", "", errors.New("--org and --realm flags must both be present or absent")
	}
	if org == "" && realm == "" {
		userInfo, err := anc.UserInfo(ctx)
		if err != nil {
			return "", "", err
		}
		org = userInfo.PersonalOrg.Slug

		// TODO: use personal org's default realm value from API check-in call,
		// instead of hard-coding "localhost" here
		realm = "localhost"
	}

	return org, realm, nil
}

func PerformAudit(ctx context.Context, anc *api.Session, org string, realm string) (*truststore.AuditInfo, error) {
	cfg := cli.ConfigFromContext(ctx)

	cas, err := fetchExpectedCAs(ctx, anc, org, realm)
	if err != nil {
		return nil, err
	}

	stores, _, err := loadStores(cfg)
	if err != nil {
		return nil, err
	}

	audit := &truststore.Audit{
		Expected: cas,
		Stores:   stores,
		SelectFn: checkAnchorCert,
	}
	auditInfo, err := audit.Perform()
	if err != nil {
		return nil, err
	}

	return auditInfo, nil
}

func fetchExpectedCAs(ctx context.Context, anc *api.Session, org, realm string) ([]*truststore.CA, error) {
	creds, err := anc.FetchCredentials(ctx, org, realm)
	if err != nil {
		return nil, err
	}

	var cas []*truststore.CA
	for _, item := range creds {
		blk, _ := pem.Decode([]byte(item.TextualEncoding))

		cert, err := x509.ParseCertificate(blk.Bytes)
		if err != nil {
			return nil, err
		}

		uniqueName := cert.SerialNumber.Text(16)

		ca := &truststore.CA{
			Certificate: cert,
			UniqueName:  uniqueName,
		}

		// TODO: make this variable based on cli.Config
		if ca.PublicKeyAlgorithm == x509.Ed25519 {
			continue
		}

		cas = append(cas, ca)
	}
	return cas, nil
}

func loadStores(cfg *cli.Config) ([]truststore.Store, *SudoManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	rootFS := truststore.RootFS()

	noSudo := cfg.Trust.NoSudo
	sysFS := &SudoManager{
		CmdFS:  rootFS,
		NoSudo: noSudo,
	}

	trustStores := cfg.Trust.Stores

	var stores []truststore.Store
	for _, storeName := range trustStores {
		switch storeName {
		case "system":
			systemStore := &truststore.Platform{
				HomeDir: homeDir,

				DataFS: rootFS,
				SysFS:  sysFS,
			}

			stores = append(stores, systemStore)
		case "nss":
			nssStore := &truststore.NSS{
				HomeDir: homeDir,

				DataFS: rootFS,
				SysFS:  sysFS,
			}

			if available, _ := nssStore.Check(); available {
				stores = append(stores, nssStore)
			}
		case "homebrew":
			brewStore := &truststore.Brew{
				RootDir: "/",

				DataFS: rootFS,
				SysFS:  sysFS,
			}

			if available, _ := brewStore.Check(); available {
				stores = append(stores, brewStore)
			}
		case "mock":
			stores = append(stores, new(truststore.Mock))
		}
	}

	return stores, sysFS, nil
}

func checkAnchorCert(ca *truststore.CA) (bool, error) {
	for _, ext := range ca.Extensions {
		if ext.Id.Equal(oid.AnchorCertificateExtension) {
			var ac ext509.AnchorCertificate
			if err := ac.Unmarshal(ext); err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, nil
}

func writeCAFile(ca *truststore.CA, dir string) error {
	fileName := filepath.Join(ca.UniqueName + ".pem")
	file, err := os.Create(filepath.Join(dir, fileName))
	if err != nil {
		return err
	}
	defer file.Close()

	blk := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.Raw,
	}

	if err := pem.Encode(file, blk); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	ca.FilePath = file.Name()
	return nil
}

type SudoManager struct {
	truststore.CmdFS

	NoSudo bool

	AroundSudo func(sudoExec func())
}

func (s *SudoManager) SudoExec(cmd *exec.Cmd) ([]byte, error) {
	sudoFn := s.CmdFS.SudoExec
	if s.NoSudo {
		sudoFn = s.CmdFS.Exec
	}

	if s.AroundSudo == nil {
		return sudoFn(cmd)
	}

	var (
		out []byte
		err error
	)

	s.AroundSudo(func() {
		out, err = sudoFn(cmd)
	})

	return out, err
}
