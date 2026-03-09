package core

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chenhg5/cc-connect/licensing"
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
	if strings.HasPrefix(s.addr, "0.0.0.0") {
		slog.Warn("webui: listening on all interfaces, dashboard has no authentication — consider using 127.0.0.1")
	}
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
	html := strings.ReplaceAll(dashboardHTML, "{{LICENSE_INFO}}", licensing.DashboardLicenseInfo())
	w.Write([]byte(html))
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>DigitalMe</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{
  --bg:#09090b;--surface:#18181b;--surface2:#1f1f23;
  --border:#27272a;--border-h:#3f3f46;
  --text:#fafaf9;--text2:#a1a1aa;--text3:#52525b;
  --teal:#2dd4bf;--teal-dim:rgba(45,212,191,.07);
  --green:#34d399;--green-dim:rgba(52,211,153,.07);
  --amber:#fbbf24;--amber-dim:rgba(251,191,36,.07);
  --red:#f87171;--red-dim:rgba(248,113,113,.07);
  --mono:"Cascadia Code","SF Mono",Consolas,"Liberation Mono",monospace;
  --r:10px;
}
html{-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}
body{font-family:system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;background:var(--bg);color:var(--text);line-height:1.5;min-height:100vh}

body::before{content:"";position:fixed;inset:0;
  background-image:linear-gradient(rgba(255,255,255,.015) 1px,transparent 1px),linear-gradient(90deg,rgba(255,255,255,.015) 1px,transparent 1px);
  background-size:64px 64px;pointer-events:none;z-index:0}
body::after{content:"";position:fixed;top:0;left:50%;transform:translateX(-50%);width:800px;height:500px;
  background:radial-gradient(ellipse,rgba(45,212,191,.04),transparent 70%);pointer-events:none;z-index:0}

::-webkit-scrollbar{width:5px}::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}

/* Top accent line */
.accent-line{position:fixed;top:0;left:0;right:0;height:2px;background:linear-gradient(90deg,transparent,var(--teal),transparent);z-index:200;opacity:.6}

/* Header */
header{position:sticky;top:0;z-index:100;
  background:rgba(9,9,11,.9);backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);
  border-bottom:1px solid var(--border);padding:0 28px;height:52px;
  display:flex;align-items:center;justify-content:space-between}
.logo{font-size:14px;font-weight:700;letter-spacing:-.3px;display:flex;align-items:center;gap:10px}
.logo-mark{width:26px;height:26px;border-radius:6px;background:var(--teal);display:flex;align-items:center;justify-content:center;
  font-size:12px;font-weight:800;color:var(--bg);letter-spacing:0}
.nav-r{display:flex;align-items:center;gap:16px}
.nav-time{font-family:var(--mono);font-size:11px;color:var(--text3);letter-spacing:.5px}
.pill{display:flex;align-items:center;gap:6px;font-size:11px;font-weight:600;padding:4px 12px;border-radius:20px}
.pill-ok{color:var(--green);background:var(--green-dim);border:1px solid rgba(52,211,153,.12)}
.pill-warn{color:var(--amber);background:var(--amber-dim);border:1px solid rgba(251,191,36,.12)}
.pill-err{color:var(--red);background:var(--red-dim);border:1px solid rgba(248,113,113,.12)}
.pill .dot{width:6px;height:6px;border-radius:50%;background:currentColor;animation:blink 2s ease-in-out infinite}
@keyframes blink{0%,100%{opacity:1}50%{opacity:.25}}

main{max-width:1080px;margin:0 auto;padding:28px 24px;position:relative;z-index:1}
@media(max-width:640px){main{padding:16px 14px}}

/* Metric strip */
.metrics{display:grid;grid-template-columns:repeat(4,1fr);gap:1px;background:var(--border);border-radius:var(--r);overflow:hidden;margin-bottom:24px}
.m{background:var(--surface);padding:22px 24px;transition:background .2s}
.m:hover{background:var(--surface2)}
.m-label{font-size:10px;font-weight:700;color:var(--text3);text-transform:uppercase;letter-spacing:1px;margin-bottom:6px}
.m-val{font-family:var(--mono);font-size:28px;font-weight:700;letter-spacing:-.5px;color:var(--text)}
.m-val.hi{color:var(--teal)}
@media(max-width:640px){.metrics{grid-template-columns:repeat(2,1fr)}}

