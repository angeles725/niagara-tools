package writeops

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// backupNow is the clock of the zip name (tests replace it).
var backupNow = time.Now

// backupPlan reads n4_save_station's backup argument into d: with backup true the station needs a
// --station-home (the directory zipped) and the server a state dir; else a refusal before any
// station call. The notes say where the zip goes and what it leaves out.
func backupPlan(d *saveData, args *pyjson.Object) ([]any, error) {
	if v, _ := args.Get("backup"); v != true {
		return nil, nil
	}
	if d.home == "" {
		return nil, plan.ToolError("backup needs --station-home " + d.station + "=<station directory>: the backup zips that " +
			"directory after the save; without it, use Workbench's Backup Station (platform or Fox)")
	}
	if d.w.StateDir == "" {
		return nil, plan.ToolError("backup needs the server's state dir (--state-dir)")
	}
	d.backup = true
	return []any{"after the save, " + d.home + " is zipped to " + d.w.StateDir + "/" + backupSubdir +
		"/<station>-<UTC time>.zip (0600) without what the station's BackupService leaves out: files " +
		pyjson.Repr(stringsAny(backupExcludeFiles)) + " and the directories " + pyjson.Repr(stringsAny(backupExcludeDirs)) +
		"; the zip holds config.bog as found on disk, so read persisted before relying on it"}, nil
}

// backupAfterSave zips the station directory and reports it, or the failure as {error}.
func backupAfterSave(d saveData) *pyjson.Object {
	out, err := zipStationHome(d.home, d.w.StateDir, d.station, backupNow())
	if err != nil {
		return obj("error", "the backup zip failed: "+err.Error())
	}
	return out
}

func stringsAny(l []string) []any {
	out := make([]any, len(l))
	for i, s := range l {
		out[i] = s
	}
	return out
}
