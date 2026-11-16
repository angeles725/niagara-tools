package blackbox

// Write mode, session and write scope (port of test_tools_write TestMode, TestIdentity and
// TestScope).

import (
	"slices"
	"strings"
	"testing"
)

func TestReadOnlyServerHasNoWriteToolsAndCallingOneIs32601(t *testing.T) {
	r := NewRig(t, nil)
	listed := listedTools(r.Srv)
	if len(listed) == 0 {
		t.Fatal("tools/list is empty")
	}
	for _, name := range append(slices.Clone(writeTools), destructiveTools...) {
		if _, ok := listed[name]; ok {
			t.Errorf("%s is listed by a read-only server", name)
		}
		if code := r.Srv.CallError(name, Args{}); code != -32601 {
			t.Errorf("%s: code %d, want -32601", name, code)
		}
	}
}

// annotations is a listed tool's annotations; it also checks the token properties every write
// tool's schema has.
func annotations(t *testing.T, tool map[string]any, name string) map[string]any {
	t.Helper()
	props, _ := Dig(tool, "inputSchema", "properties").(map[string]any)
	for _, p := range []string{"dry_run", "confirmation_token"} {
		if _, ok := props[p]; !ok {
			t.Errorf("%s: no %s property", name, p)
		}
	}
	ann, _ := tool["annotations"].(map[string]any)
	if ann == nil {
		t.Fatalf("%s: no annotations: %v", name, tool)
	}
	return ann
}

func TestWritesAllowedServerListsWriteToolsWithAnnotations(t *testing.T) {
	tools := listedTools(writeRig(t, nil).Srv)
	for _, name := range writeTools {
		ann := annotations(t, tools[name], name)
		idempotent := ann["idempotentHint"]
		if idempotent == nil {
			idempotent = false
		}
		if ann["readOnlyHint"] != false || ann["destructiveHint"] != false || ann["openWorldHint"] != false ||
			idempotent != (name == "n4_set_slot") {
			t.Errorf("%s: annotations %v", name, ann)
		}
	}
}

func TestDestructiveToolsAreListedWithDestructiveAnnotations(t *testing.T) {
	tools := listedTools(writeRig(t, nil).Srv)
	for _, name := range destructiveTools {
		ann := annotations(t, tools[name], name)
		if ann["readOnlyHint"] != false || ann["destructiveHint"] != true || ann["openWorldHint"] != false {
			t.Errorf("%s: annotations %v", name, ann)
		}
	}
}

func TestWriteWithoutASessionIsRefused(t *testing.T) {
	r := writeRig(t, nil)
	if text := r.Srv.Err("n4_create_component", createPump); !strings.Contains(text, "not connected") {
		t.Errorf("refusal = %s", text)
	}
}

func TestOutOfScopeParentIsRefused(t *testing.T) {
	r := verifiedRig(t)
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"parent_ord": "station:|slot:/"}))
	if !strings.Contains(text, "outside the write scope") || len(r.Children()) != 0 {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

func TestNoScopeConfiguredRefusesEveryWrite(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = nil })
	r.ConnectVerified()
	if text := r.Srv.Err("n4_create_component", createPump); !strings.Contains(text, "no --write-scope") {
		t.Errorf("refusal = %s", text)
	}
}

func TestPrefixBoundaryIsRespected(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = []string{"station:|slot:/Fold"} })
	r.ConnectVerified()
	if text := r.Srv.Err("n4_create_component", createPump); !strings.Contains(text, "outside the write scope") {
		t.Errorf("refusal = %s", text)
	}
}

func TestALinkTargetOutsideTheScopeIsRefused(t *testing.T) {
	r := verifiedRig(t)
	r.AddWithOut("A", 1)
	text := r.Srv.Err("n4_create_link", Args{"source_ord": Folder + "/A", "source_slot": "out",
		"target_ord": "station:|slot:/Other", "target_slot": "in10"})
	if !strings.Contains(text, "outside the write scope") {
		t.Errorf("refusal = %s", text)
	}
}
