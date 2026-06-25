package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

// Spec is the Agent CLI Guidelines version this tool conforms to (declared in `schema`).
const Spec = "0.4.0"

// version is a plain literal so -ldflags "-X .../version.version=vX" can override it.
// It MUST NOT be initialized from a function call (golang/go#64246).
var version = "dev"

// String returns the build version, falling back to VCS build info for `go install`
// (which does not run ldflags).
func String() string {
	if version != "dev" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return v
		}
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" {
				return s.Value
			}
		}
	}
	return version
}

// repoSlug parses "owner/repo" from the module path (github.com/owner/repo), or "" if
// it can't be determined.
func repoSlug() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if rest, found := strings.CutPrefix(bi.Main.Path, "github.com/"); found {
			parts := strings.Split(rest, "/")
			if len(parts) >= 2 {
				return parts[0] + "/" + parts[1]
			}
		}
	}
	return ""
}

// UpgradeHint returns the recommended upgrade command for this tool. The main package lives at
// ./cmd/<tool> (the scaffold layout), so the install path includes /cmd/<tool> — a bare
// module@latest would fail for tools with no root main.
func UpgradeHint() string {
	slug := repoSlug()
	if slug == "" {
		return ""
	}
	repo := slug[strings.LastIndex(slug, "/")+1:]
	return "go install github.com/" + slug + "/cmd/" + repo + "@latest"
}

// safeReleaseURL allows a *_RELEASES_URL override only over https (any host) or http to
// localhost (for tests). A misconfigured/hostile env var (file://, http://169.254.169.254, …)
// is ignored — version --check falls back to the default — so the override can't be used for
// SSRF or local-file reads. Returns "" for empty/disallowed input.
func safeReleaseURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := neturl.Parse(raw)
	if err != nil {
		return ""
	}
	switch u.Scheme {
	case "https":
		return raw
	case "http":
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return raw
		}
	}
	return ""
}

// Latest returns the latest released version tag (from GitHub Releases by default).
// Network, short timeout, **fail-silent**: returns ("", err) on any problem so a
// `version --check` never errors or blocks an agent loop. The release source can be
// overridden with VABC_RELEASES_URL (used in tests); a disallowed value is ignored.
func Latest(ctx context.Context) (string, error) {
	url := safeReleaseURL(os.Getenv("VABC_RELEASES_URL"))
	if url == "" {
		slug := repoSlug()
		if slug == "" {
			return "", fmt.Errorf("unknown repository")
		}
		url = "https://api.github.com/repos/" + slug + "/releases/latest"
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "vabc-version-check") // GitHub's REST API rejects requests with no UA
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release source: %s", resp.Status)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.TagName, nil
}

// UpdateAvailable reports whether latest is a different release than current.
// Dev/source builds (current == "dev") never report an update — don't nag them.
func UpdateAvailable(latest, current string) bool {
	if latest == "" || current == "" || current == "dev" {
		return false
	}
	return strings.TrimPrefix(latest, "v") != strings.TrimPrefix(current, "v")
}
