package mail

import (
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"mail-mcp/config"
)

// ListEmails lists recent emails in a folder with pagination.
func ListEmails(cfg *config.Config, folder string, limit, page int) ([]EmailSummary, int, error) {
	c, err := ConnectIMAP(cfg)
	if err != nil {
		return nil, 0, err
	}
	defer c.Logout()

	mbox, err := c.Select(folder, false)
	if err != nil {
		return nil, 0, fmt.Errorf("select folder %s: %w", folder, err)
	}

	total := int(mbox.Messages)
	if total == 0 {
		return []EmailSummary{}, 0, nil
	}

	// Calculate sequence range for pagination
	start := uint32(total - (page-1)*limit)
	end := uint32(total - (page-1)*limit - limit + 1)
	if start < 1 {
		start = 1
	}
	if end < 1 {
		end = 1
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(end, start)

	messages := make(chan *imap.Message, limit)
	err = c.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch messages: %w", err)
	}

	var results []EmailSummary
	for msg := range messages {
		results = append(results, msgToSummary(msg))
	}

	// Results come in reverse order, reverse them
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, total, nil
}

// ReadEmail fetches a single email by UID with full body and attachments.
func ReadEmail(cfg *config.Config, folder string, uid uint32) (*EmailDetail, error) {
	c, err := ConnectIMAP(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	_, err = c.Select(folder, false)
	if err != nil {
		return nil, fmt.Errorf("select folder %s: %w", folder, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// First fetch envelope and flags
	messages := make(chan *imap.Message, 1)
	err = c.UidFetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	if err != nil {
		return nil, fmt.Errorf("fetch envelope: %w", err)
	}

	msg := <-messages
	if msg == nil {
		return nil, fmt.Errorf("message UID %d not found", uid)
	}

	detail := &EmailDetail{
		UID:     msg.Uid,
		From:    addrToString(msg.Envelope.From),
		To:      addrToString(msg.Envelope.To),
		Cc:      addrToString(msg.Envelope.Cc),
		Subject: decodeMIMEHeader(msg.Envelope.Subject),
		Date:    msg.Envelope.Date,
		Unread:  !hasFlag(msg, imap.SeenFlag),
	}

	// Fetch full body
	var bodySection imap.BodySectionName
	bodySection.Peek = true
	bodyItems := []imap.FetchItem{bodySection.FetchItem()}

	bodyMessages := make(chan *imap.Message, 1)
	err = c.UidFetch(seqSet, bodyItems, bodyMessages)
	if err != nil {
		return nil, fmt.Errorf("fetch body: %w", err)
	}

	bodyMsg := <-bodyMessages
	if bodyMsg != nil {
		section := &bodySection
		r := bodyMsg.GetBody(section)
		if r != nil {
			rawBody, _ := io.ReadAll(r)
			body, attachments, _ := parseMessageBody(rawBody)
			detail.Body = body
			detail.Attachments = attachments
			if len(attachments) > 0 {
				detail.ContentType = "multipart"
			} else if strings.Contains(body, "<") && strings.Contains(body, ">") {
				detail.ContentType = "text/html"
			} else {
				detail.ContentType = "text/plain"
			}
		}
	}

	return detail, nil
}

// ListFolders lists all mailbox folders.
func ListFolders(cfg *config.Config) ([]string, error) {
	c, err := ConnectIMAP(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	mailboxes := make(chan *imap.MailboxInfo, 100)
	err = c.List("", "*", mailboxes)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}

	var folders []string
	for mbox := range mailboxes {
		folders = append(folders, mbox.Name)
	}

	return folders, nil
}

// DeleteEmail marks an email as deleted and expunges.
func DeleteEmail(cfg *config.Config, folder string, uid uint32) error {
	c, err := ConnectIMAP(cfg)
	if err != nil {
		return err
	}
	defer c.Logout()

	_, err = c.Select(folder, false)
	if err != nil {
		return fmt.Errorf("select folder %s: %w", folder, err)
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	err = c.UidStore(seqSet, imap.AddFlags, []interface{}{imap.DeletedFlag}, nil)
	if err != nil {
		return fmt.Errorf("mark deleted: %w", err)
	}

	if err := c.Expunge(nil); err != nil {
		return fmt.Errorf("expunge: %w", err)
	}

	return nil
}

// decodeMIMEHeader decodes RFC 2047 encoded header values (e.g. =?GBK?B?...?=)
func decodeMIMEHeader(s string) string {
	if s == "" {
		return s
	}
	dec := &mime.WordDecoder{
		CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
			charset = strings.ToUpper(charset)
			switch charset {
			case "GBK", "GB2312", "GB18030":
				return simplifiedchinese.GBK.NewDecoder().Reader(input), nil
			default:
				return nil, fmt.Errorf("unsupported charset: %s", charset)
			}
		},
	}
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

func msgToSummary(msg *imap.Message) EmailSummary {
	subject := ""
	from := ""
	to := ""
	var date time.Time
	if msg.Envelope != nil {
		subject = decodeMIMEHeader(msg.Envelope.Subject)
		from = addrToString(msg.Envelope.From)
		to = addrToString(msg.Envelope.To)
		date = msg.Envelope.Date
	}

	return EmailSummary{
		UID:     msg.Uid,
		From:    from,
		To:      to,
		Subject: subject,
		Date:    date,
		Unread:  !hasFlag(msg, imap.SeenFlag),
	}
}

func addrToString(addrs []*imap.Address) string {
	var parts []string
	for _, a := range addrs {
		name := decodeMIMEHeader(a.PersonalName)
		if name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s@%s>", name, a.MailboxName, a.HostName))
		} else {
			parts = append(parts, fmt.Sprintf("%s@%s", a.MailboxName, a.HostName))
		}
	}
	return strings.Join(parts, ", ")
}

func hasFlag(msg *imap.Message, flag string) bool {
	for _, f := range msg.Flags {
		if f == flag {
			return true
		}
	}
	return false
}
