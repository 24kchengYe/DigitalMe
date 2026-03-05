package licensing

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Tier represents a license tier.
type Tier string

const (
	TierFree Tier = "free"
	TierPro  Tier = "pro"
)

// Feature constants for gating.
const (
	FeatureVoice        = "voice"
	FeatureFileSendback = "file_sendback"
	FeatureCron         = "cron"
	FeatureWebDashboard = "web_dashboard"
	FeatureScreenshot   = "screenshot"
	FeatureTaskNotify   = "task_notify"
)

// proFeatures lists all features that require a Pro license.
var proFeatures = map[string]bool{
	FeatureVoice:        true,
	FeatureFileSendback: true,
	FeatureCron:         true,
	FeatureWebDashboard: true,
	FeatureScreenshot:   true,
	FeatureTaskNotify:   true,
}

// Manager is the global license manager singleton.
type Manager struct {
	mu      sync.RWMutex
	tier    Tier
	payload *Payload
	keyStr  string // raw license key for re-verification
	msgCounter atomic.Int64
}

var mgr = &Manager{tier: TierFree}

// Init validates the license key and sets the global tier.
// Call this at startup with the key from config.
func Init(keyStr string) error {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	mgr.keyStr = keyStr

	if keyStr == "" {
		mgr.tier = TierFree
		mgr.payload = nil
		slog.Info("license: no key configured, running in Free tier")
		return nil
	}

	payload, err := verifyKey(keyStr)
	if err != nil {
		mgr.tier = TierFree
		mgr.payload = nil
		slog.Warn("license: invalid key, falling back to Free tier", "error", err)
		return fmt.Errorf("license verification failed: %w", err)
	}

	if payload.IsExpired() {
		mgr.tier = TierFree
		mgr.payload = payload
		slog.Warn("license: key expired, falling back to Free tier",
			"licensee", payload.Licensee, "expires", payload.Expires)
		return nil
	}

	mgr.tier = payload.Tier
	mgr.payload = payload
	slog.Info("license: verified",
		"tier", payload.Tier,
		"licensee", payload.Licensee,
		"expires", payload.Expires,
	)
	return nil
}

// IsFeatureEnabled checks whether a feature is available under the current license.
func IsFeatureEnabled(feature string) bool {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if !proFeatures[feature] {
		return true // non-pro feature, always enabled
	}

	if mgr.tier == TierPro {
		// Check if feature is in the payload's feature list (if specified)
		if mgr.payload != nil && len(mgr.payload.Features) > 0 {
			for _, f := range mgr.payload.Features {
				if f == feature {
					return true
				}
			}
			return false
		}
		return true // Pro with no feature restrictions
	}

	return false
}

// GetTier returns the current license tier.
func GetTier() Tier {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.tier
}

// GetPayload returns the current license payload (may be nil for free tier).
func GetPayload() *Payload {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	return mgr.payload
}

// IncrementMessageCount increments the global message counter and returns the new count.
func IncrementMessageCount() int64 {
	return mgr.msgCounter.Add(1)
}

// StartPeriodicCheck re-verifies the license key at the given interval.
// This catches clock tampering and key revocation.
func StartPeriodicCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			mgr.mu.RLock()
			key := mgr.keyStr
			mgr.mu.RUnlock()

			if key == "" {
				continue
			}

			payload, err := verifyKey(key)
			if err != nil {
				slog.Warn("license: periodic check failed, downgrading to Free", "error", err)
				mgr.mu.Lock()
				mgr.tier = TierFree
				mgr.mu.Unlock()
				continue
			}

			if payload.IsExpired() {
				slog.Warn("license: key expired during periodic check, downgrading to Free")
				mgr.mu.Lock()
				mgr.tier = TierFree
				mgr.payload = payload
				mgr.mu.Unlock()
				continue
			}

			mgr.mu.Lock()
			mgr.tier = payload.Tier
			mgr.payload = payload
			mgr.mu.Unlock()
		}
	}()
}

// StatusString returns a human-readable license status for display.
func StatusString() string {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()

	if mgr.payload == nil {
		return "Free tier (no license key)"
	}

	status := fmt.Sprintf("Licensed to: %s | Tier: %s", mgr.payload.Licensee, mgr.tier)
	if mgr.payload.Expires != "" {
		if mgr.payload.IsExpired() {
			status += " | EXPIRED"
		} else {
			status += fmt.Sprintf(" | Expires: %s", mgr.payload.Expires)
		}
	}
	return status
}
