package writeops

import (
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// lowerScanBudget sets LinkScanNodeBudget to n for one test and restores it after.
func lowerScanBudget(t *testing.T, n int) {
	t.Helper()
	saved := LinkScanNodeBudget
	LinkScanNodeBudget = n
	t.Cleanup(func() { LinkScanNodeBudget = saved })
}

// folderTree is a flat load of a Folder root holding n plain folders (n+1 entries).
func folderTree(t *testing.T, n int) *pyjson.Object {
	t.Helper()
	nodes := pyjson.NewObject()
	kids := []any{}
	for i := range n {
		nm := "F" + strconv.Itoa(i)
		kid := obj("n", nm, "t", "baja:Folder", "h", strconv.Itoa(100+i))
		kids = append(kids, kid)
		nodes.Set(nm, kid)
	}
	nodes.Set("", obj("t", "baja:Folder", "h", "3", "s", kids))
	return nodes
}

func scanArgs(scan string) *pyjson.Object {
	args := obj("parent_ord", "station:|slot:/Folder", "name", "F0")
	if scan != "" {
		args.Set("link_scan_ord", scan)
	}
	return args
}

// The budget is sized for a station root: the lab station's root loads 11894 entries to depth 6.
func TestTheLinkScanBudgetDefaultHoldsAStationRoot(t *testing.T) {
	if LinkScanNodeBudget != DefaultLinkScanNodeBudget || DefaultLinkScanNodeBudget != 100000 {
		t.Fatalf("budget %d, default %d", LinkScanNodeBudget, DefaultLinkScanNodeBudget)
	}
	text := scanTooLarge("station:|slot:/", 100001, false)
	for _, want := range []string{"the scope root station:|slot:/", "too large to scan", "100001 node(s) to depth 6",
		"more than 100000", "pass link_scan_ord", "refusing"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal misses %q: %s", want, text)
		}
	}
}

func TestAScanRootOverTheBudgetRefusesAndAsksForLinkScanOrd(t *testing.T) {
	lowerScanBudget(t, 3)
	scope, err := ord.NewScope([]string{"station:|slot:/Folder"})
	if err != nil {
		t.Fatal(err)
	}
	nodes := folderTree(t, 3) // 4 entries: one over the budget
	for _, c := range []struct{ scan, want string }{
		{"", "the scope root station:|slot:/Folder is too large to scan"},
		{"station:|slot:/Folder", "the link_scan_ord root station:|slot:/Folder is too large to scan"},
	} {
		_, _, err := linkScan(scanRoot{nodes}, scanArgs(c.scan), scope, nodes, map[string]string{"s100": ""},
			pyjson.NewObject(), nil, nil)
		text := plan.PyStr(err)
		if err == nil || !strings.Contains(text, c.want) || !strings.Contains(text, "4 node(s)") ||
			!strings.Contains(text, "more than 3") || !strings.Contains(text, "link_scan_ord") {
			t.Errorf("scan %q: %v", c.scan, err)
		}
	}
	lowerScanBudget(t, 4) // at the budget: scans
	if _, _, err := linkScan(scanRoot{nodes}, scanArgs(""), scope, nodes, map[string]string{"s100": ""},
		pyjson.NewObject(), nil, nil); err != nil {
		t.Errorf("a root at the budget refused: %v", err)
	}
}
