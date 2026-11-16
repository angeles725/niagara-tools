package ntplmanifest

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// bogOf is a template.bog that uses baja and is padded with a comment to size bytes in all: it deflates to a
// few KiB, so a small archive holds it.
func bogOf(size int) string {
	head := `<bajaObjectGraph version="4.0"><p m="b=baja" t="b:Folder"/><!--`
	return head + strings.Repeat(" ", size-len(head)-len("--></bajaObjectGraph>")) + "--></bajaObjectGraph>"
}

// T26 (the #710 verifier's A4): a jar of 200 templates, each holding a bog that inflates to the full
// per-entry bound, is a small file that asked for 200 x 32 MiB of reads when each template kept its own
// budget (132.6 s on the base). One budget for the run ends it with one FAIL row naming the first entry not
// read. The work is pinned by where it stops, not by the clock: maxInflate/maxUsesRead full bogs would fill the
// budget exactly, and the templates' archives and manifests draw a little more, so the bog of template
// maxInflate/maxUsesRead - 1 (T003 at 128 and 32 MiB) is the first entry not read, and no later template is
// read. The stop is derived from the two bounds (T34: a change of either moves it, not the test's reason to
// fail). The wall-time bound is a generous backstop only (the base took minutes; -race on a busy host: 74 s).
func TestAmplifyingJarSpendsOneRunBudget(t *testing.T) {
	if maxInflate%maxUsesRead != 0 || maxInflate/maxUsesRead < 2 {
		t.Fatalf("the stop below assumes a budget of at least two whole bogs: %d / %d", maxInflate, maxUsesRead)
	}
	stop := int(maxInflate/maxUsesRead) - 1
	one := ntpl(t, entry{bogEntry, bogOf(maxUsesRead)}, entry{manifestEntry, manifestXML("")})
	entries := make([]entry, stop+10)
	for i := range entries {
		entries[i] = entry{fmt.Sprintf("t/T%03d.ntpl", i), one}
	}
	jar := jarAt(t, entries...)
	if st, err := os.Stat(jar); err != nil || st.Size() > 1<<20 {
		t.Fatalf("the amplifying jar is not small: %v %v", st, err)
	}
	start := time.Now()
	code, out, errText := run(homeOption, installation(t, stock...), jar)
	elapsed := time.Since(start)
	want := fmt.Sprintf("FAIL  ntpl-uses  %s!/t/T%03d.ntpl: the templates and their entries inflate past %d bytes in all, the bound of one run -- template.bog and everything after it not read\n",
		jar, stop, maxInflate)
	if code != 1 || out != want || errText != "" {
		t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit 1\nstdout %q", code, out, errText, want)
	}
	if elapsed > 300*time.Second { // a hang guard only: the row above is the bound
		t.Errorf("the amplifying jar took %s; the base took minutes and one run budget reads %d templates", elapsed, stop+1)
	}
}

// T26: a module of more templates than the lint reads checks the first maxTemplates in path order; one FAIL
// row names the first one not read, and none after it is opened.
func TestTemplateCountBound(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	one := template(t, "")
	for i := 0; i <= maxTemplates; i++ {
		files[fmt.Sprintf("ModA-rt/src/t/T%04d.ntpl", i)] = one
	}
	expect(t, "templates over the bound", moduleRoot(t, files), 1, fmt.Sprintf(
		"FAIL  ntpl-entries  @ROOT@/ModA-rt/src/t/T%04d.ntpl: %d templates, over the %d this lint reads -- this one and every one after it not read\n",
		maxTemplates, maxTemplates+1, maxTemplates))
}

// T26: inside a jar, the same bound on the number of templates, past it one row on the first one not read.
func TestJarTemplateCountBound(t *testing.T) {
	t.Parallel()
	one := template(t, "")
	entries := make([]entry, maxTemplates+1)
	for i := range entries {
		entries[i] = entry{fmt.Sprintf("t/T%04d.ntpl", i), one}
	}
	expect(t, "jar templates over the bound", jarAt(t, entries...), 1, fmt.Sprintf(
		"FAIL  ntpl-entries  @ROOT@!/t/T%04d.ntpl: %d templates, over the %d this lint reads -- this one and every one after it not read\n",
		maxTemplates, maxTemplates+1, maxTemplates))
}

// The run budget draws on the manifest, the bog and Px entries and, in a jar, each template: lowered so the
// second archive of a module no longer fits, its row names the entry the budget stopped at, and the third
// archive is not opened (it would be a lacks row).
func TestRunBudgetStopsTheRun(t *testing.T) {
	defer func(all int64) { maxInflate = all }(maxInflate)
	man := manifestXML("")
	maxInflate = int64(len(man) + len(bog) + len(man))
	root := moduleRoot(t, map[string]string{
		"ModA-rt/src/a/A.ntpl": template(t, ""),
		"ModA-rt/src/b/B.ntpl": template(t, ""),
		"ModA-rt/src/c/C.ntpl": ntpl(t, entry{manifestEntry, man}),
	})
	expect(t, "run budget", root, 1,
		"FAIL  ntpl-uses  @ROOT@/ModA-rt/src/b/B.ntpl: the templates and their entries inflate past "+fmt.Sprint(maxInflate)+
			" bytes in all, the bound of one run -- template.bog and everything after it not read\n")
	maxInflate = int64(len(man) - 1)
	expect(t, "run budget at the manifest", root, 1,
		"FAIL  ntpl-manifest  @ROOT@/ModA-rt/src/a/A.ntpl: the templates and their entries inflate past "+fmt.Sprint(maxInflate)+
			" bytes in all, the bound of one run -- template-manifest.xml and everything after it not read\n")
}

