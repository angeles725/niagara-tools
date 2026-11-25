package readtools

// ReadTags (#1049): over the real BOX client, which batches its loads (LoadMany), the reply and
// the first error are those of the same read over a client that loads one ord at a time.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// treeOnly is a box.Client seen through LoadTree alone: ReadTags cannot batch on it.
type treeOnly struct{ c *box.Client }

func (f treeOnly) Open() (any, error) { return f.c.Open() }
func (f treeOnly) LoadTree(o string, d int, opt box.LoadOptions) (*pyjson.Object, error) {
	return f.c.LoadTree(o, d, opt)
}
func (f treeOnly) GetOrd(o string, t float64, n int) (string, bool, error) {
	return f.c.GetOrd(o, t, n)
}
func (f treeOnly) GetFile(p string, t float64, n int) (*box.File, error) { return f.c.GetFile(p, t, n) }
func (f treeOnly) BaseURL() string                                       { return f.c.BaseURL() }
func (f treeOnly) Close()                                                { f.c.Close() }

const tagPoints = 3

// tagStation holds Folder/Pt0..Pt2, each with its own marker tag and string tag.
func tagStation(t *testing.T) *boxstation.Station {
	st := boxstation.New("admin", "pw", "S").Start(t)
	for i := range tagPoints {
		_, h, err := st.Add("3", fmt.Sprintf("Pt%d", i), "kitControl:NumericConst")
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []map[string]any{
			{"nm": "p", "t": "baja:Marker", "v": "M"}, {"nm": "p", "t": "baja:String", "v": fmt.Sprint("v", i)},
		} {
			name := "hs$3apoint"
			if s["t"] == "baja:String" {
				name = "tst$3alabel"
			}
			if _, err := st.AddSlot(h, name, s, "m"); err != nil {
				t.Fatal(err)
			}
		}
	}
	return st
}

// numberReplies serves st with each reply entry of a frame carrying the r of its message: the
// station fake answers every entry r 0, which a frame of several messages cannot match back.
func numberReplies(t *testing.T, st *boxstation.Station) string {
	target, _ := url.Parse(st.URL)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ModifyResponse = func(resp *http.Response) error {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		var frame map[string]any
		if json.Unmarshal(body, &frame) == nil {
			if entries, ok := frame["m"].([]any); ok {
				for i, e := range entries {
					e.(map[string]any)["r"] = i
				}
				body, _ = json.Marshal(frame)
			}
		}
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		return nil
	}
	srv := httptest.NewServer(proxy)
	t.Cleanup(srv.Close)
	return srv.URL
}

// readTags is ReadTags(ords) with implied tags off, on a fresh open client of st (batching unless
// single), and the BOX frames it took.
func readTags(t *testing.T, st *boxstation.Station, single bool, ords ...string) (string, error, int) {
	t.Helper()
	c, err := box.New(box.Config{BaseURL: numberReplies(t, st), Username: "admin", Password: "pw", AllowHTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	var client session.Client = c
	if single {
		client = treeOnly{c}
	}
	tools := &Tools{Ctx: &session.Context{Clock: func() float64 { return 0 }, Session: &session.Session{Client: client}}}
	list := make([]any, len(ords))
	for i, o := range ords {
		list[i] = o
	}
	base := c.Frames()
	out, err := tools.ReadTags(obj("ords", list, "implied", false))
	if err != nil {
		return "", err, c.Frames() - base
	}
	return pyjson.Repr(out), nil, c.Frames() - base
}

func TestReadTagsOverABatchingClientReplyIsTheLoadTreeReply(t *testing.T) {
	st := tagStation(t)
	ords := []string{"station:|slot:/Folder/Pt2", "station:|slot:/Folder/Pt0", "station:|slot:/Folder/Pt1"}
	batched, err, nBatch := readTags(t, st, false, ords...)
	if err != nil {
		t.Fatal(err)
	}
	single, err, nSingle := readTags(t, st, true, ords...)
	if err != nil {
		t.Fatal(err)
	}
	if batched != single {
		t.Errorf("batched reply\n%s\nLoadTree reply\n%s", batched, single)
	}
	if nBatch >= nSingle {
		t.Errorf("batched read took %d frames, one at a time %d", nBatch, nSingle)
	}
}

func TestReadTagsOverABatchingClientFailsOnTheFirstBadOrdInOrder(t *testing.T) {
	st := tagStation(t)
	pt := func(i int) string { return fmt.Sprintf("station:|slot:/Folder/Pt%d", i) }
	for _, ords := range [][]string{
		{pt(0), "station:|slot:/Folder/Nope2", pt(1), "station:|slot:/Folder/Nope1"},
		{"station:|slot:/Folder/Nope1", pt(0), "station:|slot:/Folder/Nope2"},
	} {
		_, errBatch, _ := readTags(t, st, false, ords...)
		_, errSingle, _ := readTags(t, st, true, ords...)
		if errBatch == nil || errSingle == nil || errBatch.Error() != errSingle.Error() {
			t.Errorf("%v: batched error %v, LoadTree error %v", ords, errBatch, errSingle)
		}
	}
}
