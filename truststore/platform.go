package truststore

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
