package writeops

import (
	"math"
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// pySort is sorted(items, key=key) in place for decoded values, with Python's tuple and value
// ordering and its TypeError when two keys cannot be ordered. It is CPython's list sort for
// fewer than 64 items (one run: count_run, then binary insertion), so the comparisons made, and
// the operands of a TypeError, are CPython's; longer lists merge as stably.
func pySort(items []any, key func(any) []any) error {
	less := func(i, j int) (bool, error) { return seqLess(key(items[i]), key(items[j])) }
	n, err := countRun(items, less)
	if err != nil {
		return err
	}
	for ; n < len(items); n++ { // binarysort from start n
		pivot, k := items[n], key(items[n])
		lo, hi := 0, n
		for lo < hi {
			mid := lo + (hi-lo)/2
			lt, err := seqLess(k, key(items[mid]))
			if err != nil {
				return err
			}
			if lt {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		copy(items[lo+1:n+1], items[lo:n])
		items[lo] = pivot
	}
	return nil
}

// countRun is CPython's count_run (3.13+): the length of the monotone run items starts with,
// a descending one (equal items kept in order) reversed in place into an ascending one.
func countRun(items []any, less func(i, j int) (bool, error)) (int, error) {
	total, n := len(items), 1
	for ; n < total; n++ { // ascending first
		if lt, err := less(n, n-1); err != nil || lt {
			if err != nil {
				return 0, err
			}
			break
		}
	}
	if n >= total {
		return total, nil
	}
	if n > 1 {
		lt, err := less(0, n-1)
		if err != nil || lt {
			return n, err
		}
		reverse(items[:n]) // all equal
	}
	n++
	neq := 0
	flush := func() { // reverse the last neq+1 equal items
		if neq > 0 {
			reverse(items[n-neq-1 : n])
			neq = 0
		}
	}
	for ; n < total; n++ {
		smaller, err := less(n, n-1)
		if err != nil {
			return 0, err
		}
		if smaller {
			flush()
			continue
		}
		larger, err := less(n-1, n)
		if err != nil {
			return 0, err
		}
		if larger {
			break
		}
		neq++
	}
	flush()
	reverse(items[:n])
	for ; n < total; n++ { // an ascending suffix may extend it
		lt, err := less(n, n-1)
		if err != nil {
			return 0, err
		}
		if lt {
			break
		}
	}
	return n, nil
}

func reverse(s []any) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// seqLess is a < b for tuples or lists: the first unequal items decide, else the shorter.
func seqLess(a, b []any) (bool, error) {
	for i := 0; i < len(a) && i < len(b); i++ {
		if !pyEq(a[i], b[i]) {
			return pyLess(a[i], b[i])
		}
	}
	return len(a) < len(b), nil
}

// pyEq is a == b as the sort sees it: two NaNs json.loads returns are one object (identity
// makes them equal) and two equal infinities are equal.
func pyEq(a, b any) bool {
	if x, ok := num(a); ok {
		if y, ok := num(b); ok && (x.r == nil || y.r == nil) {
			return x.r == nil && y.r == nil && (x.f == y.f || math.IsNaN(x.f) && math.IsNaN(y.f))
		}
	}
	return box.PyEqual(a, b)
}

// pyLess is a < b for decoded values: strings by code point, numbers (and bools) by value,
// lists element-wise; anything else is Python's TypeError.
func pyLess(a, b any) (bool, error) {
	if x, ok := a.(string); ok {
		if y, ok := b.(string); ok {
			return x < y, nil // WTF-8 bytes sort as code points
		}
	}
	if x, ok := a.([]any); ok {
		if y, ok := b.([]any); ok {
			return seqLess(x, y)
		}
	}
	if x, ok := num(a); ok {
		if y, ok := num(b); ok {
			return numLess(x, y), nil
		}
	}
	return false, &plan.PyError{Type: "TypeError", Msg: "'<' not supported between instances of '" +
		plan.TypeName(a) + "' and '" + plan.TypeName(b) + "'"}
}

// num is a bool or Number as a float (for nan and the infinities) and, when finite, exactly.
type number struct {
	f float64
	r *big.Rat // nil for nan and the infinities
}

func num(v any) (number, bool) {
	switch x := v.(type) {
	case bool:
		if x {
			return number{1, big.NewRat(1, 1)}, true
		}
		return number{0, new(big.Rat)}, true
	case pyjson.Number:
		if i, ok := x.BigInt(); ok {
			return number{0, new(big.Rat).SetInt(i)}, true
		}
		f, _ := x.Float64()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return number{f, nil}, true
		}
		return number{f, new(big.Rat).SetFloat64(f)}, true
	}
	return number{}, false
}

func numLess(x, y number) bool {
	if x.r != nil && y.r != nil {
		return x.r.Cmp(y.r) < 0
	}
	if math.IsNaN(x.f) || math.IsNaN(y.f) {
		return false
	}
	switch {
	case x.r == nil && math.IsInf(x.f, 1): // +inf is less than nothing
		return false
	case x.r == nil: // -inf
		return !(y.r == nil && math.IsInf(y.f, -1))
	case math.IsInf(y.f, 1):
		return true
	}
	return false // y is -inf
}
