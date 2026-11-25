package writeops

import (
	"math"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/files"
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
	observed.Set("temp_output", sampleTempOutput(c, d))
	observed.Set("hint", "the job is still "+pyjson.Str(state)+" after "+strconv.Itoa(d.wait)+" s plus "+
		strconv.Itoa(int(discoverSlack))+" s; read it later under "+job+" (its templateFileOrd holds the file)")
	return requested, replies[0], observed, "unverified", nil
}

const (
	// makeSampleGap is the nominal seconds between the two reads of the ^temp output; makeSampleCap the
	// bytes each read asks for (n4_read_file's max_bytes): the file servlet's Content-Length gives the
	// whole size, and a read that hits the cap without one proves only "at least" that many bytes.
	makeSampleGap = 5
	makeSampleCap = 1 << 20
	// makeSampleTimeout is the HTTP timeout, in seconds, of each read.
	makeSampleTimeout = 30
)

// fileGetter is what the real station client offers besides the plan.Client calls (readtools' GetFile).
type fileGetter interface {
	GetFile(urlPath string, timeout float64, maxBytes int) (*box.File, error)
}

// tempSize is one read of the output: its size, and atLeast when only a capped read of it is known.
type tempSize struct {
	bytes   int64
	atLeast bool
}

func readTempSize(g fileGetter, urlPath string) (tempSize, error) {
	f, err := g.GetFile(urlPath, makeSampleTimeout, makeSampleCap)
	if err != nil {
		return tempSize{}, err
	}
	if n, err := strconv.ParseInt(pyjson.Str(f.ContentLength), 10, 64); err == nil && n >= 0 {
		return tempSize{bytes: n}, nil
	}
	return tempSize{bytes: int64(len(f.Body)), atLeast: f.Truncated}, nil
}

// sampleTempOutput reads ^temp/<station>.ntpl twice, makeSampleGap s apart, and reports its size and growth
// (the station's own file name may differ when made unique: then the read says it could not be read).
func sampleTempOutput(c plan.Client, d makeData) *pyjson.Object {
	g, ok := c.(fileGetter)
	if !ok || d.station == "" {
		return obj("skipped", "the output size was not read: no station file read available here; "+
			"read ^temp/<template file name>.ntpl with n4_read_file twice a few seconds apart")
	}
	fileOrd := "file:^temp/" + d.station + ".ntpl"
	out := obj("file_ord", fileOrd)
	path, err := files.StationPath(fileOrd)
	var urlPath string
	if err == nil {
		urlPath, err = files.URLPath(path)
	}
	var first, second tempSize
	var t0, t1 float64
	if err == nil {
		t0 = d.w.Monotonic()
		if first, err = readTempSize(g, urlPath); err == nil {
			d.w.Sleep(makeSampleGap)
			if second, err = readTempSize(g, urlPath); err == nil {
				t1 = d.w.Monotonic()
			}
		}
	}
	if err != nil {
		out.Set("hint", "the output could not be read ("+err.Error()+"): look for the partial .ntpl in ^temp")
		return out
	}
	return describeTempGrowth(out, first, second, int(math.Round(t1-t0)))
}

func describeTempGrowth(out *pyjson.Object, first, second tempSize, apart int) *pyjson.Object {
	for i, s := range []tempSize{first, second} {
		name := []string{"first", "second"}[i]
		out.Set(name+"_bytes", s.bytes)
		if s.atLeast {
			out.Set(name+"_at_least", true)
			out.Set(name+"_note", "at least "+strconv.FormatInt(s.bytes, 10)+" bytes (the read hit its cap)")
		}
	}
	out.Set("seconds_apart", apart)
	switch {
	case !first.atLeast && !second.atLeast:
		out.Set("growth_bytes", second.bytes-first.bytes)
		out.Set("growing", second.bytes > first.bytes)
		if apart > 0 {
			out.Set("bytes_per_second", (second.bytes-first.bytes)/int64(apart))
		}
	case !first.atLeast && second.bytes > first.bytes:
		// an exact first size below the second's lower bound proves growth; a capped first read does not
		out.Set("growing", true)
	}
	if g, _ := out.Get("growing"); g == true {
		out.Set("warning", "RUNAWAY: the output is still growing while the job runs: "+msgMakeCause+"; "+msgMakeRecovery)
	}
	return out
}
