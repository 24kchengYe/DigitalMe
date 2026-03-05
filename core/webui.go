package core

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// WebUIServer serves a status dashboard and REST API on a TCP port.
type WebUIServer struct {
	addr         string
	listener     net.Listener
	server       *http.Server
	mux          *http.ServeMux
	engines      map[string]*Engine
	heartbeat    *HeartbeatMonitor
	idleReminder *IdleReminder
	startedAt    time.Time
	mu           sync.RWMutex
}

// NewWebUIServer creates a web UI server on the given address.
func NewWebUIServer(addr string, heartbeat *HeartbeatMonitor, idleReminder *IdleReminder) (*WebUIServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("webui listen: %w", err)
	}

	s := &WebUIServer{
		addr:         addr,
		listener:     listener,
		mux:          http.NewServeMux(),
		engines:      make(map[string]*Engine),
		heartbeat:    heartbeat,
		idleReminder: idleReminder,
		startedAt:    time.Now(),
	}

	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/engines", s.handleEngines)
	s.mux.HandleFunc("/api/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("/api/activity", s.handleActivity)
	s.mux.HandleFunc("/", s.handleDashboard)

	return s, nil
}

func (s *WebUIServer) RegisterEngine(name string, e *Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.engines[name] = e
}

func (s *WebUIServer) Start() {
	s.server = &http.Server{Handler: s.mux}
	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			slog.Error("webui server error", "error", err)
		}
	}()
	slog.Info("webui server started", "addr", "http://"+s.addr)
}

func (s *WebUIServer) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

// ── API ──────────────────────────────────────────────────────

type statusResponse struct {
	Status    HeartbeatStatus `json:"status"`
	Version   string          `json:"version"`
	Uptime    string          `json:"uptime"`
	UptimeSec int64           `json:"uptime_sec"`
	StartedAt time.Time       `json:"started_at"`
	Projects  int             `json:"projects"`
	GoVersion string          `json:"go_version"`
	OS        string          `json:"os"`
	Arch      string          `json:"arch"`
}

func (s *WebUIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	overall := StatusHealthy
	if latest := s.heartbeat.Latest(); latest != nil {
		overall = latest.Status
	}
	uptime := time.Since(s.startedAt)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse{
		Status:    overall,
		Version:   CurrentVersion,
		Uptime:    formatDuration(uptime),
		UptimeSec: int64(uptime.Seconds()),
		StartedAt: s.startedAt,
		Projects:  len(s.engines),
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	})
}

func (s *WebUIServer) handleEngines(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.heartbeat.EngineStatuses())
}

func (s *WebUIServer) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.heartbeat.History())
}

type activityEntry struct {
	SessionKey   string    `json:"session_key"`
	LastActivity time.Time `json:"last_activity"`
	IdleSec      int64     `json:"idle_sec"`
	IdleStr      string    `json:"idle_str"`
}

