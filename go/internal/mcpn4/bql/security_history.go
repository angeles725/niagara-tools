package bql

// Login records (station-audits S14, n4_users_audit): BAuthenticationService audits each login
// attempt as an AuditEvent "Login" or "Login Failure" (BAuthenticationService.auditLoginAttempt);
// BAuditHistoryService.checkSecurityAudit sends Login, Logout, Login Failure and Logout (Timeout)
// to the security auditor instead of the AuditHistory, and BSecurityAuditHistorySource keeps them
// in /<station>/SecurityHistory as BSecurityAuditRecord {timestamp, operation, userName, message}.
// The query is fixed: no operator text reaches it.

// SecurityHistoryName is the history name BSecurityAuditHistorySource gives the security trail.
const SecurityHistoryName = "SecurityHistory"

// The fixed login query: one row per successful login, newest first, read by position.
const (
	LoginRecordsQuery = "select timestamp.encodeToString, userName"
	LoginWhere        = "operation = 'Login'"
)