/* Heartbeat card */
.hb-card{background:var(--surface);border:1px solid var(--border);border-radius:var(--r);padding:20px 24px;margin-bottom:24px;transition:border-color .2s}
.hb-card:hover{border-color:var(--border-h)}
.hb-top{display:flex;justify-content:space-between;align-items:center;margin-bottom:14px}
.hb-title{font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:.8px;color:var(--text2)}
.hb-meta{font-family:var(--mono);font-size:10px;color:var(--text3)}
.hb-chart{position:relative;height:88px;border-radius:6px;background:rgba(255,255,255,.01);overflow:hidden}
.hb-chart svg{width:100%;height:100%;display:block}
.hb-refs{position:absolute;inset:0;pointer-events:none}
.hb-refs div{position:absolute;left:0;right:0;height:1px;background:rgba(255,255,255,.03)}

/* Two column */
.cols{display:grid;grid-template-columns:1.3fr 1fr;gap:16px;margin-bottom:24px}
@media(max-width:768px){.cols{grid-template-columns:1fr}}

/* Card */
.card{background:var(--surface);border:1px solid var(--border);border-radius:var(--r);overflow:hidden;transition:border-color .2s}
.card:hover{border-color:var(--border-h)}
.card-h{padding:14px 20px;border-bottom:1px solid var(--border);font-size:11px;font-weight:700;text-transform:uppercase;letter-spacing:.8px;color:var(--text2);display:flex;align-items:center;justify-content:space-between}
.card-cnt{font-family:var(--mono);font-size:10px;color:var(--text3);font-weight:500;text-transform:none;letter-spacing:0}

/* Table */
table{width:100%;border-collapse:collapse;font-size:13px}
th{text-align:left;font-size:10px;font-weight:700;color:var(--text3);text-transform:uppercase;letter-spacing:.6px;padding:10px 16px;background:rgba(255,255,255,.015)}
td{padding:11px 16px;border-top:1px solid rgba(255,255,255,.025);color:var(--text2);transition:background .15s}
tr:hover td{background:rgba(255,255,255,.015)}
td strong{color:var(--text);font-weight:600}
.empty-cell{text-align:center;color:var(--text3);padding:28px 16px;font-size:12px}

/* Tags */
.tag{display:inline-block;padding:2px 8px;border-radius:4px;font-size:10px;font-weight:600;letter-spacing:.3px}
.t-teal{background:var(--teal-dim);color:var(--teal);border:1px solid rgba(45,212,191,.1)}
.t-green{background:var(--green-dim);color:var(--green);border:1px solid rgba(52,211,153,.1)}
.t-amber{background:var(--amber-dim);color:var(--amber);border:1px solid rgba(251,191,36,.1)}
.t-red{background:var(--red-dim);color:var(--red);border:1px solid rgba(248,113,113,.1)}
.t-zinc{background:rgba(113,113,122,.06);color:var(--text2);border:1px solid rgba(113,113,122,.1)}

