package trust

import (
	"io"
	"strings"

	"github.com/anchordotdev/cli"
)

func isVMOrContainer(cfg *cli.Config) bool {
	switch cfg.GOOS() {
	case "linux":
		// only WSL is detected for now.
		return isWSL(cfg)
	default:
		return false
	}
}

func isWSL(cfg *cli.Config) bool {
	f, err := cfg.ProcFS().Open("version")
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
