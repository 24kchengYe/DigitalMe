# cc-connect-enhanced

> **cc-connect on steroids** — Web UI monitoring, heartbeat health checks, and idle session reminders.

Based on [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect), this fork adds features for **production monitoring and user engagement**.

```
                    ┌─────────────────────────────┐
                    │    Web Dashboard (:9315)     │
                    │  ┌───────────────────────┐  │
                    │  │ ✓ All Systems Online  │  │
                    │  │ Uptime: 3d 12h        │  │
                    │  │ Sessions: 2           │  │
                    │  │ ♥♥♥♥♥♥♥♥♥♥ Heartbeat │  │
                    │  └───────────────────────┘  │
                    └──────────┬──────────────────┘
                               │
  Phone/Tablet ──► Feishu ──► cc-connect ──► Claude Code
                               │
                    ┌──────────┴──────────────────┐
                    │  💤 Idle 30min? Auto-remind  │
                    └─────────────────────────────┘
```

## New Features

### 1. Web UI Dashboard

Real-time status dashboard accessible from any browser.

- **System status** — healthy / degraded / unhealthy at a glance
- **Engine overview** — agent type, platforms, active sessions, uptime
- **Session activity** — last message time, idle duration per session
- **Heartbeat history** — visual strip of health check results
- **Auto-refresh** — updates every 8 seconds

Access at `http://<your-ip>:9315` — works from phone too!

### 2. Heartbeat Monitor

Periodic health checks running in the background.

- Checks every 30 seconds (configurable)
- Tracks active session count over time
- Stores last 100 check results
- Feeds data to Web UI and REST API

### 3. Idle Session Reminder

Proactively reminds users when their session has been idle.

- Sends a Feishu/chat message after N minutes of inactivity
- Configurable threshold (default: 30 minutes)
- Only reminds once per idle period — won't spam
- Resets when user sends a new message

## Quick Start

### Prerequisites

- [Go 1.22+](https://go.dev/dl/) installed
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) (or other supported agents) configured
- A messaging platform bot (Feishu, Telegram, Slack, etc.) set up

### Build

```bash
git clone https://github.com/24kchengYe/cc-connect-enhanced.git
cd cc-connect-enhanced
go build -o cc-connect.exe ./cmd/cc-connect/
```

### Configure

Edit `~/.cc-connect/config.toml`:

```toml
[[projects]]
name = "my-project"

[projects.agent]
type = "claudecode"

[projects.agent.options]
work_dir = "/path/to/your/project"
mode = "default"

[[projects.platforms]]
type = "feishu"

[projects.platforms.options]
app_id = "your-app-id"
app_secret = "your-app-secret"

# ── Enhanced Features ──

[webui]
enabled = true
addr = "0.0.0.0:9315"        # "127.0.0.1:9315" for local only
heartbeat_interval = 30       # seconds

[idle]
enabled = true
idle_minutes = 30             # remind after 30 min idle
```

### Run

```bash
./cc-connect.exe
```

Open `http://localhost:9315` to see the dashboard.

## REST API

| Endpoint | Description |
|----------|-------------|
| `GET /api/status` | Overall system status, version, uptime |
| `GET /api/engines` | List engines with agent type, platforms, sessions |
| `GET /api/heartbeat` | Heartbeat history (last 100 checks) |
| `GET /api/activity` | Session activity and idle times |

## All cc-connect Features (inherited)

This fork includes **everything** from the original cc-connect:

- **6 AI Agents** — Claude Code, Codex, Cursor, Gemini CLI, Qoder, OpenCode
- **8 Chat Platforms** — Feishu, DingTalk, Telegram, Slack, Discord, LINE, WeChat Work, QQ
- **Slash commands** — `/new`, `/list`, `/switch`, `/model`, `/mode`, `/stop`, `/help`, etc.
- **Session management** — multiple sessions per user, history, persistence
- **Permission control** — default, acceptEdits, plan, YOLO modes
- **Voice & images** — speech-to-text, multimodal support
- **Scheduled tasks** — cron jobs via natural language
- **Provider management** — multiple API keys, runtime switching
- **Daemon mode** — systemd / launchd integration

See [original README](https://github.com/chenhg5/cc-connect) for full documentation.

## Credits

- [chenhg5/cc-connect](https://github.com/chenhg5/cc-connect) — the original project
- Enhanced features by [@24kchengYe](https://github.com/24kchengYe)

## License

MIT
