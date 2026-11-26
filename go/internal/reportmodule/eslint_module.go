package reportmodule

import (
	"strings"
)

// moduleOnlyParse is the message ESLint (espree) gives a file with top-level import/export parsed as a classic
// script; the rows formatter prints it as "<file>:<line>  parse: Parsing error: <message>". [ev: issue #1157]
const moduleOnlyParse = "parse: Parsing error: 'import' and 'export' may appear only with 'sourceType: module'"

// splitModuleOnly takes the script-mode rows apart: the rows to keep, and the files whose row is the
// module-only parse error (each once, in output order). ESLint stops at a fatal error, so such a file has no
// other row. removed says whether any row was taken out.
func splitModuleOnly(out string) (kept string, files []string, removed bool) {
	var b strings.Builder
	seen := map[string]bool{}
	for _, line := range strings.SplitAfter(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 && f[0] == string(failed) && f[1] == "eslint" &&
			strings.HasSuffix(strings.TrimRight(line, "\n"), "  "+moduleOnlyParse) {
			file := f[2]
			if i := strings.LastIndexByte(file, ':'); i > 0 {
				file = file[:i]
			}
			if !seen[file] {
				seen[file] = true
				files = append(files, file)
			}
			removed = true
			continue
		}
		b.WriteString(line)
	}
	return b.String(), files, removed
}

// lintAsModules re-lints the files the script-mode run could only parse as ES modules with
// eslint.module.config.mjs, and merges the two runs: the rows of both, and an exit code that no longer counts
// the module-only parse rows. ok is false when the second run was cut short.
func (r *report) lintAsModules(rc, bin, out string, code int) (string, int, bool) {
	kept, files, removed := splitModuleOnly(out)
	if len(files) == 0 || code > 1 {
		return out, code, true
	}
	args := append([]string{"--config", r.x.tb + "/eslint.module.config.mjs",
		"--format", r.x.tb + "/eslint/rows-formatter.cjs", "--no-warn-ignored", "--"}, files...)
	out2, code2, ok := r.x.run(rc, bin, args...)
	if !ok {
		return kept + out2, code2, false
	}
	if code == 1 && removed && !hasFailRow(kept) {
		code = 0
	}
	if code2 > code {
		code = code2
	}
	return kept + out2, code, true
}
