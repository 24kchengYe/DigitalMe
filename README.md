<div align="center">

# DigitalMe

### Your AI Coding Agent, Always On, Anywhere.

Control **Claude Code** from your phone via chat. Real-time dashboard. Heartbeat monitoring. Zero downtime.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Claude Code](https://img.shields.io/badge/Claude_Code-Compatible-7c3aed?style=flat-square&logo=anthropic&logoColor=white)](https://docs.anthropic.com/en/docs/claude-code)
[![License](https://img.shields.io/badge/License-BSL_1.1-blue?style=flat-square)](LICENSE)

[Features](#features) · [Quick Start](#quick-start) · [Dashboard](#dashboard) · [Configuration](#configuration) · [API](#rest-api)

</div>

---

## Why DigitalMe?

You've seen [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — the most powerful AI coding agent. You've seen [OpenClaw](https://github.com/anthropics/claw) — the web-based interface. But what if you want to **command Claude Code from anywhere** — your phone, your tablet, on the subway — through the chat app you already use?

**DigitalMe** bridges the gap. It keeps a persistent Claude Code session alive and connects it to **8 messaging platforms** including Feishu, Telegram, Slack, and Discord. No more SSH-ing into your machine. No more VPN. Just open your chat app and start coding.

```
  ╔══════════════════════════════════════════════════════════════╗
  ║                                                              ║
  ║   📱 Phone/Tablet                                            ║
  ║     │                                                        ║
  ║     ▼                                                        ║
  ║   💬 Feishu / Telegram / Slack / Discord / ...               ║
  ║     │                                                        ║
  ║     ▼                          ┌──────────────────────┐      ║
  ║   ⚡ DigitalMe ──────────────► │  🖥️ Web Dashboard    │      ║
  ║     │                          │  Heartbeat ♥♥♥♥♥♥♥  │      ║
  ║     │                          │  Sessions: 3        │      ║
  ║     │                          │  Uptime: 7d 14h     │      ║
  ║     ▼                          └──────────────────────┘      ║
  ║   🤖 Claude Code / Codex / Gemini CLI / Cursor               ║
  ║     │                                                        ║
  ║     ▼                                                        ║
  ║   📂 Your Codebase                                           ║
  ║                                                              ║
  ╚══════════════════════════════════════════════════════════════╝
```

## Features

### Core

- **6 AI Agents** — Claude Code, Codex, Cursor, Gemini CLI, Qoder, OpenCode
- **8 Chat Platforms** — Feishu, DingTalk, Telegram, Slack, Discord, LINE, WeChat Work, QQ
- **Persistent Sessions** — Claude Code process stays alive, no cold-start per message
- **Multi-session** — multiple conversations per user, switch freely with `/list` and `/switch`
- **Slash Commands** — `/new`, `/model`, `/mode`, `/stop`, `/cd`, `/help`, and more
- **Remote Directory Switching** — `/cd` lets you search and switch Claude Code's working directory from your phone, no need to remember paths
- **Permission Modes** — default, acceptEdits, plan-only, YOLO
- **Voice & Images** — speech-to-text, screenshot analysis via multimodal support
- **Scheduled Tasks** — cron jobs described in natural language
- **Provider Management** — multiple API keys, switch models at runtime

### Monitoring & Reliability (DigitalMe Enhanced)

- **Real-time Dashboard** — precision dark-themed UI with SVG heartbeat line, accessible from any browser
- **Heartbeat Monitor** — periodic health checks with visual history chart
- **Idle Session Reminder** — auto-reminds users when sessions go idle, repeating until response
- **Smart Status Detection** — heartbeat degrades to warning when idle reminders get no response
- **Remote Screenshot** — `/screenshot` command captures your screen and sends it to chat
- **File Sendback** — `/sendback` sends files to chat; Claude Code auto-sends generated files (PDF, images, etc.)
- **Task Completion Notify** — automatic notification when Claude Code finishes a task with tool usage summary
- **Local Voice Recognition** — speech-to-text via local whisper.cpp model, no API key needed
- **REST API** — query system status, engine health, session activity programmatically

## Dashboard

The built-in web dashboard provides real-time visibility into your DigitalMe instance:

- **System Status** — healthy / degraded / unhealthy at a glance
- **KPI Cards** — uptime, active projects, session count, version
- **Engine Table** — agent type, connected platforms, session count per engine
- **Session Activity** — per-session last message time and idle duration
- **Heartbeat Strip** — visual bar chart of health checks (green = active, amber = idle warning, red = down)
- **Auto-refresh** — updates every 6 seconds

## Quick Start

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated (`claude` command available in terminal)
- A messaging platform bot configured (see [Feishu Setup Guide](#feishu-setup-guide) below)

### Build

```bash
git clone https://github.com/24kchengYe/DigitalMe.git
cd DigitalMe
go build -o digitalme ./cmd/cc-connect/
```

### Configure

Create `~/.cc-connect/config.toml`:

```toml
language = "zh"          # "en", "zh", "ja", "es"

[[projects]]
name = "my-project"

[projects.agent]
type = "claudecode"      # or "codex", "cursor", "gemini", "qoder", "opencode"

[projects.agent.options]
work_dir = "/path/to/your/project"
mode = "bypassPermissions"  # "default", "acceptEdits", "plan", "bypassPermissions"

# Pick your platform (at least one)
[[projects.platforms]]
type = "feishu"

[projects.platforms.options]
app_id = "cli_xxxxxxxxxx"
app_secret = "your-app-secret"

# ── Monitoring ──

[webui]
enabled = true
addr = "0.0.0.0:9315"
heartbeat_interval = 30   # seconds

[idle]
enabled = true
idle_minutes = 20          # remind after N minutes idle

# ── Voice Recognition (optional) ──

[speech]
enabled = true
provider = "local"         # "local", "openai", "groq"
language = "zh"

[speech.local]
exe_path = "/path/to/whisper-cli"
model_path = "/path/to/ggml-base.bin"
```

### Run

```bash
./digitalme
```

Open `http://localhost:9315` for the dashboard.

---

## Feishu Setup Guide

Step-by-step guide to connect DigitalMe with Feishu (飞书). Takes about 10 minutes.

### Step 1: Create a Feishu App

1. Go to [Feishu Open Platform](https://open.feishu.cn/app) and log in
2. Click **"Create Custom App"** (创建企业自建应用)
3. Fill in:
   - **App Name**: `DigitalMe` (or any name you like)
   - **Description**: AI coding assistant bridge
   - **App Icon**: upload any icon
4. Click **Create**

### Step 2: Get App Credentials

1. In the app dashboard, go to **"Credentials & Basic Info"** (凭证与基础信息)
2. Copy the **App ID** (格式: `cli_xxxxxxxxxx`)
3. Copy the **App Secret**
4. Paste both into your `config.toml`:

```toml
[projects.platforms.options]
app_id = "cli_xxxxxxxxxx"        # ← your App ID
app_secret = "your-app-secret"   # ← your App Secret
```

### Step 3: Add Bot Capability

1. In the left sidebar, go to **"Add Capabilities"** (添加应用能力)
2. Click **"Bot"** (机器人) → **Add**
3. This enables your app to receive and send messages

### Step 4: Configure Event Subscription (WebSocket)

1. Go to **"Event Subscriptions"** (事件订阅) in the left sidebar
2. **Choose connection method**: Select **"WebSocket"** (长连接) — this is critical!
   - WebSocket mode means **no public IP needed**, no webhook URL required
   - Your local machine connects outbound to Feishu servers
3. Add the following event:
   - `im.message.receive_v1` — Receive messages

### Step 5: Add Permissions

Go to **"Permissions & Scopes"** (权限管理) and add these permissions:

| Permission | Scope ID | Purpose |
|---|---|---|
| Send messages as bot | `im:message:send_as_bot` | Send replies to users |
| Read private messages | `im:message.p2p_msg:readonly` | Receive user messages |
| Upload images | `im:resource` | Screenshot and image features |

> **Tip**: Search for the scope ID in the search box to find each permission quickly.

**Optional permissions** (for advanced features):

| Permission | Scope ID | Purpose |
|---|---|---|
| Upload files | `im:file` | `/sendback` file sending (if `im:resource` alone doesn't work) |
| Get user info | `contact:user.base:readonly` | Display user names in logs |

### Step 6: Publish the App

1. Go to **"Version Management"** (版本管理与发布)
2. Click **"Create Version"** (创建版本)
3. Fill in version number (e.g., `1.0.0`) and update description
4. Set **availability**: choose which users/departments can use the bot
5. Click **"Submit for Review"** (提交审核)
6. If you are the org admin, approve it immediately in the admin console

### Step 7: Start Chatting

1. Open Feishu on your phone or desktop
2. Search for your bot name (e.g., `DigitalMe`)
3. Start a private chat with it
4. Make sure `digitalme` is running on your computer
5. Send a message — you should see Claude Code respond!

### Troubleshooting

| Problem | Solution |
|---|---|
| Bot doesn't respond | Check that `digitalme` is running; check terminal logs for errors |
| `permission denied` error | Go back to Step 5 and add the missing permission scope, then re-publish |
| `event not received` | Make sure you selected **WebSocket** mode (not Webhook) in Step 4 |
| Messages delayed | Normal on first message — Claude Code needs ~5s to start a session |
| Voice messages not working | Install ffmpeg + whisper.cpp, enable `[speech]` in config.toml |

### Architecture

```
  📱 Feishu App (your phone)
     │
     │  WebSocket (outbound from your machine, no public IP needed)
     │
     ▼
  ⚡ DigitalMe (on your PC/server)
     │
     ▼
  🤖 Claude Code CLI (persistent process)
     │
     ▼
  📂 Your Codebase
```

Key points:
- **No public IP required** — WebSocket connects outbound from your machine to Feishu servers
- **No webhook URL** — unlike Slack or Telegram webhook mode
- **Always-on** — as long as `digitalme` is running, the bot is active
- **Multi-session** — use `/new` to create multiple conversations, `/list` to switch

## REST API

| Endpoint | Description |
|---|---|
| `GET /api/status` | System health, version, uptime, OS info |
| `GET /api/engines` | Engine list with agent type, platforms, sessions |
| `GET /api/heartbeat` | Last 100 heartbeat check records |
| `GET /api/activity` | Session activity with idle times |

## Comparison

| Feature | Claude Code CLI | OpenClaw | DigitalMe |
|---|---|---|---|
| AI coding agent | Yes | Yes | Yes (6 agents) |
| Web UI | No | Yes | Yes (dark theme) |
| Mobile access via chat | No | No | **Yes (8 platforms)** |
| Persistent sessions | Yes | Yes | Yes |
| Heartbeat monitoring | No | No | **Yes** |
| Idle reminders | No | No | **Yes** |
| Remote dir switching | No | No | **Yes (`/cd`)** |
| Remote screenshot | No | No | **Yes** |
| File sendback | No | No | **Yes** |
| Task completion notify | No | No | **Yes** |
| Local voice recognition | No | No | **Yes** |
| Multi-platform chat | No | No | **Yes** |
| Self-hosted | Yes | Cloud | **Yes** |

## Remote Directory Switching (`/cd`)

Switch Claude Code's working directory from your phone — no need to SSH in or remember full paths.

```
You:  /cd digital
Bot:  🔍 Searching for folders matching: digital
      📂 Found 3 matching folder(s):
      1. D:\projects\digital-garden
      2. D:\pythonPycharms\055digitalme
      3. C:\Users\dev\digital-notes
      Reply /cd <number> to select.

You:  /cd 2
Bot:  ✅ Switched to: D:\pythonPycharms\055digitalme
      New sessions will run in this directory.
```

- **Fuzzy search** — just type a keyword, DigitalMe scans your drives for matching folders
- **Two-level deep** — searches top-level directories and one level of subdirectories
- **Interactive selection** — multiple matches? Pick by number
- **Direct path** — `/cd D:\exact\path` works too
- **Safe** — only reads directory names, never touches file contents

## Slash Commands

| Command | Description |
|---|---|
| `/new` | Start a new session |
| `/list` | List all sessions |
| `/switch <id>` | Switch to a session |
| `/stop` | Stop the current agent process |
| `/cd <keyword>` | Search & switch working directory from chat |
| `/model <name>` | Change the AI model |
| `/mode <mode>` | Change permission mode |
| `/screenshot` `/ss` | Capture screen and send to chat |
| `/sendback <path>` | Send a file to chat (PDF, images, etc.) |
| `/version` | Show version info |
| `/help` | Show all commands |

## Tech Stack

- **Language**: Go 1.22+
- **Agent SDK**: Claude Code CLI, Codex CLI, Cursor, Gemini CLI
- **Platforms**: Feishu SDK (WebSocket), Telegram Bot API, Slack API, Discord Gateway, and more
- **Dashboard**: Embedded HTML with glassmorphism CSS, vanilla JS, REST API backend
- **Storage**: SQLite-based session persistence

## Changelog

### v0.6.0 — Remote Directory Switching
- `/cd <keyword>` command: search and switch Claude Code's working directory from chat
- Fuzzy directory search across all drives, two levels deep
- Interactive numbered selection for multiple matches
- Automatic session cleanup on directory change

### v0.5.0 — Smart File Sendback + Voice + Task Notify
- `/sendback` command: send files (PDF, images, xlsx, etc.) to chat
- `cc-connect sendback` CLI: Claude Code proactively sends generated files back to user
- Task completion notification: automatic summary when Claude Code finishes (tools used, duration)
- Local voice recognition via whisper.cpp — no API key, no server, fully offline
- `FileSender` interface for platform file upload
- Feishu file upload and send support
- Agent system prompt instructs Claude Code to auto-sendback files

### v0.4.0 — Remote Screenshot
- `/screenshot` (`/ss`) command: capture your screen and receive it in chat
- `ImageSender` interface for platforms to send image messages
- Feishu image upload and send support

### v0.3.0 — Precision Dashboard
- Complete UI redesign: dark theme with SVG heartbeat line chart
- Teal accent palette replacing generic purple gradients
- Live clock, animated status pill, grid texture background
- Staggered entrance animations, responsive two-column layout

### v0.2.0 — Heartbeat + Idle Integration
- Heartbeat status degrades when idle reminders get no response
- Idle reminder repeats every interval (not just once)
- SVG heartbeat chart: full height = healthy, 40% = degraded, 10% = unhealthy

### v0.1.0 — Initial Enhanced Features
- Web UI dashboard with real-time status monitoring
- Heartbeat monitor with configurable check interval
- Idle session reminder with configurable threshold
- REST API: `/api/status`, `/api/engines`, `/api/heartbeat`, `/api/activity`
- Cloudflare Tunnel support for external access

## Credits

Built on top of the excellent [cc-connect](https://github.com/chenhg5/cc-connect) multi-agent platform. Enhanced with monitoring, dashboard, screenshot, and reliability features.

## License

[BSL 1.1](LICENSE) (Business Source License)

- **Personal / non-commercial use**: Free
- **Commercial use**: Paid license required
- **Branding**: All versions must retain DigitalMe branding (watermark, dashboard footer, startup banner)
- **Change Date**: 2030-01-01 (converts to Apache 2.0)

### Feature Tiers

| Feature | Free | Pro |
|---------|------|-----|
| Text chat | Yes | Yes |
| Multi-session /new /list /switch | Yes | Yes |
| Permission handling | Yes | Yes |
| Basic commands /model /mode /help | Yes | Yes |
| Voice recognition | - | Yes |
| /sendback file transfer | - | Yes |
| /cron scheduled tasks | - | Yes |
| Web Dashboard | - | Yes |
| /screenshot | - | Yes |
| Task completion notify | - | Yes |
| Message watermark | Every 5 msgs | Every 5 msgs |
