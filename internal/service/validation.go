package service

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// --- Port validation ---

// ValidatePort checks that a port number is in the legal TCP range.
// It does NOT check whether the port is currently in use; that is the
// caller's responsibility (e.g. via net.Listen probe).
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

// --- MTProxy secret validation ---

// mtproxySecretRe matches the two valid MTProxy secret formats:
//
//	plain:   32 lowercase hex characters          e.g. "deadbeef..."
//	faketls: "ee" + 30 hex + domain (≤ 253 chars) e.g. "ee<hex><domain>"
//
// We accept both here; the proxy process will reject invalid secrets at
// startup, but catching them early gives a better UX.
var (
	plainSecretRe   = regexp.MustCompile(`^[0-9a-f]{32}$`)
	fakeTLSSecretRe = regexp.MustCompile(`^ee[0-9a-f]{30}[a-zA-Z0-9._-]+$`)
)

// ValidateMTProxySecret validates a proxy secret.
// Returns a descriptive error if invalid.
func ValidateMTProxySecret(secret string) error {
	if secret == "" {
		return errors.New("secret must not be empty")
	}
	s := strings.ToLower(secret)
	if plainSecretRe.MatchString(s) {
		return nil
	}
	if fakeTLSSecretRe.MatchString(s) {
		return nil
	}
	return errors.New("secret must be 32 hex chars (plain) or 'ee' + 30 hex + domain (FakeTLS)")
}

// --- Proxy link label sanitization ---

// maxLabelLen is the maximum permitted length for a proxy link label.
const maxLabelLen = 64

// SanitizeLabel trims whitespace and rejects labels that contain control
// characters or characters that could break URI encoding.
// Returns the sanitized label and an error if the label is unacceptable.
func SanitizeLabel(label string) (string, error) {
	label = strings.TrimSpace(label)

	if label == "" {
		return "", errors.New("label must not be empty")
	}
	if len(label) > maxLabelLen {
		return "", errors.New("label must not exceed 64 characters")
	}

	for _, r := range label {
		if unicode.IsControl(r) {
			return "", errors.New("label must not contain control characters")
		}
		// Reject characters that are problematic in a tg:// URI query param.
		if r == '#' || r == '&' || r == '=' || r == '?' || r == '\\' {
			return "", errors.New("label contains disallowed character: " + string(r))
		}
	}

	return label, nil
}

// --- Hostname / IP validation ---

// ValidateHost checks that a host string is a non-empty domain name or IP
// address. It does not resolve DNS.
var hostRe = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$|^(\d{1,3}\.){3}\d{1,3}$`)

func ValidateHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return errors.New("host must not be empty")
	}
	if len(host) > 253 {
		return errors.New("host too long")
	}
	if !hostRe.MatchString(host) {
		return errors.New("host must be a valid domain name or IP address")
	}
	return nil
}

// --- Command injection prevention ---

// SafeArgs validates that a slice of command-line arguments contains no
// shell meta-characters. Use this as a last-resort sanity check before
// passing args to exec.Command. The primary defence is always to use
// exec.Command(binary, args...) without shell=true — this function is a
// belt-and-suspenders layer.
func SafeArgs(args []string) error {
	// Characters that are dangerous in a shell context. We reject them even
	// though exec.Command does not invoke a shell, because they may indicate
	// a confused operator or a misconfigured config value.
	const dangerous = ";|&$`><!\\"
	for _, arg := range args {
		for _, c := range dangerous {
			if strings.ContainsRune(arg, c) {
				return errors.New("argument contains disallowed shell character: " + string(c))
			}
		}
	}
	return nil
}
