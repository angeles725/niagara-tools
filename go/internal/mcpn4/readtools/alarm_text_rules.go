package readtools

import (
	"regexp"
	"strings"
)

// The LC9 rules of n4_alarm_text_check (build-n4-module-kit/types/logic.md LC9), the live twin of
// bog-nav alarm-text (niagara-go internal/bognav/alarmtext.go): the same rule names, the same
// three text slots, the same tokens judged, the same rows. The texts of a BAlarmSourceExt are
// stored raw and resolved against the ALARM RECORD when read, never against the source, so a
// %parent...% (or any source-slot path) never resolves; on every return to normal the ext sets
// the record's msgText = toNormalText (BAlarmSourceExt.java:815, 4.14.0.162 vineflower), so an
// empty toNormalText blanks the text the kept record shows.

// Rule names, identical to bog-nav alarm-text.
const (
	ruleParentPath    = "parent-path"
	ruleEmptyToNormal = "empty-tonormal"
)

// alarmTextSlots are the ext's text slots, in the order findings are listed and the order of the
// select's text columns (alarmTextQuery is built from this slice). BAlarmSourceExt (4.14 and
// 4.15.6.20) declares the same three, as toFaultText, toOffnormalText, toNormalText. Each
// defaults to "".
var alarmTextSlots = []string{"toOffnormalText", "toFaultText", "toNormalText"}

// alarmRecordFields are the properties of BAlarmRecord: a %field...% path rooted at one resolves.
var alarmRecordFields = map[string]bool{
	"timestamp": true, "uuid": true, "sourceState": true, "ackState": true, "ackRequired": true,
	"source": true, "alarmClass": true, "priority": true, "normalTime": true, "ackTime": true,
	"user": true, "alarmData": true, "alarmTransition": true, "lastUpdate": true,
}

// formatCallRE is a BFormat function call (%lexicon(alarm:x)%): it reads no slot path.
var formatCallRE = regexp.MustCompile(`^[A-Za-z_]\w*\(`)

// unresolvedAlarmTokens are the %...% tokens of a BFormat text whose path is not rooted at a
// field of the alarm record (%parent...% included), split as BFormat splits them (formatScripts):
// %% is a literal percent sign, and a text with a lone % has none (BFormat shows it as
// "ERROR <text>").
func unresolvedAlarmTokens(text string) []string {
	scripts, _ := formatScripts(text)
	var out []string
	for _, body := range scripts {
		if formatCallRE.MatchString(body) {
			continue
		}
		root, _, _ := strings.Cut(body, ".")
		if !alarmRecordFields[root] {
			out = append(out, "%"+body+"%")
		}
	}
	return out
}

// alarmTextFinding is one row, keys in bog-nav's column order.
func alarmTextFinding(rule, path, slot, token, text string) any {
	return obj("level", "WARN", "rule", rule, "path", path, "slot", slot, "token", token, "text", text)
}

// alarmTextFindings are the rows of one ext whose texts are texts[slot]: one parent-path row per
// unresolved token, then empty-tonormal when toOffnormalText is set and toNormalText is empty.
func alarmTextFindings(path string, texts map[string]string) (rows []any, parents, empties int) {
	for _, slot := range alarmTextSlots {
		for _, tok := range unresolvedAlarmTokens(texts[slot]) {
			rows = append(rows, alarmTextFinding(ruleParentPath, path, slot, tok, texts[slot]))
			parents++
		}
	}
	if texts["toOffnormalText"] != "" && texts["toNormalText"] == "" {
		rows = append(rows, alarmTextFinding(ruleEmptyToNormal, path, "toNormalText", "", ""))
		empties++
	}
	return rows, parents, empties
}
