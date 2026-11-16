package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// licenseFiles is what the local licence files add to the licences of the page: per licence
// name the file that states its facts, and the note of what could not be added ("" when every
// licence has its file). Only with --niagara-home for the connected station; a file issued for
// another host than the page's is never used, and a file that matches no licence of the page
// is neither read nor named.
type licenseFiles struct {
	byName map[string]license.File
	note   string
}

func (t *Tools) readLicenseFiles(info *license.Info) licenseFiles {
	station := t.Ctx.Session.StationName
	home, ok := t.NiagaraHomes[station]
	if !ok {
		return licenseFiles{note: fmt.Sprintf(licenseNoHome, station)}
	}
	wanted := make([]string, len(info.Licenses))
	for i, e := range info.Licenses {
		wanted[i] = e.Name
	}
	files, skipped, err := license.ReadFiles(home, wanted)
	if err != nil {
		return licenseFiles{note: fmt.Sprintf(licenseNoDir, station)}
	}
	out := licenseFiles{byName: map[string]license.File{}}
	problems, seen := map[string][]string{}, map[string]bool{}
	for _, name := range wanted {
		if seen[name] {
			continue
		}
		seen[name] = true
		f, found := files[name]
		switch why := skipped[name]; {
		case why != "":
			problems[why] = append(problems[why], name)
		case !found:
			problems[licenseFileAbsent] = append(problems[licenseFileAbsent], name)
		case !f.ForHost(info.HostID):
			problems[licenseFileOtherHost] = append(problems[licenseFileOtherHost], name)
		default:
			out.byName[name] = f
		}
	}
	var notes []string
	for _, why := range licenseFileReasons {
		if names := problems[why]; len(names) > 0 {
			notes = append(notes, why+": "+someNames(names))
		}
	}
	if len(notes) > 0 {
		out.note = fmt.Sprintf(licenseFilesNote, joinNotes(notes))
		if len(problems[license.SkipCap]) > 0 {
			out.note += fmt.Sprintf(licenseFileCapHit, license.MaxFiles)
		}
	}
	return out
}

// add puts generated and version on a licence entry when its file states them.
func (l licenseFiles) add(entry *pyjson.Object, name string) {
	f, ok := l.byName[name]
	if !ok {
		return
	}
	if f.Generated != "" {
		entry.Set("generated", f.Generated)
	}
	if f.Version != "" {
		entry.Set("version", f.Version)
	}
}
