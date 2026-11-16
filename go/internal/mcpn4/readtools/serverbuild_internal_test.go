package readtools

import (
	"runtime/debug"
	"testing"
)

func TestVcsStampIsTheRevisionMarkedWhenDirtyAndItsCommitTime(t *testing.T) {
	set := func(kv ...string) []debug.BuildSetting {
		var out []debug.BuildSetting
		for i := 0; i < len(kv); i += 2 {
			out = append(out, debug.BuildSetting{Key: kv[i], Value: kv[i+1]})
		}
		return out
	}
	cases := []struct {
		settings       []debug.BuildSetting
		build, builtAt string
	}{
		{set("vcs.revision", "abc123", "vcs.time", "2026-11-15T10:00:00Z", "vcs.modified", "false"), "abc123", "2026-11-15T10:00:00Z"},
		{set("vcs.revision", "abc123", "vcs.modified", "true"), "abc123+modified", ""},
		{set("-trimpath", "true"), "", ""},
	}
	for _, c := range cases {
		if build, builtAt := vcsStamp(c.settings); build != c.build || builtAt != c.builtAt {
			t.Errorf("vcsStamp(%v) = %q, %q; want %q, %q", c.settings, build, builtAt, c.build, c.builtAt)
		}
	}
}
