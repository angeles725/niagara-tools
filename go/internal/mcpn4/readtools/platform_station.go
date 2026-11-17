package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
)

// The station's own key store certificates: each Services/SecurityService/certificates child is
// a CertificateInfo with alias and an expiry CertificateExpiryPoint, a numeric point whose value
// is the days to the certificate's notAfter and whose status is down once it expired
// (BCertificateExpiryPoint.onExecute). A BQL select of nss:CertificateExpiryPoint answers null
// columns on 4.15 (live 2026-11-17), so they are read by one BOX load.

// stationCerts are the station's certificates (store "station"): expires is the station's day
// plus the point's days, "" when that day is unknown. note says why none were read.
func (t *Tools) stationCerts(clock licenseClock) ([]platformCert, string, error) {
	nodes, _, err := t.timedLoad(stationCertsOrd, 2)
	if err != nil {
		section, fatal := t.sectionError(err)
		if fatal != nil {
			return nil, "", fatal
		}
		return nil, fmt.Sprintf(stationCertsFailed, lookup(section, "error")), nil
	}
	var certs []platformCert
	for _, name := range box.Children(nodes, "").Keys() {
		expiry := box.Node(nodes, name+"/"+stationExpirySlot)
		if expiry == nil {
			continue
		}
		c := platformCert{store: stationStore, alias: name}
		if node := box.Node(nodes, name+"/"+stationAliasSlot); node != nil {
			if alias := pointText(node); alias != "" {
				c.alias = alias
			}
		}
		if days, status, ok := expiryDays(pointText(expiry)); ok {
			c.days, c.known, c.status = days, true, status
			if clock.known {
				c.expires = clock.today.AddDate(0, 0, days).Format("2006-01-02")
			}
		}
		certs = append(certs, c)
	}
	if len(certs) == 0 {
		return nil, stationCertsNone, nil
	}
	return certs, "", nil
}

// pointText is a loaded point's value as text: its value, else its display.
func pointText(node any) string {
	n, ok := node.(interface{ Get(string) (any, bool) })
	if !ok {
		return ""
	}
	for _, k := range []string{"v", "d"} {
		if v, _ := n.Get(k); v != nil {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}
