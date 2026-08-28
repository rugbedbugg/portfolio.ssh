// Package server constructs the hardened SSH transport for the portfolio.
package server

import (
	"errors"
	"log/slog"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/bubbletea"
	"charm.land/wish/v2/logging"
	wishrecover "charm.land/wish/v2/recover"
	gossh "golang.org/x/crypto/ssh"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/ui"
)

// Config contains the SSH listener and resource limits.
type Config struct {
	ListenAddress       string
	HostKeyPath         string
	IdleTimeout         time.Duration
	MaxSession          time.Duration
	MaxConnectionsPerIP int
}

// Validate rejects configurations that would disable a required boundary.
func Validate(cfg Config) error {
	switch {
	case strings.TrimSpace(cfg.ListenAddress) == "":
		return errors.New("listen address is required")
	case strings.TrimSpace(cfg.HostKeyPath) == "":
		return errors.New("host key path is required")
	case cfg.IdleTimeout <= 0:
		return errors.New("idle timeout must be positive")
	case cfg.MaxSession <= 0:
		return errors.New("maximum session duration must be positive")
	case cfg.MaxConnectionsPerIP <= 0:
		return errors.New("maximum connections per IP must be positive")
	default:
		return nil
	}
}

// New builds a Wish server that exposes only the interactive portfolio.
func New(cfg Config, portfolio content.Portfolio) (*ssh.Server, error) {
	if err := Validate(cfg); err != nil {
		return nil, err
	}

	limiter := newIPLimiter(cfg.MaxConnectionsPerIP)
	return wish.NewServer(
		wish.WithAddress(cfg.ListenAddress),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithIdleTimeout(cfg.IdleTimeout),
		wish.WithMaxTimeout(cfg.MaxSession),
		hardenedBindings(),
		wish.WithMiddleware(
			bubbletea.Middleware(sessionModelHandler(portfolio)),
			requirePTYMiddleware(),
			connectionLimitMiddleware(limiter),
			logging.StructuredMiddleware(),
			wishrecover.Middleware(),
		),
	)
}

func hardenedBindings() ssh.Option {
	return func(srv *ssh.Server) error {
		srv.ChannelHandlers = map[string]ssh.ChannelHandler{
			"session": hardenedSessionHandler,
		}
		srv.RequestHandlers = map[string]ssh.RequestHandler{}
		srv.SubsystemHandlers = map[string]ssh.SubsystemHandler{}
		srv.LocalPortForwardingCallback = nil
		srv.ReversePortForwardingCallback = nil
		srv.SessionRequestCallback = func(_ ssh.Session, requestType string) bool {
			return requestType == "shell"
		}
		return nil
	}
}

func hardenedSessionHandler(srv *ssh.Server, conn *gossh.ServerConn, newChannel gossh.NewChannel, ctx ssh.Context) {
	ssh.DefaultSessionHandler(srv, conn, filteredSessionChannel{NewChannel: newChannel}, ctx)
}

type filteredSessionChannel struct {
	gossh.NewChannel
}

func (c filteredSessionChannel) Accept() (gossh.Channel, <-chan *gossh.Request, error) {
	channel, requests, err := c.NewChannel.Accept()
	if err != nil {
		return nil, nil, err
	}

	filtered := make(chan *gossh.Request)
	go filterSessionRequests(requests, filtered)
	return channel, filtered, nil
}

func filterSessionRequests(requests <-chan *gossh.Request, filtered chan<- *gossh.Request) {
	defer close(filtered)
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("panic filtering SSH session request",
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
		}
	}()

	for request := range requests {
		if request.Type == "auth-agent-req@openssh.com" {
			_ = request.Reply(false, nil)
			continue
		}
		filtered <- request
	}
}

func sessionModelHandler(portfolio content.Portfolio) bubbletea.Handler {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		pty, _, _ := sess.Pty()
		return newSessionModel(portfolio, pty.Window.Width, pty.Window.Height), nil
	}
}

func newSessionModel(portfolio content.Portfolio, width, height int) tea.Model {
	return ui.New(portfolio, width, height)
}

func requirePTYMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			if _, _, ok := sess.Pty(); !ok {
				wish.Fatalln(sess, "interactive terminal required")
				return
			}
			next(sess)
		}
	}
}

func connectionLimitMiddleware(limiter *ipLimiter) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			ip := remoteIP(sess.RemoteAddr())
			if !limiter.Acquire(ip) {
				wish.Fatalln(sess, "connection limit reached for this address")
				return
			}
			defer limiter.Release(ip)
			next(sess)
		}
	}
}

func remoteIP(addr net.Addr) string {
	if addr == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return host
	}
	return addr.String()
}

type ipLimiter struct {
	mu     sync.Mutex
	active map[string]int
	max    int
}

func newIPLimiter(max int) *ipLimiter {
	return &ipLimiter{active: make(map[string]int), max: max}
}

func (l *ipLimiter) Acquire(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[ip] >= l.max {
		return false
	}
	l.active[ip]++
	return true
}

func (l *ipLimiter) Release(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[ip] <= 1 {
		delete(l.active, ip)
		return
	}
	l.active[ip]--
}
