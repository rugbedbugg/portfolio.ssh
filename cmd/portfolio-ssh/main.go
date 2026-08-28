// Command portfolio-ssh serves the interactive SSH portfolio.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"charm.land/ssh"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/server"
)

const (
	defaultListenAddress              = ":23234"
	defaultHostKeyPath                = ".ssh/portfolio_ed25519"
	defaultIdleTimeout                = 10 * time.Minute
	defaultMaxSession                 = time.Hour
	defaultMaxConnectionsPerIP        = 5
	defaultMaxConnectionAttemptsPerIP = 10
	defaultConnectionAttemptWindow    = time.Minute
	shutdownTimeout                   = 10 * time.Second
)

func main() {
	if code := run(os.Getenv, os.Args[1:], os.Stderr); code != 0 {
		os.Exit(code)
	}
}

func run(getenv func(string) string, args []string, stderr io.Writer) int {
	cfg, err := loadConfig(getenv, args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(stderr)
			return 0
		}
		fmt.Fprintf(stderr, "invalid configuration: %s\n", err)
		return 2
	}

	srv, err := server.New(cfg, content.Default())
	if err != nil {
		slog.Error("create SSH server", "error", err)
		return 1
	}

	if err := serveUntilSignal(srv); err != nil {
		slog.Error("SSH server stopped unexpectedly", "error", err)
		return 1
	}
	return 0
}

func loadConfig(getenv func(string) string, args []string) (server.Config, error) {
	cfg := server.Config{
		ListenAddress:              valueOrDefault(getenv("PORTFOLIO_SSH_LISTEN"), defaultListenAddress),
		HostKeyPath:                valueOrDefault(getenv("PORTFOLIO_SSH_HOST_KEY"), defaultHostKeyPath),
		IdleTimeout:                defaultIdleTimeout,
		MaxSession:                 defaultMaxSession,
		MaxConnectionsPerIP:        defaultMaxConnectionsPerIP,
		MaxConnectionAttemptsPerIP: defaultMaxConnectionAttemptsPerIP,
		ConnectionAttemptWindow:    defaultConnectionAttemptWindow,
	}

	var err error
	if value := getenv("PORTFOLIO_SSH_IDLE_TIMEOUT"); value != "" {
		cfg.IdleTimeout, err = time.ParseDuration(value)
		if err != nil {
			return server.Config{}, errors.New("PORTFOLIO_SSH_IDLE_TIMEOUT must be a duration")
		}
	}
	if value := getenv("PORTFOLIO_SSH_MAX_SESSION"); value != "" {
		cfg.MaxSession, err = time.ParseDuration(value)
		if err != nil {
			return server.Config{}, errors.New("PORTFOLIO_SSH_MAX_SESSION must be a duration")
		}
	}
	if value := getenv("PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP"); value != "" {
		cfg.MaxConnectionsPerIP, err = strconv.Atoi(value)
		if err != nil {
			return server.Config{}, errors.New("PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP must be an integer")
		}
	}
	if value := getenv("PORTFOLIO_SSH_MAX_CONNECTION_ATTEMPTS_PER_IP"); value != "" {
		cfg.MaxConnectionAttemptsPerIP, err = strconv.Atoi(value)
		if err != nil {
			return server.Config{}, errors.New("PORTFOLIO_SSH_MAX_CONNECTION_ATTEMPTS_PER_IP must be an integer")
		}
	}
	if value := getenv("PORTFOLIO_SSH_CONNECTION_ATTEMPT_WINDOW"); value != "" {
		cfg.ConnectionAttemptWindow, err = time.ParseDuration(value)
		if err != nil {
			return server.Config{}, errors.New("PORTFOLIO_SSH_CONNECTION_ATTEMPT_WINDOW must be a duration")
		}
	}

	flags := flag.NewFlagSet("portfolio-ssh", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.ListenAddress, "listen", cfg.ListenAddress, "SSH listen address")
	flags.StringVar(&cfg.HostKeyPath, "host-key", cfg.HostKeyPath, "SSH host key path")
	flags.DurationVar(&cfg.IdleTimeout, "idle-timeout", cfg.IdleTimeout, "idle connection timeout")
	flags.DurationVar(&cfg.MaxSession, "max-session", cfg.MaxSession, "maximum session duration")
	flags.IntVar(&cfg.MaxConnectionsPerIP, "max-connections-per-ip", cfg.MaxConnectionsPerIP, "maximum simultaneous connections per IP")
	flags.IntVar(&cfg.MaxConnectionAttemptsPerIP, "max-connection-attempts-per-ip", cfg.MaxConnectionAttemptsPerIP, "maximum connection attempts per IP in one window")
	flags.DurationVar(&cfg.ConnectionAttemptWindow, "connection-attempt-window", cfg.ConnectionAttemptWindow, "connection-attempt rate-limit window")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return server.Config{}, err
		}
		return server.Config{}, sanitizedFlagError(err)
	}
	if err := server.Validate(cfg); err != nil {
		return server.Config{}, err
	}
	return cfg, nil
}

func writeUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "Usage: portfolio-ssh [options]")
	fmt.Fprintln(stderr, "  -listen string\n    \tSSH listen address")
	fmt.Fprintln(stderr, "  -host-key string\n    \tSSH host key path")
	fmt.Fprintln(stderr, "  -idle-timeout duration\n    \tidle connection timeout")
	fmt.Fprintln(stderr, "  -max-session duration\n    \tmaximum session duration")
	fmt.Fprintln(stderr, "  -max-connections-per-ip int\n    \tmaximum simultaneous connections per IP")
	fmt.Fprintln(stderr, "  -max-connection-attempts-per-ip int\n    \tmaximum connection attempts per IP in one window")
	fmt.Fprintln(stderr, "  -connection-attempt-window duration\n    \tconnection-attempt rate-limit window")
}

func sanitizedFlagError(parseErr error) error {
	name := flagNameFromParseError(parseErr.Error())
	switch name {
	case "idle-timeout", "max-session", "connection-attempt-window":
		return fmt.Errorf("invalid value for -%s; expected duration", name)
	case "max-connections-per-ip", "max-connection-attempts-per-ip":
		return fmt.Errorf("invalid value for -%s; expected integer", name)
	case "listen", "host-key":
		return fmt.Errorf("invalid value for -%s", name)
	default:
		return errors.New("invalid command-line option")
	}
}

func flagNameFromParseError(message string) string {
	for _, marker := range []string{" for flag -", "flag needs an argument: -"} {
		index := strings.LastIndex(message, marker)
		if index == -1 {
			continue
		}
		name := message[index+len(marker):]
		name, _, _ = strings.Cut(name, ":")
		return name
	}
	return ""
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func serveUntilSignal(srv *ssh.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return serveUntilContext(ctx, srv)
}

func serveUntilContext(ctx context.Context, srv *ssh.Server) error {
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- srv.Serve(listener)
	}()
	slog.Info("SSH portfolio server listening", "address", listener.Addr().String())

	select {
	case err := <-listenErr:
		if errors.Is(err, ssh.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down SSH portfolio server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		if err := <-listenErr; !errors.Is(err, ssh.ErrServerClosed) {
			return err
		}
		slog.Info("SSH portfolio server stopped")
		return nil
	}
}
