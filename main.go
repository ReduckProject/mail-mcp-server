package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"mail-mcp/config"
	"mail-mcp/mail"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	s := server.NewMCPServer(
		"mail-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// 1. send_email
	s.AddTool(mcp.NewTool("send_email",
		mcp.WithDescription("Send an email with optional CC, BCC, HTML body, and file attachments"),
		mcp.WithString("to",
			mcp.Required(),
			mcp.Description("Recipient email addresses, comma-separated"),
		),
		mcp.WithString("subject",
			mcp.Required(),
			mcp.Description("Email subject"),
		),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("Email body content"),
		),
		mcp.WithString("cc",
			mcp.Description("CC recipients, comma-separated"),
		),
		mcp.WithString("bcc",
			mcp.Description("BCC recipients, comma-separated"),
		),
		mcp.WithString("content_type",
			mcp.Description("Body content type: text/plain or text/html (default: text/plain)"),
			mcp.Enum("text/plain", "text/html"),
		),
		mcp.WithArray("attachments",
			mcp.Description("List of file paths to attach"),
			mcp.WithStringItems(),
		),
	), makeSendHandler(cfg))

	// 2. list_emails
	s.AddTool(mcp.NewTool("list_emails",
		mcp.WithDescription("List recent emails in a mailbox folder with pagination"),
		mcp.WithString("folder",
			mcp.Description("Mailbox folder name (default: INBOX)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of emails per page (default: 20)"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number, starting from 1 (default: 1)"),
		),
	), makeListHandler(cfg))

	// 3. read_email
	s.AddTool(mcp.NewTool("read_email",
		mcp.WithDescription("Read full content of a specific email by UID, including body and attachment list"),
		mcp.WithNumber("uid",
			mcp.Required(),
			mcp.Description("UID of the email to read"),
		),
		mcp.WithString("folder",
			mcp.Description("Mailbox folder name (default: INBOX)"),
		),
	), makeReadHandler(cfg))

	// 4. search_emails
	s.AddTool(mcp.NewTool("search_emails",
		mcp.WithDescription("Search emails by various criteria: sender, recipient, subject, body text, date range, read status"),
		mcp.WithString("from",
			mcp.Description("Filter by sender address or name"),
		),
		mcp.WithString("to",
			mcp.Description("Filter by recipient address"),
		),
		mcp.WithString("subject",
			mcp.Description("Filter by subject text"),
		),
		mcp.WithString("body",
			mcp.Description("Filter by body text content"),
		),
		mcp.WithString("since",
			mcp.Description("Emails after this date (format: YYYY-MM-DD)"),
		),
		mcp.WithString("before",
			mcp.Description("Emails before this date (format: YYYY-MM-DD)"),
		),
		mcp.WithBoolean("unread",
			mcp.Description("Filter by read status: true=unread only, false=read only"),
		),
		mcp.WithString("folder",
			mcp.Description("Mailbox folder name (default: INBOX)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 20)"),
		),
	), makeSearchHandler(cfg))

	// 5. list_folders
	s.AddTool(mcp.NewTool("list_folders",
		mcp.WithDescription("List all mailbox folders/labels"),
	), makeListFoldersHandler(cfg))

	// 6. delete_email
	s.AddTool(mcp.NewTool("delete_email",
		mcp.WithDescription("Permanently delete an email by UID"),
		mcp.WithNumber("uid",
			mcp.Required(),
			mcp.Description("UID of the email to delete"),
		),
		mcp.WithString("folder",
			mcp.Description("Mailbox folder name (default: INBOX)"),
		),
	), makeDeleteHandler(cfg))

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func splitAddresses(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, addr := range strings.Split(s, ",") {
		a := strings.TrimSpace(addr)
		if a != "" {
			result = append(result, a)
		}
	}
	return result
}

func toJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

func getFolder(request mcp.CallToolRequest) string {
	return mcp.ParseString(request, "folder", "INBOX")
}

func makeSendHandler(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		to := mcp.ParseString(request, "to", "")
		subject := mcp.ParseString(request, "subject", "")
		body := mcp.ParseString(request, "body", "")

		if to == "" {
			return mcp.NewToolResultError("'to' is required"), nil
		}
		if subject == "" {
			return mcp.NewToolResultError("'subject' is required"), nil
		}
		if body == "" {
			return mcp.NewToolResultError("'body' is required"), nil
		}

		cc := mcp.ParseString(request, "cc", "")
		bcc := mcp.ParseString(request, "bcc", "")
		contentType := mcp.ParseString(request, "content_type", "text/plain")

		var attachments []string
		args := request.GetArguments()
		if arr, ok := args["attachments"]; ok {
			if arrSlice, ok := arr.([]interface{}); ok {
				for _, item := range arrSlice {
					if s, ok := item.(string); ok {
						attachments = append(attachments, s)
					}
				}
			}
		}

		err := mail.SendEmail(cfg, splitAddresses(to), splitAddresses(cc), splitAddresses(bcc), subject, body, contentType, attachments)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("send email failed: %v", err)), nil
		}

		return mcp.NewToolResultText("Email sent successfully"), nil
	}
}

func makeListHandler(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		folder := getFolder(request)
		limit := mcp.ParseInt(request, "limit", 20)
		page := mcp.ParseInt(request, "page", 1)

		emails, total, err := mail.ListEmails(cfg, folder, limit, page)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list emails failed: %v", err)), nil
		}

		result := fmt.Sprintf("Folder: %s | Total: %d | Page: %d\n\n%s", folder, total, page, toJSON(emails))
		return mcp.NewToolResultText(result), nil
	}
}

func makeReadHandler(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		uid := mcp.ParseUInt32(request, "uid", 0)
		if uid == 0 {
			return mcp.NewToolResultError("'uid' is required"), nil
		}
		folder := getFolder(request)

		email, err := mail.ReadEmail(cfg, folder, uid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("read email failed: %v", err)), nil
		}

		return mcp.NewToolResultText(toJSON(email)), nil
	}
}

func makeSearchHandler(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := mail.SearchParams{
			Folder:  getFolder(request),
			From:    mcp.ParseString(request, "from", ""),
			To:      mcp.ParseString(request, "to", ""),
			Subject: mcp.ParseString(request, "subject", ""),
			Body:    mcp.ParseString(request, "body", ""),
			Since:   mcp.ParseString(request, "since", ""),
			Before:  mcp.ParseString(request, "before", ""),
			Limit:   mcp.ParseInt(request, "limit", 20),
		}

		args := request.GetArguments()
		if v, ok := args["unread"]; ok {
			if b, ok := v.(bool); ok {
				params.Unread = &b
			}
		}

		emails, err := mail.SearchEmails(cfg, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search emails failed: %v", err)), nil
		}

		return mcp.NewToolResultText(toJSON(emails)), nil
	}
}

func makeListFoldersHandler(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		folders, err := mail.ListFolders(cfg)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list folders failed: %v", err)), nil
		}

		return mcp.NewToolResultText(toJSON(folders)), nil
	}
}

func makeDeleteHandler(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		uid := mcp.ParseUInt32(request, "uid", 0)
		if uid == 0 {
			return mcp.NewToolResultError("'uid' is required"), nil
		}
		folder := getFolder(request)

		err := mail.DeleteEmail(cfg, folder, uid)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete email failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Email UID %d deleted from %s", uid, folder)), nil
	}
}
