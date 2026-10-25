// Package registry is the Python mcp-n4 surface the Go server serves verbatim: server name,
// version and protocol versions, the argparse usage and help text, and every tool schema in
// table order. It is data, not code: registry.json was generated from the Python server by
// mcp-n4-kit/tests/go_registry.py until S17 removed that server; it is now edited by hand, and
// the black-box suite (test_server: tools/list, initialize, the frozen protocol replies) pins
// what the binary serves from it.
package registry

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

//go:embed registry.json
var registryJSON []byte

// Tool is one MCP tool: its name, whether it needs an open station session, its input schema
// and the tools/list entry ({name, description, inputSchema, annotations}, as Python's
// Server._describe builds it).
type Tool struct {
	Name         string
	NeedsSession bool
	Schema       *pyjson.Object
	Descriptor   *pyjson.Object
}

// Registry is the decoded registry.json.
type Registry struct {
	Name, Version    string
	ProtocolVersions []string // newest first
	Usage, Help      string   // argparse format_usage() / format_help() at 80 columns
	Read, Write      []Tool   // tools_read.TOOLS / tools_write.TOOLS order
}

var (
	once    sync.Once
	loaded  *Registry
	loadErr error
)

// Default returns the embedded registry, decoded once. It panics when the embedded file is
// malformed: a build defect that the package tests catch.
func Default() *Registry {
	once.Do(func() { loaded, loadErr = Parse(registryJSON) })
	if loadErr != nil {
		panic(fmt.Sprintf("registry: embedded registry.json: %v", loadErr))
	}
	return loaded
}

// Parse decodes a registry document.
func Parse(data []byte) (*Registry, error) {
	v, err := pyjson.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	r := &Registry{}
	p := path{}
	root := p.object(v, "registry")
	srv, cli := p.object(p.get(root, "server"), "server"), p.object(p.get(root, "cli"), "cli")
	r.Name, r.Version = p.str(p.get(srv, "name"), "server.name"), p.str(p.get(srv, "version"), "server.version")
	for _, pv := range p.list(p.get(srv, "protocol_versions"), "server.protocol_versions") {
		r.ProtocolVersions = append(r.ProtocolVersions, p.str(pv, "protocol version"))
	}
	r.Usage, r.Help = p.str(p.get(cli, "usage"), "cli.usage"), p.str(p.get(cli, "help"), "cli.help")
	r.Read = p.tools(p.get(root, "read_tools"), "read_tools")
	r.Write = p.tools(p.get(root, "write_tools"), "write_tools")
	if p.err != nil {
		return nil, p.err
	}
	if len(r.ProtocolVersions) == 0 {
		return nil, fmt.Errorf("server.protocol_versions is empty")
	}
	return r, nil
}

// path decodes with a sticky first error, so Parse reads as the document's shape.
type path struct{ err error }

func (p *path) fail(format string, a ...any) {
	if p.err == nil {
		p.err = fmt.Errorf(format, a...)
	}
}

func (p *path) object(v any, what string) *pyjson.Object {
	o, ok := v.(*pyjson.Object)
	if !ok {
		p.fail("%s: want an object, got %T", what, v)
		return pyjson.NewObject()
	}
	return o
}

func (p *path) get(o *pyjson.Object, key string) any {
	v, ok := o.Get(key)
	if !ok {
		p.fail("missing key %q", key)
	}
	return v
}

func (p *path) str(v any, what string) string {
	s, ok := v.(string)
	if !ok {
		p.fail("%s: want a string, got %T", what, v)
	}
	return s
}

func (p *path) list(v any, what string) []any {
	l, ok := v.([]any)
	if !ok {
		p.fail("%s: want a list, got %T", what, v)
	}
	return l
}

func (p *path) tools(v any, what string) []Tool {
	var out []Tool
	for i, item := range p.list(v, what) {
		o := p.object(item, fmt.Sprintf("%s[%d]", what, i))
		t := Tool{Name: p.str(p.get(o, "name"), "tool name")}
		needs, ok := p.get(o, "needs_session").(bool)
		if !ok {
			p.fail("%s.needs_session: want a boolean", t.Name)
		}
		t.NeedsSession = needs
		t.Schema = p.object(p.get(o, "inputSchema"), t.Name+".inputSchema")
		t.Descriptor = pyjson.NewObject()
		for _, k := range []string{"name", "description", "inputSchema", "annotations"} {
			t.Descriptor.Set(k, p.get(o, k))
		}
		out = append(out, t)
	}
	return out
}
