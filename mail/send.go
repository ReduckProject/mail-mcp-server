package mail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strings"

	"mail-mcp/config"
)

// SendEmail sends an email with optional HTML body and file attachments.
func SendEmail(cfg *config.Config, to []string, cc []string, bcc []string, subject, body, contentType string, attachments []string) error {
	smtpClient, err := ConnectSMTP(cfg)
	if err != nil {
		return err
	}
	defer smtpClient.Close()

	if err := smtpClient.Mail(cfg.EmailFrom); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}

	allRecipients := append(append(to, cc...), bcc...)
	for _, r := range allRecipients {
		if err := smtpClient.Rcpt(r); err != nil {
			return fmt.Errorf("set recipient %s: %w", r, err)
		}
	}

	wc, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("SMTP data: %w", err)
	}
	defer wc.Close()

	msg, err := buildMessage(cfg.EmailFrom, to, cc, subject, body, contentType, attachments)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	return nil
}

func encodeRFC2047(s string) string {
	if isASCII(s) {
		return s
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	return fmt.Sprintf("=?utf-8?B?%s?=", encoded)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func buildMessage(from string, to, cc []string, subject, body, contentType string, attachments []string) ([]byte, error) {
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	if len(cc) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(cc, ", ")))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeRFC2047(subject)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	hasAttachments := len(attachments) > 0

	if hasAttachments {
		boundary := fmt.Sprintf("mixed-boundary-%d", os.Getpid())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

		// Body part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		bodyContentType := contentType
		if bodyContentType == "" {
			bodyContentType = "text/plain"
		}
		buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\r\n", bodyContentType))
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

		qw := quotedprintable.NewWriter(&buf)
		qw.Write([]byte(body))
		qw.Close()
		buf.WriteString("\r\n")

		// Attachment parts
		for _, filePath := range attachments {
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("read attachment %s: %w", filePath, err)
			}

			filename := filepath.Base(filePath)
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: application/octet-stream; name=\"%s\"\r\n",
				encodeRFC2047(filename)))
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n",
				encodeRFC2047(filename)))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")

			// Write base64 in 76-char lines
			encoded := base64.StdEncoding.EncodeToString(data)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				buf.WriteString(encoded[i:end])
				buf.WriteString("\r\n")
			}
		}

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		bodyCT := contentType
		if bodyCT == "" {
			bodyCT = "text/plain"
		}
		buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\r\n", bodyCT))
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

		qw := quotedprintable.NewWriter(&buf)
		qw.Write([]byte(body))
		qw.Close()
	}

	return buf.Bytes(), nil
}

// parseMessageBody extracts text body and attachments from a raw email message.
func parseMessageBody(raw []byte) (body string, attachments []Attachment, err error) {
	msg, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return string(raw), nil, nil
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		data, _ := io.ReadAll(msg.Body)
		return string(data), nil, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			data, _ := io.ReadAll(msg.Body)
			return string(data), nil, nil
		}

		mr := multipart.NewReader(msg.Body, boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			partMediaType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			disposition, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))

			if disposition == "attachment" || (strings.HasPrefix(partMediaType, "application/") && disposition != "inline") {
				filename := dispParams["filename"]
				if filename == "" {
					filename = "unnamed"
				}
				data, _ := io.ReadAll(part)
				attachments = append(attachments, Attachment{
					Filename:    filename,
					ContentType: partMediaType,
					Size:        int64(len(data)),
				})
			} else if body == "" && (strings.HasPrefix(partMediaType, "text/plain") || strings.HasPrefix(partMediaType, "text/html")) {
				data, _ := io.ReadAll(part)
				body = string(data)
			}
		}
	} else {
		data, _ := io.ReadAll(msg.Body)
		body = string(data)
	}

	return body, attachments, nil
}
