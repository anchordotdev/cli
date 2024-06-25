package stacktrace

import (
	"testing"

	"github.com/aymanbagabas/go-udiff"

	_ "github.com/anchordotdev/cli/testflags"
)

func TestCleaner(t *testing.T) {
	cleaner := &cleaner{
		GOPATH: gopath,
		GOROOT: goroot,
		HOME:   home,
		PWD:    pwd,
	}

	if want, got := cleanStack, cleaner.clean(originalStack); want != got {
		diff := udiff.Unified("want", "got", want, got)

		t.Fatalf("cleaned stack does not match.\n\nWant:\n\n%s\n\nGot:\n\n%s\n\nDiff:\n\n%s", want, got, diff)
	}
}

var (
	originalStack = `goroutine 39 [running]:
runtime/debug.Stack()
        /Users/benburkert/.asdf/installs/golang/1.22.3/go/src/runtime/debug/stack.go:24 +0x64
github.com/anchordotdev/cli/stacktrace.CapturePanic.func1()
        /Users/benburkert/src/github.com/anchordotdev/anchor/cli/stacktrace/stacktrace.go:40 +0x40
panic({0x104f45000?, 0x10503c250?})
        /Users/benburkert/.asdf/installs/golang/1.22.3/go/src/runtime/panic.go:770 +0x124
github.com/anchordotdev/cli_test.(*PanicCommand).run(0x104ee3160?, {0x105045020, 0x140002eb9e0}, 0x14000318150)
        /Users/benburkert/src/github.com/anchordotdev/anchor/cli/cli_test.go:117 +0xb0
github.com/anchordotdev/cli_test.TestPanic.func1.1()
        /Users/benburkert/src/github.com/anchordotdev/anchor/cli/cli_test.go:138 +0x48
github.com/anchordotdev/cli/stacktrace.CapturePanic(0x105045020?)
        /Users/benburkert/src/github.com/anchordotdev/anchor/cli/stacktrace/stacktrace.go:49 +0x58
github.com/anchordotdev/cli_test.TestPanic.func1(0x140001b16c0)
        /Users/benburkert/src/github.com/anchordotdev/anchor/cli/cli_test.go:138 +0x74
testing.tRunner(0x140001b16c0, 0x140001a6de0)
        /Users/benburkert/.asdf/installs/golang/1.22.3/go/src/testing/testing.go:1689 +0xec
created by testing.(*T).Run in goroutine 38
        /Users/benburkert/.asdf/installs/golang/1.22.3/go/src/testing/testing.go:1742 +0x318`

	cleanStack = `panic({<hex>, <hex>})
        <goroot>/src/runtime/panic.go:<line> +<hex>
github.com/anchordotdev/cli_test.(*PanicCommand).run(<hex>, {<hex>, <hex>}, <hex>)
        <pwd>/cli/cli_test.go:117 +<hex>
github.com/anchordotdev/cli_test.TestPanic.func1.1()
        <pwd>/cli/cli_test.go:138 +<hex>
github.com/anchordotdev/cli/stacktrace.CapturePanic(<hex>)
        <pwd>/cli/stacktrace/stacktrace.go:49 +<hex>
github.com/anchordotdev/cli_test.TestPanic.func1(<hex>)
        <pwd>/cli/cli_test.go:138 +<hex>
testing.tRunner(<hex>, <hex>)
        <goroot>/src/testing/testing.go:<line> +<hex>
created by testing.(*T).Run in gouroutine <int>
        <goroot>/src/testing/testing.go:<line> +<hex>`

	gopath = "/Users/benburkert/.asdf/installs/golang/1.22.3/packages"
	goroot = "/Users/benburkert/.asdf/installs/golang/1.22.3/go"
	home   = "/Users/benburkert"
	pwd    = "/Users/benburkert/src/github.com/anchordotdev/anchor"
)
