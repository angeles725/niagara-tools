package readtools

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// platformCert is one certificate: its store (station, user or system), expires as yyyy-mm-dd
// ("" when unknown), days its days from the station's day (known false when unknown) and, for a
// station certificate, the status of its expiry point.
type platformCert struct {
	store, alias, issuer, expires, status string
	days                                  int
	known                                 bool
}

// readCerts is the certificates of the Cryptography Info trust store tables; ok is false when
// the page has no such table.
func readCerts(tables []spy.Table, clock licenseClock) ([]platformCert, bool) {
	var certs []platformCert
	found := false
	for _, tb := range tables {
		store := certStore(tb.Title)
		alias, issuer, expires := slices.Index(tb.Header, certAliasColumn), slices.Index(tb.Header, certIssuerColumn),
			slices.Index(tb.Header, certExpiresColumn)
		if store == "" || alias < 0 || expires < 0 {
			continue
		}
		found = true
		for _, row := range tb.Rows {
			if len(row) != len(tb.Header) {
				continue
			}
			c := platformCert{store: store, alias: row[alias]}
			if issuer >= 0 {
				c.issuer = row[issuer]
			}
			if day, ok := certificateDay(row[expires]); ok {
				c.expires = day
				if days, ok := clock.daysLeft(day).(int); ok {
					c.days, c.known = days, true
				}
			}
			certs = append(certs, c)
		}
	}
	return certs, found
}

// sortCerts orders certificates by store (station, user, system), then nearest expiry first,
// an undated one last in its store.
func sortCerts(certs []platformCert) {
	slices.SortStableFunc(certs, func(a, b platformCert) int {
		if r := cmp.Compare(storeRank[a.store], storeRank[b.store]); r != 0 {
			return r
		}
		if a.known != b.known {
			if a.known {
				return -1
			}
			return 1
		}
		if a.known && a.days != b.days {
			return cmp.Compare(a.days, b.days)
		}
		if (a.expires == "") != (b.expires == "") {
			return cmp.Compare(len(b.expires), len(a.expires))
		}
		return cmp.Compare(a.expires, b.expires)
	})
}

// certStore is the store a table title names ("" when it names none).
func certStore(title string) string {
	for _, s := range certStores {
		if strings.HasPrefix(title, s[0]) {
			return s[1]
		}
	}
	return ""
}

// platformCertificates is the certificates item (the first platformMaxCerts) and its note, and
// every certificate read, for the warnings: the station's own (one BOX load) and the trust
// stores (the Cryptography Info page). The item is null only when neither source answered.
func (t *Tools) platformCertificates(timeout int, clock licenseClock) (any, string, []platformCert, error) {
	certs, stationNote, err := t.stationCerts(clock)
	if err != nil {
		return nil, "", nil, err
	}
	page, pageNote, err := t.platformPage(platformCryptoPage, timeout)
	if err != nil {
		return nil, "", nil, err
	}
	trusted := page != nil
	if trusted {
		read, ok := readCerts(page.Tables, clock)
		trusted = ok
		if !ok {
			pageNote = certNoTable
		}
		certs = append(certs, read...)
	}
	notes := []string{certScopeNote}
	for _, n := range []string{stationNote, pageNote} {
		if n != "" {
			notes = append(notes, n)
		}
	}
	if certs == nil && !trusted {
		return nil, strings.Join(notes[1:], "; "), nil, nil
	}
	sortCerts(certs)
	items, unread := make([]any, 0, min(len(certs), platformMaxCerts)), 0
	for i, c := range certs {
		if c.expires == "" {
			unread++
		}
		if i < platformMaxCerts {
			items = append(items, certEntry(c))
		}
	}
	if len(certs) > platformMaxCerts {
		notes = append(notes, fmt.Sprintf(certCutNote, platformMaxCerts, len(certs)))
	}
	if unread > 0 {
		notes = append(notes, fmt.Sprintf(certUnreadNote, unread))
	}
	if page != nil {
		if cut := pageRowsCut(page); cut != "" {
			notes = append(notes, fmt.Sprintf(platformPageCut, platformCryptoPage, cut, "certificates may be missing"))
		}
	}
	return items, strings.Join(notes, "; "), certs, nil
}

// certEntry is one certificate as the reply lists it.
func certEntry(c platformCert) any {
	entry := obj("alias", c.alias, "store", c.store, "issued_by", c.issuer, "expires", nil, "days_left", nil)
	if c.store == stationStore {
		entry = obj("alias", c.alias, "store", c.store, "expires", nil, "days_left", nil, "status", c.status)
	}
	if c.expires != "" {
		entry.Set("expires", c.expires)
	}
	if c.known {
		entry.Set("days_left", c.days)
	}
	return entry
}

// certWarnings are the station and user certificates expired or expiring within days, one by
// one up to platformWarnCerts and then counted, and the system trust store's (the JDK's roots)
// in one summary line.
func certWarnings(certs []platformCert, days int) []string {
	var out []string
	more, system := 0, 0
	for _, c := range certs {
		switch {
		case !c.known || c.days > days:
			continue
		case c.store == systemStore:
			system++
			continue
		case len(out) >= platformWarnCerts:
			more++
			continue
		case c.days < 0:
			out = append(out, fmt.Sprintf(certWarnPast, c.alias, c.store, c.expires, -c.days))
		default:
			out = append(out, fmt.Sprintf(certWarnSoon, c.alias, c.store, c.expires, c.days, days))
		}
	}
	if more > 0 {
		out = append(out, fmt.Sprintf(certWarnMore, more, days))
	}
	if system > 0 {
		out = append(out, fmt.Sprintf(certSystemSummary, system))
	}
	return out
}
