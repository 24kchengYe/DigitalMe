package core

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// HeartbeatStatus represents the overall health state.
type HeartbeatStatus string

const (
	StatusHealthy   HeartbeatStatus = "healthy"
	StatusDegraded  HeartbeatStatus = "degraded"
	StatusUnhealthy HeartbeatStatus = "unhealthy"
)

// HeartbeatRecord stores a single heartbeat check result.
type HeartbeatRecord struct {
	Timestamp      time.Time       `json:"timestamp"`
	Status         HeartbeatStatus `json:"status"`
	ActiveSessions int             `json:"active_sessions"`
	Uptime         string          `json:"uptime"`
}

// EngineStatus holds the current status of one engine.
type EngineStatus struct {
	Name           string          `json:"name"`
	Status         HeartbeatStatus `json:"status"`
	AgentType      string          `json:"agent_type"`
	Platforms      []string        `json:"platforms"`
	ActiveSessions int             `json:"active_sessions"`
	StartedAt      time.Time       `json:"started_at"`
	Uptime         string          `json:"uptime"`
}

// HeartbeatMonitor periodically checks engine health.
type HeartbeatMonitor struct {
	engines      map[string]*Engine
	interval     time.Duration
	history      []HeartbeatRecord
	mu           sync.RWMutex
	stopCh       chan struct{}
	maxHist      int
	idleReminder *IdleReminder
}

// NewHeartbeatMonitor creates a monitor with the given check interval.
func NewHeartbeatMonitor(interval time.Duration) *HeartbeatMonitor {
	return &HeartbeatMonitor{
		engines:  make(map[string]*Engine),
		interval: interval,
		history:  make([]HeartbeatRecord, 0, 100),
		stopCh:   make(chan struct{}),
		maxHist:  100,
	}
}

// RegisterEngine adds an engine to be monitored.
func (hb *HeartbeatMonitor) RegisterEngine(name string, e *Engine) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.engines[name] = e
}

// SetIdleReminder links the idle reminder so heartbeat can detect unresponsive sessions.
func (hb *HeartbeatMonitor) SetIdleReminder(ir *IdleReminder) {
	hb.mu.Lock()
	defer hb.mu.Unlock()
	hb.idleReminder = ir
}

// Start begins periodic health checks.
func (hb *HeartbeatMonitor) Start() {
	go func() {
		// Run immediately on start
		hb.check()
		ticker := time.NewTicker(hb.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hb.check()
			case <-hb.stopCh:
				return
			}
		}
	}()
	slog.Info("heartbeat monitor started", "interval", hb.interval)
}

// Stop halts the periodic checks.
func (hb *HeartbeatMonitor) Stop() {
	close(hb.stopCh)
}

func (hb *HeartbeatMonitor) check() {
	hb.mu.Lock()
	defer hb.mu.Unlock()

	totalSessions := 0
	overallStatus := StatusHealthy

	for _, e := range hb.engines {
		e.interactiveMu.Lock()
		count := len(e.interactiveStates)
		e.interactiveMu.Unlock()
		totalSessions += count
	}

	// If idle reminder has detected unresponsive sessions, degrade status
	if hb.idleReminder != nil {
		unresponsive := hb.idleReminder.UnresponsiveCount()
		if unresponsive > 0 {
			overallStatus = StatusDegraded
		}
	}

	record := HeartbeatRecord{
		Timestamp:      time.Now(),
		Status:         overallStatus,
		ActiveSessions: totalSessions,
		Uptime:         hb.uptimeStr(),
	}

	hb.history = append(hb.history, record)
	if len(hb.history) > hb.maxHist {
		hb.history = hb.history[len(hb.history)-hb.maxHist:]
	}
}

// History returns recent heartbeat records.
func (hb *HeartbeatMonitor) History() []HeartbeatRecord {
	hb.mu.RLock()
	defer hb.mu.RUnlock()
	out := make([]HeartbeatRecord, len(hb.history))
	copy(out, hb.history)
	return out
}

// Latest returns the most recent heartbeat, or nil.
func (hb *HeartbeatMonitor) Latest() *HeartbeatRecord {
	hb.mu.RLock()
	defer hb.mu.RUnlock()
	if len(hb.history) == 0 {
		return nil
	}
	r := hb.history[len(hb.history)-1]
	return &r
}

// EngineStatuses returns the current status of all engines.
func (hb *HeartbeatMonitor) EngineStatuses() []EngineStatus {
	hb.mu.RLock()
	defer hb.mu.RUnlock()

	var statuses []EngineStatus
	for name, e := range hb.engines {
		e.interactiveMu.Lock()
		activeSessions := len(e.interactiveStates)
		e.interactiveMu.Unlock()

		var platformNames []string
		for _, p := range e.platforms {
			platformNames = append(platformNames, p.Name())
		}

		uptime := time.Since(e.startedAt)

		statuses = append(statuses, EngineStatus{
			Name:           name,
			Status:         StatusHealthy,
			AgentType:      e.agent.Name(),
			Platforms:      platformNames,
			ActiveSessions: activeSessions,
			StartedAt:      e.startedAt,
			Uptime:         formatDuration(uptime),
		})
	}
	return statuses
}

// uptimeStr returns uptime based on earliest engine start. Caller must hold hb.mu.
func (hb *HeartbeatMonitor) uptimeStr() string {
	var earliest time.Time
	for _, e := range hb.engines {
		if earliest.IsZero() || e.startedAt.Before(earliest) {
			earliest = e.startedAt
		}
	}
	if earliest.IsZero() {
		return "0s"
	}
	return formatDuration(time.Since(earliest))
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
