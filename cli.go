package cli

import (
	"context"
	"fmt"
	"go/build"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/anchordotdev/cli/models"
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

var (
	stackHexRegexp = regexp.MustCompile(`0x[0-9a-f]{2,}\??`)
	stackNilRegexp = regexp.MustCompile(`0x0`)

	stackPathReplacer *strings.Replacer
)

func init() {
	goPaths := strings.Split(os.Getenv("GOPATH"), string(os.PathListSeparator))
	if len(goPaths) == 0 {
		goPaths = append(goPaths, build.Default.GOPATH)
	}
	if goPaths[0] == "" {
		goPaths[0] = build.Default.GOPATH
	}

	joinedGoPaths := strings.Join(goPaths, ",<gopath>,") + ",<gopath>"
	replacements := strings.Split(joinedGoPaths, ",")
	replacements = append(replacements, runtime.GOROOT(), "<goroot>")

	if pwd, _ := os.Getwd(); pwd != "" {
		replacements = append(replacements, pwd, "<pwd>")
	}

	stackPathReplacer = strings.NewReplacer(replacements...)
}

func normalizeStack(stack string) string {
	// TODO: more nuanced replace for other known values like true/false, maybe empty string?
	stack = stackPathReplacer.Replace(stack)
	stack = stackHexRegexp.ReplaceAllString(stack, "<hex>")
	stack = stackNilRegexp.ReplaceAllString(stack, "<nil>")
	stack = strings.TrimRight(stack, "\n")

	lines := strings.Split(stack, "\n")
	for i, line := range lines {
		if strings.Contains(line, "<goroot>") {
			// for lines like: `<goroot>/src/runtime/debug/stack.go:24 +<hex>`
			lines[i] = fmt.Sprintf("%s:<line> +<hex>", strings.Split(line, ":")[0])
		}
		if strings.Contains(line, "in goroutine") {
			lines[i] = fmt.Sprintf("%s in gouroutine <int>", strings.Split(line, " in goroutine ")[0])
		}
	}

	return strings.Join(lines, "\n")
}

func ReportError(ctx context.Context, drv *ui.Driver, cmd *cobra.Command, args []string, msg any, stack string) {
	cfg := ConfigFromContext(ctx)

	var flags []string
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		flags = append(flags, flag.Name)
	})

	q := url.Values{}
	q.Add("title", fmt.Sprintf("Error: %s", msg))

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
	if stack != "" {
		fmt.Fprintf(&body, "**Stack:**\n```\n%s\n```\n", normalizeStack(stack))
	}
	fmt.Fprintf(&body, "**Stdout:**\n```\n%s\n```\n", strings.TrimRight(drv.ErrorView(), "\n"))
	q.Add("body", body.String())

	reportErrorConfirmCh := make(chan struct{})
	drv.Activate(ctx, &models.ReportError{
		ConfirmCh: reportErrorConfirmCh,
		Cmd:       cmd,
		Args:      args,
		Msg:       msg,
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
