package readtools

// The contract reader under failures and concurrent listings (review of #678): a spec is
// remembered as undescribed only from a reply that answered, never after a failed call; nothing
// is asked for when every spec is remembered; one remembered set serves concurrent listings.

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ctfLoader is a registry and a plan client: each LoadTypes call is recorded and answered by
// answer (its 1-based call number and the specs asked).
type ctfLoader struct {
	mu     sync.Mutex
	calls  [][]string
	answer func(call int, specs []string) (any, error)
}

func (f *ctfLoader) LoadTypes(specs []string) (any, error) {
	f.mu.Lock()
	f.calls = append(f.calls, append([]string(nil), specs...))
	n := len(f.calls)
	f.mu.Unlock()
	return f.answer(n, specs)
}

func (f *ctfLoader) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) {
	return nil, errors.New("no tree")
}
func (f *ctfLoader) SSC(string, any) (any, error) { return nil, nil }
func (f *ctfLoader) Sync(any) (any, error)        { return nil, nil }
func (f *ctfLoader) ClearHandles()                {}

func (f *ctfLoader) asked() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.calls...)
}

// ctfAnswer answers every call with the same entries.
func ctfAnswer(entries *pyjson.Object) func(int, []string) (any, error) {
	return func(int, []string) (any, error) { return entries, nil }
}

func TestEverySpecRememberedAsksTheRegistryNothing(t *testing.T) {
	f := &ctfLoader{answer: ctfAnswer(obj("a:A", obj()))}
	var gone undescribed
	if _, err := loadContracts(f, &gone, []string{"a:A", "a:Gone"}); err != nil {
		t.Fatal(err)
	}
	before := len(f.asked())
	types, err := loadContracts(f, &gone, []string{"a:Gone"})
	if err != nil || types == nil || box.Node(types, "a:Gone") != nil {
		t.Errorf("types %v, err %v", types, err)
	}
	if got := f.asked()[before:]; len(got) != 0 {
		t.Errorf("asked %v with every spec remembered, want no call", got)
	}
}

func TestASpecIsRememberedOnlyFromAReplyThatAnswered(t *testing.T) {
	for _, tc := range []struct {
		name   string
		answer func(int, []string) (any, error)
		again  []string // what the second listing asks
	}{
		{"a failed call", func(n int, _ []string) (any, error) {
			if n == 1 {
				return nil, errors.New("connection reset")
			}
			return obj("a:A", obj()), nil
		}, []string{"a:A", "a:Gone"}},
		{"a reply that answered none", ctfAnswer(obj()), []string{"a:A", "a:Gone"}},
		{"a reply that answered another spec", ctfAnswer(obj("a:A", obj())), []string{"a:A"}},
		{"an error entry", ctfAnswer(obj("a:A", obj(), "a:Gone", obj("isErr", true, "m", "a:Gone"))), []string{"a:A"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &ctfLoader{answer: tc.answer}
			var gone undescribed
			_, _ = loadContracts(f, &gone, []string{"a:A", "a:Gone"})
			before := len(f.asked())
			if _, err := loadContracts(f, &gone, []string{"a:A", "a:Gone"}); err != nil {
				t.Fatal(err)
			}
			if got := f.asked()[before:]; len(got) == 0 || !reflect.DeepEqual(got[0], tc.again) {
				t.Errorf("second listing asked %v, want %v first", got, tc.again)
			}
		})
	}
}

// Run with -race: two listings share one remembered set.
func TestConcurrentListingsShareTheRememberedSpecs(t *testing.T) {
	f := &ctfLoader{answer: ctfAnswer(obj("a:A", obj()))}
	var gone undescribed
	var wg sync.WaitGroup
	for g := 0; g < 2; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				_, _ = loadContracts(f, &gone, []string{"a:A", "a:Gone" + string(rune('0'+g))})
			}
		}(g)
	}
	wg.Wait()
	if !gone.has("a:Gone0") || !gone.has("a:Gone1") {
		t.Errorf("remembered %v", gone.specs)
	}
}
