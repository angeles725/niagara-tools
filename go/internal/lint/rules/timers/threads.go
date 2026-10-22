package timers

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// threadPass is jdk-thread (FAIL / WARN) and job-cancel (WARN). Per file: the job-cancel row, then for a
// B* class that uses JDK concurrency, FAIL when it modifies a thread and the nearest module-permissions.xml
// requests no MANAGE_EXECUTION under type station/all (R8-D7), else WARN.
func (r *run) threadPass(srcs []*source) error {
	for _, s := range srcs {
		t := r.threadFacts(s)
		if t.class == "" {
			continue
		}
		if t.jobLoop != 0 {
			r.row(lint.Warn, "job-cancel", s, fmt.Sprintf("%s run() loops (line %d) with no isAlive() check -- cancel only interrupts the job thread; add if (!isAlive()) throw new JobCancelException(); (types/logic-authoring.md Job contract)", t.class, t.jobLoop))
		}
		if !t.jdk {
			continue
		}
		g, err := r.permsGrant(s.Path)
		if err != nil {
			return err
		}
		// the bash `read -r _pf _pg` split of "<path> <yes|no>": a path with spaces shifts into _pg
		pf, pg := splitTwo(g.path + " " + g.answer)
		if t.modify != "" && pg != "yes" {
			if pf == "-" {
				pf = "no module-permissions.xml above the file"
			}
			r.row(lint.Fail, "jdk-thread", s, fmt.Sprintf("%s modifies a thread (%s) but %s requests no MANAGE_EXECUTION (type station/all) -- modifyThread is denied at stop and the thread outlives the component (types/logic.md Off-engine work)", t.class, t.modify, pf))
		} else {
			r.row(lint.Warn, "jdk-thread", s, t.class+" uses a JDK thread/executor -- allowed (Tridium Lab 9); prefer Clock.schedule, an existing BWorker, a BSimpleJob or a named ExecutorUtil.newSingleThreadBackgroundExecutor; a shutdown/interrupt needs MANAGE_EXECUTION")
		}
	}
	return nil
}

// threadFacts is threads.awk over one file.
type threadFacts struct {
	class   string // the B* class name; "" when the file holds none (no rows)
	jdk     bool   // uses ExecutorService, Executors., ExecutorUtil. or new Thread(
	modify  string // "<call>() at line <n>": the first thread-modifying call (read only when jdk)
	jobLoop int    // the first loop line of a BSimpleJob run() with no isAlive() (the last such run() wins)
}

func (r *run) threadFacts(s *source) threadFacts {
	code, ms := s.blanked()
	var t threadFacts
	job := false
	for i := 0; i < len(code) && t.class == ""; i++ {
		if !strings.Contains(code[i], "class") || !r.rx.has(code[i], notWord+`class`+sp+`+B[A-Za-z0-9_]+`) {
			continue
		}
		// the header runs from the line naming `class BX` to the first `{`, joined
		var h strings.Builder
		for j := i; j < len(code); j++ {
			h.WriteString(" " + code[j])
			if strings.Contains(code[j], "{") {
				break
			}
		}
		head := r.rx.sub(h.String(), `\{.*`)
		name := r.rx.sub(r.rx.sub(head, `.*`+notWord+`class`+sp+`+`), `[^A-Za-z0-9_].*`)
		if r.rx.has(head, `(extends|implements)`+sp+`+B[A-Za-z0-9_]`) {
			t.class = name
		}
		if r.rx.has(head, `extends`+sp+`+BSimpleJob([^A-Za-z0-9_]|$)`) {
			job = true
		}
	}
	if t.class == "" {
		return t
	}
	for i, c := range code {
		if !t.jdk && r.rx.has(c, notWord+`(Scheduled)?ExecutorService([^A-Za-z0-9_]|$)|`+notWord+`Executors`+sp+`*\.|`+notWord+`ExecutorUtil`+sp+`*\.|`+notWord+`new`+sp+`+Thread`+sp+`*\(`) {
			t.jdk = true
		}
		if t.modify != "" || !strings.Contains(c, ".") {
			continue
		}
		c = r.rx.get(`currentThread`+sp+`*\(`+sp+`*\)`+sp+`*\.`+sp+`*interrupt`+sp+`*\(`).ReplaceAllLiteralString(c, "currentThread().selfInterrupt(")
		if m, ok := r.rx.find(c, `\.`+sp+`*(shutdownNow|shutdown|interrupt|setPriority|setDaemon)`+sp+`*\(`); ok {
			t.modify = r.rx.gsub(m, `[.([:space:]]`) + "() at line " + strconv.Itoa(i+1)
		}
	}
	if job {
		t.jobLoop = r.jobLoop(code, ms)
	}
	return t
}

