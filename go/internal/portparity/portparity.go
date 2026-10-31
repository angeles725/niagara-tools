// Package portparity diffs a reference frontend tree against its port: every reference file (by
// basename), DOM id and named function the port lacks is a missing item, and every missing item must
// be waived in the module's parity file with a recorded user decision. A port of a reference viewer
// once lost a whole view (about 22 functions) while its design table said "same"; this check makes
// such an absence visible before hand-off.
//
// Extraction is textual (regular expressions over the raw file, not a JavaScript parser): ids are
// `id="x"` / `id='x'` / `.id = "x"`, also inside JS strings with escaped quotes; functions are
// `function f`, `function* f` and `const|let|var f = function | (...) => | x =>`. A commented-out
// definition still counts as present: the check catches dropped features, not dead code.
package portparity

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Kind is the kind of a parity item.
type Kind string

// Item kinds, in report order; KindRename appears only in waivers.
const (
	KindFile     Kind = "file"
	KindID       Kind = "id"
	KindFunction Kind = "function"
	KindRename   Kind = "rename"
)

var kindOrder = map[Kind]int{KindFile: 0, KindID: 1, KindFunction: 2}

// Item is one file, DOM id or function of a tree, at its first occurrence (root-relative path, 1-based line).
type Item struct {
	Kind Kind
	Name string
	Path string
	Line int
}

// Site is the item's `<path>:<line>`.
func (it Item) Site() string { return fmt.Sprintf("%s:%d", it.Path, it.Line) }

// Tree is what Scan read under one root, each map keyed by name (files by basename).
type Tree struct {
	Root              string
	Files, IDs, Funcs map[string]Item
}

var (
	// Mutation: PP-id-boundary -- dropping the [^\w-] guard reads data-id="x" as the DOM id x
	idRe = regexp.MustCompile(`(?:^|[^\w-])id\s*=\s*\\?["']([A-Za-z][\w-]*)\\?["']`)
	fnRe = regexp.MustCompile(`(?:^|[^\w$.])function(?:\s*\*\s*|\s+)([A-Za-z_$][\w$]*)`)
	// Mutation: PP-const-arrow -- dropping the arrow alternatives loses every `const f = () =>` function
	varFnRe = regexp.MustCompile(`(?:^|[^\w$.])(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s+)?(?:function\b|\([^)]*\)\s*=>|[A-Za-z_$][\w$]*\s*=>)`)
)

// skipDirs are directory basenames never scanned: third-party and sample data, not the port's own features.
var skipDirs = map[string]bool{"vendor": true, "node_modules": true, "data": true}

var exts = map[string]bool{".js": true, ".mjs": true, ".html": true, ".css": true}

// match is one extracted name at a byte offset of the scanned text.
type match struct {
	name string
	at   int
}

// extract returns the DOM ids and function names of text, each in order of appearance.
func extract(text string) (ids, funcs []match) {
	for _, m := range idRe.FindAllStringSubmatchIndex(text, -1) {
		ids = append(ids, match{text[m[2]:m[3]], m[2]})
	}
	for _, re := range []*regexp.Regexp{fnRe, varFnRe} {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			funcs = append(funcs, match{text[m[2]:m[3]], m[2]})
		}
	}
	sort.SliceStable(funcs, func(i, j int) bool { return funcs[i].at < funcs[j].at })
	return ids, funcs
}

// Scan reads every .js/.mjs/.html/.css file under root (lexical order; vendor/, node_modules/, data/,
// dot-directories and *.min.js skipped). A root that is not a directory, an unreadable file or a root
// with no such file is an error: an empty scan would make every reference item look waived or absent.
func Scan(root string) (Tree, error) {
	t := Tree{Root: root, Files: map[string]Item{}, IDs: map[string]Item{}, Funcs: map[string]Item{}}
	if st, err := os.Stat(root); err != nil {
		return t, err
	} else if !st.IsDir() {
		return t, fmt.Errorf("%s: not a directory", root)
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (skipDirs[name] || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !exts[filepath.Ext(name)] || strings.HasSuffix(name, ".min.js") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if _, ok := t.Files[name]; !ok {
			t.Files[name] = Item{KindFile, name, rel, 1}
		}
		text := string(b)
		ids, funcs := extract(text)
		add(t.IDs, KindID, ids, text, rel)
		add(t.Funcs, KindFunction, funcs, text, rel)
		return nil
	})
	if err != nil {
		return t, err
	}
	if len(t.Files) == 0 {
		return t, fmt.Errorf("%s: no .js, .mjs, .html or .css file (wrong path or empty tree)", root)
	}
	return t, nil
}

// add records each match not seen before (first occurrence wins) with its 1-based line.
func add(into map[string]Item, k Kind, ms []match, text, rel string) {
	for _, m := range ms {
		if _, ok := into[m.name]; !ok {
			into[m.name] = Item{k, m.name, rel, 1 + strings.Count(text[:m.at], "\n")}
		}
	}
}

// Missing lists every reference item the port lacks, sorted by kind (file, id, function) then name.
func Missing(ref, port Tree) []Item {
	var out []Item
	for _, pair := range []struct{ r, p map[string]Item }{{ref.Files, port.Files}, {ref.IDs, port.IDs}, {ref.Funcs, port.Funcs}} {
		for _, name := range sortedKeys(pair.r) {
			if _, ok := pair.p[name]; !ok {
				out = append(out, pair.r[name])
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return kindOrder[out[i].Kind] < kindOrder[out[j].Kind] })
	return out
}

func sortedKeys(m map[string]Item) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
