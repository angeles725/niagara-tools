package reportmodule

import (
	"strings"
	"testing"
)

// TestSchemaRiskVerdictTellsFirstDeployFromMissingBaseline (R2-002): with no .deploy-baseline the verdict of
// --first-deploy and of a missing baseline differ; only the second reports the baseline missing.
func TestSchemaRiskVerdictTellsFirstDeployFromMissingBaseline(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	first := schemaRiskVerdict(runner{}, dir, true)
	missing := schemaRiskVerdict(runner{}, dir, false)
	if first.missingBaseline || first.status != skip || !strings.HasPrefix(first.detail, "N/A (--first-deploy") {
		t.Errorf("--first-deploy = %+v, want a SKIP N/A that is not a missing baseline", first)
	}
	if !missing.missingBaseline || missing.status != skip || !strings.HasPrefix(missing.detail, "no .deploy-baseline") {
		t.Errorf("no baseline = %+v, want a SKIP naming the missing baseline", missing)
	}
}
