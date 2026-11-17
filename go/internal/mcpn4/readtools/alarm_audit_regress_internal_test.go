package readtools

// Regression tests of the PR #788 verifier: no recipient value outside the allowlist is
// returned (B1, B2), one source in both stored spellings is one source (B3), "more than N days"
// is strict, and an offnormal source is listed once.

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// slotNodes is a recipient load: name, type, value ("\x00" for no value) per slot.
func slotNodes(slots ...[3]string) *pyjson.Object {
	n := pyjson.NewObject()
	n.Set("", pyjson.NewObject())
	for _, s := range slots {
		o := pyjson.NewObject()
		if s[1] != "" {
			o.Set("t", s[1])
		}
		if s[2] != "\x00" {
			o.Set("v", s[2])
		}
		n.Set(s[0], o)
	}
	return n
}

func TestRecipientAuditReturnsNoValueOutsideTheAllowlist(t *testing.T) {
	leaky := [][3]string{
		{"accessKey", "baja:String", "L1-LEAK"},
		{"authorization", "baja:String", "Bearer L2-LEAK"},
		{"botId", "baja:String", "123456:L3-LEAK"},
		{"url", "baja:String", "https://h/x?access_token=L4-LEAK&auth=L5-LEAK&sig=L6-LEAK"},
		{"url2", "baja:String", "https://hooks.example/services/T0/B0/L7-LEAKPATH"},
		{"pin", "baja:String", "L8-1234"},
		{"cred", "acme:SecureString", "L9-LEAK"},
		{"blob", "baja:Component", "L10-COMPONENT-V auth/password=x"},
		{"hook1", "baja:String", "https://user:pa/ss@host/L11-LEAK"},
		{"hook2", "baja:String", "https://user:p@ss@host/L12-LEAK"},
		{"hook3", "baja:String", "https://user:p ss@host/L13-LEAK"},
		{"hook4", "baja:String", "HTTPS://USER:PW@HOST/X?Token=L14-LEAK&API_KEY=L15-LEAK"},
		{"lastFailureCause", "baja:String", "POST https://h/bot1:L16-LEAK failed"},
		{"subject", "baja:Format", "L17-LEAK"},
	}
	for _, kind := range []string{"acme:WebhookRecipient", "email:EmailRecipient", "alarm:ConsoleRecipient"} {
		out := pyjson.Repr(recipientAudit(slotNodes(leaky...), kind))
		for i := 1; i <= 17; i++ {
			if mark := "L" + itoa(i) + "-"; strings.Contains(out, mark) {
				t.Errorf("%s returned %q: %s", kind, mark, out)
			}
		}
		if !strings.Contains(out, "'accessKey': '<set>'") || !strings.Contains(out, "'url': '<set>'") {
			t.Errorf("%s: a set slot is not shown as <set>: %s", kind, out)
		}
	}
}

func itoa(i int) string { return strings.TrimSpace(pyjson.Repr(i)) }

func TestTallySourcesJoinsBothStoredSpellingsOfOneSource(t *testing.T) {
	var rows []*pyjson.Object
	for i := 0; i < 6; i++ {
		src := "slot:/Drivers/Net/Pump"
		if i%2 == 0 {
			src = "local:|station:|slot:/Drivers/Net/Pump"
		}
		rows = append(rows, row("source", src, "normal_time", "2026-11-17T08:00:00.000-06:00"))
	}
	tallies := tallySources(rows)
	if got := pyjson.Repr(chatteringSources(tallies, 10)); got != "[{'source': 'station:|slot:/Drivers/Net/Pump', 'transitions': 12, 'records': 6}]" {
		t.Errorf("chattering %s", got)
	}
	if got := pyjson.Repr(busiestSources(tallies)); got != "[{'source': 'station:|slot:/Drivers/Net/Pump', 'count': 6}]" {
		t.Errorf("top %s", got)
	}
}

func TestOpenFindingsIsStrictOnDaysAndListsAnOffnormalSourceOnce(t *testing.T) {
	now := time.Date(2026, 11, 17, 18, 0, 0, 0, time.UTC)
	rows := []*pyjson.Object{
		row("source", "slot:/P", "source_state", "offnormal", "ack_state", "unacked", "alarm_class", "c", "alarm_time", "2026-11-14T12:00:00.000-06:00"),
		row("source", "local:|station:|slot:/P", "source_state", "offnormal", "ack_state", "acked", "alarm_class", "c", "alarm_time", "2026-11-12T12:00:00.000-06:00"),
	}
	off, unacked, _ := openFindings(rows, now, 3) // the unacked record is exactly 3 days old
	if got := pyjson.Repr(off); got != "[{'source': 'station:|slot:/P', 'source_state': 'offnormal', 'alarm_class': 'c', "+
		"'since': '2026-11-12T12:00:00.000-06:00', 'records': 2}]" {
		t.Errorf("offnormal %s", got)
	}
	if len(unacked) != 0 {
		t.Errorf("exactly 3 days is not more than 3: %s", pyjson.Repr(unacked))
	}
}
