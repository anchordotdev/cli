package truststore

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
)

const certPath = "etc/ca-certificates/cert.pem"

type Brew struct {
	RootDir string

	DataFS DataFS
	SysFS  CmdFS

	brewPath   string
	prefixPath string

	certPath string
}

func (s *Brew) Check() (bool, error) {
	if s.brewPath == "" {
		var err error
		if s.brewPath, err = s.SysFS.LookPath("brew"); err != nil {
			return false, err
		}
	}

	if s.prefixPath == "" {
		path, err := s.SysFS.Exec(s.SysFS.Command(s.brewPath, "--prefix"))
		if err != nil {
			return false, err
		}
		s.prefixPath = strings.TrimPrefix(strings.TrimSpace(string(path)), s.RootDir)
	}

	if s.certPath == "" {
		path := filepath.Join(s.prefixPath, certPath)
		if _, err := s.SysFS.Stat(path); err != nil {
			return false, err
		}
		s.certPath = path
	}

	return true, nil
}

func (s *Brew) CheckCA(ca *CA) (bool, error) {
	if ok, err := s.Check(); !ok {
		return ok, err
	}

	buf, err := s.DataFS.ReadFile(s.certPath)
	if err != nil {
		return false, err
	}

	var blk *pem.Block
	for len(buf) > 0 {
		if blk, buf = pem.Decode(buf); blk == nil {
			return false, nil
		}

		if bytes.Equal(ca.Raw, blk.Bytes) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Brew) Description() string { return "Homebrew OpenSSL (ca-certificates)" }

func (s *Brew) ListCAs() ([]*CA, error) {
	if ok, err := s.Check(); !ok {
		return nil, err
	}

	buf, err := s.DataFS.ReadFile(s.certPath)
	if err != nil {
		return nil, err
	}

	var cas []*CA
	for p, buf := pem.Decode(buf); p != nil; p, buf = pem.Decode(buf) {
		cert, err := x509.ParseCertificate(p.Bytes)
		if err != nil {
			return nil, err
		}

		ca := &CA{
			Certificate: cert,
			UniqueName:  cert.SerialNumber.Text(16),
		}

		cas = append(cas, ca)
	}

	return cas, nil
}

func (s *Brew) InstallCA(ca *CA) (bool, error) {
	if ok, err := s.Check(); !ok {
		return ok, err
	}

	blk := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.Raw,
	}

	if err := s.DataFS.AppendToFile(s.certPath, pem.EncodeToMemory(blk)); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Brew) UninstallCA(ca *CA) (bool, error) {
	if ok, err := s.Check(); !ok {
		return false, err
	}

	tmpf := s.certPath + ".tmp"
	if _, err := s.DataFS.Stat(tmpf); err != nil && !os.IsNotExist(err) {
		return false, err
	}

	odata, err := s.DataFS.ReadFile(s.certPath)
	if err != nil {
		return false, err
	}
	ndata := make([]byte, 0, len(odata))

	for buf := odata; len(buf) > 0; {
		blk, rem := pem.Decode(buf)
		if blk == nil {
			break
		}

		data := buf[:len(buf)-len(rem)]
		buf = rem

		cert, err := x509.ParseCertificate(blk.Bytes)
		if err != nil {
			return false, err
		}
		if bytes.Equal(cert.Raw, ca.Raw) {
			continue
		}

		ndata = append(ndata, data...)
	}

	if err := s.DataFS.AppendToFile(tmpf, ndata); err != nil {
		_ = s.DataFS.Remove(tmpf)
		return false, err
	}
	if err := s.DataFS.Rename(tmpf, s.certPath); err != nil {
		return false, err
	}
	return true, nil
}
