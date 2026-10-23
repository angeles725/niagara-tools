package lint

import (
	"errors"
	"reflect"
	"testing"
)

// rootFake is a RootRule: it records the roots it is given, reports one finding per root and fails
// when broken is set.
type rootFake struct {
	seen   *[]string
	broken bool
}

func (rootFake) ID() string         { return "lint-r" }
func (rootFake) Severity() Severity { return Fail }
func (r rootFake) CheckRoot(root string) ([]Finding, error) {
	*r.seen = append(*r.seen, root)
	out := []Finding{{Path: root, Msg: "root seen"}}
	if r.broken {
		return out, errors.New("cannot scan " + root + "/x: broken")
	}
	return out, nil
}

// rootAndCheck implements RootRule and Rule, which registration refuses.
type rootAndCheck struct{ fakeRule }

func (rootAndCheck) CheckRoot(string) ([]Finding, error) { return nil, nil }

func rootSpec(r RootRule) Spec {
	return Spec{Root: r, Mode: Raw, Usage: "usage: lint-r.sh <module-root>"}
}

func TestRunRootRule(t *testing.T) {
	skipIfRoot(t)
	empty := tree(t, map[string]string{"README": "x\n"})
	full := tree(t, map[string]string{"A.java": "MARK\n", "build/B.java": "MARK\n"})
	locked, lockedDir := lockedTree(t, map[string]string{"A.java": "MARK\n", "locked/B.java": ""}, "locked")
	java := javaSpec(fakeRule{"lint-a", "MARK", Warn}, Each)
	java.Select.Prune = []string{"build"}
	for _, tc := range []struct {
		name      string
		specs     []Spec
		roots     []string
		code      int
		out, err  string
		seenRoots []string
	}{
		// No walk, no EmptyError: a root without any selected file is the rule's own business.
		{"alone on an empty root", []Spec{rootSpec(rootFake{})}, []string{empty}, 1,
			"FAIL  lint-r  " + empty + ": root seen\n", "", []string{empty}},
		{"error fails closed", []Spec{rootSpec(rootFake{broken: true})}, []string{empty}, 3,
			"FAIL  lint-r  " + empty + ": root seen\n", "n4lint: cannot scan " + empty + "/x: broken\n", []string{empty}},
		// Beside a walking rule: rows stay contiguous in spec order, and the root rule leaves the
		// walking rule's Prune intact (union is over walking specs only).
		{"after a walking rule", []Spec{java, rootSpec(rootFake{})}, []string{full}, 1,
			"WARN  lint-a  " + full + "/A.java:1  saw: MARK\nFAIL  lint-r  " + full + ": root seen\n", "", []string{full}},
		// A walk error stops the walking rule only; the root rule still reads its root.
		{"walk error spares it", []Spec{java, rootSpec(rootFake{})}, []string{locked}, 3,
			"FAIL  lint-r  " + locked + ": root seen\n",
			"n4lint: cannot list every file under " + locked + ": find: '" + lockedDir + "': Permission denied\n", []string{locked}},
	} {
		var seen []string
		for i, sp := range tc.specs {
			if rf, ok := sp.Root.(rootFake); ok {
				rf.seen = &seen
				tc.specs[i].Root = rf
			}
		}
		s, o, e := stdio()
		code := Run(tc.specs, "n4lint", Args{Roots: tc.roots}, s)
		if code != tc.code || o.String() != tc.out || e.String() != tc.err || !reflect.DeepEqual(seen, tc.seenRoots) {
			t.Errorf("%s: exit %d out %q err %q roots %q, want %d, %q, %q and %q",
				tc.name, code, o, e, seen, tc.code, tc.out, tc.err, tc.seenRoots)
		}
	}
}
