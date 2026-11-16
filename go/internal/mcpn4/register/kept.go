package register

import (
	"fmt"
	"path/filepath"
	"strings"
)

// checkKept is the rule for keeping an existing launcher when no server flags follow --: an
// explicit --bin or --env is never ignored. Each must equal what the launcher already execs or
// sources, or register refuses (even with --force: without the server flags it cannot re-render
// the launcher), naming the launcher, both values and how to rewrite it. An explicit --bin inside
// a git checkout is refused unless --force, as on the rewrite path.
func checkKept(o options, launcher, text string) error {
	for _, c := range []struct{ flag, prefix, want string }{
		{"bin", "exec ", o.bin}, {"env", "set -a; . ", o.env},
	} {
		if !o.explicit[c.flag] {
			continue
		}
		cur, ok := launcherWord(text, c.prefix)
		if !ok {
			cur = "(unreadable)"
		}
		if !ok || filepath.Clean(cur) != filepath.Clean(c.want) {
			return fmt.Errorf("launcher %s uses %s, not the requested --%s %s; pass the server flags after -- to rewrite it",
				launcher, cur, c.flag, c.want)
		}
	}
	if git := gitTreeOf(o.bin); o.explicit["bin"] && git != "" && !o.force {
		return fmt.Errorf("--bin %s is inside a git checkout (%s); run 'mcp-n4 install' and use its path, or rerun with --force", o.bin, git)
	}
	return nil
}

// launcherWord is the first shell word (bare, '...' or "...", backslash escapes) after prefix on
// the first launcher line that starts with prefix, and false when there is none.
func launcherWord(text, prefix string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		rest, ok := strings.CutPrefix(strings.TrimLeft(line, " \t"), prefix)
		if !ok {
			continue
		}
		rest = strings.TrimLeft(rest, " \t")
		var b strings.Builder
		for i := 0; i < len(rest); i++ {
			switch c := rest[i]; {
			case c == '\'' || c == '"':
				j := strings.IndexByte(rest[i+1:], c)
				if j < 0 {
					return "", false
				}
				b.WriteString(rest[i+1 : i+1+j])
				i += j + 1
			case c == '\\' && i+1 < len(rest):
				i++
				b.WriteByte(rest[i])
			case c == ' ' || c == '\t' || c == ';':
				return b.String(), b.Len() > 0
			default:
				b.WriteByte(c)
			}
		}
		return b.String(), b.Len() > 0
	}
	return "", false
}
