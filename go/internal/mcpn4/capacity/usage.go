package capacity

import "fmt"

// Usage is one rule measured: the components counted (nil when unknown, Note says why) and the
// limit (nil: Unlimited, or unknown with a Note), with the modules the count left out.
type Usage struct {
	Rule      string
	Used      *int
	Limit     *int
	Unlimited bool
	Excluded  []string
	Note      string
}

// Headroom is how many more the limit allows (negative when already over), nil when the count
// or the limit is unknown, or there is no limit.
func (u Usage) Headroom() *int {
	if u.Used == nil || u.Limit == nil {
		return nil
	}
	h := *u.Limit - *u.Used
	return &h
}

// Over is a rule a plan would exceed, and by how many.
type Over struct {
	Rule string
	By   int
}

// Result is whether a plan fits: Fits false when a rule is surely over, nil when none is but a
// rule the plan names has an unknown count or limit (never guessed), true otherwise.
type Result struct {
	Fits  *bool
	Over  []Over
	Notes []string
}

// Check is the plan add (rule name -> components added) against usages. A rule the plan does not
// name is not checked; an unlimited rule always fits.
func Check(usages []Usage, add map[string]int) Result {
	var res Result
	unknown := false
	for _, u := range usages {
		n, named := add[u.Rule]
		switch {
		case !named || u.Unlimited:
		case u.Used == nil || u.Limit == nil:
			unknown = true
			res.Notes = append(res.Notes, fmt.Sprintf(notePlanUnknown, u.Rule))
		case *u.Used+n > *u.Limit:
			res.Over = append(res.Over, Over{Rule: u.Rule, By: *u.Used + n - *u.Limit})
		}
	}
	fits := len(res.Over) == 0
	if fits && unknown {
		return res
	}
	res.Fits = &fits
	return res
}
