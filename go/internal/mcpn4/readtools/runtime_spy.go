package readtools

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// loadedPart is one row of the module manager page.
type loadedPart struct{ name, vendor, version string }

// runtimeSpy reads one spy page for the tool: the section so far (status, and http_status and
// hint for a refused read) and the page when status is ok.
func (t *Tools) runtimeSpy(path string, timeout int) (*pyjson.Object, *spy.Page, error) {
	got, page, err := t.spyFetch(path, spy.Limits{MaxRows: runtimePageRows}, box.MaxSpyBytes, timeout)
	if err != nil {
		return nil, nil, err
	}
	out := obj("status", lookup(got, "status"))
	if page == nil {
		for _, k := range []string{"http_status", "hint"} {
			if v, ok := got.Get(k); ok {
				out.Set(k, v)
			}
		}
	}
	return out, page, nil
}

// runtimeUptime is the uptime section and the uptime in seconds (known false when the page did
// not answer or holds no tick pair). The seconds come from the ticks, never from the localized
// text, which is returned as the station wrote it.
func (t *Tools) runtimeUptime(timeout int) (*pyjson.Object, float64, bool, error) {
	out, page, err := t.runtimeSpy(runtimeSysInfoPage, timeout)
	if err != nil || page == nil {
		return out, 0, false, err
	}
	props := map[string]string{}
	for _, tb := range page.Tables {
		for _, row := range tb.Rows {
			if len(row) == 2 {
				props[row[0]] = row[1]
			}
		}
	}
	for _, kv := range runtimeUptimeText {
		if v, ok := props[kv[0]]; ok {
			out.Set(kv[1], v)
		}
	}
	boot, okB := ticks(props[runtimeBootTicks])
	now, okN := ticks(props[runtimeCurrentTicks])
	if !okB || !okN || now < boot {
		out.Set("status", runtimeUnrecognized)
		out.Set("uptime_s", nil)
		out.Set("hint", runtimeUptimeNoTicks)
		return out, 0, false, nil
	}
	secs := float64(now-boot) / 1000
	out.Set("uptime_s", secs)
	return out, secs, true, nil
}

// ticks reads a tick count the station may have written with digit grouping.
func ticks(text string) (int64, bool) {
	digits := strings.Map(func(r rune) rune {
		if strings.ContainsRune(", . ", r) {
			return -1
		}
		return r
	}, strings.TrimSpace(text))
	n, err := strconv.ParseInt(digits, 10, 64)
	return n, err == nil && digits != ""
}

// runtimeLoaded is the loaded_modules section (up to maxRows parts listed) and every part the
// page lists; complete is false when the page did not answer, is not recognized or was cut.
func (t *Tools) runtimeLoaded(timeout, maxRows int) (*pyjson.Object, []loadedPart, bool, error) {
	out, page, err := t.runtimeSpy(runtimeModulesPage, timeout)
	if err != nil || page == nil {
		return out, nil, false, err
	}
	var parts []loadedPart
	found := false
	for _, tb := range page.Tables {
		cols := make([]int, len(runtimeModuleColumns))
		for i, name := range runtimeModuleColumns {
			cols[i] = slices.Index(tb.Header, name)
		}
		if slices.Contains(cols, -1) {
			continue
		}
		found = true
		for _, row := range tb.Rows {
			if len(row) == len(tb.Header) {
				parts = append(parts, loadedPart{row[cols[0]], row[cols[1]], row[cols[2]]})
			}
		}
	}
	if !found {
		out.Set("status", runtimeUnrecognized)
		out.Set("hint", runtimeModulesNoTable)
		return out, nil, false, nil
	}
	listed := make([]any, 0, min(len(parts), maxRows))
	for _, p := range parts[:min(len(parts), maxRows)] {
		listed = append(listed, obj("name", p.name, "vendor", p.vendor, "version", p.version))
	}
	cut := pageRowsCut(page)
	out.Set("count", len(parts))
	out.Set("modules", listed)
	out.Set("truncated", len(parts) > maxRows || cut != "")
	if cut != "" {
		out.Set("note", fmt.Sprintf(runtimeLoadedPageCut, cut))
	}
	return out, parts, cut == "", nil
}

// pageRowsCut names the caps that cut rows of page ("" when none did); a cut of links, actions
// or text loses no row; a cut cell may hide a version.
func pageRowsCut(page *spy.Page) string {
	var cut []string
	for _, c := range page.Cut {
		if c == cutBytes || c == spy.CutRows || c == spy.CutTables || c == spy.CutCells || c == spy.CutCell {
			cut = append(cut, c)
		}
	}
	return strings.Join(cut, ", ")
}