// The archive the budget stops at reads nothing more: its manifest does not fit, and its small bog, which
// would fit and uses a module nothing lists, is not inflated. No row can show it (an archive without its
// manifest prints no ntpl-uses row), so the read itself is checked: no member, and the budget untouched by
// the bog. Guard: readZip's return once the budget is spent.
func TestRunBudgetStopsTheArchiveAtItsManifest(t *testing.T) {
	man := manifestXML("")
	small := `<bajaObjectGraph><p m="v=vendorx"/></bajaObjectGraph>`
	if len(small) >= len(man)-1 {
		t.Fatalf("the bog (%d bytes) must fit where the manifest (%d) does not", len(small), len(man))
	}
	data := ntpl(t, entry{bogEntry, small}, entry{manifestEntry, man})
	b := &budget{left: int64(len(man) - 1)}
	a, err := readZip(strings.NewReader(data), int64(len(data)), b)
	if err != nil || !b.spent || a.uses != nil || b.left != int64(len(man)-1) || a.unread != b.spentMsg(manifestEntry) {
		t.Errorf("err %v spent %v uses %+v left %d unread %q: want the manifest's budget row, nothing read after it", err, b.spent, a.uses, b.left, a.unread)
	}
}

// The jar the budget stops in reads no later template: after A's bog does not fit, B, whose template and
// manifest would still fit and which needs a module the installation does not ship, gives no row. Guard:
// checkJar's break once the budget is spent.
func TestRunBudgetStopsTheJar(t *testing.T) {
	defer func(all int64) { maxInflate = all }(maxInflate)
	man, needs := manifestXML(""), manifestXML("", "local-rt")
	b := template(t, "", "local-rt")
	slack := int64(len(b) + len(needs) + 10) // room left for B's template and manifest once A is stopped
	a := ntpl(t, entry{bogEntry, bogOf(int(slack) + 64)}, entry{manifestEntry, man})
	maxInflate = int64(len(a)+len(man)) + slack
	jar := jarAt(t, entry{"t/A.ntpl", a}, entry{"t/B.ntpl", b})
	expect(t, "stop in the jar", jar, 1,
		"FAIL  ntpl-uses  @ROOT@!/t/A.ntpl: the templates and their entries inflate past "+fmt.Sprint(maxInflate)+
			" bytes in all, the bound of one run -- template.bog and everything after it not read\n")
}

// #710 advisory: the size bound of a jar is read from its metadata before it is opened, as for a .ntpl, so an
// unreadable jar over the bound is the FAIL row (exit 1), never exit 3.
func TestOversizeUnreadableJarIsTheBoundRow(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a file without read permission")
	}
	jar := jarAt(t, entry{"t/A.ntpl", template(t, "")})
	if err := os.Truncate(jar, maxArchive+1); err != nil { // a sparse file: no data is written
		t.Fatal(err)
	}
	if err := os.Chmod(jar, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(jar, 0o600) })
	expect(t, "an unreadable jar over the bound", jar, 1,
		"FAIL  ntpl-entries  @ROOT@: 33554433 bytes on disk, over the 33554432 this lint reads -- archive refused\n")
}

// #710 advisory: a template entry of a jar is selected by its extension in any letter case.
func TestJarTemplateNamesAnyCase(t *testing.T) {
	t.Parallel()
	jar := jarAt(t, entry{"t/T.NTPL", template(t, "", "local-rt")}, entry{"t/U.Ntpl", template(t, "", "other-rt")})
	expect(t, "any case", jar, 1,
		"FAIL  ntpl-deps  @ROOT@!/t/T.NTPL: manifest dependencies not shipped by the installation @HOME@ (a template module jar declares no build dependencies): local-rt\n"+
			"FAIL  ntpl-deps  @ROOT@!/t/U.Ntpl: manifest dependencies not shipped by the installation @HOME@ (a template module jar declares no build dependencies): other-rt\n")
}

// #710 advisory: without an installation the rows of a jar are WARN, after the one row saying the module list
// is not known, and the run exits 0.
func TestJarWarnRowsWithoutInstallation(t *testing.T) {
	t.Parallel()
	jar := jarAt(t, entry{"t/A.ntpl", ntpl(t, entry{bogEntry, bog}, entry{manifestEntry, manifestXML("", "local-rt")},
		entry{"px/P.px", pxOf("vendorx", 200)})})
	code, out, errText := run(jar)
	want := "WARN  lint-ntpl-manifest-deps  " + jar + ": " + noHome + "\n" +
		"WARN  ntpl-deps  " + jar + "!/t/A.ntpl: manifest dependencies a template module jar does not declare, and the installation's module list is not known: local-rt\n" +
		"WARN  ntpl-uses  " + jar + "!/t/A.ntpl: px/P.px uses vendorx: no <dependency> of template-manifest.xml names them\n"
	if code != 0 || out != want || errText != "" {
		t.Errorf("exit %d\nstdout %q\nstderr %q\nwant exit 0\nstdout %q", code, out, errText, want)
	}
}
