package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// makeTemplateReadback waits for the job the action returned and reads its state, its log
// (readLog, as spyderDownload's) and the file it saved (templateFileOrd, templateName). success
// with a file is verified; failed, canceled, or success with no file a mismatch; a job still
// running, or no job at all (not a super user), unverified with why.
func makeTemplateReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	d := p.Data.(makeData)
	requested := obj("action", makeAction)
	job := jobOrd(replies[0])
	if job == "" {
		return requested, replies[0], obj("job", nil, "hint", "the station returned no job ORD: "+msgMakeSuperUser), "unverified", nil
	}
	state, nodes, err := waitJob(c, job, discoverData{wait: d.wait, w: d.w})
	if err != nil {
		return nil, nil, nil, "", err
	}
	log, logErr := jobLog(c, nodes)
	props := box.Children(nodes, "")
	file := box.ChildValue(props, "templateFileOrd", nil)
	if s, _ := file.(string); s == "" || s == "null" {
		file = nil
	}
	observed := obj("job", job, "job_state", state, "log", log, "file_ord", file,
		"template_name", box.ChildValue(props, "templateName", nil))
	if logErr != "" {
		observed.Set("log_error", logErr)
	}
	switch {
	case state == "success" && file != nil:
		observed.Set("hint", "fetch it with n4_read_file(file_ord="+pyjson.Str(file)+")")
		return requested, replies[0], observed, "verified", nil
	case state == "success":
		observed.Set("hint", "the job ended success but its templateFileOrd is empty: no file was reported")
		return requested, replies[0], observed, "mismatch", nil
	case jobComplete(state):
		observed.Set("hint", "the job ended "+pyjson.Str(state)+": its log above is what the station kept; no file was written")
		return requested, replies[0], observed, "mismatch", nil
	}
	observed.Set("hint", "the job is still "+pyjson.Str(state)+" after "+strconv.Itoa(d.wait)+" s plus "+
		strconv.Itoa(int(discoverSlack))+" s; read it later under "+job+" (its templateFileOrd holds the file)")
	return requested, replies[0], observed, "unverified", nil
}
