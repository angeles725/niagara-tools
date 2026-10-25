package agentonincludesync

import (
	"bytes"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// Sub-check test-type-in-prod (FAIL, retro r10-lint-test-types-in-prod): a part's production module-include.xml
// must not register a test type. Niagara gradle plugin 7.3.40 (iC-Niagara 4.13) runs the annotation processor of
// :<part>:compileModuleTestJava against the production module-include.xml, so every @NiagaraType class under
// srcTest/ is appended there and an ordinary build afterwards ships it in the production jar's module.xml
// (types/moduleTest.md). A <type> line is a leak when its class is declared in a *.java under the part's srcTest/,
// or when the part's moduleTest-include.xml lists the same class.

// TestCheck is the check column of a test-type-in-prod row.
const TestCheck = "test-type-in-prod"

const testInclude = "moduleTest-include.xml"

// TestLeak is one production <type> line that registers a test type: its class, the 1-based line of the <type>
// element and why it is a test type ("source <srcTest file>" or "also listed in <moduleTest-include.xml>").
type TestLeak struct {
	Class string
	Line  int
	Why   string
}

// Msg is the row message of a leak, shared by this lint and verify-module's moduletest row.
func (l TestLeak) Msg() string {
	return l.Class + ": test type in the production " + include + " (" + l.Why + ") -- move the <type> line to " +
		testInclude + "; compileModuleTestJava appends it here (types/moduleTest.md)"
}

// TestLeaks is every <type> of prod (a part's module-include.xml) whose class is a key of sources (srcTest binary
// name -> its file) or a <type> of test (the part's moduleTest-include.xml, nil when it has none), in prod order.
func TestLeaks(prod, test *xmlsrc.Doc, sources map[string]string) []TestLeak {
	listed := map[string]bool{}
	if test != nil {
		for _, i := range test.Find("type") {
			if class, ok := test.Elements()[i].Attr("class"); ok {
				listed[class] = true
			}
		}
	}
	var out []TestLeak
	elems := prod.Elements()
	for _, i := range prod.Find("type") {
		class, ok := elems[i].Attr("class")
		if !ok {
			continue
		}
		if src, ok := sources[class]; ok {
			out = append(out, TestLeak{Class: class, Line: elems[i].Line, Why: "source " + src})
		} else if listed[class] {
			out = append(out, TestLeak{Class: class, Line: elems[i].Line, Why: "also listed in " + test.Path})
		}
	}
	return out
}

// TestClasses is the binary name (pkg.Outer$Inner) of every named type declared in toks (javasrc.Tokens of the
// Strip view), in source order.
func TestClasses(toks []javasrc.Token) []string {
	pkg := packageName(toks)
	classes := javasrc.Classes(toks)
	names := make([]string, len(classes))
	var outer []int // indexes of the enclosing classes, innermost last
	for i, c := range classes {
		for len(outer) > 0 && c.At >= classes[outer[len(outer)-1]].Body[1] {
			outer = outer[:len(outer)-1]
		}
		switch {
		case len(outer) > 0:
			names[i] = names[outer[len(outer)-1]] + "$" + c.Name
		case pkg != "":
			names[i] = pkg + "." + c.Name
		default:
			names[i] = c.Name
		}
		outer = append(outer, i)
	}
	return names
}

// packageName is the name of the package declaration, "" for the default package.
func packageName(toks []javasrc.Token) string {
	for k, t := range toks {
		if t.Kind != javasrc.TokIdent || t.Text != "package" {
			continue
		}
		var b strings.Builder
		for _, n := range toks[k+1:] {
			if n.Text == ";" {
				break
			}
			b.WriteString(n.Text)
		}
		return b.String()
	}
	return ""
}

// testPart is the deepest directory above a "/srcTest/" of path that is a part (a key of parts).
func testPart[V any](path string, parts map[string]V) (string, bool) {
	for i := strings.LastIndex(path, "/srcTest/"); i >= 0; i = strings.LastIndex(path[:i], "/srcTest/") {
		if _, ok := parts[path[:i]]; ok {
			return path[:i], true
		}
	}
	return "", false
}

// PartTestLeaks reads one part directory (its module-include.xml, moduleTest-include.xml and srcTest/ sources) and
// returns its leaks, for verify-module. A part with no module-include.xml has none; a malformed XML file or an
// unreadable source is the error.
func PartTestLeaks(part string) ([]TestLeak, error) {
	if !lint.IsFile(part + "/" + include) {
		return nil, nil
	}
	prod, err := xmlsrc.Open(part + "/" + include)
	if err != nil {
		return nil, err
	}
	var test *xmlsrc.Doc
	if lint.IsFile(part + "/" + testInclude) {
		if test, err = xmlsrc.Open(part + "/" + testInclude); err != nil {
			return nil, err
		}
	}
	sources := map[string]string{}
	if lint.IsDir(part + "/srcTest") {
		files, err := lint.Walk(part+"/srcTest", lint.Selection{Names: []string{"*.java"}, Prune: []string{"build"}})
		if err != nil {
			return nil, err
		}
		for _, p := range files {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			lines, _ := javasrc.Lines(bytes.NewReader(data)) // a bytes.Reader never fails
			for _, c := range TestClasses(javasrc.Tokens(javasrc.NewFile(p, lines).Stripped())) {
				sources[c] = p
			}
		}
	}
	return TestLeaks(prod, test, sources), nil
}
