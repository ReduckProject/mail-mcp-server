package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/emersion/go-imap"
	imapClient "github.com/emersion/go-imap/client"
	"mail-mcp/config"
)

// ConnectIMAP creates a new IMAP client connection.
func ConnectIMAP(cfg *config.Config) (*imapClient.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.IMAPHost, cfg.IMAPPort)

	var c *imapClient.Client
	var err error

	if cfg.IMAPSSL {
		c, err = imapClient.DialTLS(addr, nil)
	} else {
		c, err = imapClient.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("connect IMAP %s: %w", addr, err)
	}

	if err := c.Login(cfg.IMAPUser, cfg.IMAPPass); err != nil {
		c.Logout()
		return nil, fmt.Errorf("IMAP login: %w", err)
	}

	return c, nil
}

// ConnectSMTP creates an SMTP connection and authenticates.
func ConnectSMTP(cfg *config.Config) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var c *smtp.Client
	var err error

	if cfg.SMTPSSL {
		tlsConfig := &tls.Config{ServerName: cfg.SMTPHost}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("SMTP TLS dial %s: %w", addr, err)
		}
		c, err = smtp.NewClient(conn, cfg.SMTPHost)
		if err != nil {
			return nil, fmt.Errorf("SMTP client: %w", err)
		}
	} else {
		c, err = smtp.Dial(addr)
		if err != nil {
			return nil, fmt.Errorf("SMTP dial %s: %w", addr, err)
		}
	}

	// Try STARTTLS if not already on TLS
	if !cfg.SMTPSSL {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: cfg.SMTPHost}); err != nil {
				return nil, fmt.Errorf("SMTP STARTTLS: %w", err)
			}
		}
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	if err := c.Auth(auth); err != nil {
		return nil, fmt.Errorf("SMTP auth: %w", err)
	}

	return c, nil
}

// hasAttachment checks if a message section contains attachments.
func hasAttachment(msg *imap.Message) bool {
	if msg == nil || msg.Envelope == nil {
		return false
	}
	// Check if the message structure has parts (multipart = possible attachments)
	return msg.BodyStructure != nil
}
