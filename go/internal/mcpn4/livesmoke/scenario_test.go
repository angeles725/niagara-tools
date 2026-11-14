package livesmoke

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestEveryWriteInAFullRunTargetsTheScratchFolderOrItsRootCreateRemove(t *testing.T) {
	client, sc := stubScenario(nil)
	sc.Run()
	var writes []stubCall
	for _, c := range client.calls {
		if stubWrites[c.tool] && c.executing {
			writes = append(writes, c)
		}
	}
	if len(writes) < 12 {
		t.Fatalf("%d executed writes, want >= 12", len(writes))
	}
	for _, w := range writes {
		var ords []string
		for _, k := range []string{"ord", "source_ord", "target_ord"} {
			if v, ok := w.args.Get(k); ok {
				ords = append(ords, v.(string))
			}
		}
		parent, hasParent := w.args.Get("parent_ord")
		name, _ := w.args.Get("name")
		if hasParent {
			ords = append(ords, strings.TrimRight(parent.(string), "/")+"/"+name.(string))
		}
		for _, o := range ords {
			if o != FolderOrd && !strings.HasPrefix(o, FolderOrd+"/") {
				t.Errorf("%s writes %s, outside the scratch folder", w.tool, o)
			}
		}
		if parent == RootOrd && (name != Scratch || (w.tool != "n4_create_component" && w.tool != "n4_remove_component")) {
			t.Errorf("%s writes the root for %v", w.tool, name)
		}
	}
}

func TestTheRunnerRefusesAWriteOutsideTheScratchFolder(t *testing.T) {
	client, sc := stubScenario(nil)
	for _, c := range []struct {
		tool string
		args *pyjson.Object
	}{
		{"n4_create_component", kwargs("parent_ord", RootOrd, "name", "Other", "type", "baja:Folder")},
		{"n4_remove_component", kwargs("parent_ord", RootOrd, "name", "X")},
		{"n4_invoke_action", kwargs("ord", "station:|slot:/Other", "action", "set")},
		{"n4_create_link", kwargs("source_ord", FolderOrd+"/A", "target_ord", "station:|slot:/B")},
	} {
		_, err := sc.Write(pyjson.NewObject(), c.tool, nil, c.args)
		var se *SmokeError
		if !errors.As(err, &se) {
			t.Errorf("%s %s: err = %v, want a SmokeError", c.tool, pyjson.Repr(c.args), err)
		}
	}
	if len(client.calls) != 0 {
		t.Errorf("calls reached the client: %v", client.calls)
	}
}

func TestTimeoutAfterTheFolderCreateAppliedStillCleansUp(t *testing.T) {
	client, sc := stubScenario(func(c *stubClient, tool string, executing bool, args *pyjson.Object) (bool, bool, any, error) {
		if name, _ := args.Get("name"); tool == "n4_create_component" && executing && name == Scratch {
			c.present = true // the station applied it, the reply never arrived
			return true, false, nil, smokef("timeout waiting for tools/call")
		}
		return false, false, nil, nil
	})
	sc.Run()
	if client.present {
		t.Error("the folder is still present")
	}
	v := verdicts(sc)
	if !contains(names(sc), "cleanup") || v["cleanup"] != "verified" {
		t.Errorf("cleanup: %v in %v", v["cleanup"], names(sc))
	}
	if v["create Folder McpSmoke"] != "failed" {
		t.Errorf("create Folder McpSmoke = %v, want failed", v["create Folder McpSmoke"])
	}
}

func TestARefusedDryRunMarksNothingAndDoesNotCleanUp(t *testing.T) {
	_, sc := stubScenario(func(_ *stubClient, tool string, executing bool, _ *pyjson.Object) (bool, bool, any, error) {
		return tool == "n4_create_component" && !executing, true, "refused", nil
	})
	sc.Run()
	for _, n := range names(sc) {
		if strings.HasPrefix(n, "cleanup") {
			t.Errorf("cleanup step %q ran after a refused dry run", n)
		}
	}
}

func TestAMalformedReplyIsAFailedStepNotAnException(t *testing.T) {
	_, sc := stubScenario(func(_ *stubClient, tool string, _ bool, _ *pyjson.Object) (bool, bool, any, error) {
		return tool == "n4_navigate", false, kwargs("children", []any{kwargs("nom", "x")}), nil
	})
	sc.Run()
	if v := verdicts(sc)["folder absent before"]; v != "failed" {
		t.Errorf("folder absent before = %v, want failed", v)
	}
	if d, _ := sc.Steps[1].Get("detail"); d != "KeyError: 'name'" {
		t.Errorf("detail = %v, want Python's KeyError text", d)
	}
}

