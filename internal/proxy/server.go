package proxy

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Adeelp1/vigil-gateway/config"
	"github.com/Adeelp1/vigil-gateway/internal/middleware"
	"github.com/Adeelp1/vigil-gateway/internal/store"
)

// Server owns the TCP listener and the worker pool. It deliberately does NOT
// embed net/http.Server — we want explicit control over the accept loop so
type Server struct {
	cfg      config.Config
	listener net.Listener
	handler  http.Handler // the composed middleware chain

	// wg tracks in-flight requests so Shutdown can drain before exit.
	wg sync.WaitGroup

	// sem is a buffered channel used as a counting semaphore.
	// Accepting a connection requires taking a slot; releasing the slot when done.
	// This bounds concurrency to WorkerPoolSize without spawning/killing goroutines.
	sem chan struct{}
}

func New(cfg config.Config, rdb store.Redis) (*Server, error) {
	// Build the final upstream handler.
	upstream := NewHandler(cfg.UpstreamAddr, cfg.ReadTimeout, cfg.WriteTimeout)

	// Compose the middleware chain (outermost first = first to run)
	chain := middleware.Chain(
		middleware.RequestID,
		middleware.JWTAuth(cfg),
		middleware.Logger(cfg, rdb),
		middleware.ThreatCheck(rdb),
		middleware.RateLimiter(cfg),
	)(upstream)

	l, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return nil, err
	}

	slog.Info("gateway listening", "addr", cfg.ListenAddr, "upstream", cfg.UpstreamAddr)

	return &Server{
		cfg:      cfg,
		listener: l,
		handler:  chain,
		sem:      make(chan struct{}, cfg.WorkerPoolSize),
	}, nil
}

// Serve runs the accept loop. It blocks until the listener is closed.
func (s *Server) Serve(ctx context.Context) error {
	// Watch the context: if cancelled, close the listener so Accept() unblocks.
	go func() {
		<-ctx.Done()
		s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// A closed listener return an error with "use of closed network connection".
			// Treat this as a clean shutdown signal, not a fatal error.
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			// Transient accept errors (e.g., EMFILE) - log and keep accepting.
			slog.Warn("accept error", "err", err)
			continue
		}

		// Acquire a worker slot. This blocks if all slots are taken, providing
		// natural back-pressure instead of unbounding goroutine creation.
		s.sem <- struct{}{}
		s.wg.Add(1)

		go s.serveConn(conn)
	}
}

// Shutdown drains in-flight requests within the configured timeout window.
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down - draining in-flight requests")
	s.listener.Close()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("all request drained - clean exit")
		return nil
	case <-ctx.Done():
		slog.Warn("shutdown timeout exceeded - forcing exit")
		return ctx.Err()
	}
}

// serveConn handles a single TCP connection. We delegate to net/http's
// Server machinery (http.Serve on a single-conn listener) so we get HTTP/1.1
// Keep-Alive, chunked decoding, etc. for free. The key insight: net/http is
// excellent at HTTP framing - we just control what happens before and after.
func (s *Server) serveConn(conn net.Conn) {
	defer func() {
		conn.Close()
		<-s.sem
		s.wg.Done()
	}()

	// Apply read/write deadline to the TCP connection.
	conn.SetReadDeadline(time.Now().Add(s.cfg.ReadTimeout))
	conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout))

	// Wrap the single connectino as net.Listener so http.Serve can consume it.
	httpServer := &http.Server{
		Handler:      s.handler,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	// oneConnListener lets us reuse http.Server's full HTTP/1.1 implementation
	// on a single pre-accepted connection.
	httpServer.Serve(&oneConnListener{conn: conn})
}

// oneConnListener adapts a single net.Conn into a net.Listener.
// htpp.Server calls Accept() once and then closes the listener, so this is safe.
type oneConnListener struct {
	conn net.Conn
	once sync.Once
	done chan struct{}
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	var conn net.Conn
	l.once.Do(func() {
		conn = l.conn
		l.done = make(chan struct{})
	})
	if conn != nil {
		return conn, nil
	}

	// Second call blocks forever - http.Server's accept loop will be unblocked
	// by the listener being closed.
	<-l.done
	return nil, net.ErrClosed
}

func (l *oneConnListener) Close() error {
	if l.done != nil {
		select {
		case <-l.done:
		default:
			close(l.done)
		}
	}
	return nil
}

func (l *oneConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}
