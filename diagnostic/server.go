package diagnostic

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/inetaf/tcpproxy"
)

var discardLogger = log.New(io.Discard, "", 0)

type Server struct {
	Addr string

	LclHostURL string

	GetCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)

	proxy  *tcpproxy.Proxy
	server *http.Server

	ln net.Listener

	rchanmu sync.Mutex
	rchan   chan string

	tlso sync.Once
	tlsc chan struct{}
}

func (s *Server) Start(ctx context.Context) error {
	var err error
	if s.ln, err = new(net.ListenConfig).Listen(ctx, "tcp", s.Addr); err != nil {
		return err
	}
	s.Addr = s.ln.Addr().String()

	s.proxy = &tcpproxy.Proxy{
		ListenFunc: func(string, string) (net.Listener, error) {
			return s.ln, nil
		},
	}

	lnTLS := &tcpproxy.TargetListener{
		Address: s.Addr,
	}

	s.proxy.AddSNIRouteFunc(s.Addr, func(context.Context, string) (tcpproxy.Target, bool) {
		return lnTLS, true
	})

	lnHTTP := &tcpproxy.TargetListener{
		Address: s.Addr,
	}

	s.proxy.AddRoute(s.Addr, lnHTTP)

	s.server = &http.Server{
		Addr: s.Addr,

		Handler: http.HandlerFunc(s.serveHTTP),

		BaseContext: func(net.Listener) context.Context { return ctx },
		ErrorLog:    discardLogger,
	}

	go s.server.Serve(lnHTTP)
	go s.server.Serve(tls.NewListener(lnTLS, &tls.Config{
		NextProtos:     []string{"h2", "http/1.1"},
		GetCertificate: s.getCertificate,
	}))

	s.tlsc = make(chan struct{})

	return s.proxy.Start()
}

func (s *Server) EnableTLS() {
	s.tlso.Do(func() { close(s.tlsc) })
}

func (s *Server) Close() error {
	if err := s.ln.Close(); err != nil {
		return err
	}

	if err := s.proxy.Close(); err != nil {
		return err
	}

	if err := s.server.Shutdown(context.Background()); err != nil {
		return err
	}

	return s.proxy.Wait()
}

func (s *Server) getCertificate(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	<-s.tlsc

	return s.GetCertificate(chi)
}

func (s *Server) RequestChan() <-chan string {
	s.rchanmu.Lock()
	defer s.rchanmu.Unlock()

	s.rchan = make(chan string)

	return s.rchan
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil {
		s.notifyRequest("https")

		w.Write([]byte(`<html>
  <body>
    <iframe src="` + s.lclHostURL("/secure") + `" frameborder="0" allowfullscreen style="position:absolute;top:0;left:0;width:100%;height:100%;">
      <p>Congrats!</p>
      <p>You connected with lcl.host encrypted HTTPS, powered by Anchor.dev!</p>
      <p>You can now close this tab and return to the CLI.</p>
    </iframe>
  </body>
</html>`))
	} else {
		s.notifyRequest("http")

		w.Write([]byte(`<html>
  <body>
    <iframe src="` + s.lclHostURL("/notsecure") + `" frameborder="0" allowfullscreen style="position:absolute;top:0;left:0;width:100%;height:100%;">
      <p>Welcome!</p>
      <p>You connected with non-encrypted local HTTP.</p>
      <p>You can now close this tab and return to the CLI.</p>
    </iframe>
  </body>
</html>`))
	}
}

func (s *Server) notifyRequest(scheme string) {
	s.rchanmu.Lock()
	defer s.rchanmu.Unlock()

	if s.rchan != nil {
		select {
		case s.rchan <- scheme:
		default:
		}
	}
}

func (s *Server) lclHostURL(path string) string {
	return s.LclHostURL + path
}