func (s *WebUIServer) handleActivity(w http.ResponseWriter, r *http.Request) {
	var entries []activityEntry
	if s.idleReminder != nil {
		now := time.Now()
		for k, t := range s.idleReminder.AllActivity() {
			idle := now.Sub(t)
			entries = append(entries, activityEntry{
				SessionKey:   k,
				LastActivity: t,
				IdleSec:      int64(idle.Seconds()),
				IdleStr:      formatDuration(idle),
			})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// ── Dashboard ────────────────────────────────────────────────

func (s *WebUIServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>cc-connect Monitor</title>
<style>
:root{--bg:#0b0f1a;--surface:#131926;--surface2:#1a2235;--border:#242d42;--text:#e4e8f1;--text2:#8892a8;--accent:#6c8cff;--green:#3dd68c;--yellow:#f0c050;--red:#f06060;--radius:14px}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,"SF Pro Display","Segoe UI",Roboto,sans-serif;background:var(--bg);color:var(--text);min-height:100vh}

.topbar{background:var(--surface);border-bottom:1px solid var(--border);padding:18px 28px;display:flex;align-items:center;justify-content:space-between}
.topbar h1{font-size:18px;font-weight:700;letter-spacing:-.3px}
.topbar h1 em{font-style:normal;color:var(--accent)}
.topbar .meta{display:flex;gap:16px;align-items:center;font-size:12px;color:var(--text2)}
.live-dot{width:7px;height:7px;border-radius:50%;background:var(--green);animation:blink 2s infinite}
@keyframes blink{0%,100%{opacity:1}50%{opacity:.3}}

.wrap{max-width:1040px;margin:0 auto;padding:28px 20px}

/* Status Hero */
.hero{background:linear-gradient(135deg,#0d2137 0%,#0a1628 100%);border:1px solid var(--border);border-radius:var(--radius);padding:28px 32px;margin-bottom:24px;display:flex;align-items:center;gap:20px}
.hero-icon{width:52px;height:52px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:24px}
.hero-icon.ok{background:rgba(61,214,140,.12);color:var(--green)}
.hero-icon.warn{background:rgba(240,192,80,.12);color:var(--yellow)}
.hero-icon.fail{background:rgba(240,96,96,.12);color:var(--red)}
.hero-info h2{font-size:20px;font-weight:700;margin-bottom:4px}
.hero-info p{font-size:13px;color:var(--text2)}

/* Metric Cards */
.metrics{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin-bottom:24px}
@media(max-width:640px){.metrics{grid-template-columns:repeat(2,1fr)}}
.metric{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:20px}
.metric .lbl{font-size:11px;color:var(--text2);text-transform:uppercase;letter-spacing:.8px;margin-bottom:6px}
.metric .val{font-size:26px;font-weight:800;letter-spacing:-.5px}
.metric .val.accent{color:var(--accent)}

/* Sections */
.section{background:var(--surface);border:1px solid var(--border);border-radius:var(--radius);padding:22px;margin-bottom:20px}
.section h3{font-size:14px;font-weight:600;margin-bottom:14px;display:flex;align-items:center;gap:8px}
.section h3 .icon{font-size:16px}

/* Table */
table{width:100%;border-collapse:collapse}
th{text-align:left;font-size:11px;color:var(--text2);text-transform:uppercase;letter-spacing:.6px;padding:8px 10px;border-bottom:1px solid var(--border)}
td{padding:12px 10px;font-size:13px;border-bottom:1px solid rgba(255,255,255,.03)}
tr:hover td{background:rgba(108,140,255,.03)}
.badge{display:inline-block;padding:3px 10px;border-radius:20px;font-size:11px;font-weight:600}
.badge-ok{background:rgba(61,214,140,.12);color:var(--green)}
.badge-blue{background:rgba(108,140,255,.12);color:var(--accent)}

/* Heartbeat strip */
.hb-strip{display:flex;gap:2px;height:32px;align-items:flex-end}
.hb-strip .dot{flex:1;border-radius:3px;min-width:5px;transition:all .2s}
.dot-ok{background:var(--green)}
.dot-warn{background:var(--yellow)}
.dot-fail{background:var(--red)}

/* Activity */
.activity-item{display:flex;justify-content:space-between;align-items:center;padding:10px 0;border-bottom:1px solid rgba(255,255,255,.04)}
.activity-item:last-child{border:none}
.activity-key{font-size:13px;font-family:monospace;color:var(--accent)}
.activity-idle{font-size:12px;color:var(--text2)}

.footer{text-align:center;padding:20px;font-size:11px;color:var(--text2);letter-spacing:.3px}
</style>
</head>
<body>
<div class="topbar">
  <h1><em>cc-connect</em> Monitor</h1>
  <div class="meta"><div class="live-dot"></div><span id="autoRefresh">LIVE</span></div>
</div>
<div class="wrap">
  <div class="hero" id="hero">
    <div class="hero-icon ok" id="heroIcon">&#10003;</div>
    <div class="hero-info">
      <h2 id="heroTitle">Loading...</h2>
      <p id="heroSub">Checking system status</p>
    </div>
  </div>

  <div class="metrics">
    <div class="metric"><div class="lbl">Uptime</div><div class="val" id="uptime">-</div></div>
    <div class="metric"><div class="lbl">Projects</div><div class="val accent" id="projects">-</div></div>
    <div class="metric"><div class="lbl">Sessions</div><div class="val accent" id="sessions">-</div></div>
    <div class="metric"><div class="lbl">Version</div><div class="val" id="version">-</div></div>
  </div>

  <div class="section">
    <h3><span class="icon">&#9881;</span> Engines</h3>
    <table>
      <thead><tr><th>Project</th><th>Agent</th><th>Platform</th><th>Sessions</th><th>Status</th><th>Uptime</th></tr></thead>
      <tbody id="engTbody"></tbody>
    </table>
  </div>

  <div class="section">
    <h3><span class="icon">&#128338;</span> Recent Activity</h3>
    <div id="activityList"><p style="color:var(--text2);font-size:13px">No activity yet</p></div>
  </div>

  <div class="section">
    <h3><span class="icon">&#9829;</span> Heartbeat (last 60 checks)</h3>
    <div class="hb-strip" id="hbStrip"></div>
  </div>
</div>
<div class="footer">cc-connect enhanced &middot; <span id="fv"></span> &middot; <span id="sysInfo"></span></div>

<script>
const $ = id => document.getElementById(id);
const F = async u => (await fetch(u)).json();

const heroMap = {
  healthy: ['&#10003;', 'ok', 'All Systems Operational', 'Everything is running smoothly'],
  degraded: ['&#9888;', 'warn', 'Performance Degraded', 'Some components need attention'],
  unhealthy: ['&#10007;', 'fail', 'System Issues', 'Critical problems detected'],
};

async function refresh() {
  try {
    const [st, en, hb, ac] = await Promise.all([F('/api/status'), F('/api/engines'), F('/api/heartbeat'), F('/api/activity')]);

    // Hero
    const h = heroMap[st.status] || heroMap.healthy;
    $('heroIcon').innerHTML = h[0];
    $('heroIcon').className = 'hero-icon ' + h[1];
    $('heroTitle').textContent = h[2];
    $('heroSub').textContent = h[3] + ' \u00b7 started ' + new Date(st.started_at).toLocaleString();

    // Metrics
    $('uptime').textContent = st.uptime || '0m';
    $('projects').textContent = st.projects;
    $('version').textContent = st.version || 'dev';
    $('fv').textContent = st.version || 'dev';
    $('sysInfo').textContent = st.os + '/' + st.arch;

    let total = 0;
    (en||[]).forEach(e => total += e.active_sessions || 0);
    $('sessions').textContent = total;

    // Engines
    const tb = $('engTbody');
    tb.innerHTML = '';
    if (!en || en.length === 0) {
      tb.innerHTML = '<tr><td colspan="6" style="color:var(--text2);text-align:center">No engines running</td></tr>';
    } else {
      en.forEach(e => {
        const pl = (e.platforms||[]).map(p=>'<span class="badge badge-blue">'+p+'</span>').join(' ');
        tb.innerHTML += '<tr><td><b>'+e.name+'</b></td><td><span class="badge badge-blue">'+e.agent_type+'</span></td><td>'+pl+'</td><td>'+
          (e.active_sessions||0)+'</td><td><span class="badge badge-ok">'+e.status+'</span></td><td>'+
          (e.uptime||'-')+'</td></tr>';
      });
    }

    // Activity
    const al = $('activityList');
    if (!ac || ac.length === 0) {
      al.innerHTML = '<p style="color:var(--text2);font-size:13px">No activity yet \u2014 send a message in Feishu to start</p>';
    } else {
      al.innerHTML = '';
      ac.sort((a,b) => new Date(b.last_activity) - new Date(a.last_activity));
      ac.forEach(a => {
        const t = new Date(a.last_activity).toLocaleTimeString();
        al.innerHTML += '<div class="activity-item"><span class="activity-key">'+a.session_key+'</span><span class="activity-idle">Last active: '+t+' (idle '+a.idle_str+')</span></div>';
      });
    }

    // Heartbeat
    const strip = $('hbStrip');
    strip.innerHTML = '';
    const dots = (hb||[]).slice(-60);
    dots.forEach(d => {
      const el = document.createElement('div');
      el.className = 'dot dot-' + ({healthy:'ok',degraded:'warn',unhealthy:'fail'}[d.status]||'ok');
      el.style.height = '100%';
      el.title = new Date(d.timestamp).toLocaleTimeString() + ' | sessions: ' + d.active_sessions;
      strip.appendChild(el);
    });
  } catch(e) { console.error(e); }
}
refresh();
setInterval(refresh, 8000);
</script>
</body>
</html>`
