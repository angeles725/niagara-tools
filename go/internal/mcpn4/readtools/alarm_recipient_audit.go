package readtools

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The recipient side of n4_list_alarm_classes audit=true (station-audits S5): each recipient's
// settings as its routeAlarm load already holds them, and the problems that stop delivery. A
// slot whose name or type looks like a credential is never returned, only named; a URL keeps
// no user info, Telegram bot token or secret query value.

// settingMaxRunes bounds a returned setting (a mail body is a long format).
const settingMaxRunes = 200

var (
	// sensitiveSlotRe names a slot whose value is never returned.
	sensitiveSlotRe = regexp.MustCompile(`(?i)pass|secret|token|credential|api_?key|private`)
	// endpointSlotRe names a slot a custom recipient (webhook, Telegram) delivers to.
	endpointSlotRe = regexp.MustCompile(`(?i)(url|chatid|webhook)$`)
	emailRe        = regexp.MustCompile(`^[^@\s<>"(),;:|]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}$`)
	redactions     = []struct {
		re   *regexp.Regexp
		with string
	}{
		{regexp.MustCompile(`://[^/@\s]+@`), "://<redacted>@"},
		{regexp.MustCompile(`/bot[0-9]+:[A-Za-z0-9_-]+`), "/bot<redacted>"},
		{regexp.MustCompile(`(?i)([?&](?:token|key|secret|password|apikey|api_key)=)[^&\s]*`), "${1}<redacted>"},
	}
	// emptyDefaults are the route slots a load omits at their type default, which is empty
	// (BEmailRecipient to, cc, bcc and emailAccount; BStationRecipient remoteStation).
	emptyDefaults = map[string][]string{"EmailRecipient": {"to", "cc", "bcc", "emailAccount"},
		"StationRecipient": {"remoteStation"}}
)

// Problems of a recipient, as data.
const (
	probNoEmail      = "no email address: to, cc and bcc are empty"
	probBadEmail     = "%s holds %q, which is not an email address"
	probNoStation    = "remoteStation is empty: no station receives the alarms"
	probNoTransition = "transitions is empty: no alarm transition is routed"
	probNoDay        = "daysOfWeek is empty: no day is routed"
	probNoEndpoint   = "%s is empty: the recipient has nowhere to deliver"
	probLastFailure  = "the last delivery failed: %s"
)

// recipientAudit is {settings, at_default, omitted_slots, problems} of the recipient whose
// load (two levels from the recipient) is nodes, kind its type.
func recipientAudit(nodes *pyjson.Object, kind string) *pyjson.Object {
	settings, omitted, atDefault := pyjson.NewObject(), []any{}, []any{}
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		node := box.Node(kids, name)
		v, has := node.Get("v")
		if !has || isLinkType(get(node, "t")) {
			continue
		}
		if sensitiveSlotRe.MatchString(name) || strings.Contains(strings.ToLower(pyjson.Str(get(node, "t"))), "password") {
			omitted = append(omitted, name)
			continue
		}
		settings.Set(name, redact(pyjson.Str(v)))
	}
	short := kind[strings.LastIndexByte(kind, ':')+1:]
	for _, name := range emptyDefaults[short] {
		if _, ok := settings.Get(name); !ok {
			settings.Set(name, "")
			atDefault = append(atDefault, name)
		}
	}
	return obj("settings", settings, "at_default", atDefault, "omitted_slots", omitted,
		"problems", recipientProblems(settings, short))
}

// recipientProblems are the reasons the recipient cannot deliver, from its settings.
func recipientProblems(settings *pyjson.Object, short string) []any {
	problems := []any{}
	text := func(name string) (string, bool) {
		v, ok := settings.Get(name)
		return strings.TrimSpace(pyjson.Str(v)), ok
	}
	if short == "EmailRecipient" {
		found := 0
		for _, slot := range []string{"to", "cc", "bcc"} {
			v, _ := text(slot)
			for _, addr := range emailAddresses(v) {
				found++
				if !emailRe.MatchString(addr) {
					problems = append(problems, fmt.Sprintf(probBadEmail, slot, addr))
				}
			}
		}
		if found == 0 {
			problems = append(problems, probNoEmail)
		}
	}
	if v, ok := text("remoteStation"); ok && v == "" && short == "StationRecipient" {
		problems = append(problems, probNoStation)
	}
	for _, c := range []struct{ slot, msg string }{{"transitions", probNoTransition}, {"daysOfWeek", probNoDay}} {
		if v, ok := text(c.slot); ok && (v == "0" || v == "{}") {
			problems = append(problems, c.msg)
		}
	}
	for _, name := range settings.Keys() {
		if v, _ := text(name); v == "" && endpointSlotRe.MatchString(name) {
			problems = append(problems, fmt.Sprintf(probNoEndpoint, name))
		}
	}
	if v, ok := text("lastFailureCause"); ok && v != "" {
		problems = append(problems, fmt.Sprintf(probLastFailure, v))
	}
	return problems
}

// emailAddresses are the addresses of a BEmailAddressList text: ';' (or ',') separated,
// each "address" or "address|personal" (BEmailAddress.encodeToString) or "Personal <address>".
func emailAddresses(list string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(list, func(r rune) bool { return r == ';' || r == ',' }) {
		addr, _, _ := strings.Cut(strings.TrimSpace(part), "|")
		if open, close := strings.LastIndexByte(addr, '<'), strings.LastIndexByte(addr, '>'); open >= 0 && close > open {
			addr = addr[open+1 : close]
		}
		if addr = strings.TrimSpace(addr); addr != "" {
			out = append(out, addr)
		}
	}
	return out
}

// redact is a setting without user info, bot token or secret query value, cut to settingMaxRunes.
func redact(v string) string {
	for _, r := range redactions {
		v = r.re.ReplaceAllString(v, r.with)
	}
	if runes := []rune(v); len(runes) > settingMaxRunes {
		v = string(runes[:settingMaxRunes]) + "..."
	}
	return v
}
