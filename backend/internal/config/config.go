// Package config parses command-line flags and environment fallbacks.
package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Config holds the runtime configuration of the v2home server.
type Config struct {
	Listen          string
	LogLevel        string
	WSInterval      time.Duration
	ShutdownTimeout time.Duration
	ShowVersion     bool
}

// Load parses args (usually os.Args[1:]).
func Load(args []string) (Config, error) {
	flags := flag.NewFlagSet("v2home", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var cfg Config
	flags.StringVar(&cfg.Listen, "listen", env("V2HOME_LISTEN", "0.0.0.0:8080"), "HTTP listen address")
	flags.StringVar(&cfg.LogLevel, "log-level", env("V2HOME_LOG_LEVEL", "info"), "log level: debug, info, warn, error")
	flags.DurationVar(&cfg.WSInterval, "ws-interval", envDuration("V2HOME_WS_INTERVAL", 2*time.Second), "status push interval for WebSocket clients")
	flags.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", 10*time.Second, "graceful shutdown timeout")
	flags.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")

	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}
	if cfg.Listen == "" {
		return Config{}, fmt.Errorf("listen address must not be empty")
	}
	if cfg.WSInterval < 250*time.Millisecond {
		return Config{}, fmt.Errorf("ws-interval must be at least 250ms")
	}
	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("shutdown-timeout must be positive")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
