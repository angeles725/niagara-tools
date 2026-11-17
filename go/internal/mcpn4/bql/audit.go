package bql

import (
	"fmt"
	"regexp"
	"strings"
)

// Audit trail reads (station-audits S7, "who changed what"): the station's AuditHistory, one
// BAuditRecord per audited change (history-rt BAuditRecord: timestamp, operation, target,
// slotName, oldValue, value, userName). BAuditHistoryService names it /<station>/AuditHistory
// (getHistoryId). ComplexSlotMap.audit stores target as the component's slot path body
// (/Folder/Point, no slot: scheme) and slotName as the property, or the property path a/b for a
// nested one. Every filter value is checked on its decoded form before it reaches the query.

// AuditHistoryName is the history name BAuditHistoryService gives the audit trail.
const AuditHistoryName = "AuditHistory"

// The fixed audit query: one row per record, read by position, the time as machine text.
const (
	AuditRecordsQuery = "select timestamp.encodeToString, operation, target, slotName, oldValue, value, userName"
	AuditOrder        = " order by timestamp desc"
	// auditTargetClause matches a target at a slot path or below it.
	auditTargetClause = "(target like '%[1]s' or target like '%[1]s/*')"
)

// AuditOperations are the AuditEvent operations the AuditHistory keeps (baja AuditEvent
// constants; Login, Logout, Login Failure, Logout (Timeout) and Recategorized go to the security
// audit history instead: BAuditHistoryService.checkSecurityAudit).
var AuditOperations = []string{"Changed", "Added", "Removed", "Renamed", "Reordered",
	"Flags Changed", "Facets Changed", "Invoked"}

// auditUserRe is a user filter: user-name characters only (no quote, space, '$', '*' or '|').
var auditUserRe = regexp.MustCompile(`^[A-Za-z0-9_.@-]{1,100}$`)

// Messages of the audit filter checks, as data.
const (
	auditOneTarget  = "give at most one of target (a component ORD) or target_glob (a like pattern)"
	auditTargetRule = "target must be a component ORD such as station:|slot:/Folder/Point (each segment a slot name), got %q"
	auditGlobRule   = "target_glob must be a like pattern over the stored slot path, of ORD characters and '*' " +
		"(no quote, space, '$' or '|'), e.g. /Folder/*/Point, got %q"
	auditSlotRule = "slot must be a slot name, or a property path such as facets/units (each segment a letter, " +
		"then letters, digits or '_', other characters $-escaped), got %q"
	auditUserRule = "user must be a user name of letters, digits, '_', '.', '@' or '-' (at most 100), got %q"
	auditOpRule   = "operation must be one of %s, got %q"
)

// AuditFilter is what a caller may narrow the audit read with; "" is no filter.
type AuditFilter struct {
	Target, TargetGlob, Slot, User, Operation string
}

// AuditWhere is the where clause of f ("" for none).
func AuditWhere(f AuditFilter) (string, error) {
	if f.Target != "" && f.TargetGlob != "" {
		return "", valueError(auditOneTarget)
	}
	var conds []string
	if f.Target != "" {
		path, err := alarmSourcePath(f.Target)
		if err != nil {
			return "", valueError(fmt.Sprintf(auditTargetRule, f.Target))
		}
		if err := checkLiteral("target", path, f.Target); err != nil {
			return "", err
		}
		conds = append(conds, fmt.Sprintf(auditTargetClause, path))
	}
	if f.TargetGlob != "" {
		if !alarmGlobRe.MatchString(f.TargetGlob) {
			return "", valueError(fmt.Sprintf(auditGlobRule, f.TargetGlob))
		}
		conds = append(conds, "target like '"+f.TargetGlob+"'")
	}
	if f.Slot != "" {
		if !isPropertyPath(f.Slot) {
			return "", valueError(fmt.Sprintf(auditSlotRule, f.Slot))
		}
		if err := checkLiteral("slot", f.Slot, f.Slot); err != nil {
			return "", err
		}
		conds = append(conds, "slotName = '"+f.Slot+"'")
	}
	if f.User != "" {
		if !auditUserRe.MatchString(f.User) {
			return "", valueError(fmt.Sprintf(auditUserRule, f.User))
		}
		if err := checkLiteral("user", f.User, f.User); err != nil {
			return "", err
		}
		conds = append(conds, "userName = '"+f.User+"'")
	}
	if f.Operation != "" {
		if !contains(AuditOperations, f.Operation) {
			return "", valueError(fmt.Sprintf(auditOpRule, strings.Join(AuditOperations, ", "), f.Operation))
		}
		conds = append(conds, "operation = '"+f.Operation+"'")
	}
	return strings.Join(conds, " and "), nil
}

// ComposeFilteredHistoryOrd is the ORD of query over the history id, its range parameters after
// '?', with where and the order suffix appended.
func ComposeFilteredHistoryOrd(id, params, query, where, order string) (string, error) {
	if where != "" {
		query += " where " + where
	}
	return ComposeHistoryOrd(id, params, query+order)
}

// isPropertyPath is a slot name, or slot names joined by '/' (a nested property).
func isPropertyPath(s string) bool {
	for _, seg := range strings.Split(s, "/") {
		if !slotNameRe.MatchString(seg) {
			return false
		}
	}
	return true
}
