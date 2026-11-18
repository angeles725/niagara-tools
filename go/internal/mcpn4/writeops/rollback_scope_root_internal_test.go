package writeops

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
)

// A create of the write-scope root (allowed by that scope, createScope) can be rolled back under the same
// scope: the rollback undoes exactly the component that batch created, so it is checked on the created
// path, as the create was, not on the parent its inverse removes from (live finding of #808: scope
// /AckLab created AckLab but refused its rollback).
func TestTheCreateOfTheScopeRootRollsBackUnderTheSameScope(t *testing.T) {
	r := newTRig(t, 200)
	r.state.Scope, _ = ord.NewScope([]string{tFolder + "/New"})
	_, isErr, out := r.write(t, "n4_create_component", obj("parent_ord", tFolder, "name", "New", "type", "baja:Folder"))
	if isErr {
		t.Fatalf("create: %v", out)
	}
	bid := get(out, "batch_id").(string)
	text, isErr, back := r.write(t, "n4_rollback", obj("batch_id", bid))
	if isErr || get(back, "verdict") != "verified" {
		t.Fatalf("rollback of the scope-root create: %v %s", isErr, text)
	}
}

// Only the created root is admitted: under another scope the same rollback stays refused.
func TestTheScopeRootRollbackStillNeedsTheScope(t *testing.T) {
	r := newTRig(t, 200)
	r.state.Scope, _ = ord.NewScope([]string{tFolder + "/New"})
	_, _, out := r.write(t, "n4_create_component", obj("parent_ord", tFolder, "name", "New", "type", "baja:Folder"))
	bid := get(out, "batch_id").(string)
	r.state.Scope, _ = ord.NewScope([]string{tFolder + "/Other"})
	if text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", bid)); !isErr || !strings.Contains(text, "outside the write scope") {
		t.Fatalf("rollback under another scope: %v %s", isErr, text)
	}
}

// n4_remove_component of the scope root stays refused under that scope: only the rollback of the batch that
// created it is admitted.
func TestRemovingTheScopeRootIsStillRefused(t *testing.T) {
	r := newTRig(t, 200)
	r.state.Scope, _ = ord.NewScope([]string{tFolder + "/New"})
	r.write(t, "n4_create_component", obj("parent_ord", tFolder, "name", "New", "type", "baja:Folder"))
	if text, isErr, _ := r.call(t, "n4_remove_component", obj("parent_ord", tFolder, "name", "New")); !isErr ||
		!strings.Contains(text, "outside the write scope") {
		t.Fatalf("remove of the scope root: %v %s", isErr, text)
	}
}

// checkTargetScope admits an out-of-scope parent only when every inverse op on it removes the scope root:
// another op on that parent (a value restore), a remove of another child, or no op at all keep the refusal.
func TestCheckTargetScopeRefusesAnythingButRemovingTheRoot(t *testing.T) {
	s, _ := ord.NewScope([]string{tFolder + "/New"})
	for _, c := range []struct {
		name  string
		ops   []any
		admit bool
		tool  string
	}{
		{"remove of the root", []any{obj("nm", "v", "h", "3", "n", "New")}, true, ""},
		{"remove of the root by a copy batch", []any{obj("nm", "v", "h", "3", "n", "New")}, false, "n4_copy_component"},
		{"remove of the root and a value restore", []any{obj("nm", "v", "h", "3", "n", "New"), obj("nm", "s", "h", "3", "n", "x")}, false, ""},
		{"remove of another child", []any{obj("nm", "v", "h", "3", "n", "Other")}, false, ""},
		{"no op on the target", []any{obj("nm", "v", "h", "9", "n", "New")}, false, ""},
	} {
		tool := c.tool
		if tool == "" {
			tool = createTool
		}
		err := checkTargetScope(s, tFolder, "3", c.ops, tool)
		if (err == nil) != c.admit {
			t.Errorf("%s: err %v, want admit %v", c.name, err, c.admit)
		}
	}
}
