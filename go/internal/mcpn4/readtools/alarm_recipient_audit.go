package readtools

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The recipient side of n4_list_alarm_classes audit=true (station-audits S5): each recipient's
// settings as its routeAlarm load already holds them, and the problems that stop delivery.
// Fail closed (PR #788 verifier B1, B2): a value is returned only for a slot on the allowlist of
// a known recipient type; every other slot, and every slot of an unknown type, shows only
// "<set>" or "". The problem checks read the raw values and never quote one off the allowlist.

// settingMaxRunes bounds a returned setting.
const settingMaxRunes = 200

// The text of a slot whose value is not returned.
const (
	maskedSet   = "<set>"
	maskedEmpty = ""
)

var (
	// routeSlots are the non-secret delivery slots of every BAlarmRecipient.
	routeSlots = []string{"transitions", "daysOfWeek", "timeRange", "routeAcks"}
	// safeSlots are the slots of a known recipient type whose value is returned.
	safeSlots = map[string][]string{
		"email:EmailRecipient":   append([]string{"to", "cc", "bcc", "emailAccount", "language"}, routeSlots...),
		"alarm:StationRecipient": append([]string{"remoteStation", "lastFailureCause", "status", "persistent", "retryInterval"}, routeSlots...),
		"alarm:ConsoleRecipient": append([]string{"status", "defaultTimeRange"}, routeSlots...),
	}
	// emptyDefaults are the route slots a load omits at their type default, which is empty
	// (BEmailRecipient to, cc, bcc and emailAccount; BStationRecipient remoteStation).
	emptyDefaults = map[string][]string{"email:EmailRecipient": {"to", "cc", "bcc", "emailAccount"},
		"alarm:StationRecipient": {"remoteStation"}}
	// endpointSlotRe names a slot a custom recipient (webhook, Telegram) delivers to.
	endpointSlotRe = regexp.MustCompile(`(?i)(url|chatid|webhook)$`)
	emailRe        = regexp.MustCompile(`^[^@\s<>"(),;:|]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}$`)
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
	probLastFailed   = "the last delivery failed (lastFailureCause is set; its text is not returned for this type)"
)

// recipientAudit is {settings, at_default, masked_slots, problems} of the recipient whose load
// (two levels from the recipient) is nodes, kind its type.
func recipientAudit(nodes *pyjson.Object, kind string) *pyjson.Object {
	raw, settings, masked, atDefault := map[string]string{}, pyjson.NewObject(), []any{}, []any{}
	safe := map[string]bool{}
	for _, name := range safeSlots[kind] {
		safe[name] = true
	}
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		node := box.Node(kids, name)
		v, has := node.Get("v")
		if !has || isLinkType(get(node, "t")) {
			continue
		}
		raw[name] = pyjson.Str(v)
		switch {
		case safe[name]:
			settings.Set(name, cutSetting(raw[name]))
		case strings.TrimSpace(raw[name]) == "":
			settings.Set(name, maskedEmpty)
			masked = append(masked, name)
		default:
			settings.Set(name, maskedSet)
			masked = append(masked, name)
		}
	}
	for _, name := range emptyDefaults[kind] {
		if _, ok := raw[name]; !ok {
			raw[name] = ""
			settings.Set(name, "")
			atDefault = append(atDefault, name)
		}
	}
	return obj("settings", settings, "at_default", atDefault, "masked_slots", masked,
		"problems", recipientProblems(raw, settings.Keys(), kind, safe))
}

// recipientProblems are the reasons the recipient cannot deliver, from its raw values (names in
// order); a problem quotes a value only when safe holds its slot.
func recipientProblems(raw map[string]string, names []string, kind string, safe map[string]bool) []any {
	problems := []any{}
	text := func(name string) (string, bool) {
		v, ok := raw[name]
		return strings.TrimSpace(v), ok
	}
	if kind == "email:EmailRecipient" {
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
	if v, ok := text("remoteStation"); ok && v == "" && kind == "alarm:StationRecipient" {
		problems = append(problems, probNoStation)
	}
	for _, c := range []struct{ slot, msg string }{{"transitions", probNoTransition}, {"daysOfWeek", probNoDay}} {
		if v, ok := text(c.slot); ok && (v == "0" || v == "{}") {
			problems = append(problems, c.msg)
		}
	}
	for _, name := range names {
		if v, _ := text(name); v == "" && endpointSlotRe.MatchString(name) {
			problems = append(problems, fmt.Sprintf(probNoEndpoint, name))
		}
	}
	if v, ok := text("lastFailureCause"); ok && v != "" {
		if safe["lastFailureCause"] {
			problems = append(problems, fmt.Sprintf(probLastFailure, cutSetting(v)))
		} else {
			problems = append(problems, probLastFailed)
		}
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

// cutSetting is v cut to settingMaxRunes.
func cutSetting(v string) string {
	if runes := []rune(v); len(runes) > settingMaxRunes {
		return string(runes[:settingMaxRunes]) + "..."
	}
	return v
}
