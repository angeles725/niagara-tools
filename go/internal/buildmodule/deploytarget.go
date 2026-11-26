package buildmodule

import (
	"os"
	"regexp"
	"strings"
)

// deployTargetsFile is the record of what each station runs, kept in the module root (the client repository):
// one line per target and module, "<target> <MOD>@<vendorVersion>=<commit>", "#" starts a comment. It is read,
// never written, here; the mirror (<niagara_home>/modules) is a build sandbox and is never the deployed state.
const deployTargetsFile = ".deploy-targets"

var deployRecordRe = regexp.MustCompile(`^(\S+)@(\S+)=([0-9a-f]{7,64})$`)

// deployRecord is the recorded deploy of one module on one target; the zero value is "no recorded deploy".
type deployRecord struct{ version, commit string }

// resolveTarget is --target <name>: the commit and vendorVersion .deploy-targets records for the module on that
// target. The commit becomes the schema-risk baseline. A target with no line for the module, an unreadable
// file or a line that does not read is a usage error (exit 2) before gradle.
func (b *build) resolveTarget() (rec deployRecord, code int) {
	name, path := b.opt.deployTarget, b.root+"/"+deployTargetsFile
	data, _ := os.ReadFile(path) // an absent file is "no line"
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.SplitN(line, "#", 2)[0])
		if len(fields) == 0 || fields[0] != name || specModule(fields) != b.opt.mod {
			continue // a comment, another target or another module
		}
		m := deployRecordRe.FindStringSubmatch(fields[1])
		if len(fields) != 2 || m == nil {
			b.warn(msgTargetUnreadable, name, path, strings.TrimSpace(line))
			return rec, ExitUsage
		}
		b.say(msgTargetRecord, name, m[1]+"@"+m[2], m[3], deployTargetsFile)
		return deployRecord{version: m[2], commit: m[3]}, proceed
	}
	b.warn(msgTargetNoLine, name, b.opt.mod, path)
	return rec, ExitUsage
}

// specModule is the module a line names: its second field up to the "@" (or "=" of a line that lacks one).
func specModule(fields []string) string {
	if len(fields) < 2 {
		return ""
	}
	if i := strings.IndexAny(fields[1], "@="); i >= 0 {
		return fields[1][:i]
	}
	return fields[1]
}
