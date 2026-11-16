package niagaratest

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// The source half of the ignored gate. testng-results.xml counts the ignored test-methods but never names them
// (ignored.go), so run names them from the source: every <type class> of <part>/moduleTest-include.xml is looked up
// among the classes declared under <part>/srcTest/ (the shared javasrc scanner, binary names as the
// test-type-in-prod lint derives them), and each method-level @Test(enabled = false) a registered concrete class
// declares or inherits through srcTest/ superclasses is one ignored method, named <SimpleName>.<method> (the class
// TestNG runs it on). How each case counts:
//   - an abstract registered type adds nothing itself: TestNG runs its methods on the concrete subclasses;
//   - an inherited method counts on each registered concrete subclass, unless a nearer class declares a method of
//     that name (the nearest declaration decides);
//   - a superclass outside srcTest/ (BTestNg, a library) is not read: a disable there makes the source and TestNG
//     counts disagree, which FAILs, and the count fallback (--allow-ignored-count) is the way out;
//   - a class-level @Test(enabled = false) and an enabled value that is no true/false literal are refused, as is a
//     registered source the scanner cannot read to its end: the scan never guesses (run exit 3).
//
// JUnit's @Ignore is no TestNG disable and is not read.

// maxSourceBytes is the largest moduleTest-include.xml or srcTest/*.java the scan reads; a larger one is refused.
const maxSourceBytes = 4 << 20

const testIncludeName = "moduleTest-include.xml"

// disabledInSource is the sorted list of the ignored test-methods the registered types of part (a part directory)
// declare, one entry per method TestNG ignores. A part with no moduleTest-include.xml registers none.
func disabledInSource(part string) ([]string, error) {
	inc := filepath.Join(part, testIncludeName)
	if !lint.IsFile(inc) {
		return nil, nil
	}
	data, err := readSource(inc)
	if err != nil {
		return nil, err
	}
	doc, err := xmlsrc.Parse(inc, data)
	if err != nil {
		return nil, err
	}
	var registered []string
	for _, i := range doc.Find("type") {
		if class, ok := doc.Elements()[i].Attr("class"); ok && !slices.Contains(registered, class) {
			registered = append(registered, class)
		}
	}
	if len(registered) == 0 {
		return nil, nil
	}
	idx, err := indexSrcTest(filepath.Join(part, "srcTest"))
	if err != nil {
		return nil, err
	}
	simple := map[string]string{}
	var out []string
	for _, class := range registered {
		short := class[strings.LastIndex(class, ".")+1:]
		if other, dup := simple[short]; dup {
			return nil, fmt.Errorf("registered types %s and %s share the simple name %s: their methods cannot be named <Class>.<method>", other, class, short)
		}
		simple[short] = class
		names, err := idx.disabled(class, short, part)
		if err != nil {
			return nil, fmt.Errorf("cannot name the disabled test-methods of %s: %v", class, err)
		}
		out = append(out, names...)
	}
	slices.Sort(out)
	return out, nil
}

// readSource reads a file of at most maxSourceBytes; the read stops at the cap whatever its size says.
func readSource(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxSourceBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if len(data) > maxSourceBytes {
		return nil, fmt.Errorf("%s: larger than %d MiB, not read", path, maxSourceBytes>>20)
	}
	return data, nil
}
