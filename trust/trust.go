package trust

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/ext509"
	"github.com/anchordotdev/cli/ext509/oid"
	"github.com/anchordotdev/cli/truststore"
)

const (
	sudoWarning = "Anchor needs sudo access to install certificates in your local trust stores."
)

type Command struct {
	Config *cli.Config
}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTTY: c.run,
	}
}

func (c *Command) run(ctx context.Context, tty termenv.File) error {
	output := termenv.DefaultOutput()
	cp := output.ColorProfile()

	fmt.Fprintln(tty,
		output.String("# Run `anchor trust`").Bold(),
	)

	anc, err := api.NewClient(c.Config)
	if err != nil {
		return err
	}

	org, realm, err := fetchOrgAndRealm(ctx, c.Config, anc)
	if err != nil {
		return err
	}

	res, err := anc.Get("/orgs/" + url.QueryEscape(org) + "/realms/" + url.QueryEscape(realm) + "/x509/credentials")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response code: %d", res.StatusCode)
	}

	var certs *api.Credentials
	if err := json.NewDecoder(res.Body).Decode(&certs); err != nil {
		return err
	}

	rootDir, err := os.MkdirTemp("", "add-cert")
	if err != nil {
		return err
	}
	defer os.RemoveAll(rootDir)

	fmt.Fprintln(tty,
		" ",
		output.String("!").Background(cp.Color("#7000ff")),
		sudoWarning,
	)

	stores, _, err := loadStores(c.Config)
	if err != nil {
		return err
	}

	for _, cert := range certs.Items {
		blk, _ := pem.Decode([]byte(cert.TextualEncoding))

		cert, err := x509.ParseCertificate(blk.Bytes)
		if err != nil {
			log.Fatal(err)
		}

		uniqueName := cert.SerialNumber.Text(16)
		fileName := filepath.Join(uniqueName + ".pem")
		file, err := os.Create(filepath.Join(rootDir, fileName))
		if err != nil {
			log.Fatal(err)
		}
		if err := pem.Encode(file, blk); err != nil {
			log.Fatal(err)
		}
		if err := file.Close(); err != nil {
			log.Fatal(err)
		}

		ca := &truststore.CA{
			Certificate: cert,

			FilePath:   file.Name(),
			UniqueName: uniqueName,
		}

		fmt.Fprintln(tty,
			" ",
			"# Installing",
			"\""+output.String(ca.Subject.CommonName).Underline().String()+"\"",
			ca.PublicKeyAlgorithm,
			output.String("("+uniqueName+")").Faint(),
			"certificate",
		)

		if ca.SignatureAlgorithm == x509.PureEd25519 {
			fmt.Fprintf(tty, "    - skipped awaiting broader support.\n")
			continue
		}

		if c.Config.Trust.MockMode {
			fmt.Fprintf(tty, "    - installed in the mock store.\n")
			continue
		}

		for _, store := range stores {
			if err := install(tty, ca, store); err != nil {
				return err
			}
		}
	}

	return nil
}

func install(tty termenv.File, ca *truststore.CA, store truststore.Store) error {
	if ok, err := store.Check(); !ok {
		if err != nil {
			fmt.Fprintf(tty, "    - skipping the %s store: %s\n", store.Description(), err)
		} else {
			fmt.Fprintf(tty, "    - skipping the %s store\n", store.Description())
		}
		return nil
	}

	if ok, err := store.CheckCA(ca); err != nil {
		fmt.Fprintf(tty, "    - skipping the %s store: %s\n", store.Description(), err)
		return nil
	} else if ok {
		fmt.Fprintf(tty, "    - already installed in the %s store.\n", store.Description())
		return nil
	}

	if installed, err := store.InstallCA(ca); err != nil {
		return err
	} else if installed {
		fmt.Fprintf(tty, "    - installed in the %s store.\n", store.Description())
	}
	return nil
}

func fetchOrgAndRealm(ctx context.Context, cfg *cli.Config, anc *api.Session) (string, string, error) {
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

func PerformAudit(ctx context.Context, cfg *cli.Config, anc *api.Session, org string, realm string) (*truststore.AuditInfo, error) {
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

func getHandle(ctx context.Context, anc *api.Session) (string, error) {
	userInfo, err := anc.UserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.Whoami, nil
}

func loadStores(cfg *cli.Config) ([]truststore.Store, *SudoManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	rootFS := truststore.RootFS()

	sysFS := &SudoManager{
		CmdFS:  rootFS,
		NoSudo: cfg.Trust.NoSudo,
	}

	var stores []truststore.Store
	for _, storeName := range cfg.Trust.Stores {
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

			stores = append(stores, nssStore)
		case "homebrew":
			brewStore := &truststore.Brew{
				RootDir: "/",

				DataFS: rootFS,
				SysFS:  sysFS,
			}

			stores = append(stores, brewStore)
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
