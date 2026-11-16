package blackbox

// The stale-binary warning of n4_connect and n4_describe_session (retro row P5): the build id of
// the binary on disk decides first, its mtime only when either build id is unreadable.

import (
	"os"
	"strings"
	"testing"
	"time"
)

// staleRig is a fake station and a server run from a copy of the test binary that reports build
// running (buildEnv), so a test can replace the binary on disk under the running server.
func staleRig(t *testing.T, running string) (*Rig, string) {
	t.Helper()
	bin := copiedExe(t)
	r := NewRig(t, func(_ *Rig, o *Options) {
		o.Exe = bin
		o.Env[buildEnv] = running
	})
	return r, bin
}

// warnings are the warning fields of n4_connect and then n4_describe_session.
func warnings(r *Rig) (connect, describe string) {
	return str(r.Connect(nil)["warning"]), str(r.Srv.OK("n4_describe_session", nil)["warning"])
}

func TestConnectWarnsWhenTheBinaryOnDiskIsAnotherBuild(t *testing.T) {
	stamped, revision := stampedBinary(t)
	r, bin := staleRig(t, "0000aaaa")
	replaceBinary(t, bin, stamped, -time.Hour) // an older mtime: the build id decides, not the mtime
	connect, describe := warnings(r)
	want := "the binary on disk (" + bin + ") is build " + revision +
		", but this server runs build 0000aaaa: restart the MCP server to serve it"
	if connect != want || describe != want {
		t.Errorf("connect warning %q\ndescribe warning %q\nwant %q", connect, describe, want)
	}
}

func TestConnectDoesNotWarnWhenTheBinaryOnDiskIsTheSameBuild(t *testing.T) {
	stamped, revision := stampedBinary(t)
	r, bin := staleRig(t, revision)
	replaceBinary(t, bin, stamped, time.Hour) // a newer mtime, but the same build
	if connect, describe := warnings(r); connect != "" || describe != "" {
		t.Errorf("connect warning %q, describe warning %q", connect, describe)
	}
}

func TestConnectFallsBackToTheModTimeWhenTheBinaryOnDiskHasNoBuildId(t *testing.T) {
	notGo := writeScratch(t, "#!/bin/sh\n")
	r, bin := staleRig(t, "0000aaaa")
	replaceBinary(t, bin, notGo, -time.Hour)
	if connect, describe := warnings(r); connect != "" || describe != "" {
		t.Fatalf("an older binary without a build id warned: %q, %q", connect, describe)
	}
	touch(t, bin, time.Hour)
	connect, describe := warnings(r)
	if !strings.HasPrefix(connect, "the binary on disk ("+bin+") was modified at ") ||
		!strings.Contains(connect, "restart the MCP server") || describe != connect {
		t.Errorf("connect warning %q, describe warning %q", connect, describe)
	}
}

func TestConnectNeitherWarnsNorFailsWhenTheBinaryOnDiskIsGone(t *testing.T) {
	r, bin := staleRig(t, "0000aaaa")
	if err := os.Remove(bin); err != nil {
		t.Fatal(err)
	}
	out := r.Connect(nil)
	if out["station_name"] != StationName || out["warning"] != nil {
		t.Errorf("connect = %v", out)
	}
	if w := r.Srv.OK("n4_describe_session", nil)["warning"]; w != nil {
		t.Errorf("describe warning %v", w)
	}
}
