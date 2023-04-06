package apitest

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"path/filepath"

	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gofrs/flock"
	"github.com/muesli/termenv"
	"golang.org/x/sync/errgroup"

	"github.com/anchordotdev/cli"
)

var (
	proxy      = flag.Bool("prism-proxy", false, "run prism in proxy mode")
	verbose    = flag.Bool("prism-verbose", false, "run prism in proxy mode")
	oapiConfig = flag.String("oapi-config", "config/openapi.yml", "openapi spec file path")
	lockfile   = flag.String("api-lockfile", "tmp/apitest.lock", "rails server lockfile path")
)

type Server struct {
	RootDir string

	URL string

	proxy   bool
	verbose bool

	stopfn func()
	waitfn func()
}

func (s *Server) Close() {
	s.stopfn()
	s.waitfn()
}

func (s *Server) Start(ctx context.Context) error {
	if *proxy {
		return s.StartProxy(ctx)
	}
	return s.StartMock(ctx)
}

func (s *Server) StartMock(ctx context.Context) error {
	ctx, stopfn := context.WithCancel(ctx)

	addr, waitfn, err := s.startMock(ctx)
	if err != nil {
		stopfn()
		return err
	}

	if err := s.waitTCP(addr); err != nil {
		stopfn()
		return err
	}

	s.URL = "http://" + addr + "/"
	s.stopfn = stopfn
	s.waitfn = func() { waitfn() }

	return nil
}

func (s *Server) StartProxy(ctx context.Context) error {
	ctx, stopfn := context.WithCancel(ctx)

	lock := flock.New(filepath.Join(s.RootDir, *lockfile))
	if err := lock.Lock(); err != nil {
		stopfn()
		return err
	}

	addrRails, waitRails, err := s.startRails(ctx)
	if err != nil {
		lock.Unlock()
		stopfn()
		return err
	}

	addr, waitPrism, err := s.startProxy(ctx, addrRails)
	if err != nil {
		lock.Unlock()
		stopfn()
		return err
	}

	group := new(errgroup.Group)

	group.Go(func() error { return s.waitTCP(addr) })
	group.Go(func() error { return s.waitTCP(addrRails) })

	if err := group.Wait(); err != nil {
		lock.Unlock()
		stopfn()
		return err
	}

	group = new(errgroup.Group)

	group.Go(waitPrism)
	group.Go(waitRails)

	s.URL = "http://" + addr + "/"
	s.stopfn = stopfn
	s.waitfn = func() {
		defer lock.Unlock()

		group.Wait()
	}

	return nil
}

func (s *Server) startMock(ctx context.Context) (string, func() error, error) {
	addr, err := unusedPort()
	if err != nil {
		return "", nil, err
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", nil, err
	}

	args := []string{
		"npx", "prism", "mock", *oapiConfig,
		"--port", port, "--host", host,
	}

	wait, err := s.startCmd(ctx, args)
	if err != nil {
		return "", nil, err
	}
	return addr, wait, nil
}

func (s *Server) startCmd(ctx context.Context, args []string) (func() error, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = s.RootDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return nil
	}

	wait := cmd.Wait
	if *verbose {
		var err error
		if wait, err = drainCmd(cmd); err != nil {
			return nil, err
		}
	}

	return wait, cmd.Start()
}

func (s *Server) startRails(ctx context.Context) (string, func() error, error) {
	addr, err := unusedPort()
	if err != nil {
		return "", nil, err
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", nil, err
	}

	args := []string{
		"rails", "server", "--port", port, "--binding", host,
	}

	wait, err := s.startCmd(ctx, args)
	if err != nil {
		return "", nil, err
	}
	return addr, wait, nil
}

func (s *Server) startProxy(ctx context.Context, upstreamAddr string) (string, func() error, error) {
	addr, err := unusedPort()
	if err != nil {
		return "", nil, err
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", nil, err
	}

	upstreamURL := "http://" + upstreamAddr + "/"

	args := []string{
		"npx", "prism", "proxy", *oapiConfig, upstreamURL,
		"--port", port, "--host", host,
	}

	wait, err := s.startCmd(ctx, args)
	if err != nil {
		return "", nil, err
	}
	return addr, wait, nil
}

func (s *Server) waitTCP(addr string) error {
	for {
		if conn, err := net.Dial("tcp4", addr); err == nil {
			conn.Close()
			return nil
		} else if !errors.Is(err, syscall.ECONNREFUSED) {
			return err
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func (s *Server) waitHTTP(errc chan<- error) {
	for {
		res, err := http.Get(s.URL)
		if err != nil {
			log.Printf("err = %q %#v", err, err)
		} else {
			log.Printf("res = %#v", res)
			return
		}
	}
}

func drainCmd(cmd *exec.Cmd) (func() error, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	errc := make(chan error)
	drain := func(dst io.Writer, src io.Reader) error {
		_, err := io.Copy(dst, src)
		return err
	}

	go func() { errc <- drain(os.Stdout, stdout) }()
	go func() { errc <- drain(os.Stderr, stderr) }()

	return func() error {
		if err := cmd.Wait(); err != nil {
			return err
		}
		if err := <-errc; err != nil {
			return err
		}
		return <-errc
	}, nil
}

func unusedPort() (string, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return "", err
	}
	ln.Close()

	return ln.Addr().String(), nil
}

func RunTUI(ctx context.Context, tui cli.TUI) (*bytes.Buffer, error) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		return nil, err
	}
	defer pts.Close()

	tty := &testTTY{
		File: ptmx,
	}
	tty.w = io.MultiWriter(ptmx, &tty.buf)

	output := termenv.NewOutput(tty)
	if err := tui.Run(ctx, output.TTY()); err != nil {
		return nil, err
	}
	return &tty.buf, nil
}

type testTTY struct {
	termenv.File

	w   io.Writer
	buf bytes.Buffer
}

func (t *testTTY) Write(p []byte) (int, error) {
	return t.w.Write(p)
}