// jobLoop is the first for/while/do line of a run() body with no isAlive() call; with several run()
// methods the last one reported wins (the bash kept the last JOBLOOP line).
func (r *run) jobLoop(code []string, ms []javasrc.Method) int {
	last := 0
	for _, m := range ms {
		if m.Name != "run" {
			continue
		}
		loop, alive := 0, false
		for i := m.Start; i <= m.End; i++ {
			if loop == 0 && r.rx.has(code[i-1], notWord+`(for|while)`+sp+`*\(|`+notWord+`do`+sp+`*(\{|$)`) {
				loop = i
			}
			if r.rx.has(code[i-1], notWord+`isAlive`+sp+`*\(`) {
				alive = true
			}
		}
		if loop != 0 && !alive {
			last = loop
		}
	}
	return last
}

// grant is perms_grant's answer: the module-permissions.xml nearest above a file ("-" for none) and
// whether it requests MANAGE_EXECUTION under type="station" or type="all" ("yes" / "no").
type grant struct{ path, answer string }

// permsGrant walks up from the file's directory (made absolute like `cd dir && pwd`) to /. A file that
// cannot be read is an environment error, never "no grant" guessed from a missing read.
func (r *run) permsGrant(path string) (grant, error) {
	d := filepath.Dir(path)
	if !filepath.IsAbs(d) {
		wd, err := os.Getwd()
		if err != nil {
			return grant{}, err
		}
		d = filepath.Join(wd, d)
	}
	if g, ok := r.perms[d]; ok {
		return g, nil
	}
	g := grant{"-", "no"}
	for dir := d; ; dir = filepath.Dir(dir) {
		p := dir + "/module-permissions.xml"
		if st, err := os.Stat(p); err == nil {
			g = grant{p, "no"}
			if st.IsDir() {
				break // awk skips a directory operand and prints "no"
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return grant{}, fmt.Errorf("cannot read %s: awk: fatal: cannot open file `%s' for reading: %s", p, p, strerror(err))
			}
			if r.managesExecution(data) {
				g.answer = "yes"
			}
			break
		}
		if dir == "/" {
			break
		}
	}
	r.perms[d] = g
	return g, nil
}

// managesExecution is the perms_grant awk: a <name>MANAGE_EXECUTION</name> line inside a
// <niagara-permission-groups> element whose type (read on its opening line) is station or all.
func (r *run) managesExecution(data []byte) bool {
	lines, _ := javasrc.Lines(bytes.NewReader(data))
	ty, inGroup := "", false
	for _, ln := range lines {
		if strings.Contains(ln, "<niagara-permission-groups") {
			ty = ""
			if m, ok := r.rx.find(ln, `type`+sp+`*=`+sp+`*"[^"]*"`); ok {
				ty = r.rx.sub(r.rx.sub(m, `^[^"]*"`), `"$`)
			}
			inGroup = true
		}
		if inGroup && (ty == "station" || ty == "all") && r.rx.has(ln, `<name>`+sp+`*MANAGE_EXECUTION`+sp+`*</name>`) {
			return true
		}
		if strings.Contains(ln, "</niagara-permission-groups>") {
			inGroup = false
		}
	}
	return false
}

// splitTwo is `read -r a b` over one line: a is the first IFS word, b the rest with its outer blanks trimmed.
func splitTwo(s string) (string, string) {
	s = strings.Trim(s, " \t\n")
	i := strings.IndexAny(s, " \t\n")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.Trim(s[i:], " \t\n")
}

// strerror is the C library text of err (Go's errno text with a capital first letter), as awk prints it.
func strerror(err error) string {
	if pe, ok := err.(*os.PathError); ok {
		err = pe.Err
	}
	s := err.Error()
	return strings.ToUpper(s[:1]) + s[1:]
}
