package readtools

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/spy"
)

// platformCert is one certificate of a trust store table: expires is yyyy-mm-dd ("" when the
// cell is not a date) and days its days from the station's day (known false when either is
// unknown).
type platformCert struct {
	store, alias, issuer, expires string
	days                          int
	known                         bool
}

// readCerts is the certificates of the Cryptography Info trust store tables, nearest expiry
// first (an undated one last); ok is false when the page has no such table.
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
	slices.SortStableFunc(certs, func(a, b platformCert) int {
		if (a.expires == "") != (b.expires == "") {
			return cmp.Compare(len(b.expires), len(a.expires)) // the dated first
		}
		return cmp.Compare(a.expires, b.expires)
	})
	return certs, found
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
// every certificate read, for the warnings.
func (t *Tools) platformCertificates(timeout int, clock licenseClock) (any, string, []platformCert, error) {
	page, note, err := t.platformPage(platformCryptoPage, timeout)
	if page == nil {
		return nil, note, nil, err
	}
	certs, ok := readCerts(page.Tables, clock)
	if !ok {
		return nil, certNoTable, nil, nil
	}
	items, unread := make([]any, 0, min(len(certs), platformMaxCerts)), 0
	for i, c := range certs {
		if c.expires == "" {
			unread++
		}
		if i >= platformMaxCerts {
			continue
		}
		entry := obj("alias", c.alias, "store", c.store, "issued_by", c.issuer, "expires", nil, "days_left", nil)
		if c.expires != "" {
			entry.Set("expires", c.expires)
		}
		if c.known {
			entry.Set("days_left", c.days)
		}
		items = append(items, entry)
	}
	notes := []string{certScopeNote}
	if len(certs) > platformMaxCerts {
		notes = append(notes, fmt.Sprintf(certCutNote, platformMaxCerts, len(certs)))
	}
	if unread > 0 {
		notes = append(notes, fmt.Sprintf(certUnreadNote, unread))
	}
	if cut := pageRowsCut(page); cut != "" {
		notes = append(notes, fmt.Sprintf(platformPageCut, platformCryptoPage, cut, "certificates may be missing"))
	}
	return items, strings.Join(notes, "; "), certs, nil
}

// certWarnings are the certificates expired or expiring within days, one by one up to
// platformWarnCerts and then counted.
func certWarnings(certs []platformCert, days int) []string {
	var out []string
	more := 0
	for _, c := range certs {
		if !c.known || c.days > days {
			continue
		}
		if len(out) >= platformWarnCerts {
			more++
			continue
		}
		if c.days < 0 {
			out = append(out, fmt.Sprintf(certWarnPast, c.alias, c.store, c.expires, -c.days))
		} else {
			out = append(out, fmt.Sprintf(certWarnSoon, c.alias, c.store, c.expires, c.days, days))
		}
	}
	if more > 0 {
		out = append(out, fmt.Sprintf(certWarnMore, more, days))
	}
	return out
}

// certCount is the count of certificates read, beside the listed ones.
func certCount(out *pyjson.Object, certs []platformCert) {
	if certs != nil {
		out.Set("certificates_count", len(certs))
	}
}
