package ntplmanifest

import (
	"os"
	"strings"
	"testing"
)

// TestUnsafeEntryNameRefusesTheArchive (NTPL-unsafe): an entry name that could leave the directory the
// archive is unpacked into, or that holds a control character, refuses the whole archive in one row; the
// rest of it is not judged (the archive below also lacks template.bog, which is not reported).
func TestUnsafeEntryNameRefusesTheArchive(t *testing.T) {
	t.Parallel()
	for name, quoted := range map[string]string{
		"../evil.bog":     `"../evil.bog"`,
		"px/../../x":      `"px/../../x"`,
		"/abs/x":          `"/abs/x"`,
		`px\x.px`:         `"px\\x.px"`,
		"C:x":             `"C:x"`,
		"px/a\nFAIL  x.x": `"px/a\nFAIL  x.x"`,
		"":                `""`,
	} {
		root := moduleRoot(t, map[string]string{tpl: ntpl(t, entry{manifestEntry, manifestXML("", "ghost-rt")}, entry{name, "x"})})
		expect(t, quoted, root, 1, "FAIL  ntpl-entries  @ROOT@/"+tpl+": unsafe entry name "+quoted+
			` (an absolute path, a ".." element, a backslash, a drive letter or a control character) -- archive refused`+"\n")
	}
	expect(t, "dots that are no parent element", moduleRoot(t, map[string]string{
		tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML("")}, entry{"px/..hidden/a..b", "x"}, entry{"img/", ""})}), 0, "")
}

// TestNotAnArchive: a .ntpl that is no zip is refused, never skipped.
func TestNotAnArchive(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{"plain text": "<bajaObjectGraph/>\n", "an empty file": "", "a cut archive": template(t, "")[:40]} {
		root := moduleRoot(t, map[string]string{tpl: body})
		code, out, errText := run(root)
		prefix := "FAIL  ntpl-entries  " + root + "/" + tpl + ": not a zip archive ("
		if code != 1 || !strings.HasPrefix(out, prefix) || !strings.HasSuffix(out, ") -- archive refused\n") || strings.Count(out, "\n") != 1 || errText != "" {
			t.Errorf("%s: exit %d stdout %q stderr %q, want exit 1 and one row starting %q", name, code, out, errText, prefix)
		}
	}
}

// TestReadBounds: an archive over a bound is refused by a row naming the bound, and nothing past the bound
// is read: a manifest that inflates past 4 MiB, a directory of more than 10000 entries, a file over 32 MiB.
func TestReadBounds(t *testing.T) {
	t.Parallel()
	huge := manifestXML("", "ghost-rt") + "<!--" + strings.Repeat(" ", maxManifest) + "-->\n"
	expect(t, "a manifest over the bound", moduleRoot(t, map[string]string{tpl: ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, huge})}), 1,
		"FAIL  ntpl-manifest  @ROOT@/"+tpl+": template-manifest.xml inflates past 4194304 bytes -- not read\n")

	many := []entry{{bogEntry, bog}, {manifestEntry, manifestXML("")}}
	for i := 0; len(many) <= maxEntries; i++ {
		many = append(many, entry{"px/" + strings.Repeat("a", i%7+1) + string(rune('a'+i%26)) + "/" + itoa(i), ""})
	}
	expect(t, "a directory over the bound", moduleRoot(t, map[string]string{tpl: ntpl(t, many...)}), 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": 10001 entries, over the 10000 this lint reads -- archive refused\n")

	root := moduleRoot(t, map[string]string{tpl: template(t, "")})
	if err := os.Truncate(root+"/"+tpl, maxArchive+1); err != nil { // a sparse file: no data is written
		t.Fatal(err)
	}
	expect(t, "a file over the bound", root, 1,
		"FAIL  ntpl-entries  @ROOT@/"+tpl+": 33554433 bytes on disk, over the 33554432 this lint reads -- archive refused\n")
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for ; i > 0; i /= 10 {
		b = append([]byte{byte('0' + i%10)}, b...)
	}
	return string(b)
}
