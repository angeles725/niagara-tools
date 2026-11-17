package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// maxReplicas bounds one replication (points x targets). Each add spends one write of --max-writes
// (200 by default); its rollback is outside the budget. A failure at add k leaves the batch in doubt: its
// n4_rollback removes the copies the station accepted that the read-back finds as added, and keeps
// add k and the ones never sent, each with its verdict (rollback_replica.go).
const maxReplicas = 200

// replicaTarget is one targets[] entry: the parent that receives the copies and, by source point
// name, the copy's own name and address when its copies[] entry gives them.
type replicaTarget struct {
	ord    string
	names  map[string]string
	addrOf map[string]addressOverride
}

// newName is the name the copy of point gets under the target: its copies[] new_name, else the
// source's own name.
func (t replicaTarget) newName(point string) string {
	if n, ok := t.names[point]; ok {
		return n
	}
	return point
}

// replicaArgs validates points (1 or more distinct existing slot names), targets (distinct
// parent ORDs) and their copies[]; points x targets must stay within maxReplicas.
func replicaArgs(args *pyjson.Object) ([]string, []replicaTarget, error) {
	rawPoints, _ := args.Get("points")
	pointList, _ := rawPoints.([]any)
	rawTargets, _ := args.Get("targets")
	targetList, _ := rawTargets.([]any)
	if len(pointList) == 0 || len(targetList) == 0 {
		return nil, nil, plan.ToolError("points and targets must each list at least one entry")
	}
	if n := len(pointList) * len(targetList); n > maxReplicas {
		return nil, nil, plan.ToolError(strconv.Itoa(len(pointList)) + " point(s) x " + strconv.Itoa(len(targetList)) +
			" target(s) = " + strconv.Itoa(n) + " copies; one replication holds at most " + strconv.Itoa(maxReplicas) +
			": split it by targets")
	}
	points, seen := []string{}, map[string]bool{}
	for _, v := range pointList {
		p, err := plan.ExistingName("points[]", v)
		if err != nil {
			return nil, nil, err
		}
		if seen[p] {
			return nil, nil, plan.ToolError("points lists " + pyjson.StrRepr(p) + " twice")
		}
		seen[p] = true
		points = append(points, p)
	}
	targets, ords := []replicaTarget{}, map[string]bool{}
	for _, v := range targetList {
		entry, _ := v.(*pyjson.Object)
		o, _ := get(entry, "parent_ord").(string)
		key := strings.TrimRight(strings.TrimSpace(o), "/") // "<ord>" and "<ord>/" name one parent
		if key == "" || ords[key] {
			return nil, nil, plan.ToolError("targets[].parent_ord must be distinct, non-empty ORDs; got " + pyjson.Repr(get(entry, "parent_ord")))
		}
		ords[key] = true
		t, err := replicaCopies(o, get(entry, "copies"), seen)
		if err != nil {
			return nil, nil, err
		}
		targets = append(targets, t)
	}
	return points, targets, nil
}

// replicaCopies reads one target's copies[]: each entry names one of points, at most once, with a
// plain new_name and/or a non-empty data_address (address_format optional).
func replicaCopies(o string, raw any, points map[string]bool) (replicaTarget, error) {
	t := replicaTarget{ord: o, names: map[string]string{}, addrOf: map[string]addressOverride{}}
	list, _ := raw.([]any)
	done := map[string]bool{}
	for _, e := range list {
		entry, _ := e.(*pyjson.Object)
		p, _ := get(entry, "point").(string)
		where := "copies[] of target " + o
		if !points[p] {
			return t, plan.ToolError(where + ": point " + pyjson.StrRepr(p) + " is not one of points")
		}
		if done[p] {
			return t, plan.ToolError(where + ": point " + pyjson.StrRepr(p) + " is given twice")
		}
		done[p] = true
		if n, has := entry.Get("new_name"); has {
			name, err := plan.Name("new_name", n)
			if err != nil {
				return t, err
			}
			t.names[p] = name
		}
		if a, has := entry.Get("data_address"); has {
			address, _ := a.(string)
			if strings.TrimSpace(address) == "" {
				return t, plan.ToolError(where + ": data_address of " + pyjson.StrRepr(p) + " must be a non-empty address")
			}
			format, _ := get(entry, "address_format").(string)
			t.addrOf[p] = addressOverride{address, format}
		}
	}
	return t, nil
}
