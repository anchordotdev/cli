package cli

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/anchordotdev/cli/models"
	"github.com/anchordotdev/cli/stacktrace"
	"github.com/anchordotdev/cli/ui"
	"github.com/cli/browser"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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

type UI struct {
	RunTUI func(context.Context, *ui.Driver) error
}

type ContextKey string

func ConfigFromContext(ctx context.Context) *Config {
	return ctx.Value(ContextKey("Config")).(*Config)
}

func ConfigFromCmd(cmd *cobra.Command) *Config {
	return ConfigFromContext(cmd.Context())
}

func ContextWithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ContextKey("Config"), cfg)
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

	switch err {
	case context.Canceled:
		return false
	}

	return true
}
