package blackbox

// n4_users_audit (station-audits S14): the user accounts of a fake station, their roles, last
// login and the unused ones, read-only, and never a credential, a hash or a contact value.

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// Values planted on the station that must never reach a reply.
var uaSecrets = []string{"SECRET-PASSWORD-HASH", "AUTHENTICATOR-SALT", "DEEP-HASH-VALUE", "ana@plant.example",
	"+52-555-0100", "super"}

const uaAbsTime = "2006-01-02T15:04:05.000-07:00"

// uaRig is a servlet rig whose UserService holds admin (admin role, logged in yesterday, with a
// password slot, an authenticator holding secrets and contact data), off (disabled), old
// (expired), idle (never logged in) and ghost (a role the RoleService lacks); the SecurityHistory
// answers logins, when given.
func uaRig(t *testing.T, logins string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	r.Station.Do(func(st *boxstation.Station) {
		services := st.Component(st.Root(), "Services", "baja:ServiceContainer", "")
		roles := st.Component(services, "RoleService", "baja:RoleService", "")
		st.Component(roles, "admin", "baja:AdminRole", "")
		op := st.Component(roles, "operator", "baja:Role", "")
		op.Children = append(op.Children, &boxstation.Node{Name: "permissions", Type: "baja:PermissionsMap", Value: "1=r"})
		users := st.Component(services, "UserService", "baja:UserService", "")
		users.Children = append(users.Children, &boxstation.Node{Name: "maxBadLoginsBeforeLockOut", Type: "baja:Integer", Value: "5"})
		user := func(name string, slots ...*boxstation.Node) *boxstation.Node {
			u := st.Component(users, name, "baja:User", "")
			u.Children = append(u.Children, slots...)
			return u
		}
		str := func(name, v string) *boxstation.Node {
			return &boxstation.Node{Name: name, Type: "baja:String", Value: v}
		}
		admin := user("admin", str("fullName", "Ana Admin"), str("roles", "admin"), str("email", "ana@plant.example"),
			str("cellPhoneNumber", "+52-555-0100"), &boxstation.Node{Name: "password", Type: "baja:Password", Value: "SECRET-PASSWORD-HASH"},
			&boxstation.Node{Name: "permissions", Type: "baja:PermissionsMap", Value: "super"})
		auth := st.Component(admin, "authenticator", "baja:PasswordAuthenticator", "")
		auth.Value = "AUTHENTICATOR-SALT"
		auth.Children = append(auth.Children, &boxstation.Node{Name: "passwordHash", Type: "baja:String", Value: "DEEP-HASH-VALUE"})
		// a decoded boolean, which the fake would otherwise omit at the baja:Boolean default
		user("off", str("roles", "operator"), &boxstation.Node{Name: "enabled", Type: "baja:Boolean", Value: false})
		user("old", str("roles", "operator"), &boxstation.Node{Name: "expiration", Type: "baja:AbsTime", Value: "2020-01-01T00:00:00.000-06:00"})
		user("idle", str("roles", "operator"))
		user("ghost", str("roles", "operator,nobody"), &boxstation.Node{Name: "lockOut", Type: "baja:Boolean", Value: "true"})
	})
	if logins != "" {
		routeOrds(sv, boxstation.OrdRoute{Needle: "SecurityHistory", Body: []byte(logins)})
	}
	return r, sv
}

// uaLogins is the SecurityHistory answer: admin yesterday (and earlier), off, old and ghost
// within the week, as a localized station heads the columns.
func uaLogins() string {
	at := func(d time.Duration) string { return time.Now().Add(-d).Format(uaAbsTime) }
	return "Codificar,Usuario\n" + at(24*time.Hour) + ",admin\n" + at(48*time.Hour) + ",ghost\n" +
		at(72*time.Hour) + ",off\n" + at(96*time.Hour) + ",old\n" + at(400*24*time.Hour) + ",admin\n"
}

func uaUser(t *testing.T, out map[string]any, name string) map[string]any {
	t.Helper()
	for _, u := range objects(out["users"]) {
		if u["user"] == name {
			return u
		}
	}
	t.Fatalf("no user %s in %s", name, jsonOf(t, out["users"]))
	return nil
}

