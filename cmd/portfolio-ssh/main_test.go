package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/server"
)

func TestLoadConfigUsesDefaults(t *testing.T) {
	cfg, err := loadConfig(func(string) string { return "" }, nil)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	want := server.Config{
		ListenAddress:              ":23234",
		HostKeyPath:                ".ssh/portfolio_ed25519",
		IdleTimeout:                10 * time.Minute,
		MaxSession:                 time.Hour,
		MaxConnectionsPerIP:        5,
		MaxConnectionAttemptsPerIP: 10,
		ConnectionAttemptWindow:    time.Minute,
	}
	if cfg != want {
		t.Errorf("loadConfig() = %#v, want %#v", cfg, want)
	}
}

func TestLoadConfigUsesEnvironmentValues(t *testing.T) {
	env := map[string]string{
		"PORTFOLIO_SSH_LISTEN":                         "127.0.0.1:2222",
		"PORTFOLIO_SSH_HOST_KEY":                       "testdata/host_key",
		"PORTFOLIO_SSH_IDLE_TIMEOUT":                   "3m",
		"PORTFOLIO_SSH_MAX_SESSION":                    "30m",
		"PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP":         "12",
		"PORTFOLIO_SSH_MAX_CONNECTION_ATTEMPTS_PER_IP": "24",
		"PORTFOLIO_SSH_CONNECTION_ATTEMPT_WINDOW":      "2m",
	}

	cfg, err := loadConfig(func(key string) string { return env[key] }, nil)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	want := server.Config{
		ListenAddress:              "127.0.0.1:2222",
		HostKeyPath:                "testdata/host_key",
		IdleTimeout:                3 * time.Minute,
		MaxSession:                 30 * time.Minute,
		MaxConnectionsPerIP:        12,
		MaxConnectionAttemptsPerIP: 24,
		ConnectionAttemptWindow:    2 * time.Minute,
	}
	if cfg != want {
		t.Errorf("loadConfig() = %#v, want %#v", cfg, want)
	}
}

func TestLoadConfigFlagsOverrideEnvironment(t *testing.T) {
	env := map[string]string{
		"PORTFOLIO_SSH_LISTEN":                         "127.0.0.1:2222",
		"PORTFOLIO_SSH_HOST_KEY":                       "testdata/host_key",
		"PORTFOLIO_SSH_IDLE_TIMEOUT":                   "3m",
		"PORTFOLIO_SSH_MAX_SESSION":                    "30m",
		"PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP":         "12",
		"PORTFOLIO_SSH_MAX_CONNECTION_ATTEMPTS_PER_IP": "24",
		"PORTFOLIO_SSH_CONNECTION_ATTEMPT_WINDOW":      "2m",
	}
	args := []string{
		"-listen=127.0.0.1:2022",
		"-host-key=alternate.key",
		"-idle-timeout=4m",
		"-max-session=45m",
		"-max-connections-per-ip=8",
		"-max-connection-attempts-per-ip=16",
		"-connection-attempt-window=90s",
	}

	cfg, err := loadConfig(func(key string) string { return env[key] }, args)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	want := server.Config{
		ListenAddress:              "127.0.0.1:2022",
		HostKeyPath:                "alternate.key",
		IdleTimeout:                4 * time.Minute,
		MaxSession:                 45 * time.Minute,
		MaxConnectionsPerIP:        8,
		MaxConnectionAttemptsPerIP: 16,
		ConnectionAttemptWindow:    90 * time.Second,
	}
	if cfg != want {
		t.Errorf("loadConfig() = %#v, want %#v", cfg, want)
	}
}

func TestLoadConfigRejectsInvalidEnvironmentValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{name: "idle timeout", env: map[string]string{"PORTFOLIO_SSH_IDLE_TIMEOUT": "later"}},
		{name: "maximum session", env: map[string]string{"PORTFOLIO_SSH_MAX_SESSION": "later"}},
		{name: "connections per IP", env: map[string]string{"PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP": "many"}},
		{name: "connection attempts per IP", env: map[string]string{"PORTFOLIO_SSH_MAX_CONNECTION_ATTEMPTS_PER_IP": "many"}},
		{name: "connection attempt window", env: map[string]string{"PORTFOLIO_SSH_CONNECTION_ATTEMPT_WINDOW": "later"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadConfig(func(key string) string { return tt.env[key] }, nil)
			if err == nil {
				t.Fatal("loadConfig() error = nil, want invalid value error")
			}
		})
	}
}

