package livesmoke

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// reasonIndent aligns a reason under the step name of the line above it ("%-9s %s").
const reasonIndent = "          "

// reasonLine is the console line that says why a step that ran did not verify: its detail,
// indented under the step name. A verified step and a skipped one print none (the skip reason
// is the same for every skipped step, and the report keeps it).
func reasonLine(entry *pyjson.Object) string {
	verdict, _ := entry.Get("verdict")
	if verdict == "verified" || verdict == "skipped" {
		return ""
	}
	detail, _ := entry.Get("detail")
	if detail == nil {
		return ""
	}
	if obj, ok := detail.(*pyjson.Object); ok {
		if inner, has := obj.Get("detail"); has { // a step that reports {"detail": "..."}
			detail = inner
		}
	}
	text, ok := detail.(string)
	if !ok {
		text = pyjson.Str(detail)
	}
	return reasonIndent + strings.ReplaceAll(text, "\n", "\n"+reasonIndent)
}

// reportProblem says why the report cannot be written at path, before anything runs: its
// directory must exist. "" when no report was asked for or the directory is there.
func reportProblem(path string) string {
	if path == "" {
		return ""
	}
	dir := filepath.Dir(path)
	st, err := os.Stat(dir)
	switch {
	case os.IsNotExist(err):
		return "--report: directory " + dir + " does not exist"
	case err != nil:
		return "--report: directory " + dir + " cannot be checked: " + err.Error()
	case !st.IsDir():
		return "--report: " + dir + " is not a directory"
	}
	return ""
}
