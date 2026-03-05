package licensing

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// GenerateKey creates a signed license key from the given payload.
// The signing secret is provided as a hex-encoded string.
// If secretHex is empty, it falls back to the DIGITALME_SIGN_KEY env var.
func GenerateKey(payload Payload, secretHex string) (string, error) {
	if secretHex == "" {
		secretHex = os.Getenv("DIGITALME_SIGN_KEY")
	}
	if secretHex == "" {
		return "", fmt.Errorf("signing key required: set DIGITALME_SIGN_KEY env var or pass --secret flag")
	}

	secret, err := hex.DecodeString(secretHex)
	if err != nil {
		return "", fmt.Errorf("invalid signing key (must be hex): %w", err)
	}
	if len(secret) != 32 {
		return "", fmt.Errorf("signing key must be 32 bytes (64 hex chars), got %d bytes", len(secret))
	}

	// Validate tier
	if payload.Tier != TierFree && payload.Tier != TierPro {
		return "", fmt.Errorf("invalid tier %q (must be 'free' or 'pro')", payload.Tier)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	sig := signPayload(jsonData, secret)
	raw := string(jsonData) + "." + sig
	key := base64.StdEncoding.EncodeToString([]byte(raw))

	return key, nil
}

// VerifyKeyString validates a license key and returns its payload.
// This is a public wrapper around verifyKey for CLI use.
func VerifyKeyString(keyStr string) (*Payload, error) {
	return verifyKey(keyStr)
}
