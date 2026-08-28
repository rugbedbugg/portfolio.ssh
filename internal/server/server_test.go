package server

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2/testsession"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
)

func TestValidateRejectsUnsafeConfiguration(t *testing.T) {
	valid := Config{
		ListenAddress:       "127.0.0.1:2222",
		HostKeyPath:         "host_key",
		IdleTimeout:         time.Minute,
		MaxSession:          time.Hour,
		MaxConnectionsPerIP: 2,
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "empty listen address", mutate: func(cfg *Config) { cfg.ListenAddress = "" }},
		{name: "blank listen address", mutate: func(cfg *Config) { cfg.ListenAddress = " \t" }},
		{name: "empty host key path", mutate: func(cfg *Config) { cfg.HostKeyPath = "" }},
		{name: "blank host key path", mutate: func(cfg *Config) { cfg.HostKeyPath = " \t" }},
		{name: "zero idle timeout", mutate: func(cfg *Config) { cfg.IdleTimeout = 0 }},
		{name: "negative idle timeout", mutate: func(cfg *Config) { cfg.IdleTimeout = -time.Second }},
		{name: "zero maximum session", mutate: func(cfg *Config) { cfg.MaxSession = 0 }},
		{name: "negative maximum session", mutate: func(cfg *Config) { cfg.MaxSession = -time.Second }},
		{name: "zero per-IP limit", mutate: func(cfg *Config) { cfg.MaxConnectionsPerIP = 0 }},
		{name: "negative per-IP limit", mutate: func(cfg *Config) { cfg.MaxConnectionsPerIP = -1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)
			if err := Validate(cfg); err == nil {
				t.Fatalf("Validate(%+v) returned nil; want an error", cfg)
			}
		})
	}

	if err := Validate(valid); err != nil {
		t.Fatalf("Validate(valid) returned %v", err)
	}
}

func TestNewBuildsHardenedServerWithConfiguredHostKeyAndTimeouts(t *testing.T) {
	hostKeyPath, expectedSigner := writeHostKey(t)
	cfg := Config{
		ListenAddress:       "127.0.0.1:2222",
		HostKeyPath:         hostKeyPath,
		IdleTimeout:         45 * time.Second,
		MaxSession:          20 * time.Minute,
		MaxConnectionsPerIP: 3,
	}

	srv, err := New(cfg, content.Default())
	if err != nil {
		t.Fatalf("New returned %v", err)
	}
	if srv.Addr != cfg.ListenAddress {
		t.Fatalf("server address = %q, want %q", srv.Addr, cfg.ListenAddress)
	}
	if srv.IdleTimeout != cfg.IdleTimeout || srv.MaxTimeout != cfg.MaxSession {
		t.Fatalf("server timeouts = idle %s, max %s; want idle %s, max %s", srv.IdleTimeout, srv.MaxTimeout, cfg.IdleTimeout, cfg.MaxSession)
	}
	if len(srv.HostSigners) != 1 || !ssh.KeysEqual(srv.HostSigners[0].PublicKey(), expectedSigner.PublicKey()) {
		t.Fatalf("server host signers = %d with expected key %t; want exactly the configured host key", len(srv.HostSigners), len(srv.HostSigners) == 1 && ssh.KeysEqual(srv.HostSigners[0].PublicKey(), expectedSigner.PublicKey()))
	}
	if len(srv.ChannelHandlers) != 1 || srv.ChannelHandlers["session"] == nil {
		t.Fatalf("channel handlers = %#v; want only the session channel", srv.ChannelHandlers)
	}
	if srv.RequestHandlers == nil || len(srv.RequestHandlers) != 0 {
		t.Fatalf("global request handlers = %#v; want an explicit empty set", srv.RequestHandlers)
	}
	if srv.SubsystemHandlers == nil || len(srv.SubsystemHandlers) != 0 {
		t.Fatalf("subsystem handlers = %#v; want an explicit empty set", srv.SubsystemHandlers)
	}
	if srv.LocalPortForwardingCallback != nil || srv.ReversePortForwardingCallback != nil {
		t.Fatal("port-forwarding callbacks must remain disabled")
	}
	if srv.SessionRequestCallback == nil {
		t.Fatal("server has no session request policy")
	}
	if !srv.SessionRequestCallback(nil, "shell") {
		t.Fatal("interactive SSH session request was rejected")
	}
	for _, requestType := range []string{"exec", "subsystem"} {
		if srv.SessionRequestCallback(nil, requestType) {
			t.Fatalf("%s request was accepted; want it rejected", requestType)
		}
	}
}

func TestNewRejectsUnreadableHostKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad_host_key")
	if err := os.WriteFile(path, []byte("not a private key"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		ListenAddress:       "127.0.0.1:2222",
		HostKeyPath:         path,
		IdleTimeout:         time.Minute,
		MaxSession:          time.Hour,
		MaxConnectionsPerIP: 1,
	}

	if _, err := New(cfg, content.Default()); err == nil {
		t.Fatal("New with malformed host key returned nil error")
	}
}

func TestNewRejectsMissingHostKeyWithoutCreatingIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing_host_key")
	cfg := Config{
		ListenAddress:       "127.0.0.1:2222",
		HostKeyPath:         path,
		IdleTimeout:         time.Minute,
		MaxSession:          time.Hour,
		MaxConnectionsPerIP: 1,
	}

	if _, err := New(cfg, content.Default()); err == nil {
		t.Error("New with missing host key returned nil error")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing host key stat error = %v; want os.ErrNotExist", err)
	}
}

func TestNewRejectsNonPTYAndNonPortfolioCapabilitiesOverSSH(t *testing.T) {
	hostKeyPath, _ := writeHostKey(t)
	srv, err := New(Config{
		ListenAddress:       "127.0.0.1:2222",
		HostKeyPath:         hostKeyPath,
		IdleTimeout:         time.Minute,
		MaxSession:          time.Hour,
		MaxConnectionsPerIP: 4,
	}, content.Default())
	if err != nil {
		t.Fatal(err)
	}
	addr := testsession.Listen(t, srv)

	nonPTY, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	var nonPTYStdout, nonPTYStderr bytes.Buffer
	nonPTY.Stdout = &nonPTYStdout
	nonPTY.Stderr = &nonPTYStderr
	if err := nonPTY.Shell(); err != nil {
		t.Fatalf("start non-PTY session: %v", err)
	}
	if err := nonPTY.Wait(); err == nil {
		t.Fatal("non-PTY session exited successfully; want status 1")
	}
	if !strings.Contains(nonPTYStderr.String(), "interactive terminal required") {
		t.Fatalf("non-PTY SSH response = stdout %q, stderr %q; want an interactive-terminal explanation", nonPTYStdout.String(), nonPTYStderr.String())
	}

	execSession, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := execSession.Run("whoami"); err == nil {
		t.Fatal("exec request was accepted")
	}

	subsystemSession, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := subsystemSession.RequestSubsystem("sftp"); err == nil {
		t.Fatal("SFTP subsystem request was accepted")
	}

	agentSession, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := agent.RequestAgentForwarding(agentSession); err == nil {
		t.Fatal("SSH agent forwarding request was accepted")
	}

	client, err := gossh.Dial("tcp", addr, &gossh.ClientConfig{
		User:            "testuser",
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec // isolated test server
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	destination, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = destination.Close() })
	if forwarded, err := client.Dial("tcp", destination.Addr().String()); err == nil {
		_ = forwarded.Close()
		t.Fatal("direct TCP forwarding request was accepted")
	}
}

func TestNewSessionModelProducesIndependentState(t *testing.T) {
	portfolio := content.Default()
	first := newSessionModel(portfolio, 100, 30)
	second := newSessionModel(portfolio, 100, 30)
	if first == second {
		t.Fatal("two sessions received the same Bubble Tea model")
	}

	updated, _ := first.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if updated.View().Content == second.View().Content {
		t.Fatal("changing the first session also changed or matched the second session state")
	}
}

func TestIPLimiterRejectsBeyondLimitAndReusesReleasedSlot(t *testing.T) {
	limiter := newIPLimiter(2)
	if !limiter.Acquire("192.0.2.10") || !limiter.Acquire("192.0.2.10") {
		t.Fatal("limiter rejected a connection within the per-IP limit")
	}
	if limiter.Acquire("192.0.2.10") {
		t.Fatal("limiter accepted the N+1 connection")
	}
	if !limiter.Acquire("192.0.2.11") {
		t.Fatal("one IP exhausted another IP's independent limit")
	}

	limiter.Release("192.0.2.10")
	if !limiter.Acquire("192.0.2.10") {
		t.Fatal("limiter did not reuse a released slot")
	}
}

