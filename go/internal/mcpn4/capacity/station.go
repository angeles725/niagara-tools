package capacity

import (
	"fmt"
	"slices"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// SpyGetter reads one spy page (box.Client.GetSpy).
type SpyGetter interface {
	GetSpy(urlPath string, timeout float64, maxBytes int) (*box.File, error)
}

// ReadLicence is the licence manager page as data, nil and why when it cannot be read whole:
// no spy route, a failed or cut read, a page that is no licence page.
func ReadLicence(s SpyGetter, timeout float64) (*license.Info, string) {
	if s == nil {
		return nil, fmt.Sprintf(noteLicenceUnread, "this client cannot read spy pages")
	}
	got, err := s.GetSpy(spy.URLPath(license.Page), timeout, box.MaxSpyBytes)
	if err != nil {
		return nil, fmt.Sprintf(noteLicenceUnread, err.Error())
	}
	if got.Truncated {
		return nil, fmt.Sprintf(noteLicenceUnread, "the page was cut at the read cap")
	}
	text, _ := pyjson.DecodeUTF8(got.Body, true)
	page, err := spy.Parse(text, spy.Limits{MaxText: box.MaxSpyBytes, Deadline: time.Now().Add(time.Duration(timeout) * time.Second)})
	if err != nil || page.NotFound || len(page.Cut) > 0 {
		return nil, fmt.Sprintf(noteLicenceUnread, "the page could not be read whole")
	}
	info := license.Read(page.Text)
	if !info.Recognized() {
		return nil, fmt.Sprintf(noteLicenceUnread, "it is not a licence manager page")
	}
	return info, ""
}

// DeviceAdd checks one device to add: the usage of the devices rule (nil when the licence
// could not be read: why says so) and whether one more device surely exceeds its limit. It counts
// the station's devices only when the licence sets a limit the new device counts against.
func DeviceAdd(s SpyGetter, g OrdGetter, module string, timeout float64) (u *Usage, over bool, why string) {
	info, why := ReadLicence(s, timeout)
	if info == nil {
		return nil, false, why
	}
	u, _, over, why = Add(g, info, Devices, map[string]int{module: 1}, timeout)
	return u, over, why
}

// Add checks components to add to rule r on the licence info, counted per module of their type
// (modules -> how many): the usage of the rule, how many of them the station counts (a module the
// licence excludes is not counted, Metrics.incrementDevice) and whether they surely exceed the
// limit. The station's components are counted only when the licence sets a limit and some added
// component counts against it.
func Add(g OrdGetter, info *license.Info, r Rule, modules map[string]int, timeout float64) (u *Usage, n int, over bool, why string) {
	term := TermOf(info, r)
	uncounted := &Usage{Rule: r.Name, Limit: term.Limit, Unlimited: term.Unlimited, Excluded: term.Excluded}
	for m, k := range modules {
		if !slices.Contains(term.Excluded, m) {
			n += k
		}
	}
	if term.Unlimited || n == 0 { // nothing to exceed, so no station-wide count
		return uncounted, n, false, ""
	}
	m := MeasureRule(g, info, r, timeout)
	res := Check([]Usage{m}, map[string]int{r.Name: n})
	return &m, n, res.Fits != nil && !*res.Fits, m.Note
}
