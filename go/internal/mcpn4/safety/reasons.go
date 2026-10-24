package safety

// The leading text of each refusal (safety.py REASON_*), verbatim: the raise sites build their
// messages from these and the session retro classifies audit reasons with them.
const (
	ReasonTokenMissing  = "confirmation_token is missing"
	ReasonTokenMismatch = "confirmation_token does not match this tool"
	ReasonTokenExpired  = "confirmation_token expired"
	ReasonTokenReused   = "confirmation_token already used"
	ReasonIdentity      = "station identity not verified"
	ReasonScopeNone     = "no --write-scope configured"
	ReasonScopePlain    = "is not a plain station ORD"
	ReasonScopeOutside  = "is outside the write scope"
	ReasonBudget        = "write budget exhausted"
	ReasonBudgetSmall   = "write budget too small"
	ReasonNotConnected  = "not connected"
	ReasonPartialStatus = "Status slots are written whole"
	ReasonAction        = "is not allowed in this version"
	ReasonValueType     = "value_type must be one of"
	ReasonTypeSpec      = "type must look like"

	ReasonJournalUnreadable = "journal line unreadable"
)
