package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strconv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cert/models"
	"github.com/anchordotdev/cli/ui"
)

type Provision struct {
	Cert *tls.Certificate

	Domains     []string
	OrgAPID     string
	RealmAPID   string
	ServiceAPID string
}

func (p *Provision) RunTUI(ctx context.Context, drv *ui.Driver) error {
	return p.Perform(ctx, drv)
}

func (p *Provision) Perform(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, &models.Provision{
		Domains:     p.Domains,
		OrgAPID:     p.OrgAPID,
		RealmAPID:   p.RealmAPID,
		ServiceAPID: p.ServiceAPID,
	})

	// TODO: as a stand-alone command, it makes no sense to expect a cert as an
	// initialize value for this command, but this is only used by the 'lcl
	// diagnostic' stuff for the time being, which already provisions a cert.

	cert := p.Cert

	prefix := cert.Leaf.Subject.CommonName
	if num := len(p.Domains); num > 1 {
		prefix += "+" + strconv.Itoa(num-1)
	}

	certFile := fmt.Sprintf("./%s-cert.pem", prefix)
	chainFile := fmt.Sprintf("./%s-chain.pem", prefix)
	keyFile := fmt.Sprintf("./%s-key.pem", prefix)

	certBlock := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Certificate[0],
	}

	if !cfg.Trust.MockMode {
		if err := os.WriteFile(certFile, pem.EncodeToMemory(certBlock), 0644); err != nil {
			return err
		}
	}

	var chainData []byte
	for _, certDER := range cert.Certificate {
		chainBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER,
		}

		chainData = append(chainData, pem.EncodeToMemory(chainBlock)...)
	}

	if !cfg.Trust.MockMode {
		if err := os.WriteFile(chainFile, chainData, 0644); err != nil {
			return err
		}
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return err
	}

	keyBlock := &pem.Block{
		Type:    "PRIVATE KEY",
		Headers: make(map[string]string),
		Bytes:   keyDER,
	}

	if !cfg.Trust.MockMode {
		if err := os.WriteFile(keyFile, pem.EncodeToMemory(keyBlock), 0644); err != nil {
			return err
		}
	}

	drv.Send(models.ProvisionedFiles{certFile, chainFile, keyFile})
	return nil
}
