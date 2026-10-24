package triageconsole

import "strings"

// siteClass is the site-fault class of a foreign message ("" = unclassified, no row). Matching is byte-wise on
// the ASCII-lowered text, as the awk's regexps under LC_ALL=C.
func siteClass(txt string) string {
	t := lower(txt)
	// device / instance / address match anywhere (deviceId, device_id, DeviceInstance); only the bare id token
	// is anchored, so "valid", "invalid" and "idle" are not an id while objectId / nodeId still are
	// [polish-2026-10-02 P3/P3b, #199 WU9]
	if strings.Contains(t, "duplicate") && (containsAny(t, "device", "instance", "address",
		"objectid", "object_id", "object-id", "nodeid", "node_id", "node-id") || idToken(t)) {
		return "duplicate-device-id"
	}
	if strings.Contains(t, "histor") && containsAny(t, "full", "overflow", "flood", "capacity", "exceed", "too many", "discard") {
		return "history-flood"
	}
	if containsAny(t, "timeout", "timed out", "time out", "no response", "tiempo de espera", "sin respuesta") {
		return "comm-timeout"
	}
	if offline(t) || containsAny(t, "not responding", "ping fail") {
		return "device-offline"
	}
	return ""
}

// offline is /(offline|fuera de l.nea|unreachable)/ on lowered text; "." is any one byte (LC_ALL=C), so a
// UTF-8 "línea" (two bytes for í) does not match while a Latin-1 one does.
func offline(t string) bool {
	if containsAny(t, "offline", "unreachable") {
		return true
	}
	for i := 0; ; i++ {
		p := strings.Index(t[i:], "fuera de l")
		if p < 0 {
			return false
		}
		i += p
		if j := i + len("fuera de l") + 1; j+3 <= len(t) && t[j:j+3] == "nea" {
			return true
		}
	}
}

func containsAny(t string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
}

// idToken is /(^|[^a-z0-9])ids?([^a-z0-9]|$)/.
func idToken(t string) bool {
	alnum := func(i int) bool { return i >= 0 && i < len(t) && (t[i] >= 'a' && t[i] <= 'z' || isDigit(t[i])) }
	for i := 0; i+2 <= len(t); i++ {
		if t[i:i+2] != "id" || alnum(i-1) {
			continue
		}
		if j := i + 2; !alnum(j) || t[j] == 's' && !alnum(j+1) {
			return true
		}
	}
	return false
}

// siteRecord files an unattributed block under its site class; offline wins over the "no response" timeout
// wording when the header says offline.
func (s *scanner) siteRecord(b block) {
	cls := siteClass(b.hdrMsg + " " + b.exClass + " " + b.exMsg)
	if cls == "" {
		return
	}
	if cls == "comm-timeout" && offline(lower(b.hdrMsg)) {
		cls = "device-offline"
	}
	key := cls + subsep + norm(b.hdrMsg)
	g, ok := s.sites[key]
	if !ok {
		g = &siteGroup{group: group{level: b.level, msg: b.hdrMsg}, cls: cls, tag: b.tag}
		s.sites[key] = g
	}
	g.at(b.ts)
	if !s.raw[key+subsep+b.hdrMsg] {
		s.raw[key+subsep+b.hdrMsg] = true
		g.dist++
	}
}
