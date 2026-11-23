package retrogh

import (
	"errors"
	"strings"
	"testing"
)

type call struct {
	args  []string
	stdin string
}

func fake(out map[string]string, calls *[]call) Runner {
	return func(args []string, stdin string) (string, error) {
		*calls = append(*calls, call{args, stdin})
		for prefix, o := range out {
			if strings.HasPrefix(strings.Join(args, " "), prefix) {
				if o == "ERR" {
					return "", errors.New("gh: boom")
				}
				return o, nil
			}
		}
		return "", nil
	}
}

const listJSON = `[{"number":7,"title":"mcp-n4: t","state":"CLOSED","body":"x\n<!-- retro-delta: ALPHA Δ1 -->","labels":[{"name":"retro-delta"}],
"comments":[{"body":"also\n<!-- retro-delta: BETA ΔB2 -->"}]}]`

func TestListMarkersCoverBodyAndComments(t *testing.T) {
	var calls []call
	issues, err := ListLabeled(fake(map[string]string{"issue list": listJSON}, &calls), "retro-delta")
	if err != nil || len(issues) != 1 {
		t.Fatalf("ListLabeled = %v, %v", issues, err)
	}
	m := Markers(issues)
	if m["ALPHA Δ1"].Number != 7 || m["BETA ΔB2"].Number != 7 || m["ALPHA Δ1"].State != "CLOSED" {
		t.Errorf("markers = %+v", m)
	}
	got := strings.Join(calls[0].args, " ")
	for _, want := range []string{"--state all", "--label retro-delta", "--json number,title,state,body,comments,labels"} {
		if !strings.Contains(got, want) {
			t.Errorf("args %q lack %q", got, want)
		}
	}
}

func TestListRefusesATruncatedList(t *testing.T) {
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < ListLimit; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"number":1,"state":"OPEN"}`)
	}
	b.WriteString("]")
	if _, err := ListLabeled(fake(map[string]string{"issue list": b.String()}, new([]call)), "x"); err == nil {
		t.Error("a list that hit the cap may hide a marker: it must be an error")
	}
}

func TestListErrorsPropagate(t *testing.T) {
	if _, err := ListLabeled(fake(map[string]string{"issue list": "ERR"}, new([]call)), "x"); err == nil {
		t.Error("a gh failure must be an error")
	}
	if _, err := ListLabeled(fake(map[string]string{"issue list": "not json"}, new([]call)), "x"); err == nil {
		t.Error("bad JSON must be an error")
	}
}

func TestMarker(t *testing.T) {
	if got := Marker("ALPHA", "Δ3"); got != "<!-- retro-delta: ALPHA Δ3 -->" {
		t.Errorf("Marker = %q", got)
	}
}

func TestCreateCommentAndLabels(t *testing.T) {
	var calls []call
	run := fake(map[string]string{
		"issue create": "https://github.com/o/r/issues/42\n",
		"label list":   `[{"name":"retro-delta"}]`,
	}, &calls)
	n, err := Create(run, "T", "BODY", []string{"retro-delta", "area:mcp-n4"})
	if err != nil || n != 42 {
		t.Fatalf("Create = %d, %v", n, err)
	}
	if c := calls[0]; c.stdin != "BODY" || !strings.Contains(strings.Join(c.args, " "), "--title T") || !strings.Contains(strings.Join(c.args, " "), "--label area:mcp-n4") {
		t.Errorf("create call = %+v", c)
	}
	if err := EnsureLabels(run, map[string]string{"retro-delta": "d", "area:mcp-n4": "d"}); err != nil {
		t.Fatal(err)
	}
	var made []string
	for _, c := range calls {
		if strings.HasPrefix(strings.Join(c.args, " "), "label create") {
			made = append(made, c.args[2])
		}
	}
	if len(made) != 1 || made[0] != "area:mcp-n4" {
		t.Errorf("labels created = %v, want only the missing one", made)
	}
	if err := Comment(run, 9, "hello"); err != nil {
		t.Fatal(err)
	}
	last := calls[len(calls)-2:]
	if !strings.HasPrefix(strings.Join(last[0].args, " "), "issue comment 9") || last[0].stdin != "hello" ||
		!strings.HasPrefix(strings.Join(last[1].args, " "), "issue edit 9 --add-label retro-delta") {
		t.Errorf("comment calls = %+v", last)
	}
}

func TestState(t *testing.T) {
	var calls []call
	run := fake(map[string]string{"issue view 5": `{"state":"CLOSED"}`}, &calls)
	if s, err := State(run, 5); err != nil || s != "CLOSED" {
		t.Errorf("State = %q, %v", s, err)
	}
	if got := strings.Join(calls[0].args, " "); got != "issue view 5 --json state" {
		t.Errorf("args = %q", got)
	}
	if _, err := State(fake(map[string]string{"issue view": "ERR"}, new([]call)), 5); err == nil {
		t.Error("a gh failure must be an error")
	}
}
