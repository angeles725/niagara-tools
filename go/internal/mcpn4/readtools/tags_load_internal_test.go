package readtools

// loadTags (#1025): a client that batches gets every ord in one LoadMany call and its per-ord
// errors back unchanged; a client that cannot batch gets one LoadTree per ord, as before.

import (
	"errors"
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// singleLoader loads one ord per call and records the ords asked.
type singleLoader struct{ trees []string }

func (f *singleLoader) LoadTree(ord string, _ int, _ box.LoadOptions) (*pyjson.Object, error) {
	f.trees = append(f.trees, ord)
	if ord == "bad" {
		return nil, errors.New("no such ord")
	}
	return pyjson.NewObject(), nil
}
func (f *singleLoader) Open() (any, error)                                { return nil, nil }
func (f *singleLoader) GetOrd(string, float64, int) (string, bool, error) { return "", false, nil }
func (f *singleLoader) GetFile(string, float64, int) (*box.File, error)   { return nil, nil }
func (f *singleLoader) BaseURL() string                                   { return "" }
func (f *singleLoader) Close()                                            {}

// batchLoader also batches: LoadMany records each batch and answers "bad" with its own error.
type batchLoader struct {
	singleLoader
	batches [][]string
}

func (f *batchLoader) LoadMany(ords []string, _ int, _ box.LoadOptions) ([]box.LoadResult, error) {
	f.batches = append(f.batches, append([]string(nil), ords...))
	out := make([]box.LoadResult, len(ords))
	for i, o := range ords {
		if o == "bad" {
			out[i].Err = errors.New("no such ord")
		} else {
			out[i].Nodes = pyjson.NewObject()
		}
	}
	return out, nil
}

func toolsOn(c session.Client) *Tools {
	return &Tools{Ctx: &session.Context{Session: &session.Session{Client: c}}}
}

func TestLoadTagsBatchesWhenTheClientCan(t *testing.T) {
	c := &batchLoader{}
	ords := []string{"a", "bad", "c"}
	got, err := toolsOn(c).loadTags(ords)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.batches, [][]string{ords}) || len(c.trees) != 0 {
		t.Errorf("batches = %v, single loads = %v; want one batch of every ord, no single load", c.batches, c.trees)
	}
	if got[1].Err == nil || got[0].Err != nil || got[2].Err != nil {
		t.Errorf("errors = %v %v %v; want only the bad ord's", got[0].Err, got[1].Err, got[2].Err)
	}
}

func TestLoadTagsLoadsOneByOneWithoutBatching(t *testing.T) {
	c := &singleLoader{}
	ords := []string{"a", "bad", "c"}
	got, err := toolsOn(c).loadTags(ords)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.trees, ords) {
		t.Errorf("single loads = %v, want %v", c.trees, ords)
	}
	if got[1].Err == nil || got[0].Err != nil || got[2].Err != nil {
		t.Errorf("errors = %v %v %v; want only the bad ord's", got[0].Err, got[1].Err, got[2].Err)
	}
}
