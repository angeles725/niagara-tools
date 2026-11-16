package license

import "strings"

// Dated is one thing of the page with an expiry date.
type Dated struct{ Kind, Name, Expires string }

// Dated lists what the page dates: licences, then certificates, then features, each in page
// order. What never expires, or whose expiry is not a date, is not listed.
func (i *Info) Dated() []Dated {
	var out []Dated
	i.eachExpiry(func(kind, name, expires string) {
		if ExpiryOf(expires) == ExpiryDate {
			out = append(out, Dated{kind, name, expires})
		}
	})
	return out
}

// UnreadExpiries counts the licences, certificates and features whose expiry is neither
// "never" nor a date: their days left are unknown, not endless.
func (i *Info) UnreadExpiries() (n int) {
	i.eachExpiry(func(_, _, expires string) {
		if ExpiryOf(expires) == ExpiryUnread {
			n++
		}
	})
	return n
}

func (i *Info) eachExpiry(visit func(kind, name, expires string)) {
	for _, e := range i.Licenses {
		visit(KindLicense, e.Name, e.Expires)
	}
	for _, e := range i.Certificates {
		visit(KindCertificate, e.Name, e.Expires)
	}
	for _, f := range i.Features {
		visit(KindFeature, f.Name, f.Expires)
	}
}

// Nearest is the earliest of dated (the first listed among equal dates) and how many share its
// date; ok false when dated is empty.
func Nearest(dated []Dated) (nearest Dated, same int, ok bool) {
	for _, d := range dated {
		switch {
		case !ok || d.Expires < nearest.Expires:
			nearest, same, ok = d, 1, true
		case d.Expires == nearest.Expires:
			same++
		}
	}
	return nearest, same, ok
}

// Identity is the brand, owner and project the licence states ("" when it does not): the
// attributes of the about and brand features, the brand also from the Brand section.
func (i *Info) Identity() (brandID, owner, project string) {
	for n := range i.Features {
		switch f := &i.Features[n]; f.Name {
		case aboutFeature:
			owner, _ = f.Attr(ownerKey)
			project, _ = f.Attr(projectKey)
		case brandFeature:
			brandID, _ = f.Attr(brandKey)
		}
	}
	for _, row := range i.Brand {
		if brandID == "" && row[0] == brandKey {
			brandID = row[1]
		}
	}
	return brandID, owner, project
}

// SMAGroup is the attributes of one feature that speak of the software maintenance agreement
// (the sma.* ones, in page order); AttrsCut says the feature has attributes that were not kept.
type SMAGroup struct {
	Feature  string
	Attrs    [][2]string
	AttrsCut bool
}

// Maintenance is what the page states of the software maintenance agreement: per feature its
// sma.* attributes, the feature named after maintenance (nil when none), and the names of the
// features whose sma.exempt is true, each once. The exemption is stated per feature, never for
// the licence.
func (i *Info) Maintenance() (groups []SMAGroup, feature *Feature, exempt []string) {
	named := map[string]bool{}
	for n := range i.Features {
		f := &i.Features[n]
		if _, name, _ := strings.Cut(f.Name, ":"); feature == nil && strings.Contains(strings.ToLower(name), maintenanceName) {
			feature = f
		}
		group := SMAGroup{Feature: f.Name, AttrsCut: f.AttrsCut > 0}
		for _, a := range f.Attrs {
			if !strings.HasPrefix(a[0], smaPrefix) {
				continue
			}
			group.Attrs = append(group.Attrs, a)
			if a[0] == smaExempt && strings.EqualFold(a[1], smaTrue) && !named[f.Name] {
				named[f.Name] = true
				exempt = append(exempt, f.Name)
			}
		}
		if len(group.Attrs) > 0 {
			groups = append(groups, group)
		}
	}
	return groups, feature, exempt
}

// Match lists the features a filter names: the one whose vendor:name is the filter, else those
// whose name (after the vendor) holds it, compared without case; every feature for "".
func (i *Info) Match(filter string) []Feature {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return i.Features
	}
	var part []Feature
	for _, f := range i.Features {
		lower := strings.ToLower(f.Name)
		if lower == filter {
			return []Feature{f}
		}
		if _, name, _ := strings.Cut(lower, ":"); strings.Contains(name, filter) {
			part = append(part, f)
		}
	}
	return part
}
