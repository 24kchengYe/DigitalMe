<div align="center">

# DigitalMe

### Your AI Coding Agent, Always On, Anywhere.

Control **Claude Code** from your phone via chat. Real-time dashboard. Heartbeat monitoring. Zero downtime.

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Claude Code](https://img.shields.io/badge/Claude_Code-Compatible-7c3aed?style=flat-square&logo=anthropic&logoColor=white)](https://docs.anthropic.com/en/docs/claude-code)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

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
- **Slash Commands** — `/new`, `/model`, `/mode`, `/stop`, `/help`, and more
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
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated
- A messaging platform bot configured (e.g., Feishu, Telegram)

### Build

```bash
git clone https://github.com/24kchengYe/DigitalMe.git
cd DigitalMe
go build -o digitalme ./cmd/cc-connect/
```

### Configure

Create `~/.cc-connect/config.toml`:

```toml
language = "en"          # "en", "zh", "ja", "es"

[[projects]]
name = "my-project"

[projects.agent]
type = "claudecode"      # or "codex", "cursor", "gemini", "qoder", "opencode"

[projects.agent.options]
work_dir = "/path/to/your/project"
mode = "default"         # "default", "acceptEdits", "plan", "bypassPermissions"

# Pick your platform (at least one)
[[projects.platforms]]
type = "feishu"

[projects.platforms.options]
app_id = "your-app-id"
app_secret = "your-app-secret"

# ── Monitoring ──

[webui]
enabled = true
addr = "0.0.0.0:9315"
heartbeat_interval = 30   # seconds

[idle]
enabled = true
idle_minutes = 30          # remind after N minutes idle
```

### Run

```bash
./digitalme
```

Open `http://localhost:9315` for the dashboard.

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
| Remote screenshot | No | No | **Yes** |
| Multi-platform chat | No | No | **Yes** |
| Self-hosted | Yes | Cloud | **Yes** |

## Slash Commands

| Command | Description |
|---|---|
| `/new` | Start a new session |
| `/list` | List all sessions |
| `/switch <id>` | Switch to a session |
| `/stop` | Stop the current agent process |
| `/model <name>` | Change the AI model |
| `/mode <mode>` | Change permission mode |
| `/screenshot` `/ss` | Capture screen and send to chat |
| `/version` | Show version info |
| `/help` | Show all commands |

## Tech Stack

- **Language**: Go 1.22+
- **Agent SDK**: Claude Code CLI, Codex CLI, Cursor, Gemini CLI
- **Platforms**: Feishu SDK (WebSocket), Telegram Bot API, Slack API, Discord Gateway, and more
- **Dashboard**: Embedded HTML with glassmorphism CSS, vanilla JS, REST API backend
- **Storage**: SQLite-based session persistence

## Changelog

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

MIT
