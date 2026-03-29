package mail

import "time"

type EmailSummary struct {
	UID           uint32    `json:"uid"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Subject       string    `json:"subject"`
	Date          time.Time `json:"date"`
	Unread        bool      `json:"unread"`
	HasAttachment bool      `json:"has_attachment"`
}

type EmailDetail struct {
	UID         uint32        `json:"uid"`
	From        string        `json:"from"`
	To          string        `json:"to"`
	Cc          string        `json:"cc"`
	Subject     string        `json:"subject"`
	Date        time.Time     `json:"date"`
	Unread      bool          `json:"unread"`
	ContentType string        `json:"content_type"`
	Body        string        `json:"body"`
	Attachments []Attachment  `json:"attachments"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type SearchParams struct {
	Folder string
	From   string
	To     string
	Subject string
	Body   string
	Since  string
	Before string
	Unread *bool
	Limit  int
}
