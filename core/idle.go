package core

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// IdleReminder sends proactive messages when sessions are idle for too long.
type IdleReminder struct {
	engines       map[string]*Engine
	idleThreshold time.Duration
	checkInterval time.Duration
	lastActivity  map[string]time.Time // sessionKey → last message time
	reminded      map[string]bool      // sessionKey → already reminded
	mu            sync.Mutex
	stopCh        chan struct{}
}

// NewIdleReminder creates an idle reminder.
// idleMinutes: how many minutes of inactivity before sending a reminder.
func NewIdleReminder(idleMinutes int) *IdleReminder {
	return &IdleReminder{
		engines:       make(map[string]*Engine),
		idleThreshold: time.Duration(idleMinutes) * time.Minute,
		checkInterval: 1 * time.Minute,
		lastActivity:  make(map[string]time.Time),
		reminded:      make(map[string]bool),
		stopCh:        make(chan struct{}),
	}
}

// RegisterEngine adds an engine to monitor.
func (ir *IdleReminder) RegisterEngine(name string, e *Engine) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.engines[name] = e
}

// RecordActivity marks a session as active (called when a message is received).
func (ir *IdleReminder) RecordActivity(sessionKey string) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.lastActivity[sessionKey] = time.Now()
	ir.reminded[sessionKey] = false
}

// Start begins the idle check loop.
func (ir *IdleReminder) Start() {
	go func() {
		ticker := time.NewTicker(ir.checkInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ir.check()
			case <-ir.stopCh:
				return
			}
		}
	}()
	slog.Info("idle reminder started", "threshold", ir.idleThreshold)
}

// Stop halts the idle reminder.
func (ir *IdleReminder) Stop() {
	close(ir.stopCh)
}

func (ir *IdleReminder) check() {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	now := time.Now()
	for sessionKey, lastTime := range ir.lastActivity {
		if ir.reminded[sessionKey] {
			continue
		}
		if now.Sub(lastTime) < ir.idleThreshold {
			continue
		}

		// Session is idle, send reminder
		ir.reminded[sessionKey] = true
		go ir.sendReminder(sessionKey)
	}
}

func (ir *IdleReminder) sendReminder(sessionKey string) {
	ir.mu.Lock()
	engines := make(map[string]*Engine, len(ir.engines))
	for k, v := range ir.engines {
		engines[k] = v
	}
	ir.mu.Unlock()

	msg := "💤 检测到会话空闲中。需要我做什么吗？\n\n发送任意消息继续对话，或发送 /clear 开始新会话。"

	for _, e := range engines {
		err := e.SendToSession(sessionKey, msg)
		if err == nil {
			slog.Info("idle reminder sent", "session_key", sessionKey)
			return
		}
	}

	// If SendToSession failed (no active interactive state), try via platform
	for _, e := range engines {
		for _, p := range e.platforms {
			rc, ok := p.(ReplyContextReconstructor)
			if !ok {
				continue
			}
			replyCtx, err := rc.ReconstructReplyCtx(sessionKey)
			if err != nil {
				continue
			}
			if err := p.Send(context.Background(), replyCtx, msg); err == nil {
				slog.Info("idle reminder sent via platform", "session_key", sessionKey, "platform", p.Name())
				return
			}
		}
	}

	slog.Debug("idle reminder: no way to reach session", "session_key", sessionKey)
}

// GetLastActivity returns the last activity time for a session.
func (ir *IdleReminder) GetLastActivity(sessionKey string) (time.Time, bool) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	t, ok := ir.lastActivity[sessionKey]
	return t, ok
}

// AllActivity returns a copy of all last activity times.
func (ir *IdleReminder) AllActivity() map[string]time.Time {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	out := make(map[string]time.Time, len(ir.lastActivity))
	for k, v := range ir.lastActivity {
		out[k] = v
	}
	return out
}
