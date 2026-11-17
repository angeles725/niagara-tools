package readtools

import (
	"fmt"
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_users_audit (station-audits S14): the station's user accounts, their roles, last login and
// the accounts nobody uses, read-only and without a credential. One load of the UserService (its
// users and their slots, two levels: the authenticator's own slots are never loaded), one of the
// RoleService, and one fixed query of the SecurityHistory login records.

// The services the audit reads (BUserService and BRoleService live under Services).
const (
	userServiceOrd = "station:|slot:/Services/UserService"
	roleServiceOrd = "station:|slot:/Services/RoleService"
)

// Defaults and bounds of n4_users_audit.
const (
	usersDefaultUnusedDays = 90
	usersDefaultMax        = 500
	usersLoginRecords      = 10000 // newest login records read
	usersTimeout           = 60
)

// Notes of the reply, as data.
const (
	noteNoRoles   = "role_resolves and admin_role are null: the RoleService %s could not be read: %s"
	noteNoLogins  = "last login unavailable: the security history %s could not be read: %s"
	noteNoRecords = "last login unavailable: no login records in %s"
	notePartial   = "only the newest %d login records were read (back to %s): a user with no login among them " +
		"has last_login null and no_recorded_login means no login since then"
	noteTruncated = "only the first %d of %d users are audited (max_users)"
)

// UsersAudit is n4_users_audit.
func (t *Tools) UsersAudit(args *pyjson.Object) (*pyjson.Object, error) {
	unusedDays, maxUsers := intArg(args, "unused_days", usersDefaultUnusedDays), intArg(args, "max_users", usersDefaultMax)
	start := t.Ctx.Clock()
	nodes, err := t.Ctx.Session.Client.(Client).LoadTree(userServiceOrd, 2, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, name := range box.Children(nodes, "").Keys() {
		if userTypes[pyjson.Str(get(box.Node(nodes, name), "t"))] {
			names = append(names, name)
		}
	}
	notes := []any{}
	if len(names) > maxUsers {
		notes = append(notes, fmt.Sprintf(noteTruncated, maxUsers, len(names)))
	}
	roles, note := t.usersRoles()
	if note != "" {
		notes = append(notes, note)
	}
	logins, note, err := t.usersLogins()
	if err != nil {
		return nil, err
	}
	if note != "" {
		notes = append(notes, note)
	}
	now := bql.Now()
	users, unused, read := []any{}, []any{}, []userSlots{}
	counts := map[string]int{}
	for i, name := range names {
		if i == maxUsers {
			break
		}
		slots := readUserSlots(box.Children(nodes, name))
		read = append(read, slots)
		entry, isUnused := auditUser(name, slots, roles, logins, now, unusedDays)
		users = append(users, entry)
		if isUnused {
			unused = append(unused, obj("user", name, "last_login", get(entry, "last_login")))
		}
		for _, f := range get(entry, "flags").([]any) {
			counts[f.(string)]++
		}
	}
	t.recordRead("n4_users_audit", "", args, len(users))
	countObj := obj("users", len(users))
	for _, f := range []string{flagDisabled, flagExpired, flagLocked, flagNever, flagUnused, flagAdmin, flagUnknown} {
		countObj.Set(f, counts[f])
	}
	var since any
	if !logins.oldest.IsZero() {
		since = logins.since
	}
	return obj("users", users, "unused", unused, "unused_days", unusedDays, "logins_since", since, "counts", countObj,
		"masked_slots", maskedUnion(read), "notes", notes, "truncated", len(names) > maxUsers,
		"elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000))), nil
}

// usersRoles loads the RoleService one level of roles and their slots; a failed load is a note.
func (t *Tools) usersRoles() (roleView, string) {
	nodes, err := t.Ctx.Session.Client.(Client).LoadTree(roleServiceOrd, 2, box.LoadOptions{})
	if err != nil {
		return roleView{}, fmt.Sprintf(noteNoRoles, roleServiceOrd, err.Error())
	}
	view := roleView{read: true, admin: map[string]bool{}}
	for _, name := range box.Children(nodes, "").Keys() {
		kind := pyjson.Str(get(box.Node(nodes, name), "t"))
		if !strings.HasSuffix(kind, "Role") {
			continue
		}
		kids := box.Children(nodes, name)
		view.admin[name] = kind == "baja:AdminRole" || box.ChildValue(kids, "enabled", "true") != "false" &&
			box.ChildValue(kids, "permissions", "") == "super"
	}
	return view, ""
}

// usersLogins reads the newest successful login of each user from the SecurityHistory; an
// unreadable history is a note, except an AuthError, which fails the call.
func (t *Tools) usersLogins() (loginView, string, error) {
	id, err := bql.ResolveHistoryId(bql.SecurityHistoryName, t.Ctx.Session.StationName)
	if err != nil {
		return loginView{}, fmt.Sprintf(noteNoLogins, bql.SecurityHistoryName, err.Error()), nil
	}
	ord, err := bql.ComposeFilteredHistoryOrd(id, "", bql.LoginRecordsQuery, bql.LoginWhere, bql.AuditOrder)
	if err == nil {
		var columns []string
		var rows []*pyjson.Object
		var truncated bool
		if columns, rows, truncated, _, err = t.fetchHistory(ord, usersLoginRecords, usersTimeout); err == nil {
			var keyed []*pyjson.Object
			if keyed, err = bql.ByPosition(columns, rows, []string{"timestamp", "user"}); err == nil {
				return loginsOf(id, keyed, truncated)
			}
		}
	}
	if e := unreadable(id, err); e == err {
		return loginView{}, "", err
	}
	return loginView{}, fmt.Sprintf(noteNoLogins, id, err.Error()), nil
}

// loginsOf is the login view of rows (newest first) and its note.
func loginsOf(id string, rows []*pyjson.Object, truncated bool) (loginView, string, error) {
	view, oldest := loginView{read: true, complete: !truncated, last: map[string]string{}}, ""
	for _, row := range rows {
		text, user := cell(row, "timestamp"), cell(row, "user")
		at, ok := bql.ParseAbsTime(text)
		if !ok {
			continue
		}
		if _, seen := view.last[user]; !seen {
			view.last[user] = text
		}
		view.oldest, oldest = at, text
	}
	view.since = oldest
	switch {
	case truncated:
		return view, fmt.Sprintf(notePartial, len(rows), oldest), nil
	case len(view.last) == 0:
		return view, fmt.Sprintf(noteNoRecords, id), nil
	}
	return view, "", nil
}
