// Package info provides the build information of a command or module.
package info

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/tkrop/go-make/internal/coding"
)

const (
	// RepoPathSepNum is the number of the repository path separator.
	RepoPathSepNum = 3
	// DebugRevisionLen is the length of the debug revision.
	DebugRevisionLen = 12
)

// Regexp for semantic versioning as supported by go as tag.
var semVersionTagRegex = regexp.MustCompile(
	`v(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)` +
		`(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
		`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// Info provides the build information of a command or module.
type Info struct {
	// Path contains the package path of the command or module.
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
	// Repo contains the repository of the command or module.
	Repo string `yaml:"repo,omitempty" json:"repo,omitempty"`
	// Version contains the actual version of the command or module.
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	// Revision contains the revision of the command or module from version
	// control system.
	Revision string `yaml:"revision,omitempty" json:"revision,omitempty"`
	// Build contains the build time of the command or module.
	Build time.Time `yaml:"build,omitempty" json:"build,omitempty"`
	// Commit contains the last commit time of the command or module from the
	// version control system.
	Commit time.Time `yaml:"commit,omitempty" json:"commit,omitempty"`
	// Dirty flags whether the build of the command or module is based on a
	// dirty local repository state.
	Dirty bool `yaml:"dirty,omitempty" json:"dirty,omitempty"`
	// Checksum contains the check sum of the command or module.
	Checksum string `yaml:"checksum,omitempty" json:"checksum,omitempty"`

	// Go contains the go version the command or module was build with.
	Go string `yaml:"go,omitempty" json:"go,omitempty"`
	// Platform contains the build platform the command or module was build
	// for.
	Platform string `yaml:"platform,omitempty" json:"platform,omitempty"`
	// Compiler contains the actual compiler the command or module was build
	// with.
	Compiler string `yaml:"compiler,omitempty" json:"compiler,omitempty"`
}

// NewInfo returns the build information of a command or module using given
// custom version and custom build time using RFC3339 format. The provided
// version must follow semantic versioning as supported by go.
func NewInfo(path, version, revision, build, commit string, dirty bool) *Info {
	return (&Info{
		Path:     path,
		Version:  version,
		Revision: revision,
		Build:    timeRFC3339Parse(build),
		Commit:   timeRFC3339Parse(commit),
		Dirty:    dirty,
		Go:       runtime.Version()[2:],
		Compiler: runtime.Compiler,
		Platform: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}).UseDebug(debug.ReadBuildInfo()).AdjustVersion()
}

// UseDebug enriches the build information of a command or module using the
// given debug build information. If the debug build information is not
// available the build information is not changed.
func (info *Info) UseDebug(build *debug.BuildInfo, ok bool) *Info {
	if ok && build != nil {
		if info.Path == "" {
			info.Path = build.Main.Path
		}

		if semVersionTagRegex.MatchString(build.Main.Version) {
			info.Version = build.Main.Version
			index := strings.LastIndex(build.Main.Version, "-")
			info.Revision = build.Main.Version[index+1:]
		}

		info.Checksum = build.Main.Sum
		for _, kv := range build.Settings {
			switch kv.Key {
			case "vcs.revision":
				info.Revision = kv.Value
			case "vcs.time":
				info.Commit, _ = time.Parse(time.RFC3339, kv.Value)
			case "vcs.modified":
				info.Dirty = kv.Value == "true"
			}
		}
	}

	return info
}

// AdjustVersion adjusts the version of the build information of a command or
// module if the version does not follow semantic versioning as supported by
// go. The version is adjusted using the revision and commit time of the build
// information. If the revision is not available the version is not changed.
func (info *Info) AdjustVersion() *Info {
	if info.Path != "" {
		info.Repo = strings.Replace(splitRuneN(
			"git@"+info.Path, '/', RepoPathSepNum), "/", ":", 1)
	}

	if !semVersionTagRegex.MatchString(info.Version) {
		if info.Revision != "" && !info.Commit.Equal(time.Time{}) {
			revision := info.Revision
			if len(revision) > DebugRevisionLen {
				info.Revision = revision[0:DebugRevisionLen]
			}
			info.Version = fmt.Sprintf("v0.0.0-%s-%s",
				info.Commit.UTC().Format("20060102150405"), revision)
		}
	}

	return info
}

// String returns the build information of a command or module as structured
// JSON string.
func (info *Info) String() string {
	return coding.ToString(coding.TypeJSON, info)
}

// splitRuneN splits the string s at the `n`th occurrence of the rune ch.
func splitRuneN(s string, ch rune, n int) string {
	count := 0
	index := strings.IndexFunc(s, func(is rune) bool {
		if ch == is {
			count++
		}
		return count == n
	})

	if index >= 0 {
		return s[0:index]
	}
	return s
}

// timeRFC3339Parse parses the given time string using RFC3339 format
// swallowing errors.
func timeRFC3339Parse(t string) time.Time {
	time, _ := time.Parse(time.RFC3339, t)
	return time
}
