package mail

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap"
	"mail-mcp/config"
)

// SearchEmails searches emails matching the given criteria.
func SearchEmails(cfg *config.Config, params SearchParams) ([]EmailSummary, error) {
	c, err := ConnectIMAP(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	_, err = c.Select(params.Folder, false)
	if err != nil {
		return nil, fmt.Errorf("select folder %s: %w", params.Folder, err)
	}

	// Build search criteria
	criteria := imap.NewSearchCriteria()

	if params.From != "" {
		criteria.Header.Add("FROM", params.From)
	}
	if params.To != "" {
		criteria.Header.Add("TO", params.To)
	}
	if params.Subject != "" {
		criteria.Header.Add("SUBJECT", params.Subject)
	}
	if params.Body != "" {
		criteria.Body = []string{params.Body}
	}
	if params.Since != "" {
		if t, err := parseDate(params.Since); err == nil {
			criteria.Since = t
		}
	}
	if params.Before != "" {
		if t, err := parseDate(params.Before); err == nil {
			criteria.Before = t
		}
	}
	if params.Unread != nil && *params.Unread {
		criteria.WithoutFlags = []string{imap.SeenFlag}
	}
	if params.Unread != nil && !*params.Unread {
		criteria.WithFlags = []string{imap.SeenFlag}
	}

	uids, err := c.UidSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	if len(uids) == 0 {
		return []EmailSummary{}, nil
	}

	// Apply limit - take the most recent UIDs
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if len(uids) > limit {
		uids = uids[len(uids)-limit:]
	}

	seqSet := new(imap.SeqSet)
	for _, uid := range uids {
		seqSet.AddNum(uid)
	}

	messages := make(chan *imap.Message, len(uids))
	err = c.UidFetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	if err != nil {
		return nil, fmt.Errorf("fetch results: %w", err)
	}

	var results []EmailSummary
	for msg := range messages {
		results = append(results, msgToSummary(msg))
	}

	return results, nil
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		time.RFC3339,
		"Jan 2, 2006",
		"02-Jan-2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", s)
}
