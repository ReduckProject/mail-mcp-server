package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	SMTPHost        string `json:"smtp_host"`
	SMTPPort        int    `json:"smtp_port"`
	SMTPUser        string `json:"smtp_user"`
	SMTPPass        string `json:"smtp_pass"`
	SMTPSSL         bool   `json:"smtp_ssl"`
	SMTPSkipTLS     bool   `json:"smtp_skip_tls_verify"`

	IMAPHost        string `json:"imap_host"`
	IMAPPort        int    `json:"imap_port"`
	IMAPUser        string `json:"imap_user"`
	IMAPPass        string `json:"imap_pass"`
	IMAPSSL         bool   `json:"imap_ssl"`
	IMAPSkipTLS     bool   `json:"imap_skip_tls_verify"`

	EmailFrom string `json:"email_from"`

	// Proxy settings
	ProxyURL string `json:"proxy_url"`
}

func defaults() *Config {
	return &Config{
		SMTPPort: 465,
		SMTPSSL:  true,
		IMAPPort: 993,
		IMAPSSL:  true,
	}
}

func Load() (*Config, error) {
	cfg := defaults()

	// Try config files
	paths := []string{
		"mail-mcp.json",
		filepath.Join(os.Getenv("HOME"), ".mail-mcp.json"),
		filepath.Join(os.Getenv("USERPROFILE"), ".mail-mcp.json"),
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			if err := json.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", p, err)
			}
			break
		}
	}

	// Environment variables override config file
	envStr := func(key string, target *string) {
		if v := os.Getenv(key); v != "" {
			*target = v
		}
	}
	envInt := func(key string, target *int) {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*target = n
			}
		}
	}
	envBool := func(key string, target *bool) {
		if v := os.Getenv(key); v != "" {
			*target = v == "true" || v == "1"
		}
	}

	envStr("SMTP_HOST", &cfg.SMTPHost)
	envInt("SMTP_PORT", &cfg.SMTPPort)
	envStr("SMTP_USER", &cfg.SMTPUser)
	envStr("SMTP_PASS", &cfg.SMTPPass)
	envBool("SMTP_SSL", &cfg.SMTPSSL)
	envBool("SMTP_SKIP_TLS_VERIFY", &cfg.SMTPSkipTLS)

	envStr("IMAP_HOST", &cfg.IMAPHost)
	envInt("IMAP_PORT", &cfg.IMAPPort)
	envStr("IMAP_USER", &cfg.IMAPUser)
	envStr("IMAP_PASS", &cfg.IMAPPass)
	envBool("IMAP_SSL", &cfg.IMAPSSL)
	envBool("IMAP_SKIP_TLS_VERIFY", &cfg.IMAPSkipTLS)

	envStr("EMAIL_FROM", &cfg.EmailFrom)
	envStr("PROXY_URL", &cfg.ProxyURL)

	// If IMAP credentials not set separately, use SMTP credentials
	if cfg.IMAPUser == "" {
		cfg.IMAPUser = cfg.SMTPUser
	}
	if cfg.IMAPPass == "" {
		cfg.IMAPPass = cfg.SMTPPass
	}
	if cfg.EmailFrom == "" {
		cfg.EmailFrom = cfg.SMTPUser
	}

	return cfg, nil
}