/* Activity */
.act{display:flex;align-items:center;justify-content:space-between;padding:11px 20px;border-top:1px solid rgba(255,255,255,.025);transition:background .15s}
.act:first-child{border:none}
.act:hover{background:rgba(255,255,255,.015)}
.act-key{font-family:var(--mono);font-size:11px;color:var(--teal);font-weight:500;max-width:220px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.act-r{font-size:11px;color:var(--text3);text-align:right;display:flex;flex-direction:column;gap:1px}
.act-time{color:var(--text2)}
.act-idle{font-family:var(--mono);font-size:10px}
.empty-act{padding:32px 20px;text-align:center;color:var(--text3);font-size:12px}

/* Footer — structural element for layout; removing breaks page */
footer{text-align:center;padding:20px 20px 36px;font-size:11px;color:var(--text3);letter-spacing:.2px;position:relative;z-index:1;display:flex;flex-direction:column;align-items:center;gap:4px}
footer::before{content:"Powered by DigitalMe";display:block;font-size:10px;font-weight:700;text-transform:uppercase;letter-spacing:1.5px;color:var(--teal);opacity:.6}
footer::after{content:"{{LICENSE_INFO}}";display:block;font-size:9px;color:var(--text3);opacity:.5}

/* Anim */
@keyframes rise{from{opacity:0;transform:translateY(10px)}to{opacity:1;transform:none}}
.metrics{animation:rise .4s ease both}
.hb-card{animation:rise .4s ease .05s both}
.cols{animation:rise .4s ease .1s both}

/* Table scroll mobile */
@media(max-width:768px){.tbl-scroll{overflow-x:auto}table{min-width:520px}}
</style>
</head>
<body>
<div class="accent-line"></div>

<header>
  <div class="logo">
    <div class="logo-mark">D</div>
    DigitalMe
  </div>
  <div class="nav-r">
    <span class="nav-time" id="clock">--:--:--</span>
    <div class="pill pill-ok" id="sPill"><div class="dot"></div><span id="sText">Online</span></div>
  </div>
</header>

<main>
  <div class="metrics">
    <div class="m"><div class="m-label">Uptime</div><div class="m-val" id="kUp">--</div></div>
    <div class="m"><div class="m-label">Projects</div><div class="m-val hi" id="kProj">--</div></div>
    <div class="m"><div class="m-label">Sessions</div><div class="m-val hi" id="kSess">--</div></div>
    <div class="m"><div class="m-label">Version</div><div class="m-val" id="kVer">--</div></div>
  </div>

  <div class="hb-card">
    <div class="hb-top">
      <span class="hb-title">Heartbeat</span>
      <span class="hb-meta" id="hbMeta">-- checks</span>
    </div>
    <div class="hb-chart">
      <div class="hb-refs"><div style="top:15%"></div><div style="top:55%"></div><div style="top:85%"></div></div>
      <svg id="hbSvg" preserveAspectRatio="none" viewBox="0 0 800 88"></svg>
    </div>
  </div>

  <div class="cols">
    <div class="card">
      <div class="card-h">Engines <span class="card-cnt" id="engCnt">0</span></div>
      <div class="tbl-scroll">
        <table><thead><tr><th>Project</th><th>Agent</th><th>Platform</th><th>Sessions</th><th>Status</th></tr></thead>
        <tbody id="tEng"></tbody></table>
      </div>
    </div>
    <div class="card">
      <div class="card-h">Activity <span class="card-cnt" id="actCnt">0</span></div>
      <div id="actBody"><div class="empty-act">No activity</div></div>
    </div>
  </div>
</main>

<footer>DigitalMe &middot; <span id="fVer">dev</span> &middot; <span id="fSys">-</span></footer>

<script>
var $=function(i){return document.getElementById(i)};
var F=function(u){return fetch(u).then(function(r){return r.json()})};
setInterval(function(){$('clock').textContent=new Date().toLocaleTimeString()},1000);

var sMap={healthy:['pill-ok','Online'],degraded:['pill-warn','Degraded'],unhealthy:['pill-err','Offline']};

function drawHB(arr){
  var svg=$('hbSvg');
  if(!arr||!arr.length){svg.innerHTML='';return}
  var W=800,H=88,px=4,n=arr.length,pts=[];
  for(var i=0;i<n;i++){
    var x=px+(W-px*2)*i/(n-1||1);
    var s=arr[i].status;
    var y=s==='healthy'?14:s==='degraded'?50:76;
    pts.push([x,y]);
  }
  var d='M'+pts[0][0]+','+pts[0][1];
  for(var i=1;i<pts.length;i++){
    var cp=(pts[i][0]-pts[i-1][0])*0.4;
    d+=' C'+(pts[i-1][0]+cp)+','+pts[i-1][1]+' '+(pts[i][0]-cp)+','+pts[i][1]+' '+pts[i][0]+','+pts[i][1];
  }
  var last=pts[pts.length-1];
  var area=d+' L'+last[0]+','+H+' L'+pts[0][0]+','+H+' Z';
  var c=arr[arr.length-1].status==='healthy'?'#2dd4bf':arr[arr.length-1].status==='degraded'?'#fbbf24':'#f87171';

  svg.innerHTML=
    '<defs><linearGradient id="hfill" x1="0" y1="0" x2="0" y2="1">'+
    '<stop offset="0%" stop-color="'+c+'" stop-opacity=".18"/>'+
    '<stop offset="100%" stop-color="'+c+'" stop-opacity="0"/>'+
    '</linearGradient></defs>'+
    '<path d="'+area+'" fill="url(#hfill)"/>'+
    '<path d="'+d+'" fill="none" stroke="'+c+'" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" opacity=".9"/>'+
    '<circle cx="'+last[0]+'" cy="'+last[1]+'" r="3.5" fill="'+c+'"/>'+
    '<circle cx="'+last[0]+'" cy="'+last[1]+'" r="8" fill="'+c+'" opacity=".12"/>';
}

function R(){
  Promise.all([F('/api/status'),F('/api/engines'),F('/api/heartbeat'),F('/api/activity')])
  .then(function(res){
    var s=res[0],en=res[1],hb=res[2],ac=res[3];
    var sm=sMap[s.status]||sMap.healthy;
    $('sPill').className='pill '+sm[0];
    $('sText').textContent=sm[1];

    $('kUp').textContent=s.uptime||'0m';
    $('kProj').textContent=s.projects;
    $('kVer').textContent=s.version||'dev';
    $('fVer').textContent=s.version||'dev';
    $('fSys').textContent=(s.os||'?')+'/'+(s.arch||'?');

    var tot=0;(en||[]).forEach(function(e){tot+=e.active_sessions||0});
    $('kSess').textContent=tot;

    var tb=$('tEng');tb.innerHTML='';
    $('engCnt').textContent=(en||[]).length;
    if(!en||!en.length){
      tb.innerHTML='<tr><td colspan="5" class="empty-cell">No engines</td></tr>';
    }else{
      function esc(s){var d=document.createElement('div');d.textContent=s;return d.innerHTML}
      en.forEach(function(e){
        var pl=(e.platforms||[]).map(function(p){return '<span class="tag t-zinc">'+esc(p)+'</span>'}).join(' ');
        var sc=e.status==='degraded'?'amber':e.status==='unhealthy'?'red':'green';
        tb.innerHTML+='<tr><td><strong>'+esc(e.name)+'</strong></td><td><span class="tag t-teal">'+esc(e.agent_type)+'</span></td><td>'+pl+'</td><td style="color:var(--text)">'+
          (e.active_sessions||0)+'</td><td><span class="tag t-'+sc+'">'+esc(e.status)+'</span></td></tr>';
      });
    }

    var ab=$('actBody');
    $('actCnt').textContent=(ac||[]).length;
    if(!ac||!ac.length){
      ab.innerHTML='<div class="empty-act">No activity</div>';
    }else{
      ab.innerHTML='';
      ac.sort(function(a,b){return new Date(b.last_activity)-new Date(a.last_activity)});
      ac.forEach(function(a){
        var t=new Date(a.last_activity).toLocaleTimeString();
        ab.innerHTML+='<div class="act"><span class="act-key">'+esc(a.session_key)+'</span><div class="act-r"><span class="act-time">'+esc(t)+'</span><span class="act-idle">idle '+esc(a.idle_str)+'</span></div></div>';
      });
    }

    var hArr=(hb||[]).slice(-80);
    $('hbMeta').textContent=hArr.length+' checks';
    drawHB(hArr);
  })["catch"](function(e){console.error(e)});
}
R();setInterval(R,6000);

// Brand integrity check — DigitalMe branding protection
(function(){
  var _b=[68,105,103,105,116,97,108,77,101]; // charCodes for "DigitalMe"
  var _n='';for(var i=0;i<_b.length;i++)_n+=String.fromCharCode(_b[i]);
  var _v=[80,111,119,101,114,101,100,32,98,121]; // "Powered by"
  var _p='';for(var i=0;i<_v.length;i++)_p+=String.fromCharCode(_v[i]);
  function _ck(){
    var f=document.querySelector('footer');
    if(!f)return _fail();
    var cs=window.getComputedStyle(f,'::before');
    var t=document.title;
    var h=document.querySelector('.logo');
    if(!h||h.textContent.indexOf(_n)<0)return _fail();
    if(t.indexOf(_n)<0)return _fail();
    if(f.textContent.indexOf(_n)<0)return _fail();
  }
  function _fail(){
    document.body.innerHTML='<div style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:#09090b;color:#f87171;font-family:monospace;font-size:18px;text-align:center;padding:40px"><div><h1 style="font-size:48px;margin-bottom:20px">&#9888;</h1><p>Brand integrity violation detected.</p><p style="color:#a1a1aa;font-size:13px;margin-top:12px">'+_p+' '+_n+' branding must not be removed.</p><p style="color:#a1a1aa;font-size:11px;margin-top:8px">This is required by the license agreement.</p></div></div>';
  }
  setInterval(_ck,5000);
  _ck();
})();
</script>
</body>
</html>`