func TestConnectionLimitMiddlewareReleasesSlotWhenSessionPanics(t *testing.T) {
	limiter := newIPLimiter(1)
	sess := &stubSession{remote: &net.TCPAddr{IP: net.ParseIP("203.0.113.7"), Port: 49152}}
	handler := connectionLimitMiddleware(limiter)(func(ssh.Session) {
		panic("session failed")
	})

	panicked := false
	func() {
		defer func() {
			panicked = recover() != nil
		}()
		handler(sess)
	}()
	if !panicked {
		t.Fatal("test handler did not panic")
	}
	if !limiter.Acquire("203.0.113.7") {
		t.Fatal("middleware leaked a limiter slot after panic")
	}
}

func TestConnectionLimitMiddlewareRejectsAndReleasesOverSSH(t *testing.T) {
	limiter := newIPLimiter(1)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	srv := &ssh.Server{Handler: connectionLimitMiddleware(limiter)(func(ssh.Session) {
		entered <- struct{}{}
		<-release
	})}
	addr := testsession.Listen(t, srv)

	first, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Shell(); err != nil {
		t.Fatal(err)
	}
	waitForEntry(t, entered)

	second, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	var rejectedOutput bytes.Buffer
	second.Stderr = &rejectedOutput
	if err := second.Shell(); err != nil {
		t.Fatal(err)
	}
	if err := second.Wait(); err == nil {
		t.Fatal("N+1 live session exited successfully; want rejection")
	}
	if !strings.Contains(rejectedOutput.String(), "connection limit reached") {
		t.Fatalf("N+1 live session response = %q; want limit explanation", rejectedOutput.String())
	}

	close(release)
	if err := first.Wait(); err != nil {
		t.Fatalf("first live session exit: %v", err)
	}

	third, err := testsession.NewClientSession(t, addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := third.Shell(); err != nil {
		t.Fatalf("released live slot was not reusable: %v", err)
	}
	waitForEntry(t, entered)
	if err := third.Wait(); err != nil {
		t.Fatalf("third live session exit: %v", err)
	}
}

func TestRequirePTYRejectsNonInteractiveSessionWithExplanation(t *testing.T) {
	sess := &stubSession{remote: &net.TCPAddr{IP: net.ParseIP("198.51.100.4"), Port: 49152}}
	called := false
	handler := requirePTYMiddleware()(func(ssh.Session) { called = true })

	handler(sess)

	if called {
		t.Fatal("non-PTY session reached the interactive handler")
	}
	if !strings.Contains(sess.stderr.String(), "interactive terminal required") {
		t.Fatalf("non-PTY response = %q; want an interactive-terminal explanation", sess.stderr.String())
	}
	if len(sess.exitCodes) != 1 || sess.exitCodes[0] != 1 {
		t.Fatalf("non-PTY exit codes = %#v; want status 1", sess.exitCodes)
	}
}

func writeHostKey(t *testing.T) (string, gossh.Signer) {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	path := filepath.Join(t.TempDir(), "host_key")
	if err := os.WriteFile(path, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.ParsePrivateKey(keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return path, signer
}

func waitForEntry(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for live SSH handler")
	}
}

type stubSession struct {
	bytes.Buffer
	stderr    bytes.Buffer
	remote    net.Addr
	pty       bool
	exitCodes []int
}

func (s *stubSession) Close() error                                   { return nil }
func (s *stubSession) CloseWrite() error                              { return nil }
func (s *stubSession) SendRequest(string, bool, []byte) (bool, error) { return false, nil }
func (s *stubSession) Stderr() io.ReadWriter                          { return &s.stderr }
func (s *stubSession) User() string                                   { return "test" }
func (s *stubSession) RemoteAddr() net.Addr                           { return s.remote }
func (s *stubSession) LocalAddr() net.Addr                            { return &net.TCPAddr{} }
func (s *stubSession) Environ() []string                              { return nil }
func (s *stubSession) Exit(code int) error {
	s.exitCodes = append(s.exitCodes, code)
	return nil
}
func (s *stubSession) Command() []string            { return nil }
func (s *stubSession) RawCommand() string           { return "" }
func (s *stubSession) Subsystem() string            { return "" }
func (s *stubSession) PublicKey() ssh.PublicKey     { return nil }
func (s *stubSession) Context() ssh.Context         { return nil }
func (s *stubSession) Permissions() ssh.Permissions { return ssh.Permissions{} }
func (s *stubSession) EmulatedPty() bool            { return s.pty }
func (s *stubSession) Pty() (ssh.Pty, <-chan ssh.Window, bool) {
	return ssh.Pty{Term: "xterm", Window: ssh.Window{Width: 80, Height: 24}}, nil, s.pty
}
func (s *stubSession) Signals(chan<- ssh.Signal) {}
func (s *stubSession) Break(chan<- bool)         {}
