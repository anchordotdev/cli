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
	"path/filepath"

	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/truststore"
)

const (
	sudoWarning = "! Anchor needs sudo permission to add the specified certificates to your local trust stores."
)

type Command struct {
	Config *cli.Config
}

func (c Command) TUI() cli.TUI {
	return cli.TUI{
		Run: c.run,
	}
}

func (c *Command) run(ctx context.Context, tty termenv.File) error {
	anc, err := api.Client(c.Config)
	if err == api.ErrSignedOut {
		fmt.Fprintf(tty, "Authentication required!\n")
		return nil
	}
	if err != nil {
		return err
	}

	res, err := anc.Get("")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected response")
	}

	org, realm := c.Config.Trust.Org, c.Config.Trust.Realm
	if (org == "") != (realm == "") {
		return errors.New("--org and --realm flags must both be present or absent")
	}
	if org == "" && realm == "" {
		// TODO: use personal org value from API check-in call
		var userInfo *api.Root
		if err = json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
			return err
		}
		org = *userInfo.PersonalOrg.Slug
		realm = "localhost"
	}

	res, err = anc.Get("/orgs/" + url.QueryEscape(org) + "/realms/" + url.QueryEscape(realm) + "/x509/credentials")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected response")
	}

	var certs struct {
		Items *[]api.Credential `json:"items,omitempty"`
	}
	if err := json.NewDecoder(res.Body).Decode(&certs); err != nil {
		return err
	}

	rootDir, err := os.MkdirTemp("", "add-cert")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(rootDir)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(tty, sudoWarning)

	rootFS := truststore.RootFS()
	systemStore := &truststore.Platform{
		HomeDir: homeDir,

		DataFS: rootFS,
		SysFS:  rootFS,
	}

	nssStore := &truststore.NSS{
		HomeDir: homeDir,

		DataFS: rootFS,
		SysFS:  rootFS,
	}

	for _, cert := range *certs.Items {
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

		if ca.SignatureAlgorithm == x509.PureEd25519 {
			fmt.Fprintf(tty, "Installing \"%s\" %s (%s) certificate:\n", ca.Subject.CommonName, ca.PublicKeyAlgorithm, uniqueName)
			fmt.Fprintf(tty, "  - skipped awaiting broader support.\n")
			continue
		}

		fmt.Fprintf(tty, "Installing \"%s\" %s (%s) certificate:\n", ca.Subject.CommonName, ca.PublicKeyAlgorithm, uniqueName)

		if c.Config.Trust.MockMode {
			fmt.Fprintf(tty, "  - installed in the mock store.\n")
			continue
		}

		if installed, err := systemStore.InstallCA(ca); installed {
			fmt.Fprintf(tty, "  - installed in the system store.\n")
		} else if err != nil {
			return err
		}
		if installed, err := nssStore.InstallCA(ca); installed {
			fmt.Fprintf(tty, "  - installed in the Network Security Services (NSS) store.\n")
		} else if err != nil {
			return err
		}
	}

	return nil
}
