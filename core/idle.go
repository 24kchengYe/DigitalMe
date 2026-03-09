package core

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// IdleReminder sends proactive messages when sessions are idle for too long.
// It repeats the reminder every idleThreshold interval until the user responds.
type IdleReminder struct {
	engines       map[string]*Engine
	idleThreshold time.Duration
	checkInterval time.Duration
	lastActivity  map[string]time.Time // sessionKey → last user message time
	lastReminded  map[string]time.Time // sessionKey → last reminder sent time
	mu            sync.Mutex
	stopCh        chan struct{}
}

// NewIdleReminder creates an idle reminder.
func NewIdleReminder(idleMinutes int) *IdleReminder {
	return &IdleReminder{
		engines:       make(map[string]*Engine),
		idleThreshold: time.Duration(idleMinutes) * time.Minute,
		checkInterval: 30 * time.Second,
		lastActivity:  make(map[string]time.Time),
		lastReminded:  make(map[string]time.Time),
		stopCh:        make(chan struct{}),
	}
}

func (ir *IdleReminder) RegisterEngine(name string, e *Engine) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.engines[name] = e
}

// RecordActivity marks a session as active (called on every incoming message).
func (ir *IdleReminder) RecordActivity(sessionKey string) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	ir.lastActivity[sessionKey] = time.Now()
	delete(ir.lastReminded, sessionKey) // reset reminder on new activity
}

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

func (ir *IdleReminder) Stop() {
	close(ir.stopCh)
}

const idleEvictAfter = 24 * time.Hour

func (ir *IdleReminder) check() {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	now := time.Now()

	// Evict sessions idle for more than 24 hours to prevent map growth
	for sessionKey, lastTime := range ir.lastActivity {
		if now.Sub(lastTime) > idleEvictAfter {
			delete(ir.lastActivity, sessionKey)
			delete(ir.lastReminded, sessionKey)
		}
	}

	for sessionKey, lastTime := range ir.lastActivity {
		idle := now.Sub(lastTime)
		if idle < ir.idleThreshold {
			continue
		}

		// Check if we already reminded recently (within one threshold period)
		if lr, ok := ir.lastReminded[sessionKey]; ok {
			if now.Sub(lr) < ir.idleThreshold {
				continue
			}
		}

		ir.lastReminded[sessionKey] = now
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

	msg := "💤 会话已空闲，随时可以发消息继续对话。\n\n发送 /clear 可开始新会话。"

	for _, e := range engines {
		if err := e.SendToSession(sessionKey, msg); err == nil {
			slog.Info("idle reminder sent", "session_key", sessionKey)
			return
		}
	}

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

	slog.Debug("idle reminder: could not reach session", "session_key", sessionKey)
}

func (ir *IdleReminder) GetLastActivity(sessionKey string) (time.Time, bool) {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	t, ok := ir.lastActivity[sessionKey]
	return t, ok
}

func (ir *IdleReminder) AllActivity() map[string]time.Time {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	out := make(map[string]time.Time, len(ir.lastActivity))
	for k, v := range ir.lastActivity {
		out[k] = v
	}
	return out
}

// UnresponsiveCount returns how many sessions have been reminded but haven't
// responded yet (lastReminded exists and is after lastActivity).
func (ir *IdleReminder) UnresponsiveCount() int {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	count := 0
	for sessionKey, remindedAt := range ir.lastReminded {
		if activityAt, ok := ir.lastActivity[sessionKey]; ok {
			if remindedAt.After(activityAt) {
				count++
			}
		}
	}
	return count
}
