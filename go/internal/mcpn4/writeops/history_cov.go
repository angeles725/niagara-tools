package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// numericCovType is the one extension with a change tolerance: history-rt
// BNumericCovHistoryExt.changeTolerance (double, default 0, facet min 0), the smallest change of
// the point value that records a new sample.
const numericCovType = "history:NumericCovHistoryExt"

// fullPolicies are the history:FullPolicy tags (history-rt BFullPolicy range stop, roll); a new
// extension's BHistoryConfig.fullPolicy defaults to roll.
var fullPolicies = map[string]string{"roll": "Roll", "stop": "Stop"}

// historyTolerance is the change_tolerance argument, nil when not given: a finite number of at
// least 0, only for mode cov and never with an ext_type other than the numeric COV one.
func historyTolerance(args *pyjson.Object, mode, typ string) (any, error) {
	raw, given := args.Get("change_tolerance")
	if !given {
		return nil, nil
	}
	if mode != "Cov" {
		return nil, plan.ToolError(msgHistoryToleranceMode)
	}
	if typ != "" && typ != numericCovType {
		return nil, plan.ToolError(fmt.Sprintf(msgHistoryToleranceType, typ))
	}
	n, ok := raw.(pyjson.Number)
	f, err := n.Float64()
	if !ok || err != nil || !(f >= 0) || f-f != 0 {
		return nil, plan.ToolError(fmt.Sprintf(msgHistoryTolerance, pyjson.Repr(raw)))
	}
	return f, nil
}

// historyFullPolicy is the full_policy argument as its history:FullPolicy tag, "" when not given.
func historyFullPolicy(args *pyjson.Object) (string, error) {
	raw, given := args.Get("full_policy")
	if !given {
		return "", nil
	}
	if tag, _ := raw.(string); fullPolicies[tag] != "" {
		return tag, nil
	}
	return "", plan.ToolError(fmt.Sprintf(msgHistoryFullPolicy, pyjson.Repr(raw)))
}

// checkTolerance refuses a change tolerance on a point whose extension is not the numeric COV
// one, naming the field, the point and its family.
func checkTolerance(s historySpec, kind, pointType string) error {
	if s.tolerance == nil || s.typ == numericCovType {
		return nil
	}
	return plan.ToolError(fmt.Sprintf(msgHistoryTolerancePoint, s.ord, strings.ToLower(kind), pointType))
}

// toleranceSlot is the changeTolerance property of the add body.
func toleranceSlot(f float64) *pyjson.Object {
	return obj("nm", "p", "n", "changeTolerance", "t", "baja:Double", "v", pyjson.FloatRepr(f))
}

// fullPolicyOp sets the full policy of the extension name under the point with handle h, by a
// set op right after its add, like capacityOp (historyConfig is a component).
func fullPolicyOp(h any, name, tag string) *pyjson.Object {
	return obj("nm", "s", "h", h, "n", name+"/historyConfig/fullPolicy",
		"b", obj("nm", "p", "t", "history:FullPolicy", "v", tag))
}

// policyWord is the full policy as the capacity notes say it: Roll unless stop was asked.
func policyWord(tag string) string {
	if tag == "" {
		return "Roll"
	}
	return fullPolicies[tag]
}
