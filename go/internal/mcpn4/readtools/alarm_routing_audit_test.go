package readtools_test

// n4_list_alarm_classes audit=true through the in-memory station: the unrouted classes, each
// recipient's settings and problems, and no credential in the reply.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// auditServer is classesServer plus a class nobody routes (Orphan), a mail recipient whose
// to-list is empty and whose cc holds a bad address and two credential slots, a console routing no
// transition, and a webhook recipient with an empty chat id and a failed delivery.
func auditServer(t *testing.T) *pyjson.Object {
	t.Helper()
	s := classesServer(t)
	s.Station.OrdRoutes[0].Body += "slot:/Services/AlarmService/Orphan,Orphan,toOffnormal=1;toFault=1;toNormal=1;toAlert=1;,{},0,0,0,0\n"
	s.Station.OrdRoutes[1].Body += "slot:/Services/AlarmService/Hook,acme:WebhookRecipient\n"
	mail := s.Station.Trees["station:|slot:"+servicePath+"/Mail"]
	for _, e := range [][]any{{"cc", "v", "ops@plant.example;not-an-address|Ops"}, {"emailAccount", "v", "Outgoing"},
		{"accountPassword", "t", "baja:Password", "v", "hunter2"}, {"secretKey", "v", "s3cr3t"}} {
		mail.Set(e[0].(string), field(nodes(e), e[0].(string)))
	}
	console := s.Station.Trees["station:|slot:"+servicePath+"/Console"]
	console.Set("transitions", field(nodes([]any{"transitions", "v", "0"}), "transitions"))
	s.Station.Trees["station:|slot:"+servicePath+"/Hook"] = nodes([]any{"", "h", "r3"},
		[]any{"url", "v", "https://user:pw@api.telegram.org/bot123:ABC-def/sendMessage"}, []any{"chatId", "v", ""},
		[]any{"lastFailureCause", "v", "HTTP 401"})
	return callOK(t, s, "n4_list_alarm_classes", "audit", true)
}

func TestListAlarmClassesAuditFindsUnroutedClassesAndBrokenRecipients(t *testing.T) {
	out := auditServer(t)
	if got := pyjson.Repr(field(out, "unrouted_classes")); got != "[{'path': '/Services/AlarmService/Orphan', 'name': 'Orphan'}]" {
		t.Errorf("unrouted %s", got)
	}
	if got := pyjson.Repr(field(out, "audit_counts")); got != "{'classes': 3, 'unrouted_classes': 1, 'recipients_read': 3, 'recipients_with_problems': 3}" {
		t.Errorf("counts %s", got)
	}
	if got := field(out, "unrouted_note"); !strings.HasPrefix(pyjson.Str(got), "1 routeAlarm link(s) have a source that is not a listed class and 0 recipient(s)") {
		t.Errorf("note %v", got)
	}
	recipients := field(out, "recipients").([]any)
	want := []string{
		"{'path': '/Services/AlarmService/Console', 'type': 'alarm:ConsoleRecipient', 'classes': ['defaultAlarmClass', 'Critical'], " +
			"'settings': {'transitions': '0'}, 'at_default': [], 'masked_slots': [], " +
			"'problems': ['transitions is empty: no alarm transition is routed']}",
		"{'path': '/Services/AlarmService/Mail', 'type': 'email:EmailRecipient', 'classes': ['Critical'], " +
			"'settings': {'cc': 'ops@plant.example;not-an-address|Ops', 'emailAccount': 'Outgoing', 'accountPassword': '<set>', " +
			"'secretKey': '<set>', 'to': '', 'bcc': ''}, 'at_default': ['to', 'bcc'], 'masked_slots': ['accountPassword', 'secretKey'], " +
			"'problems': ['cc holds \"not-an-address\", which is not an email address']}",
		"{'path': '/Services/AlarmService/Hook', 'type': 'acme:WebhookRecipient', 'classes': [], " +
			"'settings': {'url': '<set>', 'chatId': '', 'lastFailureCause': '<set>'}, " +
			"'at_default': [], 'masked_slots': ['url', 'chatId', 'lastFailureCause'], " +
			"'problems': ['chatId is empty: the recipient has nowhere to deliver', " +
			"'the last delivery failed (lastFailureCause is set; its text is not returned for this type)']}",
	}
	if len(recipients) != len(want) {
		t.Fatalf("recipients %s", pyjson.Repr(recipients))
	}
	for i, w := range want {
		if got := pyjson.Repr(recipients[i]); got != w {
			t.Errorf("recipient %d\n got %s\nwant %s", i, got, w)
		}
	}
	if text := pyjson.Repr(out); strings.Contains(text, "hunter2") || strings.Contains(text, "s3cr3t") || strings.Contains(text, "ABC-def") || strings.Contains(text, "HTTP 401") {
		t.Errorf("a credential reached the reply: %s", text)
	}
}

func TestListAlarmClassesAuditFlagsAMailRecipientWithNoAddress(t *testing.T) {
	s := classesServer(t)
	out := callOK(t, s, "n4_list_alarm_classes", "audit", true)
	mail := pyjson.Repr(field(out, "recipients").([]any)[1])
	if !strings.Contains(mail, "'problems': ['no email address: to, cc and bcc are empty']") ||
		!strings.Contains(mail, "'at_default': ['to', 'cc', 'bcc', 'emailAccount']") {
		t.Errorf("mail %s", mail)
	}
	if got := pyjson.Repr(field(out, "unrouted_classes")); got != "[]" {
		t.Errorf("unrouted %s", got)
	}
}
