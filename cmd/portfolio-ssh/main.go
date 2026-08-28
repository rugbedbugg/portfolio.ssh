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
	"syscall"
	"time"

	"charm.land/ssh"

	"github.com/rugbedbugg/portfolio.ssh/internal/content"
	"github.com/rugbedbugg/portfolio.ssh/internal/server"
)

const (
	defaultListenAddress       = ":23234"
	defaultHostKeyPath         = ".ssh/portfolio_ed25519"
	defaultIdleTimeout         = 10 * time.Minute
	defaultMaxSession          = time.Hour
	defaultMaxConnectionsPerIP = 5
	shutdownTimeout            = 10 * time.Second
)

func main() {
	cfg, err := loadConfig(os.Getenv, os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		slog.Error("invalid configuration")
		os.Exit(2)
	}

	srv, err := server.New(cfg, content.Default())
	if err != nil {
		slog.Error("create SSH server", "error", err)
		os.Exit(1)
	}

	if err := serveUntilSignal(srv); err != nil {
		slog.Error("SSH server stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}

func loadConfig(getenv func(string) string, args []string) (server.Config, error) {
	cfg := server.Config{
		ListenAddress:       valueOrDefault(getenv("PORTFOLIO_SSH_LISTEN"), defaultListenAddress),
		HostKeyPath:         valueOrDefault(getenv("PORTFOLIO_SSH_HOST_KEY"), defaultHostKeyPath),
		IdleTimeout:         defaultIdleTimeout,
		MaxSession:          defaultMaxSession,
		MaxConnectionsPerIP: defaultMaxConnectionsPerIP,
	}

	var err error
	if value := getenv("PORTFOLIO_SSH_IDLE_TIMEOUT"); value != "" {
		cfg.IdleTimeout, err = time.ParseDuration(value)
		if err != nil {
			return server.Config{}, fmt.Errorf("parse PORTFOLIO_SSH_IDLE_TIMEOUT: %w", err)
		}
	}
	if value := getenv("PORTFOLIO_SSH_MAX_SESSION"); value != "" {
		cfg.MaxSession, err = time.ParseDuration(value)
		if err != nil {
			return server.Config{}, fmt.Errorf("parse PORTFOLIO_SSH_MAX_SESSION: %w", err)
		}
	}
	if value := getenv("PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP"); value != "" {
		cfg.MaxConnectionsPerIP, err = strconv.Atoi(value)
		if err != nil {
			return server.Config{}, fmt.Errorf("parse PORTFOLIO_SSH_MAX_CONNECTIONS_PER_IP: %w", err)
		}
	}

	flags := flag.NewFlagSet("portfolio-ssh", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.ListenAddress, "listen", cfg.ListenAddress, "SSH listen address")
	flags.StringVar(&cfg.HostKeyPath, "host-key", cfg.HostKeyPath, "SSH host key path")
	flags.DurationVar(&cfg.IdleTimeout, "idle-timeout", cfg.IdleTimeout, "idle connection timeout")
	flags.DurationVar(&cfg.MaxSession, "max-session", cfg.MaxSession, "maximum session duration")
	flags.IntVar(&cfg.MaxConnectionsPerIP, "max-connections-per-ip", cfg.MaxConnectionsPerIP, "maximum simultaneous connections per IP")
	if err := flags.Parse(args); err != nil {
		return server.Config{}, err
	}
	if err := server.Validate(cfg); err != nil {
		return server.Config{}, err
	}
	return cfg, nil
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
