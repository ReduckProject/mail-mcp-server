# Mail MCP Server

A Go-based [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for email operations — send, read, search, and manage emails via SMTP and IMAP.

[中文文档](#中文说明)

---

## Features

- **Send Email** - Send emails with CC/BCC, HTML body, and file attachments
- **List Emails** - Paginated listing of emails in any mailbox folder
- **Read Email** - Read full email content including body and attachment info
- **Search Emails** - Search by sender, recipient, subject, body text, date range, read status
- **List Folders** - List all mailbox folders/labels
- **Delete Email** - Permanently delete emails
- **GBK/GB2312 Decoding** - Properly decodes Chinese encoded email headers
- **TLS Skip Verify** - Supports skipping TLS certificate verification for corporate networks
- **Proxy Support** - SOCKS5 and HTTP/HTTPS proxy support for both SMTP and IMAP

## Installation

```bash
git clone https://github.com/your-username/mail-mcp.git
cd mail-mcp
go build -o mail-mcp .
```

## Configuration

### Option 1: Environment Variables (recommended)

```bash
export SMTP_HOST=smtp.exmail.qq.com
export SMTP_PORT=465
export SMTP_SSL=true
export SMTP_USER=you@example.com
export SMTP_PASS=your_password
export SMTP_SKIP_TLS_VERIFY=false

export IMAP_HOST=imap.exmail.qq.com
export IMAP_PORT=993
export IMAP_SSL=true
export IMAP_SKIP_TLS_VERIFY=false

export EMAIL_FROM=you@example.com   # default sender address

# Optional: proxy support
export PROXY_URL=socks5://127.0.0.1:1080
```

### Option 2: Config File

Create `mail-mcp.json` in the project directory or `~/.mail-mcp.json`:

```json
{
  "smtp_host": "smtp.exmail.qq.com",
  "smtp_port": 465,
  "smtp_ssl": true,
  "smtp_skip_tls_verify": false,
  "smtp_user": "you@example.com",
  "smtp_pass": "your_password",
  "imap_host": "imap.exmail.qq.com",
  "imap_port": 993,
  "imap_ssl": true,
  "imap_skip_tls_verify": false,
  "proxy_url": ""
}
```

Environment variables take priority over the config file. If `IMAP_USER` / `IMAP_PASS` are not set, `SMTP_USER` / `SMTP_PASS` will be used as fallback.

### Proxy Configuration

Supports connecting to SMTP/IMAP through a proxy. Set `PROXY_URL` (env) or `proxy_url` (config):

| Format | Example |
|--------|---------|
| SOCKS5 (no auth) | `socks5://127.0.0.1:1080` |
| SOCKS5 (with auth) | `socks5://user:pass@127.0.0.1:1080` |
| HTTP proxy | `http://127.0.0.1:8080` |
| HTTPS proxy (with auth) | `https://user:pass@proxy.example.com:443` |

When `PROXY_URL` is empty or not set, connections are made directly.

### Config Reference

| Field | Env Var | Default | Description |
|-------|---------|---------|-------------|
| `smtp_host` | `SMTP_HOST` | - | SMTP server address |
| `smtp_port` | `SMTP_PORT` | `465` | SMTP server port |
| `smtp_ssl` | `SMTP_SSL` | `true` | Use SSL/TLS for SMTP |
| `smtp_skip_tls_verify` | `SMTP_SKIP_TLS_VERIFY` | `false` | Skip TLS certificate verification |
| `smtp_user` | `SMTP_USER` | - | SMTP login username |
| `smtp_pass` | `SMTP_PASS` | - | SMTP login password |
| `imap_host` | `IMAP_HOST` | - | IMAP server address |
| `imap_port` | `IMAP_PORT` | `993` | IMAP server port |
| `imap_ssl` | `IMAP_SSL` | `true` | Use SSL/TLS for IMAP |
| `imap_skip_tls_verify` | `IMAP_SKIP_TLS_VERIFY` | `false` | Skip TLS certificate verification |
| `email_from` | `EMAIL_FROM` | `smtp_user` | Default sender address |
| `proxy_url` | `PROXY_URL` | - | Proxy URL (socks5/http/https) |

## MCP Tools

| Tool | Description |
|------|-------------|
| `send_email` | Send email with optional CC, BCC, HTML body, attachments |
| `list_emails` | List recent emails with pagination |
| `read_email` | Read full email content and attachment list |
| `search_emails` | Search emails by sender, recipient, subject, body, date, read status |
| `list_folders` | List all mailbox folders |
| `delete_email` | Permanently delete an email |

### Tool Parameters

#### send_email
| Parameter | Required | Description |
|-----------|----------|-------------|
| `to` | Yes | Recipient addresses (comma-separated) |
| `subject` | Yes | Email subject |
| `body` | Yes | Email body content |
| `cc` | No | CC recipients (comma-separated) |
| `bcc` | No | BCC recipients (comma-separated) |
| `content_type` | No | `text/plain` or `text/html` (default: `text/plain`) |
| `attachments` | No | Array of file paths to attach |

#### list_emails
| Parameter | Required | Description |
|-----------|----------|-------------|
| `folder` | No | Mailbox folder (default: `INBOX`) |
| `limit` | No | Emails per page (default: 20) |
| `page` | No | Page number, starting from 1 (default: 1) |

#### read_email
| Parameter | Required | Description |
|-----------|----------|-------------|
| `uid` | Yes | UID of the email to read |
| `folder` | No | Mailbox folder (default: `INBOX`) |

#### search_emails
| Parameter | Required | Description |
|-----------|----------|-------------|
| `from` | No | Filter by sender |
| `to` | No | Filter by recipient |
| `subject` | No | Filter by subject |
| `body` | No | Filter by body text |
| `since` | No | After this date (`YYYY-MM-DD`) |
| `before` | No | Before this date (`YYYY-MM-DD`) |
| `unread` | No | `true` = unread only, `false` = read only |
| `folder` | No | Mailbox folder (default: `INBOX`) |
| `limit` | No | Max results (default: 20) |

#### delete_email
| Parameter | Required | Description |
|-----------|----------|-------------|
| `uid` | Yes | UID of the email to delete |
| `folder` | No | Mailbox folder (default: `INBOX`) |

## Usage with Claude Code

Add to your Claude Code MCP configuration:

```json
{
  "mcpServers": {
    "mail": {
      "command": "/path/to/mail-mcp",
      "env": {
        "SMTP_HOST": "smtp.exmail.qq.com",
        "SMTP_PORT": "465",
        "SMTP_SSL": "true",
        "SMTP_SKIP_TLS_VERIFY": "true",
        "SMTP_USER": "you@example.com",
        "SMTP_PASS": "your_password",
        "IMAP_HOST": "imap.exmail.qq.com",
        "IMAP_PORT": "993",
        "IMAP_SSL": "true",
        "IMAP_SKIP_TLS_VERIFY": "true",
        "PROXY_URL": ""
      }
    }
  }
}
```

> **Note**: Set `SMTP_SKIP_TLS_VERIFY` / `IMAP_SKIP_TLS_VERIFY` to `"true"` if you encounter `x509: certificate signed by unknown authority` errors (common in corporate networks).

## Project Structure

```
mail-mcp/
  main.go              # MCP server entry point, tool registration
  config/
    config.go          # Config loading (env > config file)
  mail/
    models.go          # Data structures
    client.go          # SMTP/IMAP connection + proxy support
    send.go            # Send email with attachments
    fetch.go           # List, read, delete emails
    search.go          # Search with full criteria
```

## Tech Stack

- **Language**: Go
- **MCP SDK**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **IMAP**: [emersion/go-imap](https://github.com/emersion/go-imap)
- **SMTP**: Go standard library (`net/smtp` + `mime`)
- **Proxy**: [golang.org/x/net/proxy](https://pkg.go.dev/golang.org/x/net/proxy) for SOCKS5
- **Charset**: [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) for GBK/GB2312 decoding

## License

MIT

---

<a id="中文说明"></a>

# 中文说明

基于 Go 实现的 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 邮件服务器 — 通过 SMTP 和 IMAP 协议收发、搜索和管理邮件。

## 功能特性

- **发送邮件** - 支持 CC/BCC、HTML 正文和文件附件
- **邮件列表** - 分页浏览邮箱文件夹中的邮件
- **阅读邮件** - 读取完整邮件内容和附件信息
- **搜索邮件** - 按发件人、收件人、主题、正文、日期范围、已读状态搜索
- **文件夹列表** - 列出所有邮箱文件夹
- **删除邮件** - 永久删除邮件
- **GBK/GB2312 解码** - 正确解码中文编码的邮件头
- **跳过 TLS 验证** - 支持企业网络环境下跳过证书验证
- **代理支持** - 支持 SOCKS5 和 HTTP/HTTPS 代理，适用于 SMTP 和 IMAP

## 安装

```bash
git clone https://github.com/your-username/mail-mcp.git
cd mail-mcp
go build -o mail-mcp .
```

## 配置

### 方式一：环境变量（推荐）

```bash
export SMTP_HOST=smtp.exmail.qq.com
export SMTP_PORT=465
export SMTP_SSL=true
export SMTP_USER=you@example.com
export SMTP_PASS=your_password
export SMTP_SKIP_TLS_VERIFY=false

export IMAP_HOST=imap.exmail.qq.com
export IMAP_PORT=993
export IMAP_SSL=true
export IMAP_SKIP_TLS_VERIFY=false

export EMAIL_FROM=you@example.com   # 默认发件人地址

# 可选：代理设置
export PROXY_URL=socks5://127.0.0.1:1080
```

### 方式二：配置文件

在项目目录或 `~/.` 目录下创建 `mail-mcp.json`：

```json
{
  "smtp_host": "smtp.exmail.qq.com",
  "smtp_port": 465,
  "smtp_ssl": true,
  "smtp_skip_tls_verify": false,
  "smtp_user": "you@example.com",
  "smtp_pass": "your_password",
  "imap_host": "imap.exmail.qq.com",
  "imap_port": 993,
  "imap_ssl": true,
  "imap_skip_tls_verify": false,
  "proxy_url": ""
}
```

环境变量优先级高于配置文件。若未设置 `IMAP_USER` / `IMAP_PASS`，将使用 `SMTP_USER` / `SMTP_PASS` 作为默认值。

### 代理配置

支持通过代理连接 SMTP/IMAP。设置 `PROXY_URL`（环境变量）或 `proxy_url`（配置文件）：

| 格式 | 示例 |
|------|------|
| SOCKS5（无认证） | `socks5://127.0.0.1:1080` |
| SOCKS5（有认证） | `socks5://user:pass@127.0.0.1:1080` |
| HTTP 代理 | `http://127.0.0.1:8080` |
| HTTPS 代理（有认证） | `https://user:pass@proxy.example.com:443` |

`PROXY_URL` 为空或不设置时，使用直连。

### 配置项说明

| 字段 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `smtp_host` | `SMTP_HOST` | - | SMTP 服务器地址 |
| `smtp_port` | `SMTP_PORT` | `465` | SMTP 服务器端口 |
| `smtp_ssl` | `SMTP_SSL` | `true` | 使用 SSL/TLS |
| `smtp_skip_tls_verify` | `SMTP_SKIP_TLS_VERIFY` | `false` | 跳过 TLS 证书验证 |
| `smtp_user` | `SMTP_USER` | - | SMTP 登录用户名 |
| `smtp_pass` | `SMTP_PASS` | - | SMTP 登录密码 |
| `imap_host` | `IMAP_HOST` | - | IMAP 服务器地址 |
| `imap_port` | `IMAP_PORT` | `993` | IMAP 服务器端口 |
| `imap_ssl` | `IMAP_SSL` | `true` | 使用 SSL/TLS |
| `imap_skip_tls_verify` | `IMAP_SKIP_TLS_VERIFY` | `false` | 跳过 TLS 证书验证 |
| `email_from` | `EMAIL_FROM` | `smtp_user` | 默认发件人地址 |
| `proxy_url` | `PROXY_URL` | - | 代理地址（socks5/http/https） |

## MCP 工具列表

| 工具 | 说明 |
|------|------|
| `send_email` | 发送邮件，支持 CC/BCC/HTML/附件 |
| `list_emails` | 分页列出邮件 |
| `read_email` | 读取完整邮件内容和附件列表 |
| `search_emails` | 按发件人/收件人/主题/正文/日期/已读状态搜索 |
| `list_folders` | 列出所有邮箱文件夹 |
| `delete_email` | 永久删除邮件 |

## 在 Claude Code 中使用

将以下配置添加到 Claude Code 的 MCP 配置中：

```json
{
  "mcpServers": {
    "mail": {
      "command": "/path/to/mail-mcp",
      "env": {
        "SMTP_HOST": "smtp.exmail.qq.com",
        "SMTP_PORT": "465",
        "SMTP_SSL": "true",
        "SMTP_SKIP_TLS_VERIFY": "true",
        "SMTP_USER": "you@example.com",
        "SMTP_PASS": "your_password",
        "IMAP_HOST": "imap.exmail.qq.com",
        "IMAP_PORT": "993",
        "IMAP_SSL": "true",
        "IMAP_SKIP_TLS_VERIFY": "true",
        "PROXY_URL": ""
      }
    }
  }
}
```

> **提示**：如遇到 `x509: certificate signed by unknown authority` 错误（企业网络常见），设置 `SMTP_SKIP_TLS_VERIFY` / `IMAP_SKIP_TLS_VERIFY` 为 `"true"` 即可。

## 项目结构

```
mail-mcp/
  main.go              # MCP 服务入口，工具注册
  config/
    config.go          # 配置加载（环境变量 > 配置文件）
  mail/
    models.go          # 数据结构定义
    client.go          # SMTP/IMAP 连接 + 代理支持
    send.go            # 发送邮件（含附件）
    fetch.go           # 列表、阅读、删除邮件
    search.go          # 全条件搜索
```

## 技术栈

- **语言**: Go
- **MCP SDK**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **IMAP**: [emersion/go-imap](https://github.com/emersion/go-imap)
- **SMTP**: Go 标准库 (`net/smtp` + `mime`)
- **代理**: [golang.org/x/net/proxy](https://pkg.go.dev/golang.org/x/net/proxy) 用于 SOCKS5
- **字符集**: [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) 用于 GBK/GB2312 解码

## 许可证

MIT
