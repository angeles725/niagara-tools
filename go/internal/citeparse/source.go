package citeparse

// sourceClause reads a clause that names a source file; the source grammar is not covered yet.
func sourceClause(c, _, _ string, _ map[string]string) []Pointer {
	return unparsed(c, "kind not covered")
}

// aliases reads the per-card path aliases; none are read yet.
func aliases(_ []string, _ []bool) map[string]string { return nil }

// inline reads the prose citations of line i; none are read yet.
func (sc *scan) inline(_ int) []Pointer { return nil }

