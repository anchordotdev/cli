package trust

import (
	"testing"
	"testing/fstest"

	"github.com/anchordotdev/cli"
)

func TestIsVMOrContainer(t *testing.T) {
	tests := []struct {
		name string

		cfg *cli.Config

		result bool
	}{
		{
			name: "non-vm-or-container-linux",

			cfg: &cli.Config{
				GOOS: "linux",
				ProcFS: fstest.MapFS{
					"version": &fstest.MapFile{
						Data: unameLinuxHost,
					},
				},
			},

			result: false,
		},
		{
			name: "WSL-1",

			cfg: &cli.Config{
				GOOS: "linux",
				ProcFS: fstest.MapFS{
					"version": &fstest.MapFile{
						Data: unameWSL1,
					},
				},
			},

			result: true,
		},
		{
			name: "WSL-2",

			cfg: &cli.Config{
				GOOS: "linux",
				ProcFS: fstest.MapFS{
					"version": &fstest.MapFile{
						Data: unameWSL2,
					},
				},
			},

			result: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if want, got := test.result, isVMOrContainer(test.cfg); want != got {
				t.Errorf("want IsVMOrContainer result %t, got %t", want, got)
			}
		})
	}
}

var (
	unameWSL1 = []byte(`Linux db-d-18 4.4.0-19041-Microsoft #1237-Microsoft Sat Sep 11 14:32:00 PST 2021 x86_64 GNU/Linux`)
	unameWSL2 = []byte(`Linux db-d-18 5.4.72-microsoft-standard-WSL2 #1 SMP Wed Oct 28 23:40:43 UTC 2020 x86_64 GNU/Linux`)

	unameLinuxHost = []byte(`Linux geemus-framework 6.5.0-1020-oem #21-Ubuntu SMP PREEMPT_DYNAMIC Wed Apr  3 14:54:32 UTC 2024 x86_64 x86_64 x86_64 GNU/Linux`)
)
