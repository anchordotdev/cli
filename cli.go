package cli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cli/browser"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/anchordotdev/cli/models"
	"github.com/anchordotdev/cli/stacktrace"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
)

type Clipboard interface {
	ReadAll() (string, error)
	WriteAll(text string) error
}

var Executable string

var Version = struct {
	Version, Commit, Date string

	Os, Arch string
}{
	Version: "dev",
	Commit:  "none",
	Date:    "unknown",
	Os:      runtime.GOOS,
	Arch:    runtime.GOARCH,
}

func IsDevVersion() bool {
	return Version.Version == "dev"
}

func UserAgent() string {
	return "Anchor CLI " + VersionString()
}

func ReleaseTagName() string {
	return fmt.Sprintf("v%s", Version.Version)
}

func VersionString() string {
	return fmt.Sprintf("%s (%s/%s) Commit: %s BuildDate: %s", Version.Version, Version.Os, Version.Arch, Version.Commit, Version.Date)
}

var Defaults = defaultConfig()

func defaultConfig() *Config {
	var cfg Config
	defaults.SetDefaults(&cfg)
	return &cfg
}

type UI struct {
	RunTUI func(context.Context, *ui.Driver) error
}

type contextKey int

const (
	configKey contextKey = iota
	calledAsKey
)

func CalledAsFromContext(ctx context.Context) string {
	if calledAs, ok := ctx.Value(calledAsKey).(string); ok {
		return calledAs
	}
	return ""
}

func ConfigFromContext(ctx context.Context) *Config {
	if v := ctx.Value(configKey); v != nil {
		return v.(*Config)
	}
	return nil
}

func ConfigFromCmd(cmd *cobra.Command) *Config {
	return ConfigFromContext(cmd.Context())
}

func ContextWithCalledAs(ctx context.Context, calledAs string) context.Context {
	return context.WithValue(ctx, calledAsKey, calledAs)
}

func ContextWithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

func ReportError(ctx context.Context, err error, drv *ui.Driver, cmd *cobra.Command, args []string) {
	cfg := ConfigFromContext(ctx)

	var flags []string
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		flags = append(flags, flag.Name)
	})

	q := url.Values{}
	q.Add("title", fmt.Sprintf("Error: %s", err.Error()))

	var body strings.Builder

	fmt.Fprintf(&body, "**Are there any additional details you would like to share?**\n")
	fmt.Fprintf(&body, "\n")
	fmt.Fprintf(&body, "---\n")
	fmt.Fprintf(&body, "\n")
	fmt.Fprintf(&body, "**Command:** `%s`\n", cmd.CommandPath())
	var executable string
	if Executable != "" {
		executable = Executable
	} else {
		executable, _ = os.Executable()
	}
	if executable != "" {
		fmt.Fprintf(&body, "**Executable:** `%s`\n", executable)
	}
	fmt.Fprintf(&body, "**Version:** `%s`\n", VersionString())
	fmt.Fprintf(&body, "**Arguments:** `[%s]`\n", strings.Join(args, ", "))
	fmt.Fprintf(&body, "**Flags:** `[%s]`\n", strings.Join(flags, ", "))
	fmt.Fprintf(&body, "**Timestamp:** `%s`\n", cfg.Timestamp().Format(time.RFC3339Nano))

	var stackerr stacktrace.Error
	if errors.As(err, &stackerr) {
		fmt.Fprintf(&body, "**Stack:**\n```\n%s\n```\n", stackerr.Stack)
	}

	fmt.Fprintf(&body, "**Stdout:**\n```\n%s\n```\n", strings.TrimRight(drv.ErrorView(), "\n"))
	q.Add("body", body.String())

	reportErrorConfirmCh := make(chan struct{})
	drv.Activate(ctx, &models.ReportError{
		ConfirmCh: reportErrorConfirmCh,
		Cmd:       cmd,
		Args:      args,
		Msg:       err.Error(),
	})

	if !cfg.NonInteractive {
		<-reportErrorConfirmCh
	}

	newIssueURL := fmt.Sprintf("https://github.com/anchordotdev/cli/issues/new?%s", q.Encode())
	// FIXME: ? remove config val, switch to mocking this to always err in tests
	if cfg.Test.Browserless {
		drv.Activate(ctx, &models.Browserless{Url: newIssueURL})
	} else {
		if err := browser.OpenURL(newIssueURL); err != nil {
			drv.Activate(ctx, &models.Browserless{Url: newIssueURL})
		}
	}
}

type UserError struct {
	Err error
}

func (u UserError) Error() string { return u.Err.Error() }

func isReportable(err error) bool {
	switch err.(type) {
	case UserError:
		return false
	case ui.Error:
		return false
	}

	var terr truststore.Error
	if errors.As(err, &terr) {
		return false
	}

	eerr := new(exec.ExitError)
	if errors.As(err, &eerr) {
		return false
	}

	switch err {
	case context.Canceled:
		return false
	}

	return true
}
