package validators

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrInvalidDomain   = errors.New("invalid domain name")
	ErrInvalidPath     = errors.New("invalid path")
	ErrInvalidURL      = errors.New("invalid URL")
	ErrInvalidRealm    = errors.New("invalid auth realm")
	ErrDangerousChars  = errors.New("dangerous characters detected")
)

// Domain validation: RFC 1123 compliant hostname
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// Path validation: URL path characters only, no nginx injection chars
var pathRegex = regexp.MustCompile(`^/[a-zA-Z0-9/_\-\.]*$`)

// Auth realm: alphanumeric, spaces, basic punctuation (no quotes, semicolons, newlines)
var realmRegex = regexp.MustCompile(`^[a-zA-Z0-9 _\-\.]+$`)

// Dangerous characters for nginx config
var dangerousChars = []string{";", "\n", "\r", "'", "\"", "`", "$", "{", "}", "\\"}

// ValidateDomain validates a domain name for nginx config
func ValidateDomain(domain string) error {
	if domain == "" {
		return ErrInvalidDomain
	}

	// Check length
	if len(domain) > 253 {
		return ErrInvalidDomain
	}

	// Check for dangerous chars
	if containsDangerousChars(domain) {
		return ErrDangerousChars
	}

	// Validate format
	if !domainRegex.MatchString(domain) {
		return ErrInvalidDomain
	}

	return nil
}

// ValidatePath validates a URL path for nginx config
func ValidatePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	// Must start with /
	if !strings.HasPrefix(path, "/") {
		return ErrInvalidPath
	}

	// Check length
	if len(path) > 2048 {
		return ErrInvalidPath
	}

	// Check for dangerous chars
	if containsDangerousChars(path) {
		return ErrDangerousChars
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return ErrInvalidPath
	}

	// Validate format
	if !pathRegex.MatchString(path) {
		return ErrInvalidPath
	}

	return nil
}

// ValidateRedirectURL validates a redirect target URL
func ValidateRedirectURL(rawURL string) error {
	if rawURL == "" {
		return ErrInvalidURL
	}

	// Check length
	if len(rawURL) > 2048 {
		return ErrInvalidURL
	}

	// Check for nginx injection chars (except $ which is used in nginx vars)
	for _, char := range []string{";", "\n", "\r", "'", "\"", "`", "{", "}"} {
		if strings.Contains(rawURL, char) {
			return ErrDangerousChars
		}
	}

	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	// Reject dangerous schemes (XSS vectors)
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "" && scheme != "http" && scheme != "https" {
		return ErrInvalidURL
	}

	return nil
}

// ValidateAuthRealm validates an HTTP Basic Auth realm string
func ValidateAuthRealm(realm string) error {
	if realm == "" {
		return ErrInvalidRealm
	}

	// Check length
	if len(realm) > 128 {
		return ErrInvalidRealm
	}

	// Check for dangerous chars
	if containsDangerousChars(realm) {
		return ErrDangerousChars
	}

	// Validate format
	if !realmRegex.MatchString(realm) {
		return ErrInvalidRealm
	}

	return nil
}

// containsDangerousChars checks for nginx config injection characters
func containsDangerousChars(s string) bool {
	for _, char := range dangerousChars {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}
