// Package ndriverstaticcollection is lint-ndriver-static-collection (G66): a static collection field in a driver
// network, device or point-device-ext class. Every station component of that type shares one static field, so
// two devices of the driver see each other's subscriptions or pending requests (the course device's static
// subscribedPoints set). The contract is the deferred G66 spec (build-n4-module-kit/ledger.d/2026-10-23-r8-u5-
// c59-c62.md); toolbelt/lint-ndriver-static-collection.sh is the wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every *.java under a -rt part's src/, srcTest/, test/ and build/
// not entered, read as javasrc tokens (comments stripped, a literal one token). A class qualifies when its
// superclass is one of the driver framework bases below, or a class of the same root that qualifies. A row is a
// static field of a qualifying class whose declared type is a collection interface or a java.util (or
// java.util.concurrent) implementation of one, unless its initializer is an unmodifiable or empty collection.
package ndriverstaticcollection

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name>
// convention; the deferred spec's row wrote "ndriver-static-collection").
const ID = "lint-ndriver-static-collection"

// bases are the NDriver and driver framework classes whose instances are station components of one driver.
var bases = []string{"BNDevice", "BNNetwork", "BDevice", "BDeviceNetwork", "BNPointDeviceExt"}

// collection are the collection interfaces and their java.util / java.util.concurrent implementations.
var collection = map[string]bool{}

func init() {
	for _, n := range strings.Fields(`Set Map List Collection Queue Deque SortedSet NavigableSet SortedMap NavigableMap
		HashSet LinkedHashSet TreeSet EnumSet HashMap LinkedHashMap TreeMap EnumMap IdentityHashMap WeakHashMap Hashtable
		ArrayList LinkedList Vector Stack ArrayDeque PriorityQueue
		ConcurrentMap ConcurrentHashMap ConcurrentSkipListMap ConcurrentSkipListSet CopyOnWriteArrayList CopyOnWriteArraySet
		BlockingQueue BlockingDeque ConcurrentLinkedQueue ConcurrentLinkedDeque LinkedBlockingQueue LinkedBlockingDeque
		ArrayBlockingQueue PriorityBlockingQueue DelayQueue`) {
		collection[n] = true
	}
}

// Spec registers the rule: [--strict] [--] <module-root>... (each root on its own), WARN rows (exit 1 under
// --strict), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-ndriver-static-collection.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "test", "build"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt part's src/ (wrong path: pass the module root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var rtPart = regexp.MustCompile(`(^|/)[^/]+-rt/src/`)

type source struct {
	path    string
	toks    []javasrc.Token
	classes []javasrc.Class
}

// CheckTree finds the qualifying classes of the root, then their static collection fields.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var srcs []source
	var all []javasrc.Class
	for _, f := range files {
		if rtPart.MatchString(f.Path) {
			toks := javasrc.Tokens(f.Stripped())
			s := source{f.Path, toks, javasrc.Classes(toks)}
			srcs, all = append(srcs, s), append(all, s.classes...)
		}
	}
	if len(srcs) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	driver := javasrc.Descendants(all, bases...)
	var out []lint.Finding
	for _, s := range srcs {
		for _, c := range s.classes {
			if !driver[c.Name] {
				continue
			}
			for _, fd := range javasrc.Fields(s.toks, c.Body) {
				if fd.Static && collection[baseType(fd.Type)] && !immutable(s.toks[fd.Init[0]:fd.Init[1]]) {
					out = append(out, lint.Finding{Path: s.path, Line: fd.Line + 1,
						Msg: "static " + fd.Type + " " + fd.Name + " is shared by every instance -- keep it per instance"})
				}
			}
		}
	}
	return out, nil
}

// baseType is the simple name of a declared type: "java.util.Set<String>" is Set, "Set<String>[]" is not one.
func baseType(t string) string {
	if strings.HasSuffix(t, "]") {
		return ""
	}
	if i := strings.IndexByte(t, '<'); i >= 0 {
		t = t[:i]
	}
	return t[strings.LastIndexByte(t, '.')+1:]
}

// immutable reports an initializer that starts with Collections.unmodifiable…(…) or Collections.empty…() (the
// class qualified or not), which no instance can change. Set.of and List.of are Java 9 and a module builds as Java 8.
func immutable(init []javasrc.Token) bool {
	if len(init) > 4 && init[0].Text == "java" && init[1].Text == "." && init[2].Text == "util" && init[3].Text == "." {
		init = init[4:]
	}
	return len(init) > 2 && init[0].Text == "Collections" && init[1].Text == "." &&
		(strings.HasPrefix(init[2].Text, "unmodifiable") || strings.HasPrefix(init[2].Text, "empty"))
}
