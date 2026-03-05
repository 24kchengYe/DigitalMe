package licensing

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// GenerateKey creates a signed license key from the given payload.
// The Ed25519 private key is provided as a hex-encoded string (128 hex chars = 64 bytes).
// If privateKeyHex is empty, it falls back to the DIGITALME_SIGN_KEY env var.
//
// Security: Only the project owner has the private key.
// The public key embedded in key.go can only VERIFY, not create signatures.
func GenerateKey(payload Payload, privateKeyHex string) (string, error) {
	if privateKeyHex == "" {
		privateKeyHex = os.Getenv("DIGITALME_SIGN_KEY")
	}
	if privateKeyHex == "" {
		return "", fmt.Errorf("signing key required: set DIGITALME_SIGN_KEY env var or pass --secret flag")
	}

	privBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid signing key (must be hex): %w", err)
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("signing key must be %d bytes (%d hex chars), got %d bytes",
			ed25519.PrivateKeySize, ed25519.PrivateKeySize*2, len(privBytes))
	}

	privateKey := ed25519.PrivateKey(privBytes)

	// Validate tier
	if payload.Tier != TierFree && payload.Tier != TierPro {
		return "", fmt.Errorf("invalid tier %q (must be 'free' or 'pro')", payload.Tier)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Sign with Ed25519 private key
	sig := ed25519.Sign(privateKey, jsonData)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	// Format: base64(json + "." + base64url(signature))
	raw := string(jsonData) + "." + sigB64
	key := base64.StdEncoding.EncodeToString([]byte(raw))

	return key, nil
}

// VerifyKeyString validates a license key and returns its payload.
// This is a public wrapper around verifyKey for CLI use.
func VerifyKeyString(keyStr string) (*Payload, error) {
	return verifyKey(keyStr)
}
