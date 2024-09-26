package truststore

import (
	"crypto/x509"
	"strings"
)

func (s *Platform) Check() (bool, error) {
	ok, err := s.check()
	if err != nil {
		err = Error{
			Op: OpCheck,

			Warning: PlatformError{
				Err: err,

				NSSBrowsers: nssBrowsers,
			},
		}
	}
	return ok, err
}

func (s *Platform) CheckCA(ca *CA) (installed bool, err error) {
	if _, cerr := s.check(); cerr != nil {
		defer func() {
			err = Error{
				Op: OpCheck,

				Warning: PlatformError{
					Err: cerr,

					NSSBrowsers: nssBrowsers,
					RootCA:      ca.FilePath,
				},
			}
		}()
	}

	return s.checkCA(ca)
}

func (s *Platform) InstallCA(ca *CA) (installed bool, err error) {
	if _, cerr := s.check(); cerr != nil {
		defer func() {
			err = Error{
				Op: OpInstall,

				Warning: PlatformError{
					Err: cerr,

					NSSBrowsers: nssBrowsers,
					RootCA:      ca.FilePath,
				},
			}
		}()
	}

	return s.installCA(ca)
}

func (s *Platform) ListCAs() (cas []*CA, err error) {
	if _, cerr := s.check(); cerr != nil {
		defer func() {
			err = Error{
				Op: OpList,

				Warning: PlatformError{
					Err: cerr,

					NSSBrowsers: nssBrowsers,
				},
			}
		}()
	}

	return s.listCAs()
}

func (s *Platform) UninstallCA(ca *CA) (uninstalled bool, err error) {
	if _, cerr := s.check(); cerr != nil {
		defer func() {
			err = Error{
				Op: OpUninstall,

				Warning: PlatformError{
					Err: cerr,

					NSSBrowsers: nssBrowsers,
					RootCA:      ca.FilePath,
				},
			}
		}()
	}

	return s.uninstallCA(ca)
}

func parseCertificate(der []byte) (*x509.Certificate, error) {
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		if strings.HasPrefix(err.Error(), "x509: certificate contains duplicate extension") {
			return nil, nil
		}
		if strings.HasPrefix(err.Error(), "x509: inner and outer signature algorithm identifiers don't match") {
			return nil, nil
		}
		return nil, err
	}
	return cert, nil
}
