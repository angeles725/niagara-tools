package readtools

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The server block of n4_describe_session: which binary serves, its build, when it started, and
// whether a newer binary replaced it on disk since (new tools reach a client only after a restart).

// unknownBuild is the build of a binary without a VCS stamp (a test binary, a build outside git).
const unknownBuild = "unknown"

// warnNewerBinary is the warning when the binary on disk changed after the server started.
const warnNewerBinary = "the binary on disk (%s) was modified at %s, after this server started at %s: restart the MCP server to serve it"

// startup is what the process knew of itself when it started.
var startup = readStartup()

type startupInfo struct {
	at      time.Time
	binary  string // "" when the executable cannot be named
	modTime time.Time
	build   string
	builtAt string
}

func readStartup() startupInfo {
	s := startupInfo{at: time.Now(), build: unknownBuild}
	if exe, err := os.Executable(); err == nil {
		s.binary = exe
		if fi, err := os.Stat(exe); err == nil {
			s.modTime, s.builtAt = fi.ModTime(), stamp(fi.ModTime())
		}
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if build, builtAt := vcsStamp(info.Settings); build != "" {
			s.build = build
			if builtAt != "" {
				s.builtAt = builtAt
			}
		}
	}
	return s
}

// vcsStamp is the revision go build stamped (with +modified for a dirty tree) and its commit
// time; "" for a binary built without VCS information.
func vcsStamp(settings []debug.BuildSetting) (build, builtAt string) {
	vcs := map[string]string{}
	for _, kv := range settings {
		vcs[kv.Key] = kv.Value
	}
	if build = vcs["vcs.revision"]; build == "" {
		return "", ""
	}
	if vcs["vcs.modified"] == "true" {
		build += "+modified"
	}
	return build, vcs["vcs.time"]
}

// serverBlock is the server block and, when the binary on disk is newer than the server, the
// warning that says to restart it ("" otherwise).
func serverBlock() (*pyjson.Object, string) {
	s := startup
	out := obj("build", s.build, "built_at", nilIfEmpty(s.builtAt), "started_at", stamp(s.at),
		"binary", nilIfEmpty(s.binary))
	if s.binary == "" {
		return out, ""
	}
	fi, err := os.Stat(s.binary)
	if err != nil || !fi.ModTime().After(s.at) {
		return out, ""
	}
	return out, fmt.Sprintf(warnNewerBinary, s.binary, stamp(fi.ModTime()), stamp(s.at))
}

// stamp is t in UTC, to the second, RFC 3339.
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
