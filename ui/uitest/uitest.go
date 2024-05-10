package uitest

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aymanbagabas/go-udiff"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii) // no color for consistent golden file
	lipgloss.SetHasDarkBackground(false)    // dark background for consistent golden file
}

func TestTUI(ctx context.Context, t *testing.T) (*ui.Driver, *teatest.TestModel) {
	drv := ui.NewDriverTest(ctx)
	tm := teatest.NewTestModel(t, drv, teatest.WithInitialTermSize(128, 64))

	drv.Program = program{tm}

	return drv, tm
}

type program struct {
	*teatest.TestModel
}

func (p program) Quit() {
	err := p.TestModel.Quit()
	if err != nil {
		panic(err)
	}
}

func (p program) Run() (tea.Model, error) {
	panic("TODO")
}

func (p program) Wait() {
	// no-op, for TestError and since TestModel doesn't provide a Wait without needing a t.testing
}

func TestTUIError(ctx context.Context, t *testing.T, tui cli.UI, msgAndArgs ...interface{}) {
	_, errc := testTUI(ctx, t, tui)
	err := <-errc
	require.Error(t, err, msgAndArgs...)
}

func TestTUIOutput(ctx context.Context, t *testing.T, tui cli.UI) {
	drv, errc := testTUI(ctx, t, tui)

	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, []byte(drv.Golden()))
}

func testTUI(ctx context.Context, t *testing.T, tui cli.UI) (*ui.Driver, chan error) {
	drv := ui.NewDriverTest(ctx)
	tm := teatest.NewTestModel(t, drv, teatest.WithInitialTermSize(128, 64))

	drv.Program = program{tm}

	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		defer tm.Quit()

		errc <- tui.RunTUI(ctx, drv)
	}()

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))

	return drv, errc
}

var (
	waitDuration = 3 * time.Second
	waitInterval = 50 * time.Millisecond
)

func WaitForGoldenContains(t *testing.T, drv *ui.Driver, errc chan error, want string) {
	t.Helper()

	if len(errc) > 0 {
		t.Fatal(<-errc)
	}

	start := time.Now()
	for time.Since(start) <= waitDuration {
		if strings.Contains(drv.Golden(), want) {
			return
		}
		time.Sleep(waitInterval)
	}

	t.Fatalf("WaitFor: condition not met after %s;\n\nWant:\n\n%s\n\nGot:\n\n%s", waitDuration, want, drv.Golden())
}

func TestGolden(t *testing.T, got string) {
	t.Helper()

	got = strings.ReplaceAll(got, "\r\n", "\n")

	goldenPath := filepath.Join("testdata", t.Name()+".golden")

	update, err := pflag.CommandLine.GetBool("update")
	if err != nil {
		t.Fatal(err)
	}

	if update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	wantBytes, _ := os.ReadFile(goldenPath)
	// treating absent error the same as empty provides a nicer user experience
	// especially since the main error case seems to be for nonexistent files

	want := string(wantBytes)
	want = strings.ReplaceAll(want, "\r\n", "\n")

	diff := udiff.Unified("want", "got", want, got)
	if diff != "" {
		t.Fatalf("`%s` does not match, expected:\n\n%s\n\ngot:\n\n%s\n\ndiff:\n\n%s", goldenPath, want, got, diff)
	}
}
