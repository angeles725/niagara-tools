package readtools

import (
	"debug/buildinfo"
	"fmt"
	"os"
)

// The stale-binary warning n4_connect and n4_describe_session share (retro row P5): new tools
// reach a client only after the binary is rebuilt and its MCP server restarted.

// warnOtherBuild is the warning when the binary on disk carries another build id than the server.
const warnOtherBuild = "the binary on disk (%s) is build %s, but this server runs build %s: restart the MCP server to serve it"

// warnNewerBinary is the warning when the binary on disk changed after the server started.
const warnNewerBinary = "the binary on disk (%s) was modified at %s, after this server started at %s: restart the MCP server to serve it"

// staleBinaryWarning is the one-line warning that the binary on disk is not the one serving, or
// "": the build ids decide when both are readable, else the mtime against the start. A missing
// or unreadable binary never warns and never fails the tool.
func staleBinaryWarning(s startupInfo) string {
	if s.binary == "" {
		return ""
	}
	fi, err := os.Stat(s.binary)
	if err != nil {
		return ""
	}
	if onDisk := diskBuild(s.binary); onDisk != "" && s.build != unknownBuild {
		if onDisk == s.build {
			return ""
		}
		return fmt.Sprintf(warnOtherBuild, s.binary, onDisk, s.build)
	}
	if !fi.ModTime().After(s.at) {
		return ""
	}
	return fmt.Sprintf(warnNewerBinary, s.binary, stamp(fi.ModTime()), stamp(s.at))
}

// diskBuild is the VCS build stamped in the executable at path (vcsStamp), "" when it has none
// or is not a readable Go binary.
func diskBuild(path string) string {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return ""
	}
	build, _ := vcsStamp(info.Settings)
	return build
}
