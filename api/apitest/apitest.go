package apitest

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"io"
	"net"
	"os"
	"os/exec"
	"time"

	_ "github.com/anchordotdev/cli/testflags"
	"github.com/gofrs/flock"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
)

var (
	proxy, _      = pflag.CommandLine.GetBool("prism-proxy")
	verbose, _    = pflag.CommandLine.GetBool("prism-verbose")
	oapiConfig, _ = pflag.CommandLine.GetString("oapi-config")
	lockfile, _   = pflag.CommandLine.GetString("api-lockfile")
)

type Server struct {
	Host    string
	RootDir string

	URL       string
	RailsPort string

	stopfn func()
	waitfn func()
}

func (s *Server) Close() {
	s.stopfn()
	s.waitfn()
}

func (s *Server) IsMock() bool {
	return !proxy
}

func (s *Server) IsProxy() bool {
	return proxy
}

func (s *Server) RecreateUser(username string) error {
	if s.IsMock() {
		return nil
	}

	cmd := exec.Command("script/clitest-recreate-user", username)
	cmd.Dir = s.RootDir

	_, err := cmd.Output()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) GeneratePAT(email string) (string, error) {
	if !s.IsProxy() {
		return "test-token", nil
	}

	cmd := exec.Command("script/clitest-gen-pat", email)
	cmd.Dir = s.RootDir

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (s *Server) Start(ctx context.Context) error {
	if proxy {
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

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		stopfn()
		return err
	}

	if s.Host != "" {
		host = s.Host
	}

	s.URL = "http://" + host + ":" + port + "/v0"
	s.stopfn = stopfn
	s.waitfn = func() { _ = waitfn() }

	return nil
}

func (s *Server) StartProxy(ctx context.Context) error {
	ctx, stopfn := context.WithCancel(ctx)

	lock := flock.New(filepath.Join(s.RootDir, lockfile))
	if err := lock.Lock(); err != nil {
		stopfn()
		return err
	}

	addrRails, waitRails, err := s.startRails(ctx)
	if err != nil {
		_ = lock.Unlock()
		stopfn()
		return err
	}

	host, port, err := net.SplitHostPort(addrRails)
	if err != nil {
		stopfn()
		return err
	}
	s.RailsPort = port

	if s.Host != "" {
		host = s.Host
	}

	addr, waitPrism, err := s.startProxy(ctx, host+":"+port)
	if err != nil {
		_ = lock.Unlock()
		stopfn()
		return err
	}

	group := new(errgroup.Group)

	group.Go(func() error { return s.waitTCP(addr) })
	group.Go(func() error { return s.waitTCP(addrRails) })

	if err := group.Wait(); err != nil {
		_ = lock.Unlock()
		stopfn()
		return err
	}

	group = new(errgroup.Group)

	group.Go(waitPrism)
	group.Go(waitRails)

	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		stopfn()
		return err
	}

	if s.Host != "" {
		host = s.Host
	}

	s.URL = "http://" + host + ":" + port + "/v0"
	s.stopfn = stopfn
	s.waitfn = func() {
		_ = group.Wait()
		_ = lock.Unlock()
	}

	return nil
}

func (s *Server) startMock(ctx context.Context) (string, func() error, error) {
	addr, err := UnusedPort()
	if err != nil {
		return "", nil, err
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", nil, err
	}

	args := []string{
		"npx", "prism", "mock", oapiConfig,
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
	setpgid(cmd)
	cmd.Cancel = func() error {
		err := terminateProcess(cmd)
		if err != nil {
			return err
		}
		return nil
	}

	wait := cmd.Wait
	if verbose {
		var err error
		if wait, err = drainCmd(cmd); err != nil {
			return nil, err
		}
	}

	return wait, cmd.Start()
}

func (s *Server) startRails(ctx context.Context) (string, func() error, error) {
	addr, err := UnusedPort()
	if err != nil {
		return "", nil, err
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", nil, err
	}

	args := []string{
		"script/clitest-server", "::", port,
	}

	wait, err := s.startCmd(ctx, args)
	if err != nil {
		return "", nil, err
	}
	return addr, wait, nil
}

func (s *Server) startProxy(ctx context.Context, upstreamAddr string) (string, func() error, error) {
	addr, err := UnusedPort()
	if err != nil {
		return "", nil, err
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", nil, err
	}

	upstreamURL := "http://" + upstreamAddr + "/"

	args := []string{
		"npx", "prism", "proxy", oapiConfig, upstreamURL,
		"--port", port, "--host", host,
	}

	wait, err := s.startCmd(ctx, args)
	if err != nil {
		return "", nil, err
	}
	return addr, wait, nil
}

func (s *Server) waitTCP(addr string) error {
	var dialer net.Dialer

	duration := 16 * time.Second
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		duration,
		fmt.Errorf("waitTcp %s timeout waiting for: %s", duration, addr),
	)
	defer cancel()

	for {
		if conn, err := dialer.DialContext(ctx, "tcp4", addr); err == nil {
			conn.Close()
			return nil
		} else if !isConnRefused(err) {
			return err
		}

		time.Sleep(10 * time.Millisecond)
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

func UnusedPort() (string, error) {
	ln, err := net.Listen("tcp4", ":0")
	if err != nil {
		return "", err
	}
	ln.Close()

	return ln.Addr().String(), nil
}
