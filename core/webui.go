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

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>cc-connect Dashboard</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{
  --bg:#0a0a0f;--bg2:#0f0f18;--surface:rgba(255,255,255,.03);--glass:rgba(255,255,255,.05);
  --border:rgba(255,255,255,.06);--border-h:rgba(255,255,255,.1);
  --text:#e2e8f0;--text2:#64748b;--text3:#475569;
  --accent:#7c3aed;--accent2:#a78bfa;--accent-glow:rgba(124,58,237,.15);
  --green:#10b981;--green-glow:rgba(16,185,129,.12);
  --amber:#f59e0b;--amber-glow:rgba(245,158,11,.12);
  --red:#ef4444;--red-glow:rgba(239,68,68,.12);
  --radius:12px;--radius-lg:16px;
}
body{font-family:-apple-system,BlinkMacSystemFont,"SF Pro Display","Segoe UI",Roboto,Helvetica,sans-serif;background:var(--bg);color:var(--text);line-height:1.6;min-height:100vh}
body::before{content:"";position:fixed;top:0;left:0;right:0;height:600px;background:radial-gradient(ellipse 80% 50% at 50% -20%,rgba(124,58,237,.08),transparent);pointer-events:none;z-index:0}

/* Scrollbar */
::-webkit-scrollbar{width:6px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:rgba(255,255,255,.08);border-radius:3px}
::-webkit-scrollbar-thumb:hover{background:rgba(255,255,255,.14)}

/* Nav */
nav{background:rgba(10,10,15,.8);backdrop-filter:blur(20px) saturate(180%);-webkit-backdrop-filter:blur(20px) saturate(180%);border-bottom:1px solid var(--border);padding:0 24px;height:56px;display:flex;align-items:center;justify-content:space-between;position:sticky;top:0;z-index:100}
nav .brand{display:flex;align-items:center;gap:10px;font-weight:700;font-size:15px;color:var(--text);letter-spacing:-.2px}
nav .brand-icon{width:32px;height:32px;border-radius:8px;background:linear-gradient(135deg,var(--accent),#a78bfa);display:flex;align-items:center;justify-content:center}
nav .brand-icon svg{width:18px;height:18px;color:#fff}
nav .live{display:flex;align-items:center;gap:8px;font-size:12px;color:var(--text2);font-weight:500;padding:6px 12px;border-radius:20px;background:rgba(16,185,129,.08);border:1px solid rgba(16,185,129,.15)}
nav .live-dot{width:7px;height:7px;border-radius:50%;background:var(--green);animation:pulse 2s ease-in-out infinite}
@keyframes pulse{0%,100%{opacity:1;box-shadow:0 0 0 0 rgba(16,185,129,.5)}50%{opacity:.8;box-shadow:0 0 0 8px rgba(16,185,129,0)}}

.wrap{max-width:1040px;margin:0 auto;padding:24px 20px;position:relative;z-index:1}
@media(max-width:640px){.wrap{padding:16px 12px}}

/* Status Bar */
.status-row{display:flex;align-items:center;gap:12px;padding:14px 20px;border-radius:var(--radius);margin-bottom:24px;font-weight:600;font-size:13px;backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);transition:all .4s ease}
.status-ok{background:rgba(16,185,129,.08);border:1px solid rgba(16,185,129,.2);color:var(--green)}
.status-warn{background:rgba(245,158,11,.08);border:1px solid rgba(245,158,11,.2);color:var(--amber)}
.status-err{background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.2);color:var(--red)}
.status-dot{width:10px;height:10px;border-radius:50%;flex-shrink:0}
.status-ok .status-dot{background:var(--green);box-shadow:0 0 8px rgba(16,185,129,.5)}
.status-warn .status-dot{background:var(--amber);box-shadow:0 0 8px rgba(245,158,11,.5)}
.status-err .status-dot{background:var(--red);box-shadow:0 0 8px rgba(239,68,68,.5)}

/* KPI Grid */
.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:14px;margin-bottom:24px}
@media(max-width:640px){.grid{grid-template-columns:repeat(2,1fr);gap:10px}}
.kpi{background:var(--glass);backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);border:1px solid var(--border);border-radius:var(--radius);padding:20px;transition:border-color .3s,background .3s}
.kpi:hover{border-color:var(--border-h);background:rgba(255,255,255,.06)}
.kpi .label{font-size:11px;font-weight:600;color:var(--text2);text-transform:uppercase;letter-spacing:.8px;margin-bottom:8px}
.kpi .num{font-size:30px;font-weight:800;letter-spacing:-.5px;background:linear-gradient(135deg,var(--text) 0%,var(--text2) 100%);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}
.kpi .num.accent{background:linear-gradient(135deg,var(--accent2) 0%,var(--accent) 100%);-webkit-background-clip:text;-webkit-text-fill-color:transparent;background-clip:text}

