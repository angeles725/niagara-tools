package license

import "strings"

// Entry is one certificate or licence line: its name, its expiry as written ("never" or a
// date) and its state between braces.
type Entry struct{ Name, Expires, State string }

// Valid reports whether the station says the entry is valid.
func (e Entry) Valid() bool { return e.State == validState }

// Feature is one licensed feature: vendor:name, its expiry as written and its attributes
// (key, value) in page order, at most MaxFeatureAttrs; AttrsCut counts the ones not kept.
type Feature struct {
	Name, Expires string
	Attrs         [][2]string
	AttrsCut      int
}

// Attr is the value of the feature's attribute key, ok false when it has none.
func (f *Feature) Attr(key string) (string, bool) {
	for _, a := range f.Attrs {
		if a[0] == key {
			return a[1], true
		}
	}
	return "", false
}

// Info is the licence manager page as data. Certificates and Licenses hold at most MaxEntries
// each, CertificateCount and LicenseCount what the page has. LicenseKind is the qualifier of
// the Licenses title ("perpetual"), "" when it has none. Missing names the sections the page
// does not have, and Unparsed counts the lines that matched no pattern of their section.
type Info struct {
	HostID                         string
	Certificates, Licenses         []Entry
	CertificateCount, LicenseCount int
	LicenseKind                    string
	Features                       []Feature
	Brand                          [][2]string
	Missing                        []string
	Unparsed                       int
}

// Recognized reports whether the text was a licence manager page at all: a host id or at
// least one section.
func (i *Info) Recognized() bool { return i.HostID != "" || len(i.Missing) < len(sectionTitles) }

// Read is the page text (one item per line, as spy.Page.Text gives it) as data. It never
// fails: what it cannot place is counted in Unparsed and what is absent is named in Missing.
// Every string kept is Clean and cut to its limit.
func Read(text string) *Info {
	info, at, seen := &Info{}, preamble, map[section]bool{}
	for _, line := range strings.Split(text, "\n") {
		if line = Clean(line); line == "" {
			continue
		}
		if kind, qualifier, ok := sectionOf(line); ok && kind > at {
			at, seen[kind] = kind, true
			if kind == licenses {
				info.LicenseKind = Clip(strings.TrimSpace(qualifier), MaxNameRunes)
			}
			continue
		}
		if !info.place(at, line) {
			info.Unparsed++
		}
	}
	for _, s := range sectionTitles {
		if !seen[s.kind] {
			info.Missing = append(info.Missing, s.title)
		}
	}
	return info
}

// sectionOf is the section a title line names and its qualifier.
func sectionOf(line string) (kind section, qualifier string, ok bool) {
	m := titleLine.FindStringSubmatch(line)
	if m == nil {
		return preamble, "", false
	}
	for _, s := range sectionTitles {
		if s.title == m[1] {
			return s.kind, m[2], true
		}
	}
	return preamble, "", false
}

// place keeps a line of section at where it belongs; false when it matches no pattern there.
func (i *Info) place(at section, line string) bool {
	switch at {
	case preamble:
		if m := hostIDLine.FindStringSubmatch(line); m != nil && i.HostID == "" {
			i.HostID = Clip(strings.TrimSpace(m[1]), MaxNameRunes)
			return i.HostID != ""
		}
		return line == pageTitle
	case certificates:
		return keepEntry(&i.Certificates, &i.CertificateCount, line)
	case licenses:
		return keepEntry(&i.Licenses, &i.LicenseCount, line)
	case features:
		return i.placeFeature(line)
	case brand:
		if m := brandLine.FindStringSubmatch(line); m != nil {
			if len(i.Brand) < MaxFeatureAttrs {
				i.Brand = append(i.Brand, [2]string{Clip(m[1], MaxKeyRunes), Clip(m[2], MaxValueRunes)})
			}
			return true
		}
	}
	return false
}

// keepEntry reads a certificate or licence line into list (up to MaxEntries) and counts it.
func keepEntry(list *[]Entry, count *int, line string) bool {
	m := entryLine.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	if *count++; len(*list) < MaxEntries {
		*list = append(*list, Entry{Name: Clip(strings.TrimSpace(m[1]), MaxNameRunes),
			Expires: Clip(strings.TrimSpace(m[2]), MaxNameRunes), State: Clip(strings.TrimSpace(m[3]), MaxNameRunes)})
	}
	return true
}

// placeFeature reads a feature line, or an attribute of the last feature read.
func (i *Info) placeFeature(line string) bool {
	if m := featureLine.FindStringSubmatch(line); m != nil {
		i.Features = append(i.Features, Feature{Name: Clip(m[1], MaxNameRunes), Expires: Clip(strings.TrimSpace(m[2]), MaxNameRunes)})
		return true
	}
	m := attributeLine.FindStringSubmatch(line)
	if m == nil || len(i.Features) == 0 {
		return false
	}
	last := &i.Features[len(i.Features)-1]
	if len(last.Attrs) >= MaxFeatureAttrs {
		last.AttrsCut++
		return true
	}
	last.Attrs = append(last.Attrs, [2]string{Clip(m[1], MaxKeyRunes), Clip(m[2], MaxValueRunes)})
	return true
}
