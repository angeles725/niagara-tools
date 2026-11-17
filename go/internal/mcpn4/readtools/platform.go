package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_station_health platform=true: platform capacity and what is about to expire (station
// audits S13) as one section. heap, history_db, certificates and modules each read their spy
// page; cpu, memory and licence reuse the system and license sections already read. Each item
// that cannot be read is null with a *_note saying why; warnings gathers what is near a limit.

// platformArgs are the arguments of the section.
type platformArgs struct {
	timeout, warnDays int
	includeModules    bool
}

// healthPlatform is the platform section; err is only an error that fails the whole call.
func (t *Tools) healthPlatform(a platformArgs, system, license *pyjson.Object) (*pyjson.Object, error) {
	out := pyjson.NewObject()
	var warnings []string
	heap, note, err := t.platformHeap(a.timeout)
	if err != nil {
		return nil, err
	}
	setItem(out, "heap", heap, note)
	if h, ok := heap.(*pyjson.Object); ok {
		if f, ok := lookup(h, "fraction").(float64); ok && f >= platformHeapWarnFrac {
			warnings = append(warnings, fmt.Sprintf(heapWarning, percent(f), lookup(h, "used_kb"), lookup(h, "max_kb")))
		}
	}
	cpu, cpuWhy, memory, memoryWhy := systemItems(system)
	setItem(out, "cpu", cpu, cpuWhy)
	setItem(out, "memory", memory, memoryWhy)
	history, note, err := t.platformHistory(a.timeout)
	if err != nil {
		return nil, err
	}
	setItem(out, "history_db", history, note)
	licence, note, warn := licenceItem(license, a.warnDays)
	setItem(out, "licence", licence, note)
	warnings = append(warnings, warn...)
	certs, note, all, err := t.platformCertificates(a.timeout, stationClock(system))
	if err != nil {
		return nil, err
	}
	setItem(out, "certificates", certs, note)
	if certs != nil {
		out.Set("certificates_count", len(all))
	}
	warnings = append(warnings, certWarnings(all, a.warnDays)...)
	modules, note, err := t.platformModules(a.timeout, a.includeModules)
	if err != nil {
		return nil, err
	}
	setItem(out, "modules", modules, note)
	out.Set("expiry_warn_days", a.warnDays)
	out.Set("warnings", strs(warnings))
	return out, nil
}

// systemItems are the cpu and memory items from the system section, each null with its note
// when the section did not answer; cpu is also null, with the section's cpu_note, when the CPU
// usage was not measured.
func systemItems(system *pyjson.Object) (cpu any, cpuWhy string, memory any, memoryWhy string) {
	if text, failed := lookup(system, "error").(string); failed {
		why := fmt.Sprintf(cpuNote, text)
		return nil, why, nil, why
	}
	memory = obj("total", lookup(system, totalMemorySlot), "free", lookup(system, freeMemorySlot))
	if why, unread := lookup(system, "cpu_note").(string); unread {
		return nil, why, memory, ""
	}
	cpu = obj("current", lookup(system, cpuSlots[0]), "overall", lookup(system, cpuSlots[1]), "cpus", lookup(system, "numCpus"))
	return cpu, "", memory, ""
}

// licenceItem is the licence item from the license section's nearest expiry, and its warning
// when that expiry is within days or past. No dated expiry on the page is never_expires.
func licenceItem(license *pyjson.Object, days int) (any, string, []string) {
	if text, failed := lookup(license, "error").(string); failed {
		return nil, fmt.Sprintf(licenceNote, text), nil
	}
	nearest, ok := lookup(license, "nearest_expiry").(*pyjson.Object)
	if !ok {
		return obj("expires", nil, "days_left", nil, "never_expires", true), "", nil
	}
	kind, name, expires := lookup(nearest, "kind"), lookup(nearest, "name"), lookup(nearest, "expires")
	left := lookup(nearest, "days_left")
	item := obj("kind", kind, "name", name, "expires", expires, "days_left", left, "never_expires", false)
	n, known := left.(int)
	switch {
	case !known || n > days:
		return item, "", nil
	case n < 0:
		return item, "", []string{fmt.Sprintf(licenceWarnPast, kind, name, expires, -n)}
	}
	return item, "", []string{fmt.Sprintf(licenceWarnSoon, kind, name, expires, n, days)}
}
