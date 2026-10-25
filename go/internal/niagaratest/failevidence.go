package niagaratest

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// testngEvidence is what a failed run keeps of testng-results.xml beside its log, read before cleanup deletes
// the scratch dir: a copy and the whole <full-stacktrace> of every FAIL test-method, each cut at a line end
// below evidenceCap with a marker row. unavailable says why the file could not be read ("" when it was).
type testngEvidence struct {
	results, traces string
	failed          int
	unavailable     string
}

// gatherTestNG reads at most evidenceCap+1 bytes for the copy and decodes at most MaxTestNGBytes for the traces;
// no buffer is sized from the file.
func gatherTestNG(path string) testngEvidence {
	f, err := os.Open(path)
	if err != nil {
		return testngEvidence{unavailable: strerror.Text(err) + ": " + path}
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, evidenceCap+1))
	if err != nil {
		return testngEvidence{unavailable: strerror.Text(err) + ": " + path}
	}
	ev := testngEvidence{results: capAtLine(b, len(b) > evidenceCap, "testng-results.xml over %d bytes")}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		ev.traces = "(cannot re-read the results file: " + strerror.Text(err) + ")\n"
		return ev
	}
	ev.traces, ev.failed = failureTraces(io.LimitReader(f, MaxTestNGBytes))
	return ev
}

// capAtLine keeps b up to its last newline within evidenceCap when cut, then a marker row: a cut never ends
// inside a path, so redaction still sees every path whole.
func capAtLine(b []byte, cut bool, what string) string {
	if !cut {
		return string(b)
	}
	b = b[:min(len(b), evidenceCap)]
	b = b[:bytes.LastIndexByte(b, '\n')+1]
	return string(b) + fmt.Sprintf("(truncated: "+what+"; kept the first %d)\n", evidenceCap, len(b))
}

// cappedText collects at most evidenceCap bytes and remembers whether more came.
type cappedText struct {
	b   []byte
	cut bool
}

func (c *cappedText) add(s string) {
	if room := evidenceCap - len(c.b); len(s) > room {
		s, c.cut = s[:max(room, 0)], true
	}
	c.b = append(c.b, s...)
}

// failureTraces is one "== FAIL <class>.<method>[ (config)] ==" section per FAIL test-method, in document order,
// holding its <full-stacktrace> whole, and the count of those methods. A read that breaks off (malformed or cut
// XML) ends the text with a row saying so.
func failureTraces(r io.Reader) (string, int) {
	var out cappedText
	dec := xml.NewDecoder(r)
	class, n := "", 0
	inFail, inTrace, hasTrace := false, false, false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			out.add(fmt.Sprintf("(results file unreadable past this point: %s)\n", err))
			break
		}
		switch el := tok.(type) {
		case xml.StartElement:
			switch {
			case el.Name.Local == "class":
				class = attr(el, "name")
			case el.Name.Local == "test-method" && attr(el, "status") == "FAIL":
				n++
				inFail, hasTrace = true, false
				config := ""
				if attr(el, "is-config") == "true" {
					config = " (config)"
				}
				out.add(fmt.Sprintf("== FAIL %s.%s%s ==\n", class, attr(el, "name"), config))
			case inFail && el.Name.Local == "full-stacktrace":
				inTrace, hasTrace = true, true
			}
		case xml.CharData:
			if inTrace {
				out.add(string(el))
			}
		case xml.EndElement:
			switch {
			case el.Name.Local == "full-stacktrace":
				inTrace = false
			case inFail && el.Name.Local == "test-method":
				if !hasTrace {
					out.add("(no full-stacktrace)")
				}
				if len(out.b) > 0 && out.b[len(out.b)-1] != '\n' {
					out.add("\n")
				}
				out.add("\n")
				inFail = false
			}
		}
		if out.cut {
			break
		}
	}
	if n == 0 && !out.cut {
		out.add("(no FAIL test-method in the results file)\n")
	}
	return capAtLine(out.b, out.cut, "traces over %d bytes"), n
}

// keepTestNG writes the redacted results copy and traces beside the log named stem.log, each 0600 and made
// exclusively, and returns the log's rows naming them.
func keepTestNG(ev testngEvidence, stem string, red redactor) ([]string, error) {
	if ev.unavailable != "" {
		return []string{"testng-results.xml unavailable: " + ev.unavailable}, nil
	}
	var rows []string
	for _, f := range []struct{ suffix, body, row string }{
		{".testng-results.xml", ev.results, "testng-results.xml: copied to %s"},
		{".traces.txt", ev.traces, "failure traces: " + plural(ev.failed, "failed method", "failed methods") + " in %s"},
	} {
		name := stem + f.suffix
		if err := writeExclusive(name, red.apply(f.body)); err != nil {
			return rows, err
		}
		rows = append(rows, fmt.Sprintf(f.row, name))
	}
	return rows, nil
}

func writeExclusive(name, body string) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.WriteString(body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
