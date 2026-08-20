package config

import (
	"flag"
	"os"
)

// Config holds all runtime configuration.
type Config struct {
	Addr   string
	DB     string
	Secret string
	SMTP   string
}

// Load reads config from defaults, env, and flags.
func Load() *Config {
	c := &Config{
		Addr:   ":7777",
		DB:     "pollkit.json",
		Secret: "",
		SMTP:   "",
	}

	// Env
	if v := os.Getenv("POLLKIT_ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("POLLKIT_DB"); v != "" {
		c.DB = v
	}
	if v := os.Getenv("POLLKIT_SECRET"); v != "" {
		c.Secret = v
	}
	if v := os.Getenv("POLLKIT_SMTP"); v != "" {
		c.SMTP = v
	}

	// Flags
	flag.StringVar(&c.Addr, "addr", c.Addr, "listen address")
	flag.StringVar(&c.DB, "db", c.DB, "data file path")
	flag.StringVar(&c.Secret, "secret", c.Secret, "token signing secret (auto-generated if empty)")
	flag.StringVar(&c.SMTP, "smtp", c.SMTP, "SMTP URL for sending OTP emails (empty = log to stderr)")
	flag.Parse()

	return c
}
