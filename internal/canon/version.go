package canon

import "runtime/debug"

// Version is the canon build version, normally injected at build time with
// -ldflags "-X github.com/victorhsb/canon/internal/canon.Version=vX.Y.Z" (see
// scripts/install.sh). When empty, the version is derived from Go build info.
var Version = ""

// versionString resolves the reported version: the ldflags-injected value
// when present, then the module version for `go install pkg@version` builds,
// then the VCS revision stamped by module-aware builds, and finally "dev".
func versionString() string {
	if Version != "" {
		return Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	var revision string
	dirty := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			dirty = setting.Value == "true"
		}
	}
	if revision == "" {
		return "dev"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	version := "dev-" + revision
	if dirty {
		version += "-dirty"
	}
	return version
}
