package trust

import (
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/anchordotdev/cli"
)

func isVMOrContainer(cfg *cli.Config) bool {
	os := cfg.GOOS
	if os == "" {
		os = runtime.GOOS
	}

	switch os {
	case "linux":
		// only WSL is detected for now.
		return isWSL(cfg)
	default:
		return false
	}
}

func isWSL(cfg *cli.Config) bool {
	procFS := cfg.ProcFS
	if procFS == nil {
		procFS = os.DirFS("/proc")
	}

	f, err := procFS.Open("version")
	if err != nil {
		return false
	}

	buf, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	kernel := string(buf)

	// https://superuser.com/questions/1725627/which-linux-kernel-do-i-have-in-wsl

	return strings.Contains(kernel, "Microsoft") || // WSL 1
		strings.Contains(kernel, "microsoft") // WSL 2
}
