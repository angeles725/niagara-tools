package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// licenseReply adds a read licence page to out; it returns how many features it lists and the
// notes of the lists it had to cut.
func (t *Tools) licenseReply(out *pyjson.Object, info *license.Info, clock licenseClock, filter string, maxFeatures int) (int, []string) {
	if clock.known {
		out.Set("station_date", clock.today.Format("2006-01-02"))
	} else {
		out.Set("station_date", nil)
		out.Set("station_date_note", clock.why)
	}
	var notes []string
	listed := func(key string, list *capped, total int) {
		out.Set(key, list.items)
		if total > len(list.items) {
			out.Set(key+"_count", total)
		}
		if list.cut {
			notes = append(notes, fmt.Sprintf(licenseListCut, key))
		}
	}
	files := t.readLicenseFiles(info)
	out.Set("host_id", stated(info.HostID))
	certificates := newCapped(capEntries)
	for _, e := range info.Certificates {
		if !certificates.add(entryObject(e)) {
			break
		}
	}
	listed("certificates", certificates, info.CertificateCount)
	out.Set("license_kind", stated(info.LicenseKind))
	listed("licenses", licenseEntries(info, clock, files, true), info.LicenseCount)
	if files.note != "" {
		out.Set("generated_note", files.note)
	}
	brand, owner, project := info.Identity()
	out.Set("brand", stated(brand))
	out.Set("owner", stated(owner))
	out.Set("project", stated(project))
	maintenance := maintenanceEntry(info, clock)
	files.maintenance(maintenance, clock)
	out.Set("maintenance", maintenance)
	setExpiries(out, info, clock)
	matched := info.Match(filter)
	list := newCapped(listCap{min(maxFeatures, capFeatures.items), capFeatures.bytes})
	for _, f := range matched {
		if !list.add(featureEntry(f)) {
			break
		}
	}
	out.Set("features", obj("count", len(info.Features), "matched", len(matched), "list", list.items,
		"truncated", len(list.items) < len(matched)))
	return len(list.items), notes
}

// stated is a value the page states, nil when it does not.
func stated(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// entryObject is a certificate or licence as name, expiry and valid, with its state as the
// page writes it when that is not the valid one.
func entryObject(e license.Entry) *pyjson.Object {
	entry := obj("name", e.Name, "expires", e.Expires, "valid", e.Valid())
	if !e.Valid() {
		entry.Set("state", e.State)
	}
	return entry
}

// licenseEntries are the licences of the page under capEntries: name, expiry, valid and the
// days left, plus what their local files add when withFiles is set.
func licenseEntries(info *license.Info, clock licenseClock, files licenseFiles, withFiles bool) *capped {
	list := newCapped(capEntries)
	for _, e := range info.Licenses {
		entry := entryObject(e)
		entry.Set("days_left", clock.daysLeft(e.Expires))
		if withFiles {
			files.add(entry, e.Name)
		}
		if !list.add(entry) {
			break
		}
	}
	return list
}

// featureEntry is a feature as name, expiry and its attributes by key; attributes_truncated
// says the page gives it more attributes than are kept.
func featureEntry(f license.Feature) *pyjson.Object {
	entry := obj("name", f.Name, "expires", f.Expires, "attributes", attributes(f.Attrs))
	if f.AttrsCut > 0 {
		entry.Set("attributes_truncated", true)
	}
	return entry
}

func attributes(attrs [][2]string) *pyjson.Object {
	out := pyjson.NewObject()
	for _, a := range attrs {
		out.Set(a[0], a[1])
	}
	return out
}

// setExpiries adds nearest_expiry (the earliest dated expiry, with how many share its date) and
// expired (what is past on the station's day, under capExpired; nil when that day is unknown)
// to out.
func setExpiries(out *pyjson.Object, info *license.Info, clock licenseClock) {
	dated := info.Dated()
	out.Set("nearest_expiry", nil)
	if d, same, ok := license.Nearest(dated); ok {
		out.Set("nearest_expiry", obj("kind", d.Kind, "name", d.Name, "expires", d.Expires,
			"days_left", clock.daysLeft(d.Expires), "same_date", same))
	}
	if !clock.known {
		out.Set("expired", nil)
		return
	}
	expired, count := newCapped(capExpired), 0
	for _, d := range dated {
		days, _ := license.DaysLeft(d.Expires, clock.today)
		if days >= 0 {
			continue
		}
		count++
		expired.add(obj("kind", d.Kind, "name", d.Name, "expires", d.Expires, "days_left", days))
	}
	out.Set("expired", expired.items)
	if count > len(expired.items) {
		out.Set("expired_count", count)
	}
}
