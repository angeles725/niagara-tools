package bogaudit

// The checks still to land, one commit per group; until then they print nothing.

func (a *audit) defrostSiblings() {}
func (a *audit) roomIndex()       {}
func (a *audit) tileNumbers()     {}
func (a *audit) linkDirection()   {}
func (a *audit) numericToBool()   {}

type sources struct{}

func scanSources(string) *sources { return &sources{} }

func (a *audit) sourceChecks() {
	if a.src == nil {
		for _, c := range []string{"CHECK2", "CHECK3", "CHECK4", "CHECK5", "CHECK6", "CHECK7"} {
			a.emit(c, "SKIP", "--source-dir", "not provided; skipping source-dependent check")
		}
	}
}
