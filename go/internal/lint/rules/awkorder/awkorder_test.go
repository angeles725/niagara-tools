package awkorder

import (
	"math/rand"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// verifiedGawk is the gawk versions whose `for (k in a)` order Order was checked against (str_array.c
// hashing and table growth). A newer gawk may hash differently: the ubuntu-latest image 20261021
// ships one whose order differs from 39 keys on, so the differential runs only on these versions.
var verifiedGawk = regexp.MustCompile(`^GNU Awk 5\.[0-2]\.`)

// TestOrderDifferential checks Order against gawk's own `for (k in a)` order, table growth included.
func TestOrderDifferential(t *testing.T) {
	gawk, err := exec.LookPath("gawk")
	if err != nil {
		t.Skip("gawk not installed")
	}
	ver, err := exec.Command(gawk, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if first := strings.SplitN(string(ver), "\n", 2)[0]; !verifiedGawk.MatchString(first) {
		t.Skipf("%s: Order emulates gawk 5.0-5.2; the differential is not verified for this version", first)
	}
	rnd := rand.New(rand.NewSource(1))
	const first, rest = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_", "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789"
	for _, n := range []int{1, 2, 13, 38, 39, 40, 127, 381, 382, 1200} {
		seen := map[string]bool{}
		var keys []string
		for len(keys) < n {
			k := string(first[rnd.Intn(len(first))])
			for i := rnd.Intn(16); i > 0; i-- {
				k += string(rest[rnd.Intn(len(rest))])
			}
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
		cmd := exec.Command(gawk, `{ a[$0] = a[$0] "x" } END { for (k in a) print k }`)
		cmd.Stdin = strings.NewReader(strings.Join(keys, "\n") + "\n")
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.Fields(string(out)); !reflect.DeepEqual(Order(keys), want) {
			t.Errorf("%d keys: Order differs from gawk", n)
		}
	}
}

// TestHashSignedChar pins that a byte from 0x80 is hashed as gawk's signed char, not as an unsigned one.
func TestHashSignedChar(t *testing.T) {
	if got, want := Hash("\xff"), uint32(0xffffffff); got != want {
		t.Errorf("Hash(0xff) = %#x, want %#x", got, want)
	}
	if got, want := Hash("ab"), uint32(65599*97+98); got != want {
		t.Errorf("Hash(ab) = %d, want %d", got, want)
	}
}
