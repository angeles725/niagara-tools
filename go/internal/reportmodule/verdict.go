package reportmodule

// The causes an INCOMPLETE verdict names (#1116). Every build of a module used to end "INCOMPLETE" whatever left
// the report incomplete, so a host fault (a missing tool: an ERROR row) and a module check that was skipped (a
// floor SKIP: no built jar, no .deploy-baseline) read alike and the first taught people to ignore the second.
const (
	causeEnvironment = "environment"
	causeModule      = "module"
)

// incompleteCauses are the causes that left the run incomplete, in the order they print: a member that could
// not run (an ERROR row, whatever the mode) is the environment's; a floor check that did not run, which --lint-only
// leaves out of the verdict, is the module's.
func (r *report) incompleteCauses() []string {
	var causes []string
	if r.hadEnv {
		causes = append(causes, causeEnvironment)
	}
	if r.floorSkip && !r.o.lintOnly {
		causes = append(causes, causeModule)
	}
	return causes
}

// incomplete is "INCOMPLETE (environment)", "INCOMPLETE (module)" or "INCOMPLETE (environment, module)".
func incomplete(causes []string) string {
	out := "INCOMPLETE ("
	for i, c := range causes {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out + ")"
}
