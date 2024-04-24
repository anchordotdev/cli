package auth

import (
	"testing"

	"github.com/anchordotdev/cli/cmdtest"
)

func TestCmdAuthSignin(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuthSignin, "auth", "signin", "--help")
	})
}

func TestSignIn(t *testing.T) {
	t.Run("cli-auth-success", func(t *testing.T) {
		t.Skip("cli auth test not yet implemented")
		return
	})

	t.Run("valid-config-token", func(t *testing.T) {
		t.Skip("cli auth test not yet implemented")
		return
	})

	t.Run("invalid-config-token", func(t *testing.T) {
		t.Skip("cli auth test not yet implemented")
		return
	})
}
