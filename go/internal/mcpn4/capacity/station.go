package capacity

import (
	"fmt"
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
// could not be read: why says so) and whether one more device surely exceeds its limit.
func DeviceAdd(s SpyGetter, g OrdGetter, module string, timeout float64) (u *Usage, over bool, why string) {
	info, why := ReadLicence(s, timeout)
	if info == nil {
		return nil, false, why
	}
	term := TermOf(info, Devices)
	for _, m := range term.Excluded {
		if m == module { // the station does not count this device (Metrics.incrementDevice)
			return &Usage{Rule: Devices.Name, Limit: term.Limit, Unlimited: term.Unlimited, Excluded: term.Excluded}, false, ""
		}
	}
	m := MeasureRule(g, info, Devices, timeout)
	res := Check([]Usage{m}, map[string]int{Devices.Name: 1})
	return &m, res.Fits != nil && !*res.Fits, m.Note
}
