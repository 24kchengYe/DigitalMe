package licensing

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Ed25519 public key for license verification.
// This is SAFE to publish — it can only verify signatures, not create them.
// The corresponding private key is kept by the project owner.
var publicKey = ed25519.PublicKey{
	0x51, 0xbe, 0xa7, 0x9a, 0x27, 0x15, 0x53, 0x7c,
	0x08, 0x51, 0x96, 0x21, 0x0c, 0xbc, 0xbe, 0x6b,
	0x64, 0x1d, 0xb1, 0x29, 0xfd, 0x8e, 0x78, 0x3b,
	0xa2, 0x75, 0x68, 0x2c, 0xfa, 0xf2, 0x1d, 0xef,
}

// Payload represents the license data embedded in a key.
type Payload struct {
	Licensee string   `json:"licensee"`
	Email    string   `json:"email"`
	Tier     Tier     `json:"tier"`
	Expires  string   `json:"expires,omitempty"` // YYYY-MM-DD format
	Features []string `json:"features,omitempty"`
}

// IsExpired checks if the license has passed its expiration date.
func (p *Payload) IsExpired() bool {
	if p.Expires == "" {
		return false // no expiry = perpetual
	}
	t, err := time.Parse("2006-01-02", p.Expires)
	if err != nil {
		t, err = time.Parse(time.RFC3339, p.Expires)
		if err != nil {
			return true // unparseable = expired
		}
	}
	// Grace: license is valid through end of expiry day
	return time.Now().After(t.Add(24 * time.Hour))
}

// verifyKey validates a license key string and returns the decoded payload.
// Key format: base64(jsonPayload + "." + base64(ed25519Signature))
func verifyKey(keyStr string) (*Payload, error) {
	// Decode outer base64
	raw, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(keyStr)
		if err != nil {
			raw, err = base64.URLEncoding.DecodeString(keyStr)
			if err != nil {
				raw, err = base64.RawURLEncoding.DecodeString(keyStr)
				if err != nil {
					return nil, fmt.Errorf("invalid key encoding")
				}
			}
		}
	}

	// Split at last "." — JSON payload may contain dots (emails, dates)
	rawStr := string(raw)
	lastDot := strings.LastIndex(rawStr, ".")
	if lastDot < 0 || lastDot >= len(rawStr)-1 {
		return nil, fmt.Errorf("invalid key format")
	}

	jsonData := []byte(rawStr[:lastDot])
	sigB64 := rawStr[lastDot+1:]

	// Decode signature from base64
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding")
	}

	// Verify Ed25519 signature with embedded public key
	if !ed25519.Verify(publicKey, jsonData, sig) {
		return nil, fmt.Errorf("invalid signature")
	}

	var payload Payload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	return &payload, nil
}
