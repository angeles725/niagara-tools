package capacity

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
)

// Term is what the licence says of one rule: its limit (nil with Unlimited when it states none,
// or states no limit at all; nil without Unlimited when it cannot be read, Note says why) and the
// modules whose components are not counted (ExcludedKnown false when that list was cut).
type Term struct {
	Limit         *int
	Unlimited     bool
	Excluded      []string
	ExcludedKnown bool
	Note          string
}

// TermOf is r's term on the licence info. Without the global capacity feature the station
// enforces no global limit (GlobalGroup sets every limit to Integer.MAX_VALUE).
func TermOf(info *license.Info, r Rule) Term {
	term := Term{Unlimited: true, ExcludedKnown: true}
	var feature *license.Feature
	for i := range info.Features {
		if info.Features[i].Name == FeatureName {
			feature = &info.Features[i]
		}
	}
	if feature == nil {
		return term
	}
	switch v, ok := feature.Attr(r.LimitKey); {
	case !ok && feature.AttrsCut > 0:
		term.Unlimited, term.Note = false, fmt.Sprintf(noteLimitCut, FeatureName, r.LimitKey)
	case !ok || strings.EqualFold(strings.TrimSpace(v), noLimit):
	default:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			term.Limit, term.Unlimited = &n, false
		} else {
			term.Unlimited, term.Note = false, fmt.Sprintf(noteLimitUnread, r.LimitKey, v)
		}
	}
	v, ok := feature.Attr(r.ExcludeKey)
	switch {
	case !ok && feature.AttrsCut > 0, strings.HasSuffix(v, license.CutMarker):
		term.ExcludedKnown = false
		term.Note = joinNote(term.Note, fmt.Sprintf(noteExcludedCut, r.ExcludeKey))
	case ok && !strings.EqualFold(strings.TrimSpace(v), noLimit):
		// GlobalGroup.parseModules: a StringTokenizer on ",;"
		for _, m := range strings.FieldsFunc(v, func(c rune) bool { return c == ',' || c == ';' }) {
			term.Excluded = append(term.Excluded, m)
		}
	}
	return term
}

func joinNote(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}
