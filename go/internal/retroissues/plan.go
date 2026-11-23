package retroissues

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrodup"
	"github.com/angeles725/niagara-tools/go/internal/retrogh"
	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

// delta is one retro row that is to become (part of) an issue.
type delta struct {
	src   string // the stem of a kit retro, the alias of any other
	path  string
	inKit bool
	row   retrorows.Row
	area  string
	title string
	tok   map[string]int
}

func (d delta) key() string    { return d.src + " " + d.row.ID }
func (d delta) label() string  { return "retro " + d.key() }
func (d delta) marker() string { return retrogh.Marker(d.src, d.row.ID) }

// counts of what a run skipped.
type skipped struct {
	noArea, applied int
	tracked         []string // "tracked #N: retro <src> <id>"
}

// inKit reports whether path is a retro under <kitRoot>/retros.
func inKit(kitRoot, path string) bool {
	if kitRoot == "" {
		return false
	}
	abs, err1 := filepath.Abs(path)
	root, err2 := filepath.Abs(filepath.Join(kitRoot, "retros"))
	return err1 == nil && err2 == nil && strings.HasPrefix(abs, root+string(filepath.Separator))
}

// collect reads every retro and returns its rows that are still to be filed. A retro outside the kit with no alias is
// an error that names its position on the command line, never its stem.
func collect(o options) (ds []delta, sk skipped, err error) {
	for n, path := range o.files {
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, sk, fmt.Errorf("cannot read retro #%d of the command line: %v", n+1, strings.TrimPrefix(rerr.Error(), "open "+path+": "))
		}
		stem := retrorows.Stem(path)
		kit := inKit(o.kitRoot, path)
		src := stem
		if !kit {
			if src = o.aliases[stem]; src == "" {
				return nil, sk, fmt.Errorf("retro #%d of the command line is outside the kit: give it --alias <stem>=<ALIAS>, so its name never reaches GitHub", n+1)
			}
		}
		def := ""
		if kit {
			def = retrorows.AreaBuild
		}
		for _, r := range retrorows.Parse(string(b)) {
			d := delta{src: src, path: path, inKit: kit, row: r}
			switch {
			case len(o.only) > 0 && !o.only[r.ID] && !o.only[src+":"+r.ID]:
				continue
			case r.Done != "":
				sk.applied++
				continue
			case r.Issue != 0:
				sk.tracked = append(sk.tracked, fmt.Sprintf("tracked #%d: %s", r.Issue, d.label()))
				continue
			}
			d.area = o.area
			if d.area == "" {
				d.area = retrorows.Area(r, filepath.Base(path), def)
			}
			if d.area == "" {
				sk.noArea++
				continue
			}
			d.title = d.area + ": " + retrorows.Summary(r.Text, 98-len(d.area))
			d.tok = retrodup.Tokens(d.title, r.Text+" "+r.Target)
			ds = append(ds, d)
		}
	}
	return ds, sk, nil
}

// group is one planned issue: its members are the deltas that describe the same change.
type group struct {
	members []delta
	kind    string // create, exists, dup, fixed, refused
	issue   retrogh.Issue
	score   float64
	reason  string // refused: "<field> matches denylist rule #N"
	number  int    // the issue every member's row points at, once known
}

func (g *group) primary() delta { return g.members[0] }

func (g *group) labels() []string { return []string{retrogh.Label, "area:" + g.primary().area} }

func block(d delta) string {
	var b strings.Builder
	b.WriteString(d.row.Text + "\n")
	if d.row.Target != "" {
		b.WriteString("\n**Target:** " + d.row.Target + "\n")
	}
	if d.row.Evidence != "" {
		b.WriteString("\n**Evidence:** " + d.row.Evidence + "\n")
	}
	return b.String()
}

// body is the issue text: the primary delta, its source, every further source, then one hidden marker per source.
func (g *group) body() string {
	p := g.primary()
	b := block(p) + "\n**Source:** " + p.label() + "\n"
	for _, d := range g.members[1:] {
		b += "\n### Also proposed by " + d.label() + "\n\n" + block(d)
	}
	b += "\n"
	for _, d := range g.members {
		b += d.marker() + "\n"
	}
	return b
}

// comment is what an existing issue gets when a delta is a duplicate of it: the new source, with its hidden marker so a
// rerun skips it and the fold audit finds it.
func comment(d delta) string {
	return "also proposed by " + d.label() + "\n\n" + block(d) + "\n" + d.marker() + "\n"
}

// groupDeltas puts deltas of different retros that describe the same change (same area, token score at or above the
// threshold) into one group; a delta named by --new stays alone.
func groupDeltas(ds []delta, o options) []*group {
	var gs []*group
next:
	for _, d := range ds {
		if !o.renew[d.row.ID] && !o.renew[d.src+":"+d.row.ID] {
			for _, g := range gs {
				p := g.primary()
				if p.area == d.area && !g.has(d.src) && !o.renew[p.row.ID] && !o.renew[p.src+":"+p.row.ID] &&
					retrodup.Score(p.tok, d.tok) >= o.threshold {
					g.members = append(g.members, d)
					continue next
				}
			}
		}
		gs = append(gs, &group{members: []delta{d}, kind: "create"})
	}
	return gs
}

func (g *group) has(src string) bool {
	for _, m := range g.members {
		if m.src == src {
			return true
		}
	}
	return false
}
