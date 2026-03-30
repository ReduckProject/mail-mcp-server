package mail

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"time"

	imapClient "github.com/emersion/go-imap/client"
	"golang.org/x/net/proxy"
	"mail-mcp/config"
)

// dialer returns a proxy-aware dial function. Supports socks5/socks5h/http/https.
// If no proxy is set, uses direct connection.
func dialer(cfg *config.Config) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	if cfg.ProxyURL == "" {
		return func(ctx context.Context, network, addr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 15 * time.Second}
			return d.DialContext(ctx, network, addr)
		}, nil
	}

	proxyURL, err := url.Parse(cfg.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}

	switch proxyURL.Scheme {
	case "socks5", "socks5h":
		var auth *proxy.Auth
		if u := proxyURL.User; u != nil {
			auth = &proxy.Auth{User: u.Username()}
			if p, ok := u.Password(); ok {
				auth.Password = p
			}
		}
		dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, &net.Dialer{Timeout: 15 * time.Second})
		if err != nil {
			return nil, fmt.Errorf("create socks5 dialer: %w", err)
		}
		return func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}, nil

	case "http", "https":
		return func(ctx context.Context, network, targetAddr string) (net.Conn, error) {
			d := net.Dialer{Timeout: 15 * time.Second}
			conn, err := d.DialContext(ctx, "tcp", proxyURL.Host)
			if err != nil {
				return nil, fmt.Errorf("connect to proxy %s: %w", proxyURL.Host, err)
			}

			// Build HTTP CONNECT request
			req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr)
			if u := proxyURL.User; u != nil {
				p, _ := u.Password()
				creds := base64.StdEncoding.EncodeToString([]byte(u.Username() + ":" + p))
				req += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", creds)
			}
			req += "\r\n"

			if _, err := conn.Write([]byte(req)); err != nil {
				conn.Close()
				return nil, fmt.Errorf("send CONNECT: %w", err)
			}

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("read proxy response: %w", err)
			}

			resp := string(buf[:n])
			if len(resp) < 12 || resp[9:12] != "200" {
				conn.Close()
				return nil, fmt.Errorf("proxy CONNECT failed: %s", resp)
			}

			return conn, nil
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s (use socks5, http, or https)", proxyURL.Scheme)
	}
}

// ConnectIMAP creates an IMAP client connection, optionally through a proxy.
func ConnectIMAP(cfg *config.Config) (*imapClient.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.IMAPHost, cfg.IMAPPort)

	dialFn, err := dialer(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.IMAPSSL {
		conn, err := dialFn(context.Background(), "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("connect IMAP %s: %w", addr, err)
		}

		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         cfg.IMAPHost,
			InsecureSkipVerify: cfg.IMAPSkipTLS,
		})
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("IMAP TLS handshake: %w", err)
		}

		c, err := imapClient.New(tlsConn)
		if err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("IMAP client: %w", err)
		}

		if err := c.Login(cfg.IMAPUser, cfg.IMAPPass); err != nil {
			c.Logout()
			return nil, fmt.Errorf("IMAP login: %w", err)
		}
		return c, nil
	}

	conn, err := dialFn(context.Background(), "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect IMAP %s: %w", addr, err)
	}

	c, err := imapClient.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("IMAP client: %w", err)
	}

	if err := c.Login(cfg.IMAPUser, cfg.IMAPPass); err != nil {
		c.Logout()
		return nil, fmt.Errorf("IMAP login: %w", err)
	}
	return c, nil
}

// ConnectSMTP creates an SMTP connection, optionally through a proxy.
func ConnectSMTP(cfg *config.Config) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	dialFn, err := dialer(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.SMTPSSL {
		conn, err := dialFn(context.Background(), "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("SMTP connect %s: %w", addr, err)
		}

		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         cfg.SMTPHost,
			InsecureSkipVerify: cfg.SMTPSkipTLS,
		})
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("SMTP TLS handshake: %w", err)
		}

		c, err := smtp.NewClient(tlsConn, cfg.SMTPHost)
		if err != nil {
			tlsConn.Close()
			return nil, fmt.Errorf("SMTP client: %w", err)
		}

		auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
		if err := c.Auth(auth); err != nil {
			c.Close()
			return nil, fmt.Errorf("SMTP auth: %w", err)
		}
		return c, nil
	}

	// Non-SSL: connect plain, then upgrade with STARTTLS
	conn, err := dialFn(context.Background(), "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("SMTP connect %s: %w", addr, err)
	}

	c, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SMTP client: %w", err)
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{
			ServerName:         cfg.SMTPHost,
			InsecureSkipVerify: cfg.SMTPSkipTLS,
		}); err != nil {
			c.Close()
			return nil, fmt.Errorf("SMTP STARTTLS: %w", err)
		}
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	if err := c.Auth(auth); err != nil {
		c.Close()
		return nil, fmt.Errorf("SMTP auth: %w", err)
	}

	return c, nil
}
