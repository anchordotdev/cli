package truststore

import (
	"bytes"
	"encoding/pem"
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