func uaFlags(t *testing.T, out map[string]any, name string) string {
	t.Helper()
	return jsonOf(t, uaUser(t, out, name)["flags"])
}

func TestUsersAuditClassifiesEachAccount(t *testing.T) {
	r, sv := uaRig(t, uaLogins())
	out := r.Srv.OK("n4_users_audit", nil)
	want := map[string]string{"admin": `["admin"]`, "off": `["disabled"]`, "old": `["expired"]`,
		"idle": `["never_logged_in","unused"]`, "ghost": `["locked","unknown_role"]`}
	for name, flags := range want {
		if got := uaFlags(t, out, name); got != flags {
			t.Errorf("%s flags = %s, want %s", name, got, flags)
		}
	}
	admin := uaUser(t, out, "admin")
	if admin["full_name"] != "Ana Admin" || admin["admin_role"] != true || admin["role_resolves"] != true ||
		admin["auth_scheme"] != "DigestScheme" || admin["expiration"] != nil || admin["last_login_source"] != "SecurityHistory" {
		t.Errorf("admin = %s", jsonOf(t, admin))
	}
	if last, _ := admin["last_login"].(string); last == "" || !strings.HasPrefix(last, time.Now().Add(-24*time.Hour).Format("2006-01-02")) {
		t.Errorf("admin last_login = %v (the newest record wins)", admin["last_login"])
	}
	if c, _ := admin["contact"].(map[string]any); len(c) != 2 || c["email"] != "<set>" || c["cell_phone"] != "<set>" {
		t.Errorf("admin contact = %v", admin["contact"])
	}
	if g := uaUser(t, out, "ghost"); jsonOf(t, g["unresolved_roles"]) != `["nobody"]` || g["role_resolves"] != false {
		t.Errorf("ghost = %s", jsonOf(t, g))
	}
	if u := jsonOf(t, out["unused"]); u != `[{"last_login":null,"user":"idle"}]` {
		t.Errorf("unused = %s", u)
	}
	counts := `{"admin":1,"disabled":1,"expired":1,"locked":1,"never_logged_in":1,"unknown_role":1,"unused":1,"users":5}`
	if c := jsonOf(t, out["counts"]); c != counts {
		t.Errorf("counts = %s", c)
	}
	masked := jsonOf(t, out["masked_slots"])
	for _, name := range []string{"authenticator", "password", "permissions", "email", "cellPhoneNumber"} {
		if !strings.Contains(masked, `"`+name+`"`) {
			t.Errorf("masked_slots %s lacks %s", masked, name)
		}
	}
	if ords := strings.Join(ordRequests(sv), "\n"); !strings.Contains(ords, "/FakeStation/SecurityHistory") ||
		!strings.Contains(ords, "operation = 'Login'") {
		t.Errorf("ORDs sent %s", ords)
	}
}

func TestUsersAuditNeverRepliesACredentialOrContactValue(t *testing.T) {
	r, _ := uaRig(t, uaLogins())
	raw := jsonOf(t, r.Srv.OK("n4_users_audit", nil))
	for _, secret := range uaSecrets {
		if strings.Contains(raw, secret) {
			t.Errorf("the reply holds %q: %s", secret, raw)
		}
	}
}

func TestUsersAuditWithoutLoginRecordsLeavesLastLoginUnknown(t *testing.T) {
	r, _ := uaRig(t, "")
	out := r.Srv.OK("n4_users_audit", Args{"max_users": 2})
	idle := objects(out["users"])
	if len(idle) != 2 || out["truncated"] != true {
		t.Fatalf("max_users 2: %s", jsonOf(t, out))
	}
	for _, u := range idle {
		if u["last_login"] != nil || u["last_login_source"] != nil || strings.Contains(jsonOf(t, u["flags"]), "never") {
			t.Errorf("unread history: %s", jsonOf(t, u))
		}
	}
	if notes := jsonOf(t, out["notes"]); !strings.Contains(notes, "last login unavailable") ||
		!strings.Contains(notes, "max_users") {
		t.Errorf("notes = %s", notes)
	}
}
