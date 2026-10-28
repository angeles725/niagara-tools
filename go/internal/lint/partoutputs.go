package lint

// partNames are the module part suffixes whose own outputs PartOutputs prunes.
var partNames = []string{"rt", "ux", "wb", "se"}

// PartOutputs returns PrunePaths for a module part's own test/ and build/ directories (<module>-rt/test,
// <module>-ux/build, ...), which a rule over a module root never enters. They are pruned by path, not by name: a Java
// package named test or build under src/ (com/acme/test/) is source and is read (run-3 review R3). A test/ or build/
// directory that is not directly under a part (a group-level build/) is read too: pruning it by name would hide
// source, and no generated source is known to land there (B9 review R4-001/R3-001, decided in
// build-n4-module-kit/odd/tasks/kit-followups-rare-cases.md F9). Each call returns a fresh slice.
func PartOutputs() []string {
	var out []string
	for _, p := range partNames {
		out = append(out, "*-"+p+"/test", "*-"+p+"/build")
	}
	return out
}
