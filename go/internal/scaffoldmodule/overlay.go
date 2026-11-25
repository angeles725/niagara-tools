package scaffoldmodule

import (
	"os"
	"strings"
)

// stationTestOverlay adds templates/station-test to <dest>/<Module>/<Module>-rt for a type with link-in
// slots: a Fox-off BTestNgStation fixture base, a link-in sample test and moduleTest-include.xml
// (types/moduleTest.md).
func stationTestOverlay(toolbelt, dest string, n names, typ moduleType, withOffload bool) *failure {
	if typ.linkInSlots == "" {
		return nil
	}
	root := toolbelt + "/../" + stationTestDir
	if !isFile(root + "/" + stationTestProbe) {
		return failf(ExitEnv, stationTestMissing, root)
	}
	rt := dest + "/" + n.module + "/" + n.module + "-rt"
	if err := copyTree(root, rt, n.templatePathRules(), n.templateTextRules(typ.linkInSlots), rewriteAll, keepFixtureTestList(withOffload)); err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	return nil
}

// versionLedger writes <dest>/odd/VERSION-LEDGER.md: the kit template without its sample row, so the
// first real build adds the first row and toolbelt/ledger-check.sh passes it clean (BUILD-LOOP.md §0, §7).
func versionLedger(toolbelt, dest string) *failure {
	template := toolbelt + "/../" + ledgerTemplatePath
	if !isFile(template) {
		return failf(ExitEnv, ledgerMissing, template)
	}
	data, err := os.ReadFile(template)
	if err == nil {
		err = writeFile(dest+"/"+ledgerDest, []byte(dropSampleRow(string(data))), 0o666)
	}
	if err != nil {
		return failf(ExitEnv, writeFault, err)
	}
	return nil
}

// dropSampleRow keeps every line that does not start with the sample row, each newline-terminated (as
// `grep -v` prints them, a final line without a newline included).
func dropSampleRow(text string) string {
	var b strings.Builder
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, ledgerSampleRow) {
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
