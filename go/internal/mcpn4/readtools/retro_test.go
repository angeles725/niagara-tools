package readtools_test

// n4_session_retro_draft through the dispatcher: no session needed, the window starts at the
// server start, a dangling report is an observation of this session, a bad since is its
// ValueError.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func TestTheRetroDraftCoversThisSession(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "journal.jsonl"), []byte(`{"batch_id": "b", "phase": "intent", "ts": "2026-10-01T11:00:00+00:00"}`+"\n"), 0o600)
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: dir})
	text, isErr, out := s.Call(t, "n4_session_retro_draft")
	if isErr || !strings.Contains(text, `"candidates": [], "station": "unknown"`) ||
		!strings.Contains(text, "- Window: since "+s.Ctx.StartedAt+`\n`) {
		t.Fatalf("%v %s", isErr, text)
	}
	if _, _, out = s.Call(t, "n4_session_retro_draft", "since", ""); !strings.Contains(pyjson.Repr(out), "'candidates': []") {
		t.Errorf("an empty since is not the server start: %s", pyjson.Repr(out))
	}
	s.Connect(t)
	s.Ctx.Observations = append(s.Ctx.Observations, pyjson.NewObject())
	s.Ctx.Observations[0].Set("ts", "2026-10-01T12:00:00+00:00")
	s.Ctx.Observations[0].Set("count", 4)
	text, isErr, _ = s.Call(t, "n4_session_retro_draft", "since", "1970-01-01T00:00:00Z")
	for _, want := range []string{`"station": "FakeStation"`, `"key": "in_doubt"`, "found 4", `Retro \u2014 FakeStation \u00b7`} {
		if isErr || !strings.Contains(text, want) {
			t.Errorf("no %s in %s", want, text)
		}
	}
	if text, isErr, _ = s.Call(t, "n4_session_retro_draft", "since", "nope"); !isErr || text != "not an ISO-8601 timestamp: 'nope'" {
		t.Errorf("%v %s", isErr, text)
	}
}