/* Glass Panels */
.panel{background:var(--glass);backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);border:1px solid var(--border);border-radius:var(--radius-lg);margin-bottom:20px;overflow:hidden;transition:border-color .3s}
.panel:hover{border-color:var(--border-h)}
.panel-head{padding:16px 20px;border-bottom:1px solid var(--border);font-size:13px;font-weight:700;display:flex;align-items:center;gap:10px;color:var(--text)}
.panel-head svg{width:16px;height:16px;color:var(--accent2);opacity:.8}
.panel-body{padding:16px 20px}
.panel-empty{color:var(--text3);font-size:13px;text-align:center;padding:28px 0}

/* Table */
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;font-size:10px;font-weight:700;color:var(--text3);text-transform:uppercase;letter-spacing:.8px;padding:10px 14px;border-bottom:1px solid var(--border)}
td{padding:12px 14px;border-bottom:1px solid rgba(255,255,255,.03);color:var(--text);transition:background .2s}
tr:hover td{background:rgba(255,255,255,.02)}
td b{color:var(--text);font-weight:600}

/* Chips */
.chip{display:inline-block;padding:3px 10px;border-radius:6px;font-size:11px;font-weight:600;letter-spacing:.2px}
.chip-green{background:rgba(16,185,129,.1);color:var(--green);border:1px solid rgba(16,185,129,.15)}
.chip-purple{background:rgba(124,58,237,.1);color:var(--accent2);border:1px solid rgba(124,58,237,.15)}
.chip-blue{background:rgba(99,102,241,.1);color:#818cf8;border:1px solid rgba(99,102,241,.15)}
.chip-amber{background:rgba(245,158,11,.1);color:var(--amber);border:1px solid rgba(245,158,11,.15)}
.chip-red{background:rgba(239,68,68,.1);color:var(--red);border:1px solid rgba(239,68,68,.15)}

/* Activity */
.act-row{display:flex;justify-content:space-between;align-items:center;padding:12px 0;border-bottom:1px solid rgba(255,255,255,.03);font-size:13px;transition:opacity .2s}
.act-row:last-child{border:none}
.act-row:hover{opacity:.85}
.act-key{font-family:"SF Mono",Consolas,"Liberation Mono",monospace;color:var(--accent2);font-size:12px;font-weight:600;background:rgba(124,58,237,.06);padding:3px 8px;border-radius:5px}
.act-meta{color:var(--text2);font-size:12px}

/* Heartbeat */
.hb-wrap{position:relative}
.hb-label{display:flex;justify-content:space-between;margin-bottom:10px;font-size:11px;color:var(--text3);font-weight:500}
.hb{display:flex;gap:2px;height:48px;align-items:flex-end;padding:4px 0;position:relative}
.hb::before{content:"";position:absolute;bottom:0;left:0;right:0;height:1px;background:rgba(255,255,255,.04)}
.hb span{flex:1;border-radius:3px 3px 1px 1px;min-width:3px;cursor:default;transition:opacity .2s,transform .2s;position:relative}
.hb span:hover{opacity:.75;transform:scaleY(1.08);transform-origin:bottom}
.hb-g{background:linear-gradient(0deg,rgba(16,185,129,.6),var(--green));box-shadow:0 0 6px rgba(16,185,129,.15)}
.hb-y{background:linear-gradient(0deg,rgba(245,158,11,.5),var(--amber));box-shadow:0 0 6px rgba(245,158,11,.12)}
.hb-r{background:linear-gradient(0deg,rgba(239,68,68,.5),var(--red));box-shadow:0 0 6px rgba(239,68,68,.12)}

/* Footer */
.foot{text-align:center;padding:32px 20px 40px;font-size:11px;color:var(--text3);letter-spacing:.2px}
.foot span{color:var(--text2)}

/* Responsive table scroll */
@media(max-width:768px){
  .panel-body.tbl-wrap{overflow-x:auto;padding:0}
  table{min-width:560px}
}

/* Fade-in animation */
@keyframes fadeUp{from{opacity:0;transform:translateY(8px)}to{opacity:1;transform:translateY(0)}}
.grid,.panel,.status-row{animation:fadeUp .5s ease both}
.grid{animation-delay:.05s}
.panel:nth-child(1){animation-delay:.1s}
.panel:nth-child(2){animation-delay:.15s}
.panel:nth-child(3){animation-delay:.2s}
</style>
</head>
<body>

<nav>
  <div class="brand">
    <div class="brand-icon">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
    </div>
    cc-connect
  </div>
  <div class="live"><div class="live-dot"></div>Live</div>
</nav>

<div class="wrap">

  <div id="statusBar" class="status-row status-ok">
    <div class="status-dot"></div>
    <span id="statusMsg">Connecting...</span>
  </div>

  <div class="grid">
    <div class="kpi"><div class="label">Running Time</div><div class="num" id="kUptime">--</div></div>
    <div class="kpi"><div class="label">Projects</div><div class="num accent" id="kProjects">--</div></div>
    <div class="kpi"><div class="label">Active Sessions</div><div class="num accent" id="kSessions">--</div></div>
    <div class="kpi"><div class="label">Version</div><div class="num" id="kVersion">--</div></div>
  </div>

  <div class="panel">
    <div class="panel-head">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/></svg>
      Engine Status
    </div>
    <div class="panel-body tbl-wrap" style="padding:0">
      <table><thead><tr><th>Project</th><th>Agent</th><th>Platform</th><th>Sessions</th><th>Status</th><th>Uptime</th></tr></thead>
      <tbody id="tEngines"></tbody></table>
    </div>
  </div>

  <div class="panel">
    <div class="panel-head">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
      Session Activity
    </div>
    <div class="panel-body" id="actBody">
      <p class="panel-empty">Waiting for first message...</p>
    </div>
  </div>

  <div class="panel">
    <div class="panel-head">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.84 4.61a5.5 5.5 0 00-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 00-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 000-7.78z"/></svg>
      Heartbeat
    </div>
    <div class="panel-body">
      <div class="hb-wrap">
        <div class="hb-label"><span>Older</span><span id="hbCount">-- checks</span><span>Recent</span></div>
        <div class="hb" id="hbBar"></div>
      </div>
    </div>
  </div>

</div>

<div class="foot">cc-connect enhanced &middot; <span id="fVer">-</span> &middot; <span id="fSys">-</span></div>

<script>
const $=id=>document.getElementById(id),F=async u=>(await fetch(u)).json();
const sMap={healthy:['status-ok','All systems operational'],degraded:['status-warn','Sessions idle \u2014 awaiting response'],unhealthy:['status-err','System issues detected']};

async function R(){
  try{
    const[s,en,hb,ac]=await Promise.all([F('/api/status'),F('/api/engines'),F('/api/heartbeat'),F('/api/activity')]);

    /* Status bar */
    const[cls,msg]=sMap[s.status]||sMap.healthy;
    $('statusBar').className='status-row '+cls;
    $('statusMsg').textContent=msg+' \u2014 started '+new Date(s.started_at).toLocaleString();

    /* KPI */
    $('kUptime').textContent=s.uptime||'0m';
    $('kProjects').textContent=s.projects;
    $('kVersion').textContent=s.version||'dev';
    $('fVer').textContent=s.version||'dev';
    $('fSys').textContent=(s.os||'?')+'/'+(s.arch||'?');

    let tot=0;(en||[]).forEach(e=>tot+=e.active_sessions||0);
    $('kSessions').textContent=tot;

    /* Engines table */
    const tb=$('tEngines');tb.innerHTML='';
    if(!en||!en.length){
      tb.innerHTML='<tr><td colspan="6" class="panel-empty">No engines registered</td></tr>';
    }else{
      en.forEach(e=>{
        const pl=(e.platforms||[]).map(p=>'<span class="chip chip-blue">'+p+'</span>').join(' ');
        const sc=e.status==='degraded'?'amber':e.status==='unhealthy'?'red':'green';
        tb.innerHTML+='<tr><td><b>'+e.name+'</b></td><td><span class="chip chip-purple">'+e.agent_type+'</span></td><td>'+pl+'</td><td>'+
          (e.active_sessions||0)+'</td><td><span class="chip chip-'+sc+'">'+e.status+'</span></td><td style="color:var(--text2)">'+(e.uptime||'-')+'</td></tr>';
      });
    }

    /* Activity list */
    const ab=$('actBody');
    if(!ac||!ac.length){
      ab.innerHTML='<p class="panel-empty">No active sessions</p>';
    }else{
      ab.innerHTML='';
      ac.sort((a,b)=>new Date(b.last_activity)-new Date(a.last_activity));
      ac.forEach(a=>{
        const t=new Date(a.last_activity).toLocaleTimeString();
        ab.innerHTML+='<div class="act-row"><span class="act-key">'+a.session_key+'</span><span class="act-meta">'+t+' &middot; idle '+a.idle_str+'</span></div>';
      });
    }

    /* Heartbeat bar */
    const hArr=(hb||[]).slice(-80);
    $('hbCount').textContent=hArr.length+' checks';
    const hb2=$('hbBar');hb2.innerHTML='';
    hArr.forEach(d=>{
      const el=document.createElement('span');
      const sc={healthy:'g',degraded:'y',unhealthy:'r'}[d.status]||'g';
      el.className='hb-'+sc;
      el.style.height={g:'100%',y:'40%',r:'10%'}[sc];
      el.title=new Date(d.timestamp).toLocaleTimeString()+' | '+d.status+' | sessions: '+(d.active_sessions||0);
      hb2.appendChild(el);
    });

  }catch(e){console.error('dashboard refresh error:',e)}
}
R();setInterval(R,6000);
</script>
</body>
</html>`
