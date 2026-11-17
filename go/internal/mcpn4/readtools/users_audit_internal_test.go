package readtools

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_users_audit (station-audits S14): the allowlist and the verdicts, as pure functions.

var usersNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

// userKids is a user's loaded children, each name to its value.
func userKids(values map[string]string) *pyjson.Object {
	kids := pyjson.NewObject()
	for name, v := range values {
		kids.Set(name, obj("t", "baja:String", "v", v))
	}
	return kids
}

func TestUserSlotsReturnOnlyTheAllowlistAndMaskEverythingElse(t *testing.T) {
	kids := userKids(map[string]string{"fullName": "Ana", "roles": "operator", "email": "ana@plant.example",
		"cellPhoneNumber": " ", "password": "s3cr3t-hash", "authenticator": "pbkdf2$SALT$HASH", "permissions": "super"})
	kids.Set("noValue", obj("t", "baja:Component"))
	u := readUserSlots(kids)
	if u.safe["fullName"] != "Ana" || u.safe["roles"] != "operator" || u.safe["enabled"] != "true" ||
		u.safe["expiration"] != "null" || u.safe["authenticationSchemeName"] != "DigestScheme" {
		t.Errorf("safe = %v", u.safe)
	}
	if got := pyjson.Repr(u.contact); got != "{'email': '<set>', 'cell_phone': ''}" {
		t.Errorf("contact = %s", got)
	}
	masked := pyjson.Repr(maskedUnion([]userSlots{u}))
	for _, name := range []string{"password", "authenticator", "permissions", "noValue", "email", "cellPhoneNumber"} {
		if !strings.Contains(masked, "'"+name+"'") {
			t.Errorf("masked_slots %s lacks %s", masked, name)
		}
	}
	for _, secret := range []string{"s3cr3t", "SALT", "HASH", "ana@plant"} {
		if strings.Contains(pyjson.Repr(u.contact)+pyjson.Repr(u.safe), secret) {
			t.Errorf("a masked value leaked: %s", secret)
		}
	}
}

func TestSplitRolesTrimsAndDropsBlanks(t *testing.T) {
	if got := splitRoles(" admin, ,operator ,"); !reflect.DeepEqual(got, []string{"admin", "operator"}) {
		t.Errorf("roles %q", got)
	}
	if got := splitRoles(""); len(got) != 0 {
		t.Errorf("empty roles %q", got)
	}
}

// verdict audits one user against roles admin (super) and operator, the logins given.
func verdict(t *testing.T, values map[string]string, logins loginView) (*pyjson.Object, bool) {
	t.Helper()
	roles := roleView{read: true, admin: map[string]bool{"admin": true, "operator": false}}
	return auditUser("u", readUserSlots(userKids(values)), roles, logins, usersNow, 90)
}

func flagsOf(entry *pyjson.Object) string { return pyjson.Repr(get(entry, "flags")) }

func TestAuditUserFlagsEachCondition(t *testing.T) {
	logins := loginView{read: true, complete: true, last: map[string]string{"u": "2026-11-16T08:00:00.000-06:00"}}
	cases := []struct {
		values map[string]string
		flags  string
	}{
		{map[string]string{"roles": "admin"}, "['admin']"},
		{map[string]string{"enabled": "false", "roles": "operator"}, "['disabled']"},
		{map[string]string{"expiration": "2026-01-01T00:00:00.000-06:00", "roles": "operator"}, "['expired']"},
		{map[string]string{"lockOut": "true", "roles": "operator"}, "['locked']"},
		{map[string]string{"roles": "operator,ghost"}, "['unknown_role']"},
	}
	for _, c := range cases {
		entry, unused := verdict(t, c.values, logins)
		if flagsOf(entry) != c.flags || unused {
			t.Errorf("%v: flags %s unused %v", c.values, flagsOf(entry), unused)
		}
	}
	entry, _ := verdict(t, map[string]string{"roles": "operator,ghost"}, logins)
	if get(entry, "role_resolves") != false || pyjson.Repr(get(entry, "unresolved_roles")) != "['ghost']" ||
		get(entry, "last_login") != "2026-11-16T08:00:00.000-06:00" || get(entry, "last_login_source") != "SecurityHistory" {
		t.Errorf("entry %s", pyjson.Repr(entry))
	}
}

