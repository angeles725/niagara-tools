// Package retro is mcp-n4-kit/mcp_n4/retro.py: the session retro draft, evidence-backed
// CANDIDATE kit deltas derived from the audit log, the journal and the server's in-memory read
// observations. A candidate is built only from timestamps, tool names, batch ids, op names and
// fixed text: free-form audit reasons and args_redacted are never copied.
package retro

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// refusal is one _REFUSALS row: a class key, the reason substrings (the shared safety
// constants, so a reworded refusal cannot silently stop being classified) and its advice.
type refusal struct {
	key                      string
	needles                  []string
	change, target, priority string
}

var refusals = []refusal{
	{"token_missing", []string{safety.ReasonTokenMissing},
		"Make the dry-run -> token step impossible to skip in the skill/checklist wording", "METHODOLOGY.md · section 4", "MEDIUM"},
	{"token_mismatch", []string{safety.ReasonTokenMismatch},
		"Warn that any argument change after the dry run invalidates the token", "METHODOLOGY.md · section 2 (L1)", "MEDIUM"},
	{"token_expired", []string{safety.ReasonTokenExpired},
		"Document the token lifetime and when to raise --token-ttl for slow human approval", "README.md · running the server", "LOW"},
	{"token_reused", []string{safety.ReasonTokenReused},
		"State that a token is single use and a retry needs a fresh dry run", "METHODOLOGY.md · section 4 step 5", "LOW"},
	{"identity", []string{safety.ReasonIdentity},
		"Put expected_station in the first checklist step so the connect is not repeated", "METHODOLOGY.md · section 4 step 1", "MEDIUM"},
	{"scope", []string{safety.ReasonScopeNone, safety.ReasonScopePlain, safety.ReasonScopeOutside},
		"Explain how to choose --write-scope prefixes before the session starts", "README.md · write scope", "MEDIUM"},
	{"budget", []string{safety.ReasonBudget, safety.ReasonBudgetSmall},
		"Document sizing --max-writes for rollbacks and multi-op batches", "README.md · write budget", "LOW"},
	{"not_connected", []string{safety.ReasonNotConnected},
		"Remind the agent to call n4_connect before any station tool", "skill/SKILL.md · session checklist", "LOW"},
	{"partial_status", []string{safety.ReasonPartialStatus},
		"Teach that Status slots take value_type + status together in the write tool docs", "METHODOLOGY.md · section 3", "MEDIUM"},
	{"unsupported", []string{safety.ReasonAction},
		"Decide whether the refused action is worth supporting or documenting as unsupported", "mcp_n4/tools_write.py · allowed actions", "MEDIUM"},
	{"unsupported", []string{safety.ReasonValueType},
		"Decide whether the refused value type is worth supporting or documenting", "mcp_n4/tools_write.py · VALUE_TYPES", "MEDIUM"},
	{"tier", []string{safety.ReasonTier},
		"Record the PoC/probe outcome for this build and decide on --allow-tier-b/--allow-tier-c", "METHODOLOGY.md · section 5", "MEDIUM"},
	{"unsupported", []string{safety.ReasonTypeSpec},
		"Document the module:Type spec expected for component creation", "README.md · write tools", "LOW"},
	{"journal", []string{safety.ReasonJournalUnreadable},
		"Find what wrote the unreadable journal.jsonl line(s); repair or remove them by hand before deciding the rollback",
		"METHODOLOGY.md · section 2 (L6)", "HIGH"},
}

// verdicts is _VERDICTS, in its order: a read-back verdict and its advice.
var verdicts = []struct{ verdict, change, target, priority string }{
	{"mismatch", "Investigate why the read-back differs from the request and document the cause", "METHODOLOGY.md · section 3", "HIGH"},
	{"failed", "Investigate why the read-back failed after the station accepted the write", "mcp_n4/tools_write.py · read-back", "HIGH"},
	{"partial", "Restore by hand the configuration a rollback reported as not restored " +
		"(frozen_config_not_restored, link_inputs_not_restored)", "METHODOLOGY.md · section 3", "MEDIUM"},
	{"unverified", "Add a way to verify this write class, or document that it stays unverified", "METHODOLOGY.md · section 3", "MEDIUM"},
}

// RefusalClass is the class of a refusal reason (the key of the first _REFUSALS row with a
// needle in it), or "" when none matches.
func RefusalClass(reason string) string {
	if r := refusalRow(reason); r != nil {
		return r.key
	}
	return ""
}

func refusalRow(reason string) *refusal {
	for i := range refusals {
		for _, n := range refusals[i].needles {
			if strings.Contains(reason, n) {
				return &refusals[i]
			}
		}
	}
	return nil
}

var safeText = regexp.MustCompile(`^[A-Za-z0-9_:.+\-]{1,64}$`)

// safe is _safe: a tool name, op name or batch id; anything odd is "?" rather than echoed.
func safe(v any) string {
	if s, ok := v.(string); ok && safeText.MatchString(s) {
		return s
	}
	return "?"
}
