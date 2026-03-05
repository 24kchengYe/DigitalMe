package licensing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Obfuscated signing key: actual = maskedKey XOR keyMask.
// This prevents extraction via `strings` on the compiled binary.
var (
	maskedKey = [32]byte{
		0x03, 0x38, 0x85, 0x61, 0x12, 0x6d, 0x86, 0xe8,
		0xcf, 0xa6, 0xc5, 0xb6, 0x80, 0x1f, 0x13, 0x86,
		0x1f, 0xf2, 0xf8, 0xc9, 0x0b, 0xe5, 0x22, 0x22,
		0x30, 0x15, 0x07, 0x76, 0x6c, 0xc8, 0x4e, 0xfa,
	}
	keyMask = [32]byte{
		0xd7, 0x63, 0x0f, 0x96, 0x31, 0xac, 0xe8, 0x75,
		0x80, 0x1e, 0xc7, 0x53, 0xfa, 0x29, 0xb2, 0x4d,
		0x06, 0x7f, 0x9c, 0x3b, 0xa5, 0xd2, 0x27, 0xea,
		0x61, 0xf3, 0xbe, 0x49, 0x1c, 0x84, 0xc0, 0x58,
	}
)

// recoverKey XORs maskedKey with keyMask to produce the actual HMAC signing key.
func recoverKey() []byte {
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		key[i] = maskedKey[i] ^ keyMask[i]
	}
	return key
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
// Key format: base64(jsonPayload + "." + hex(hmacSHA256(jsonPayload)))
func verifyKey(keyStr string) (*Payload, error) {
	// Try standard base64, then URL-safe
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

	rawStr := string(raw)
	lastDot := strings.LastIndex(rawStr, ".")
	if lastDot < 0 || lastDot >= len(rawStr)-1 {
		return nil, fmt.Errorf("invalid key format")
	}

	jsonData := []byte(rawStr[:lastDot])
	sigHex := rawStr[lastDot+1:]

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding")
	}

	// Verify HMAC-SHA256
	secret := recoverKey()
	mac := hmac.New(sha256.New, secret)
	mac.Write(jsonData)
	expected := mac.Sum(nil)

	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("invalid signature")
	}

	var payload Payload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	return &payload, nil
}

// signPayload creates an HMAC-SHA256 signature for the given JSON data.
func signPayload(jsonData, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(jsonData)
	return hex.EncodeToString(mac.Sum(nil))
}