func TestAuditUserLoginVerdicts(t *testing.T) {
	old := loginView{read: true, complete: true, last: map[string]string{"u": "2026-01-01T00:00:00.000-06:00"},
		oldest: usersNow.Add(-400 * 24 * time.Hour)}
	if entry, unused := verdict(t, map[string]string{}, old); !unused || flagsOf(entry) != "['unused']" {
		t.Errorf("old login: %s %v", flagsOf(entry), unused)
	}
	// An empty SecurityHistory proves nothing about any user: no flag, not unused (review R3).
	empty := loginView{read: true, complete: true, last: map[string]string{}}
	if entry, unused := verdict(t, map[string]string{}, empty); unused || flagsOf(entry) != "[]" {
		t.Errorf("empty history: %s %v", flagsOf(entry), unused)
	}
	// Every record read, but the history rolled 5 days back: a login 10 days ago may have rolled
	// off, so the user has no recorded login and is NOT unused within 90 days (review R3).
	rolled := loginView{read: true, complete: true, last: map[string]string{}, oldest: usersNow.Add(-5 * 24 * time.Hour)}
	if entry, unused := verdict(t, map[string]string{}, rolled); unused || flagsOf(entry) != "['no_recorded_login']" {
		t.Errorf("rolled history: %s %v", flagsOf(entry), unused)
	}
	if entry, unused := verdict(t, map[string]string{"enabled": "false"}, rolled); unused ||
		flagsOf(entry) != "['disabled', 'no_recorded_login']" {
		t.Errorf("disabled, no login: %s %v", flagsOf(entry), unused)
	}
	recent := loginView{read: true, last: map[string]string{}, oldest: usersNow.Add(-24 * time.Hour)}
	if entry, unused := verdict(t, map[string]string{}, recent); unused || flagsOf(entry) != "['no_recorded_login']" {
		t.Errorf("partial records within the window: %s %v", flagsOf(entry), unused)
	}
	deep := loginView{read: true, last: map[string]string{}, oldest: usersNow.Add(-200 * 24 * time.Hour)}
	if entry, unused := verdict(t, map[string]string{}, deep); !unused || flagsOf(entry) != "['no_recorded_login', 'unused']" {
		t.Errorf("records past the window: %s %v", flagsOf(entry), unused)
	}
	entry, unused := verdict(t, map[string]string{}, loginView{})
	if unused || get(entry, "last_login") != nil || get(entry, "last_login_source") != nil || flagsOf(entry) != "[]" {
		t.Errorf("unread history: %s", pyjson.Repr(entry))
	}
}

func TestAuditUserWithoutRoleServiceLeavesRoleVerdictsNull(t *testing.T) {
	entry, _ := auditUser("u", readUserSlots(userKids(map[string]string{"roles": "ghost"})), roleView{},
		loginView{}, usersNow, 90)
	if get(entry, "role_resolves") != nil || get(entry, "admin_role") != nil || flagsOf(entry) != "[]" {
		t.Errorf("entry %s", pyjson.Repr(entry))
	}
}

func TestLoginsOfKeepsTheNewestPerUser(t *testing.T) {
	rows := []*pyjson.Object{obj("timestamp", "2026-11-16T10:00:00.000-06:00", "user", "ana"),
		obj("timestamp", "2026-11-15T10:00:00.000-06:00", "user", "bob"),
		obj("timestamp", "2026-11-14T10:00:00.000-06:00", "user", "ana")}
	view, note, _ := loginsOf("/S/SecurityHistory", rows, false)
	if !view.complete || view.last["ana"] != "2026-11-16T10:00:00.000-06:00" || note != "" {
		t.Errorf("view %v note %q", view, note)
	}
	view, note, _ = loginsOf("/S/SecurityHistory", rows, true)
	if view.complete || !strings.Contains(note, "2026-11-14T10:00:00.000-06:00") {
		t.Errorf("partial note %q", note)
	}
	if _, note, _ = loginsOf("/S/SecurityHistory", nil, false); !strings.Contains(note, "no login records") {
		t.Errorf("empty note %q", note)
	}
}

// An allowlisted slot that is typed baja:Password or whose value is not a scalar is masked, never
// echoed (verifier note on PR #807: the pyjson.Str fallback would print an object's Repr).
func TestUserSlotsMaskAnAllowlistedSlotWithAPasswordTypeOrANonScalarValue(t *testing.T) {
	kids := pyjson.NewObject()
	kids.Set("fullName", obj("t", "baja:Password", "v", "PW-TYPED-SECRET"))
	kids.Set("prototypeName", obj("t", "baja:String", "v", obj("t", "baja:Password", "v", "PW-NESTED-SECRET")))
	kids.Set("roles", obj("t", "baja:String", "v", "admin"))
	u := readUserSlots(kids)
	if u.safe["fullName"] != "" || u.safe["prototypeName"] != "" || u.safe["roles"] != "admin" {
		t.Errorf("safe = %v", u.safe)
	}
	masked := pyjson.Repr(maskedUnion([]userSlots{u}))
	for _, name := range []string{"fullName", "prototypeName"} {
		if !strings.Contains(masked, "'"+name+"'") {
			t.Errorf("masked_slots %s lacks %s", masked, name)
		}
	}
	if strings.Contains(pyjson.Repr(u.safe), "SECRET") {
		t.Errorf("secret reached the safe values: %v", u.safe)
	}
}
