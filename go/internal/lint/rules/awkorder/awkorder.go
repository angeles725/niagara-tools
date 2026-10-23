// Package awkorder reproduces the order gawk's `for (k in a)` visits the string keys of an array, for
// the rules that port gawk programs whose rows come out in that order (reference: rules/timers).
package awkorder

// Order is the order `for (k in a)` visits the string keys of a gawk 5 array that were inserted
// in the order given (str_array.c: awk_hash, chains prepended, tables of 13, 127, 1021, ... buckets
// grown when the element count over the bucket count exceeds STR_CHAIN_MAX = 2).
func Order(keys []string) []string {
	sizes := []uint32{13, 127, 1021, 8191, 16381, 32749, 65497, 131101}
	hashes := make(map[string]uint32, len(keys))
	size := sizes[0]
	buckets := make([][]string, size)
	for n, k := range keys {
		h := Hash(k)
		hashes[k] = h
		if uint32(n+1)/size > 2 && size != sizes[len(sizes)-1] {
			for _, s := range sizes {
				if s > size {
					size = s
					break
				}
			}
			grown := make([][]string, size)
			for _, chain := range buckets {
				for _, e := range chain {
					i := hashes[e] % size
					grown[i] = append([]string{e}, grown[i]...)
				}
			}
			buckets = grown
		}
		i := h % size
		buckets[i] = append([]string{k}, buckets[i]...)
	}
	out := make([]string, 0, len(keys))
	for _, chain := range buckets {
		out = append(out, chain...)
	}
	return out
}

// Hash is gawk's awk_hash (sdbm, h = 65599*h + c, kept to 32 bits).
func Hash(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h = 65599*h + uint32(int8(s[i])) // gawk hashes a signed char
	}
	return h
}
