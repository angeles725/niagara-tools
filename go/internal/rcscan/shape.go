package rcscan

// shape is what a name's value is known to be: a literal collection (an array or object literal, one narrowed
// by filter / slice / sort / reverse, or one computed from a literal collection by .map), an element of one, a
// function that returns an element, or a function that returns a collection.
type shape struct {
	kind  int // collShape, elemShape, elemFnShape or collFnShape
	trees []*lit
}

const (
	collShape = iota
	elemShape
	elemFnShape
	collFnShape
)

// elems is the elements of a collection: an array's items, an object's field values.
func (sh *shape) elems() *shape {
	out := &shape{kind: elemShape}
	for _, t := range sh.trees {
		if t.dirty {
			out.trees = append(out.trees, nil)
		}
		out.trees = append(out.trees, t.elems...)
		for _, f := range t.fields {
			out.trees = append(out.trees, f)
		}
	}
	return out
}
