package version

import "runtime/debug"

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
