package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// The heap and history_db items of the platform section, each from its spy pages. An item
// that cannot be read is nil with the note saying why; err is only an error that fails the
// whole call (an authentication failure).

// platformPage reads one spy page: the page, or nil and the note saying why it was not read.
func (t *Tools) platformPage(path string, timeout int) (*spy.Page, string, error) {
	reply, page, err := t.spyFetch(path, spy.Limits{MaxRows: platformPageRows}, box.MaxSpyBytes, timeout)
	if err != nil {
		section, fatal := t.sectionError(err)
		if fatal != nil {
			return nil, "", fatal
		}
		return nil, fmt.Sprintf(platformPageNote, path, "error", nil, ": "+lookup(section, "error").(string)), nil
	}
	if page == nil {
		return nil, fmt.Sprintf(platformPageNote, path, lookup(reply, "status"), lookup(reply, "http_status"), hintText(reply)), nil
	}
	return page, "", nil
}

// hintText is the hint of a refused spy read as the tail of a note ("" when none).
func hintText(reply *pyjson.Object) string {
	if h, ok := lookup(reply, "hint").(string); ok && h != "" {
		return ": " + h
	}
	return ""
}

// pageProps is the two-cell rows of a page as name to value.
func pageProps(page *spy.Page) map[string]string {
	props := map[string]string{}
	for _, tb := range page.Tables {
		for _, row := range tb.Rows {
			if len(row) == 2 {
				props[row[0]] = row[1]
			}
		}
	}
	return props
}

// platformHeap is the heap item: used and committed from sysInfo (the JVM's totalMemory less
// freeMemory, and totalMemory), the maximum from the heap pools of JMX Info. A maximum that
// cannot be read leaves max_kb and fraction null with max_note; the item is null only when
// sysInfo gives no memory pair.
func (t *Tools) platformHeap(timeout int) (any, string, error) {
	page, note, err := t.platformPage(platformSysInfoPage, timeout)
	if page == nil {
		return nil, note, err
	}
	props := pageProps(page)
	total, okT := kilobytes(props[sysInfoTotal])
	free, okF := kilobytes(props[sysInfoFree])
	if !okT || !okF || free > total {
		return nil, heapNoMemory, nil
	}
	used := total - free
	heap := obj("used_kb", used, "committed_kb", total, "max_kb", nil, "fraction", nil)
	jmx, note, err := t.platformPage(platformJmxPage, timeout)
	switch {
	case err != nil:
		return nil, "", err
	case jmx == nil:
		heap.Set("max_note", note)
	default:
		if max, pools, ok := heapMax(jmx.Tables); ok {
			heap.Set("max_kb", max)
			heap.Set("fraction", heapFraction(used, max))
		} else {
			heap.Set("max_note", fmt.Sprintf(heapNoMax, pools))
		}
	}
	heap.Set("source", heapSource)
	return heap, "", nil
}

// platformHistory is the history_db item: the histories the localHistory page lists and their
// records summed; bytes is always null, with the route that would give it.
func (t *Tools) platformHistory(timeout int) (any, string, error) {
	page, note, err := t.platformPage(platformHistoryPage, timeout)
	if page == nil {
		return nil, note, err
	}
	count, ok := historyRecords(page.Tables)
	if !ok {
		return nil, historyNoTable, nil
	}
	out := obj("histories", count.histories, "records", count.records, "bytes", nil, "bytes_note", historyBytesNote)
	var notes []string
	if count.unreadable > 0 {
		notes = append(notes, fmt.Sprintf(historyUnreadNote, count.unreadable))
	}
	if cut := pageRowsCut(page); cut != "" {
		notes = append(notes, fmt.Sprintf(platformPageCut, platformHistoryPage, cut, "histories and records are lower bounds"))
	}
	out.Set("truncated", pageRowsCut(page) != "")
	if len(notes) > 0 {
		out.Set("note", strings.Join(notes, "; "))
	}
	out.Set("source", historySource)
	return out, "", nil
}

// platformModules is the modules item: the parts the module manager page lists (count), and
// with include them as {name, vendor, version} up to platformMaxModules.
func (t *Tools) platformModules(timeout int, include bool) (any, string, error) {
	section, _, _, err := t.runtimeLoaded(timeout, platformMaxModules)
	if err != nil {
		failed, fatal := t.sectionError(err)
		if fatal != nil {
			return nil, "", fatal
		}
		return nil, fmt.Sprintf(platformPageNote, runtimeModulesPage, "error", nil, ": "+lookup(failed, "error").(string)), nil
	}
	if lookup(section, "status") != runtimeStatusOK {
		return nil, fmt.Sprintf(platformPageNote, runtimeModulesPage, lookup(section, "status"),
			lookup(section, "http_status"), hintText(section)), nil
	}
	out := obj("count", lookup(section, "count"), "truncated", lookup(section, "truncated"))
	if include {
		out.Set("items", lookup(section, "modules"))
	}
	if note, ok := lookup(section, "note").(string); ok {
		out.Set("note", note)
	}
	out.Set("source", modulesSource)
	return out, "", nil
}

// setItem puts an item and, when it is null, its *_note.
func setItem(out *pyjson.Object, key string, item any, note string) {
	out.Set(key, item)
	if item == nil || note != "" {
		out.Set(key+"_note", note)
	}
}
