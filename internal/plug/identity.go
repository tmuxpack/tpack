package plug

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

const (
	defaultGitHost  = "github.com"
	maxDirNameBytes = 64
	hashChars       = 12
)

// NormalizeIdentity returns a canonical host and repository path for spec.
func NormalizeIdentity(spec string) (string, error) {
	if containsControl(spec) {
		return "", fmt.Errorf("repository spec contains control characters")
	}
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", fmt.Errorf("repository spec is empty")
	}

	if strings.Contains(spec, "://") {
		u, err := url.Parse(spec)
		if err != nil || u.Hostname() == "" {
			return "", fmt.Errorf("invalid repository URL %q", spec)
		}
		host := strings.ToLower(u.Hostname())
		if port := u.Port(); port != "" {
			host += ":" + port
		}
		return normalizeHostPath(host, u.Path)
	}
	if host, repoPath, ok := parseSCPIdentity(spec); ok {
		return normalizeHostPath(host, repoPath)
	}
	return normalizeHostPath(defaultGitHost, spec)
}

func normalizeHostPath(host, repoPath string) (string, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	repoPath = strings.Trim(strings.TrimSpace(repoPath), "/")
	repoPath = strings.TrimSuffix(repoPath, ".git")
	if host == "" || repoPath == "" || repoPath == "." {
		return "", fmt.Errorf("repository identity requires host and path")
	}
	if err := validateRepositoryHost(host); err != nil {
		return "", err
	}
	if err := validateRepositoryPath(repoPath); err != nil {
		return "", err
	}
	return host + "/" + repoPath, nil
}

func parseSCPIdentity(spec string) (host, repoPath string, ok bool) {
	colon := strings.IndexByte(spec, ':')
	if colon <= 0 || colon == len(spec)-1 {
		return "", "", false
	}
	host = spec[:colon]
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		host = host[at+1:]
	}
	if host == "" || strings.ContainsAny(host, `/\`) {
		return "", "", false
	}
	return host, spec[colon+1:], true
}

func validateRepositoryHost(host string) error {
	hostname := host
	if name, port, err := strings.Cut(host, ":"); err && name != "" && port != "" {
		if _, convErr := strconv.ParseUint(port, 10, 16); convErr != nil {
			return fmt.Errorf("invalid repository host %q", host)
		}
		hostname = name
	}
	if hostname == "" || strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return fmt.Errorf("invalid repository host %q", host)
	}
	for _, r := range hostname {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '.' && r != '-' {
			return fmt.Errorf("invalid repository host %q", host)
		}
	}
	return nil
}

func validateRepositoryPath(repoPath string) error {
	if containsControl(repoPath) || strings.ContainsFunc(repoPath, unicode.IsSpace) ||
		strings.ContainsAny(repoPath, `\'";`+"`$|&<>") {
		return fmt.Errorf("invalid repository path %q", repoPath)
	}
	for segment := range strings.SplitSeq(repoPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("invalid repository path %q", repoPath)
		}
	}
	return nil
}

func containsControl(value string) bool {
	return strings.ContainsFunc(value, unicode.IsControl)
}

// RepositoryName returns the owner and repository for GitHub identities and
// preserves the host for identities from other forges.
func RepositoryName(identity string) string {
	if name, ok := strings.CutPrefix(identity, defaultGitHost+"/"); ok {
		return name
	}
	return identity
}

// LegacyPluginName extracts the basename used by existing plugin paths.
func LegacyPluginName(spec string) string {
	return strings.TrimSuffix(filepath.Base(spec), ".git")
}

// GeneratedDirName returns a filesystem-safe, bounded name derived from the
// repository basename and full identity.
func GeneratedDirName(identity string) string {
	base := sanitizeDirPrefix(path.Base(identity))
	if base == "" {
		base = "plugin"
	}
	sum := sha256.Sum256([]byte(identity))
	suffix := "-" + hex.EncodeToString(sum[:])[:hashChars]
	base = base[:min(len(base), maxDirNameBytes-len(suffix))]
	base = strings.TrimRight(base, ".-_")
	if base == "" {
		base = "plugin"
	}
	return base + suffix
}

func sanitizeDirPrefix(value string) string {
	var b strings.Builder
	separator := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			separator = false
		} else if !separator {
			b.WriteByte('-')
			separator = true
		}
	}
	return strings.Trim(b.String(), ".-_")
}
