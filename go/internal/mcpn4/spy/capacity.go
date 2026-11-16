package spy

import (
	"strconv"
	"strings"
)

// Resource is one row of the Global Capacity table: the resource, how many the station counts
// and the licence limit. Used and Limit are nil when the cell is not a count ("none" is no
// limit); the cells are kept as text beside them.
type Resource struct {
	Type                string
	Used, Limit         *int
	UsedText, LimitText string
}

// Capacity is what the metrics page says of global capacity licensing: the resource counts and
// the rows of the Recount table (name, value) in page order.
type Capacity struct {
	Resources []Resource
	Recount   [][2]string
}

// LastRun is the value of the Recount row that says when the recount last ran, ok false when
// the page has no such row; NeverRan reports whether that value says it never did.
func (c *Capacity) LastRun() (string, bool) {
	for _, row := range c.Recount {
		if strings.EqualFold(row[0], recountLastRun) {
			return row[1], true
		}
	}
	return "", false
}

// NeverRan reports whether the page says the recount has never run.
func (c *Capacity) NeverRan() bool {
	v, ok := c.LastRun()
	return ok && strings.EqualFold(strings.TrimSpace(v), recountNever)
}

// ReadCapacity is the capacity of a parsed metrics page, nil when the page holds no Global
// Capacity table with Type, Limit and Used columns and at least one row: a layout this reader
// does not know is "no capacity", never a guessed count. skipped counts the rows of such a
// table whose cell count differs from its header: they are not resources, and not silently so.
func ReadCapacity(p *Page) (capacity *Capacity, skipped int) {
	var out *Capacity
	var recount [][2]string
	for _, t := range p.Tables {
		switch strings.ToLower(t.Title) {
		case capacityTitle:
			res, odd := resources(t)
			skipped += odd
			if len(res) > 0 && out == nil {
				out = &Capacity{Resources: res}
			}
		case recountTitle:
			for _, row := range t.Rows {
				if len(row) > 0 {
					recount = append(recount, [2]string{row[0], strings.Join(row[1:], " ")})
				}
			}
		}
	}
	if out != nil {
		out.Recount = recount
	}
	return out, skipped
}

func resources(t Table) (out []Resource, skipped int) {
	col := map[string]int{}
	for i, name := range t.Header {
		col[strings.ToLower(name)] = i
	}
	ti, okT := col[columnType]
	li, okL := col[columnLimit]
	ui, okU := col[columnUsed]
	if !okT || !okL || !okU {
		return nil, 0
	}
	for _, row := range t.Rows {
		if len(row) != len(t.Header) {
			skipped++
			continue
		}
		out = append(out, Resource{Type: row[ti], Used: count(row[ui]), UsedText: row[ui],
			Limit: count(row[li]), LimitText: row[li]})
	}
	return out, skipped
}

// NoLimit reports whether a limit cell says the resource is not limited.
func NoLimit(text string) bool { return strings.EqualFold(strings.TrimSpace(text), noLimit) }

// count is a cell as a count: digits, with groups of three after a ',', '.', space or
// no-break space separator; anything else is not a count.
func count(text string) *int {
	groups := strings.FieldsFunc(strings.TrimSpace(text), func(r rune) bool {
		return r == ',' || r == '.' || r == ' ' || r == ' ' || r == ' '
	})
	if len(groups) == 0 {
		return nil
	}
	for i, g := range groups {
		if strings.Trim(g, "0123456789") != "" || i > 0 && len(g) != 3 || i == 0 && len(groups) > 1 && len(g) > 3 {
			return nil
		}
	}
	n, err := strconv.Atoi(strings.Join(groups, ""))
	if err != nil {
		return nil
	}
	return &n
}
