// Package secret detects paths that likely contain credentials.
package secret

import (
	"path/filepath"
	"strings"
)

var nameHints = []string{
	"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa",
	".pem", ".p12", ".pfx", ".key",
	"credentials", "credential", "secret", "secrets",
	".netrc", "netrc", "auth.json", "token", "passwd",
	"password", "private_key", "private-key", "service-account",
	"kubeconfig", ".env", "env.local", "secrets.yaml", "secrets.yml",
	"secrets.json", "wallet", "mnemonic",
}

var pathHints = []string{
	"/.ssh/", "/gnupg/", "/.gnupg/", "/password-store/",
	"/secrets/", "/credentials/",
}

// Path reports whether abs looks secret-sensitive.
func Path(abs string) bool {
	if abs == "" {
		return false
	}
	norm := filepath.ToSlash(strings.ToLower(abs))
	base := strings.ToLower(filepath.Base(abs))

	// Public keys are fine.
	if strings.HasSuffix(base, ".pub") {
		return false
	}

	for _, h := range pathHints {
		if strings.Contains(norm, h) {
			return true
		}
	}
	for _, h := range nameHints {
		if base == h || strings.HasPrefix(base, h) || strings.HasSuffix(base, h) {
			return true
		}
		// multi-part names like "foo.secrets.json"
		if strings.Contains(base, h) && (strings.Contains(h, ".") || strings.Contains(h, "_") || len(h) >= 6) {
			return true
		}
	}
	if strings.HasPrefix(base, "id_") {
		return true
	}
	return false
}

// Label is a short UI marker.
func Label(abs string) string {
	if Path(abs) {
		return "🔒"
	}
	return ""
}
