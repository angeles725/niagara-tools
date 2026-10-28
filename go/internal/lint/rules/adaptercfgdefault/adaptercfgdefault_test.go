package adaptercfgdefault

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the audit 2026-10-03 A7-D1 contract (odd/tasks/kit-backlog-2026-10-27.md B8), one hand-written case per
// testdata/*.txt: a Cfg field the adapter fills from a slot (`cfg.f = getSlot()`, `.getMillis()`/`.getSeconds()`
// on a BRelTime) WARNs when the slot's defaultValue differs from the field's default (its literal initializer, or
// 0/false with none); equal defaults, an unread field, a default it cannot evaluate and a commented annotation are
// clean; --strict; empty roots and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 6) }
