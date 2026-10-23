package box

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// handleCache is BoxClient._handles: ord -> node handle, in insertion order.
type handleCache struct {
	keys []string
	vals map[string]any
}

func (h *handleCache) get(k string) (any, bool) {
	v, ok := h.vals[k]
	return v, ok
}

func (h *handleCache) set(k string, v any) {
	if h.vals == nil {
		h.vals = map[string]any{}
	}
	if _, ok := h.vals[k]; !ok {
		h.keys = append(h.keys, k)
	}
	h.vals[k] = v
}

// drop deletes every key keep rejects.
func (h *handleCache) drop(match func(string) bool) {
	kept := h.keys[:0]
	for _, k := range h.keys {
		if match(k) {
			delete(h.vals, k)
		} else {
			kept = append(kept, k)
		}
	}
	h.keys = kept
}

// Handles returns the handle cache as a dict, in insertion order (a copy).
func (c *Client) Handles() *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range c.handles.keys {
		out.Set(k, c.handles.vals[k])
	}
	return out
}

// SetHandle caches handle for ord, as an earlier load of ord would.
func (c *Client) SetHandle(ord string, handle any) { c.handles.set(ord, handle) }

// InvalidateHandles forgets the cached handles of prefix and its descendants; the match is
// boundary-aware (.../A clears .../A and .../A/B, never .../AB).
func (c *Client) InvalidateHandles(prefix string) {
	base := strings.TrimRight(prefix, "/")
	c.handles.drop(func(k string) bool { return k == base || strings.HasPrefix(k, base+"/") })
}

// ClearHandles forgets every cached handle (invalidate_handles(None)).
func (c *Client) ClearHandles() { c.handles.drop(func(string) bool { return true }) }