func TestAFailedReadAfterTheRollbackStillCleansUp(t *testing.T) {
	failed := false
	client, sc := stubScenario(func(c *stubClient, tool string, _ bool, _ *pyjson.Object) (bool, bool, any, error) {
		if tool == "n4_navigate" && c.counts[countKey("n4_rollback", true)] > 0 && !failed {
			failed = true
			return true, false, nil, smokef("timeout waiting for tools/call")
		}
		return false, false, nil, nil
	})
	sc.Run()
	v := verdicts(sc)
	if v["remove folder (again)"] != "failed" {
		t.Errorf("remove folder (again) = %v, want failed", v["remove folder (again)"])
	}
	if client.present || v["cleanup"] != "verified" {
		t.Errorf("present=%v cleanup=%v", client.present, v["cleanup"])
	}
}

func TestCleanupReadsAreRecordedSteps(t *testing.T) {
	_, sc := stubScenario(func(_ *stubClient, tool string, executing bool, _ *pyjson.Object) (bool, bool, any, error) {
		if tool == "n4_invoke_action" && executing {
			return true, false, nil, smokef("boom")
		}
		return false, false, nil, nil
	})
	sc.Run()
	if !contains(names(sc), "cleanup: folder present?") {
		t.Errorf("no cleanup probe step in %v", names(sc))
	}
}

func TestCleanupAssumesPresentWhenTheProbeFails(t *testing.T) {
	afterAbort := false
	client, sc := stubScenario(func(_ *stubClient, tool string, executing bool, _ *pyjson.Object) (bool, bool, any, error) {
		if tool == "n4_invoke_action" && executing {
			afterAbort = true
			return true, false, nil, smokef("boom")
		}
		if tool == "n4_navigate" && afterAbort {
			return true, false, nil, smokef("navigate down")
		}
		return false, false, nil, nil
	})
	sc.Run()
	v := verdicts(sc)
	if v["cleanup: folder present?"] != "failed" || v["cleanup"] != "verified" || client.present {
		t.Errorf("probe=%v cleanup=%v present=%v", v["cleanup: folder present?"], v["cleanup"], client.present)
	}
}

func TestAbortAfterTheFirstSaveRemovesAndSavesAgain(t *testing.T) {
	client, sc := stubScenario(func(c *stubClient, tool string, executing bool, _ *pyjson.Object) (bool, bool, any, error) {
		if tool == "n4_remove_component" && executing && c.counts[countKey(tool, true)] == 1 {
			return true, false, nil, smokef("timeout waiting for tools/call")
		}
		return false, false, nil, nil
	})
	sc.Run()
	n := names(sc)
	if client.present || !reflect.DeepEqual(n[len(n)-2:], []string{"cleanup", "cleanup save"}) {
		t.Errorf("present=%v last steps=%v", client.present, n[len(n)-2:])
	}
	if v := verdicts(sc)["cleanup save"]; v != "verified" {
		t.Errorf("cleanup save = %v", v)
	}
	if got := client.counts[countKey("n4_save_station", true)]; got != 2 {
		t.Errorf("%d executed saves, want 2", got)
	}
}

func TestNoCleanupSaveWhenTheFolderWasNeverPersisted(t *testing.T) {
	_, sc := stubScenario(func(_ *stubClient, tool string, executing bool, _ *pyjson.Object) (bool, bool, any, error) {
		if tool == "n4_invoke_action" && executing {
			return true, false, nil, smokef("boom")
		}
		return false, false, nil, nil
	})
	sc.Run()
	if contains(names(sc), "cleanup save") {
		t.Errorf("cleanup save ran: %v", names(sc))
	}
}

func TestPlanRowsEqualTheExecutedStepNamesAndTools(t *testing.T) {
	client, sc := stubScenario(nil)
	sc.Run()
	var executed [][2]string
	for _, s := range sc.Steps {
		n, _ := s.Get("name")
		tool, _ := s.Get("tool")
		executed = append(executed, [2]string{n.(string), tool.(string)})
	}
	if !reflect.DeepEqual(PlanRows(), executed) {
		t.Errorf("plan rows %v\nexecuted %v", PlanRows(), executed)
	}
	if client.present {
		t.Error("the folder is left present")
	}
}

func TestPlanTextListsEveryRowAndTheComponents(t *testing.T) {
	text := PlanText()
	for _, r := range PlanRows() {
		if !strings.Contains(text, r[0]) {
			t.Errorf("plan text misses %q", r[0])
		}
	}
	if !strings.Contains(text, "kitControl:GreaterThan") {
		t.Error("plan text misses kitControl:GreaterThan")
	}
}

func TestSaveTakesAnExplicitFolderPresentFlag(t *testing.T) {
	_, sc := stubScenario(nil)
	entry := pyjson.NewObject()
	if _, _, err := sc.Save(true)(entry); err != nil || !sc.SavedWithFolder() {
		t.Fatalf("Save(true): err=%v saved_with_folder=%v", err, sc.SavedWithFolder())
	}
	if _, _, err := sc.Save(false)(entry); err != nil || sc.SavedWithFolder() {
		t.Fatalf("Save(false): err=%v saved_with_folder=%v", err, sc.SavedWithFolder())
	}
}
