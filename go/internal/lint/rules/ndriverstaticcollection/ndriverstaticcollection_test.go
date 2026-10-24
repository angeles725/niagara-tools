package ndriverstaticcollection

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G66 contract (ledger.d/2026-10-23-r8-u5-c59-c62.md), one hand-written case per testdata/*.txt: a
// static collection in a BNDevice, BNNetwork, BDeviceNetwork and module-local device subclass WARNs (java.util and
// java.util.concurrent types alike); an instance field, a Collections.unmodifiable…/empty… initializer, a scalar
// or array static, a commented-out or quoted field, a static method and a utility class are clean; --strict;
// only -rt src/ is read; empty roots and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 9) }
