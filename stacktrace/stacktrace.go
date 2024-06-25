package stacktrace

import (
	"fmt"
	"go/build"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
)

var defaultCleaner *cleaner

func init() {
	defaultCleaner = &cleaner{
		GOPATH: build.Default.GOPATH,
		GOROOT: build.Default.GOROOT,
	}

	if pwd, _ := os.Getwd(); pwd != "" {
		defaultCleaner.PWD = pwd
	}
	if home, _ := os.UserHomeDir(); home != "" {
		defaultCleaner.HOME = home
	}
}

func CapturePanic(fn func() error) (err error) {
	defer func() {
		if msg := recover(); msg != nil {
			stack := debug.Stack()

			err = Error{
				error: fmt.Errorf("%s", msg),
				Stack: defaultCleaner.clean(string(stack)),
			}
		}
	}()

	err = fn()
	return
}

type Error struct {
	error

	Stack string
}

type cleaner struct {
	GOPATH, GOROOT string

	// optionals
	HOME, PWD string
}

var (
	stackHexRegexp = regexp.MustCompile(`0x[0-9a-f]{2,}\??`)
	stackNilRegexp = regexp.MustCompile(`0x0`)
)

func (c *cleaner) clean(stack string) string {
	replacements := []string{
		normalizedStackPath(c.GOPATH), "<gopath>",
		normalizedStackPath(c.GOROOT), "<goroot>",
	}

	if pwd := c.PWD; len(pwd) > 0 {
		replacements = append(replacements, normalizedStackPath(pwd), "<pwd>")
	}
	if home := c.HOME; len(home) > 0 {
		replacements = append(replacements, normalizedStackPath(home), "<home>")
	}

	stackPathReplacer := strings.NewReplacer(replacements...)

	// TODO: more nuanced replace for other known values like true/false, maybe empty string?

	stack = stackPathReplacer.Replace(stack)
	stack = stackHexRegexp.ReplaceAllString(stack, "<hex>")
	stack = stackNilRegexp.ReplaceAllString(stack, "<nil>")
	stack = strings.TrimRight(stack, "\n")

	lines := strings.Split(string(stack), "\n")
	// omit lines: 0 go routine, 2-3 stack call, 4-5 cleanup call
	lines = lines[5:]

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
func normalizedStackPath(path string) string {
	return strings.ReplaceAll(path, string(os.PathSeparator), "/")
}
