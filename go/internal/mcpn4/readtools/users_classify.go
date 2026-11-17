package readtools

import (
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The verdicts of n4_users_audit (station-audits S14), from a user's allowlisted slots, the
// RoleService roles and the login records. A verdict the evidence cannot support is null, never
// guessed: role_resolves and admin_role need the RoleService, last_login needs the SecurityHistory.

// The flags of a user, as data.
const (
	flagDisabled = "disabled"
	flagExpired  = "expired"
	flagLocked   = "locked"
	flagNever    = "no_recorded_login"
	flagUnused   = "unused"
	flagAdmin    = "admin"
	flagUnknown  = "unknown_role"
)

// roleView is what the RoleService says: each role name and whether it grants every permission
// (baja:AdminRole, or an enabled role whose permissions map is "super": BUser.getPermissions).
// read is false when the RoleService could not be read.
type roleView struct {
	read  bool
	admin map[string]bool
}

// loginView is the newest login per user name from the SecurityHistory. read is false when the
// history could not be read; complete is false when only the newest records were read, oldest
// being the oldest of them.
type loginView struct {
	read, complete bool
	last           map[string]string // BAbsTime text as the station wrote it
	oldest         time.Time         // the oldest record read; zero when none
	since          string            // oldest as the station wrote it (logins_since)
}

// auditUser is the reply entry of user name, and whether it is unused.
func auditUser(name string, u userSlots, roles roleView, logins loginView, now time.Time, unusedDays int) (*pyjson.Object, bool) {
	// EqualFold: a station sends "true"/"false"; a decoded boolean reads True/False.
	enabled, locked := !strings.EqualFold(u.safe["enabled"], "false"), strings.EqualFold(u.safe["lockOut"], "true")
	flags := []any{}
	if !enabled {
		flags = append(flags, flagDisabled)
	}
	var expiration any
	if at, ok := bql.ParseAbsTime(u.safe["expiration"]); ok {
		expiration = u.safe["expiration"]
		if at.Before(now) {
			flags = append(flags, flagExpired)
		}
	}
	if locked {
		flags = append(flags, flagLocked)
	}
	names := splitRoles(u.safe["roles"])
	var resolves, admin any
	unresolved := []any{}
	if roles.read {
		isAdmin := false
		for _, r := range names {
			grants, ok := roles.admin[r]
			if !ok {
				unresolved = append(unresolved, r)
			}
			isAdmin = isAdmin || grants
		}
		resolves, admin = len(unresolved) == 0, isAdmin
	}
	lastLogin, source, unused := loginVerdict(name, enabled, logins, now, unusedDays, &flags)
	if unused {
		flags = append(flags, flagUnused)
	}
	if admin == true {
		flags = append(flags, flagAdmin)
	}
	if len(unresolved) > 0 {
		flags = append(flags, flagUnknown)
	}
	return obj("user", name, "full_name", u.safe["fullName"], "enabled", enabled, "locked_out", locked,
		"expiration", expiration, "roles", strs(names), "role_resolves", resolves, "unresolved_roles", unresolved,
		"admin_role", admin, "auth_scheme", u.safe["authenticationSchemeName"], "prototype", u.safe["prototypeName"],
		"last_login", lastLogin, "last_login_source", source, "contact", u.contact, "flags", flags), unused
}

// loginVerdict is the user's last login and its source (nil when unknown) and whether an enabled
// user is unused: no login within unusedDays. The SecurityHistory has a capacity and rolls, so
// reading every record proves nothing older than its oldest record: no_recorded_login only says
// no Login record since that oldest record (logins_since), and an empty history proves nothing.
// A user with no login record is unused only when the records read reach back past the cut-off.
func loginVerdict(name string, enabled bool, logins loginView, now time.Time, unusedDays int, flags *[]any) (any, any, bool) {
	if !logins.read {
		return nil, nil, false
	}
	cutoff := now.Add(-time.Duration(unusedDays) * 24 * time.Hour)
	if text, ok := logins.last[name]; ok {
		at, _ := bql.ParseAbsTime(text)
		return text, bql.SecurityHistoryName, enabled && at.Before(cutoff)
	}
	if logins.oldest.IsZero() {
		return nil, bql.SecurityHistoryName, false
	}
	*flags = append(*flags, flagNever)
	return nil, bql.SecurityHistoryName, enabled && logins.oldest.Before(cutoff)
}
