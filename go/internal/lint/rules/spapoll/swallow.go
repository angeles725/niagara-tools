package spapoll

import "strings"

// swallow.go is the FAIL half (issue #895): a poll catch that swallows the failure and then stamps lastOk*.

// catchWord finds a block or promise catch (exitsCatch, handler.go, reads its handler); conditional marks a
// stamp that is guarded (if/else/loop/switch/return/throw, ?:, &&, ||) and so is not this shape.
var (
	catchWord   = rx(`catch[[:space:]]*[({]`)
	conditional = rx(`(^|[^A-Za-z0-9_$])(if|else|while|for|switch|return|throw)([^A-Za-z0-9_$]|$)|\?|&&|\|\|`)
)

const tailSwallowed = " swallows a failure: its catch path neither rethrows nor returns and falls through to a" +
	" lastOk* clock stamp, so the success-time watchdog never sees the failure -- rethrow in the catch (or return)," +
	" or stamp lastOk* only on the validated success path (types/dashboard.md Poll loop reliability)"

// swallowsThenStamps: the poll body catches a failure without rethrowing or returning, and an unconditional
// lastOk* clock stamp follows that catch in the same block, so the failure reaches the stamp as a success. A
// block catch (`try {..} catch (e) {..}`) and a promise catch (`.catch(handler)`) are read alike. A stamp inside
// the try or the then-handler (before the catch), or one behind an if/?:/&&/||/return/throw, is not this shape.
func swallowsThenStamps(b string) bool {
	depth := make([]int, len(b)+1) // brace depth before each byte
	for i := 0; i < len(b); i++ {
		depth[i+1] = depth[i]
		switch b[i] {
		case '{':
			depth[i+1]++
		case '}':
			depth[i+1]--
		}
	}
	for _, at := range catchWord.FindAllStringIndex(b, -1) {
		start := at[0]
		end := start + len("catch") // just past the keyword
		var handler string
		if start > 0 && b[start-1] == '.' {
			stop := matching(b, end, '(', ')')
			if stop < 0 {
				continue
			}
			handler, end = b[end:stop], stop
		} else {
			open := strings.IndexByte(b[end:], '{')
			if open < 0 {
				continue
			}
			stop := matching(b, end+open, '{', '}')
			if stop < 0 {
				continue
			}
			handler, end = b[end+open:stop], stop
		}
		if exitsCatch(handler) {
			continue
		}
		for _, m := range okSet.FindAllStringIndex(b[end:], -1) {
			p := end + m[0]
			if depth[p] == depth[start] && !conditional.MatchString(b[end:p]) {
				return true
			}
		}
	}
	return false
}

// matching is the index of the byte that closes the open byte found at or after from (len(b) bound), or -1.
func matching(b string, from int, open, shut byte) int {
	level := 0
	for i := from; i < len(b); i++ {
		switch b[i] {
		case open:
			level++
		case shut:
			level--
			if level == 0 {
				return i
			}
		}
	}
	return -1
}
