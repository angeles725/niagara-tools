package modulemap

import (
	"path"
	"sort"
	"strconv"
	"strings"
)

// ConcernFolder is the concern folder of a part's src/**/*.java file rel (relative to the module root): the
// path up to the directory right under the first role-folder segment after src/, or "" for a file that sits
// in no concern folder (directly in a role folder, or outside every role folder).
func ConcernFolder(rel string) string {
	seg := strings.Split(rel, "/")
	if len(seg) < 4 || seg[1] != "src" || !strings.HasSuffix(rel, ".java") {
		return ""
	}
	var roles []string
	for suf, r := range roleFolders {
		if strings.HasSuffix(seg[0], suf) {
			roles = r
		}
	}
	for i := 2; i < len(seg)-2; i++ {
		for _, r := range roles {
			if seg[i] == r {
				return strings.Join(seg[:i+2], "/")
			}
		}
	}
	return ""
}

// mm7 relates concern folders and rows. First, per non-plumbing row in map order, a (concern, part) pair that an
// earlier row already uses, compared case-insensitively: one concern may have one row per part (rt and ux rows of
// battery are two rows of one concern), never two rows in one part; plumbing may repeat. Then per non-plumbing row,
// a row whose Files lie in concern folders of two or more names (one concern may span role folders: rt
// model/battery and adapter/battery); then per concern folder in path order, a folder whose files two rows list.
// Folders are gathered per row, never per concern name, so rows of one concern in two parts stay apart.
func (s *scan) mm7(rows []row, owners map[string][]int) {
	first := map[string]int{} // lower-case concern + part -> the line of its first row
	for _, r := range rows {
		if isPlumbing(r) {
			continue
		}
		k := strings.ToLower(r.concern) + "\x00" + strings.ToLower(r.part)
		if at, dup := first[k]; dup {
			s.add(s.strictSev(), s.mapPath, r.line, "MM7: concern "+r.concern+" has a second "+r.part+" row (the first is line "+
				strconv.Itoa(at)+"); one row per concern and part")
			continue
		}
		first[k] = r.line
	}
	byRow := map[int]map[string]bool{}    // row idx -> its concern folders
	byFolder := map[string]map[int]bool{} // concern folder -> the rows listing its files
	for f, idxs := range owners {
		cf := ConcernFolder(f)
		if cf == "" || !isSource(f) {
			continue
		}
		for _, i := range idxs {
			if byRow[i] == nil {
				byRow[i] = map[string]bool{}
			}
			byRow[i][cf] = true
			if byFolder[cf] == nil {
				byFolder[cf] = map[int]bool{}
			}
			byFolder[cf][i] = true
		}
	}
	for _, r := range rows {
		names := map[string]bool{} // the folders' own names: model/battery and adapter/battery are one concern
		for cf := range byRow[r.idx] {
			names[path.Base(cf)] = true
		}
		if isPlumbing(r) || len(names) < 2 {
			continue
		}
		// the count is the one decided on: distinct folder NAMES, not folders (model/x + adapter/x count once)
		s.add(s.strictSev(), s.mapPath, r.line, "MM7: concern "+r.concern+" lists files in concern folders of "+
			strconv.Itoa(len(names))+" names ("+strings.Join(keys(names), ", ")+": "+strings.Join(keys(byRow[r.idx]), ", ")+
			"); keep one concern folder per row")
	}
	for _, cf := range keys(byFolder) {
		if len(byFolder[cf]) > 1 {
			idxs := make([]int, 0, len(byFolder[cf]))
			for i := range byFolder[cf] {
				idxs = append(idxs, i)
			}
			sort.Ints(idxs) // map order
			s.add(s.strictSev(), s.abs(cf), 0, "MM7: concern folder in "+strconv.Itoa(len(idxs))+" rows ("+
				strings.Join(concerns(rows, idxs), ", ")+"); one row owns a concern folder")
		}
	}
}

// isPlumbing: r is a plumbing row (any case), which may span concern folders and repeat.
func isPlumbing(r row) bool { return strings.EqualFold(r.concern, "plumbing") }

// concerns is the concern names of the rows idxs, in the order given (a name repeats when two rows share it).
func concerns(rows []row, idxs []int) []string {
	out := make([]string, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, rows[i].concern)
	}
	return out
}

// keys is m's keys in sorted order.
func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