func TestLoadConfigRejectsInvalidFlagValues(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "idle timeout", args: []string{"-idle-timeout=later"}},
		{name: "maximum session", args: []string{"-max-session=later"}},
		{name: "connections per IP", args: []string{"-max-connections-per-ip=many"}},
		{name: "connection attempts per IP", args: []string{"-max-connection-attempts-per-ip=many"}},
		{name: "connection attempt window", args: []string{"-connection-attempt-window=later"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadConfig(func(string) string { return "" }, tt.args)
			if err == nil {
				t.Fatal("loadConfig() error = nil, want invalid value error")
			}
		})
	}
}

func TestLoadConfigReturnsFlagHelp(t *testing.T) {
	_, err := loadConfig(func(string) string { return "" }, []string{"-help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("loadConfig() error = %v, want flag.ErrHelp", err)
	}
}

func TestRunRendersUsageForHelp(t *testing.T) {
	var stderr bytes.Buffer
	code := run(func(string) string { return "" }, []string{"-help"}, &stderr)
	if code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}

	output := stderr.String()
	if !strings.Contains(output, "Usage: portfolio-ssh [options]") {
		t.Errorf("help output = %q, want usage header", output)
	}
	if !strings.Contains(output, "-listen") {
		t.Errorf("help output = %q, want -listen option", output)
	}
}

func TestRunWritesSanitizedFlagError(t *testing.T) {
	const secret = "not-a-duration-token"
	var stderr bytes.Buffer
	code := run(func(string) string { return "" }, []string{"-idle-timeout=" + secret}, &stderr)
	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid configuration: invalid value for -idle-timeout; expected duration") {
		t.Errorf("error output = %q, want sanitized idle-timeout reason", output)
	}
	if strings.Contains(output, secret) {
		t.Errorf("error output = %q, must not contain malformed value", output)
	}
}

func TestRunWritesErrorForActualLaterInvalidFlag(t *testing.T) {
	var stderr bytes.Buffer
	code := run(func(string) string { return "" }, []string{"-idle-timeout=10s", "-max-session=bad"}, &stderr)
	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid configuration: invalid value for -max-session; expected duration") {
		t.Errorf("error output = %q, want max-session reason", output)
	}
	if strings.Contains(output, "-idle-timeout; expected duration") {
		t.Errorf("error output = %q, must not report earlier valid flag", output)
	}
}

func TestRunWritesUnknownOptionErrorAfterValidFlag(t *testing.T) {
	const unknown = "-unsupported-token=secret"
	var stderr bytes.Buffer
	code := run(func(string) string { return "" }, []string{"-idle-timeout=10s", unknown}, &stderr)
	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid configuration: invalid command-line option") {
		t.Errorf("error output = %q, want unknown-option reason", output)
	}
	if strings.Contains(output, "-idle-timeout") || strings.Contains(output, "secret") {
		t.Errorf("error output = %q, must not report another option or malformed value", output)
	}
}

func TestRunWritesSanitizedEnvironmentError(t *testing.T) {
	const secret = "not-a-session-duration"
	var stderr bytes.Buffer
	code := run(func(key string) string {
		if key == "PORTFOLIO_SSH_MAX_SESSION" {
			return secret
		}
		return ""
	}, nil, &stderr)
	if code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}

	output := stderr.String()
	if !strings.Contains(output, "invalid configuration: PORTFOLIO_SSH_MAX_SESSION must be a duration") {
		t.Errorf("error output = %q, want sanitized max-session reason", output)
	}
	if strings.Contains(output, secret) {
		t.Errorf("error output = %q, must not contain malformed value", output)
	}
}

func TestServeUntilContextGracefullyStopsServer(t *testing.T) {
	privateKeyPath := writeTestHostKey(t)
	srv, err := server.New(server.Config{
		ListenAddress:              "127.0.0.1:0",
		HostKeyPath:                privateKeyPath,
		IdleTimeout:                time.Minute,
		MaxSession:                 time.Minute,
		MaxConnectionsPerIP:        1,
		MaxConnectionAttemptsPerIP: 10,
		ConnectionAttemptWindow:    time.Minute,
	}, content.Default())
	if err != nil {
		t.Fatalf("server.New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := serveUntilContext(ctx, srv); err != nil {
		t.Fatalf("serveUntilContext() error = %v", err)
	}
}

func writeTestHostKey(t *testing.T) string {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}

	path := t.TempDir() + "/host_key"
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}
