package vendorfloor

import (
	"fmt"
	"sort"
	"strings"
)

// The panel floor every vendored browser library must run on.
const (
	FloorChrome      = 83   // Chromium 83, the HMI panel browser
	FloorECMAVersion = 2020 // the newest ECMAScript syntax Chromium 83 parses
)

// APIs is the API-above-floor table: each name maps to the Chrome version that first shipped it,
// grouped by the token shape of its use. To flag one more API, add one entry to Default.
type APIs struct {
	Member     map[string]int // <expr>.name(   a method call on any receiver
	Qualified  map[string]int // Owner.name(    a static call, keyed "Owner.name"
	GlobalCall map[string]int // name(          a call not preceded by "."
	GlobalNew  map[string]int // new Name       a constructor
}

// Default is the table the lint ships with.
var Default = APIs{
	Member: map[string]int{
		"replaceAll": 85, "at": 92, "findLast": 97, "findLastIndex": 97,
		"toSorted": 110, "toReversed": 110, "toSpliced": 110,
	},
	Qualified: map[string]int{
		"Object.hasOwn": 93, "AbortSignal.timeout": 103, "Promise.any": 85, "Object.groupBy": 117,
	},
	GlobalCall: map[string]int{"structuredClone": 98},
	GlobalNew:  map[string]int{"WeakRef": 84, "FinalizationRegistry": 84},
}

// Validate reports a method name held by both Member and Qualified: one call would print two rows.
// Mutation: VF-disjoint -- dropping this check lets a name in both MEMBER and QUALIFIED double-report.
func (a APIs) Validate() error {
	for _, q := range sortedKeys(a.Qualified) {
		if m := method(q); has(a.Member, m) {
			return fmt.Errorf("MEMBER and QUALIFIED share the method name '%s' (%s)", m, q)
		}
	}
	return nil
}

// Names lists every identifier the table can flag, sorted and unique: the only name tokens the
// acorn driver has to report.
func (a APIs) Names() []string {
	seen := map[string]bool{}
	for _, m := range []map[string]int{a.Member, a.GlobalCall, a.GlobalNew} {
		for name := range m {
			seen[name] = true
		}
	}
	for q := range a.Qualified {
		seen[method(q)] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// method is the name after the last "." of a qualified name.
func method(qualified string) string {
	return qualified[strings.LastIndexByte(qualified, '.')+1:]
}

func has(m map[string]int, name string) bool {
	_, ok := m[name]
	return ok
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
