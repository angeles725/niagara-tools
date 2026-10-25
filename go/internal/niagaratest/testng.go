package niagaratest

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// MaxTestNGBytes is the largest testng-results.xml TestNGFile reads; a larger file is a FAIL, never read in
// part. TestNG reads it with a streaming decoder, so memory does not grow with the file either way.
const MaxTestNGBytes = 64 << 20

// Method is one test-method whose status is FAIL; Config marks a configuration method (is-config="true",
// such as a @BeforeClass setupStation), whose failure skips the tests after it.
type Method struct {
	Name   string
	Config bool
}

// TestNGResult is the verdict on one testng-results.xml, the file `bin\test.exe <module> -output:<dir>` writes
// (its console holds no `Total tests run` line). Root says a <testng-results> root was read; Total, Passed,
// Failed and Skipped are its attributes, -1 when missing or no count. Failing lists every FAIL test-method in
// document order. Reasons holds every cause of a FAIL.
type TestNGResult struct {
	Pass                           bool
	Root                           bool
	Total, Passed, Failed, Skipped int
	Failing                        []Method
	Reasons                        []string
}

// TestNGFile opens path and judges it with TestNG. A file that cannot be opened or is larger than
// MaxTestNGBytes is a FAIL: a run that wrote no results file is no pass.
func TestNGFile(path string) TestNGResult {
	f, err := os.Open(path)
	if err != nil {
		return unread().fail(fmt.Sprintf("cannot read %s: %s", path, strerror.Text(err)))
	}
	defer f.Close()
	if fi, err := f.Stat(); err != nil {
		return unread().fail(fmt.Sprintf("cannot read %s: %s", path, strerror.Text(err)))
	} else if fi.Size() > MaxTestNGBytes {
		return unread().fail(fmt.Sprintf("file larger than %d MiB (%d bytes): not read", MaxTestNGBytes>>20, MaxTestNGBytes))
	}
	// The cap holds again for a file that grows after Stat: the read ends at the cap, mid-document, as malformed.
	return TestNG(io.LimitReader(f, MaxTestNGBytes))
}

// unread is the result before the root is read: no counts.
func unread() TestNGResult { return TestNGResult{Total: -1, Passed: -1, Failed: -1, Skipped: -1} }

// TestNG reads a testng-results document token by token. It passes only when the root is <testng-results>
// with total > 0, failed = 0 and skipped = 0, no test-method has status FAIL (configuration methods
// included: they are not in the root counts, and their failure skips tests), the document is well-formed to
// its end, and the root counts agree with the statuses of the non-config test-methods.
func TestNG(r io.Reader) TestNGResult {
	res := unread()
	dec := xml.NewDecoder(r)
	ignored := 0
	seen := map[string]int{"PASS": 0, "FAIL": 0, "SKIP": 0}
	configFails := 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		var syn *xml.SyntaxError
		if errors.As(err, &syn) {
			return res.fail("malformed XML: " + syn.Error())
		} else if err != nil {
			return res.fail("cannot read: " + strerror.Text(err))
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if !res.Root {
			if el.Name.Local != "testng-results" {
				return res.fail(fmt.Sprintf("root element <%s>, not <testng-results>", el.Name.Local))
			}
			res.Root = true
			ignored = res.rootCounts(el)
			continue
		}
		if el.Name.Local != "test-method" {
			continue
		}
		status, name, config := attr(el, "status"), attr(el, "name"), attr(el, "is-config") == "true"
		if _, known := seen[status]; !known {
			res.Reasons = append(res.Reasons, fmt.Sprintf("test-method %s has status %q, not PASS, FAIL or SKIP", name, status))
			continue
		}
		if status == "FAIL" {
			res.Failing = append(res.Failing, Method{name, config})
		}
		if config {
			if status == "FAIL" {
				configFails++
			}
			continue
		}
		seen[status]++
	}
	if !res.Root {
		return res.fail("no testng-results root (empty file)")
	}
	res.counts(seen, ignored, configFails)
	res.Pass = len(res.Reasons) == 0
	return res
}

func (r TestNGResult) fail(reason string) TestNGResult {
	r.Reasons = append(r.Reasons, reason)
	r.Pass = false
	return r
}

// rootCounts reads the root's count attributes in a fixed order, a reason for each one missing or no count,
// and returns ignored (optional: 0 when absent).
func (r *TestNGResult) rootCounts(el xml.StartElement) int {
	ignored := 0
	for _, a := range []struct {
		key      string
		dst      *int
		optional bool
	}{{"total", &r.Total, false}, {"passed", &r.Passed, false}, {"failed", &r.Failed, false},
		{"skipped", &r.Skipped, false}, {"ignored", &ignored, true}} {
		v, found := attrOK(el, a.key)
		if !found {
			if !a.optional {
				r.Reasons = append(r.Reasons, "root attribute "+a.key+" missing")
			}
			continue
		}
		n, err := strconv.ParseUint(v, 10, 31)
		if err != nil {
			r.Reasons = append(r.Reasons, fmt.Sprintf("root attribute %s=%q is not a count", a.key, v))
			*a.dst = -1
			continue
		}
		*a.dst = int(n)
	}
	return ignored
}

// counts adds the reasons of the root counts once the whole document is read. A root attribute that is
// missing or no count already failed the run, and the cross-checks need every count, so they are skipped.
func (r *TestNGResult) counts(seen map[string]int, ignored, configFails int) {
	if r.Total < 0 || r.Passed < 0 || r.Failed < 0 || r.Skipped < 0 || ignored < 0 {
		return
	}
	if r.Total == 0 {
		r.Reasons = append(r.Reasons, "no tests ran (total=0)")
	}
	if r.Failed > 0 {
		r.Reasons = append(r.Reasons, fmt.Sprintf("%d failed", r.Failed))
	}
	if r.Skipped > 0 {
		r.Reasons = append(r.Reasons, fmt.Sprintf("%d skipped (a failed setup skips its tests)", r.Skipped))
	}
	if configFails > 0 {
		r.Reasons = append(r.Reasons, plural(configFails, "failed config method", "failed config methods"))
	}
	if seen["PASS"] != r.Passed || seen["FAIL"] != r.Failed || seen["SKIP"] != r.Skipped {
		r.Reasons = append(r.Reasons, fmt.Sprintf(
			"root counts passed %d, failed %d, skipped %d disagree with the test-methods: %d PASS, %d FAIL, %d SKIP",
			r.Passed, r.Failed, r.Skipped, seen["PASS"], seen["FAIL"], seen["SKIP"]))
	}
	if sum := r.Passed + r.Failed + r.Skipped + ignored; sum != r.Total {
		r.Reasons = append(r.Reasons, fmt.Sprintf("root total %d is not passed+failed+skipped+ignored (%d)", r.Total, sum))
	}
}

func attr(el xml.StartElement, key string) string {
	v, _ := attrOK(el, key)
	return v
}

func attrOK(el xml.StartElement, key string) (string, bool) {
	for _, a := range el.Attr {
		if a.Name.Space == "" && a.Name.Local == key {
			return a.Value, true
		}
	}
	return "", false
}
