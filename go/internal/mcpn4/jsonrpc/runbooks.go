package jsonrpc

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ResourceNotFound is the MCP error code of resources/read for an unknown URI.
const ResourceNotFound = -32002

// runbookScheme prefixes the URI under which each runbook is a resource.
const runbookScheme = "runbook://"

// Runbook is one recurring procedure the server offers both as an MCP prompt (the body with its
// {{argument}} placeholders filled) and as an MCP resource (the body as written).
type Runbook struct {
	Name, Title, Description string
	Args                     []RunbookArg
	Body                     string
}

// RunbookArg is one prompt argument; an optional one fills its placeholder with Default.
type RunbookArg struct {
	Name, Description, Default string
	Required                   bool
}

type resourceNotFound struct{ msg string }

func (e *resourceNotFound) Error() string { return e.msg }

// isRunbookMethod reports whether name is one of the prompt or resource methods; they exist only
// when the server holds runbooks (else "method not found", as before they were added).
func (s *Server) isRunbookMethod(name string) bool {
	switch name {
	case "prompts/list", "prompts/get", "resources/list", "resources/read":
		return len(s.Runbooks) > 0
	}
	return false
}

func (s *Server) runbookMethod(name string, params any) (any, error) {
	switch name {
	case "prompts/list":
		list := make([]any, len(s.Runbooks))
		for i, b := range s.Runbooks {
			list[i] = obj("name", b.Name, "title", b.Title, "description", b.Description, "arguments", promptArgs(b))
		}
		return obj("prompts", list), nil
	case "prompts/get":
		return s.getPrompt(params)
	case "resources/list":
		list := make([]any, len(s.Runbooks))
		for i, b := range s.Runbooks {
			list[i] = obj("uri", runbookScheme+b.Name, "name", b.Name, "title", b.Title,
				"description", b.Description, "mimeType", "text/markdown")
		}
		return obj("resources", list), nil
	}
	return s.readResource(params)
}

func promptArgs(b Runbook) []any {
	out := make([]any, len(b.Args))
	for i, a := range b.Args {
		d := a.Description
		if a.Default != "" {
			d += " (default: " + a.Default + ")"
		}
		out[i] = obj("name", a.Name, "description", d, "required", a.Required)
	}
	return out
}

func (s *Server) book(name string) *Runbook {
	for i := range s.Runbooks {
		if s.Runbooks[i].Name == name {
			return &s.Runbooks[i]
		}
	}
	return nil
}

func (s *Server) getPrompt(params any) (any, error) {
	p, _ := params.(*pyjson.Object)
	name, ok := lookup(p, "name").(string)
	if !ok {
		return nil, &InvalidParams{"prompts/get needs a string 'name'"}
	}
	b := s.book(name)
	if b == nil {
		names := make([]string, len(s.Runbooks))
		for i, r := range s.Runbooks {
			names[i] = r.Name
		}
		return nil, &InvalidParams{"unknown prompt: " + name + " (known: " + strings.Join(names, ", ") + ")"}
	}
	given, err := promptValues(b, lookup(p, "arguments"))
	if err != nil {
		return nil, err
	}
	text := b.Body
	for _, a := range b.Args {
		v, ok := given[a.Name]
		if !ok {
			if a.Required {
				return nil, &InvalidParams{"prompt " + b.Name + " needs argument " + a.Name}
			}
			v = a.Default
		}
		text = strings.ReplaceAll(text, "{{"+a.Name+"}}", v)
	}
	msg := obj("role", "user", "content", obj("type", "text", "text", text))
	return obj("description", b.Description, "messages", []any{msg}), nil
}

// promptValues are the string arguments of a prompts/get call; an argument the runbook does not
// declare is refused (sorted, so the first one named is stable).
func promptValues(b *Runbook, raw any) (map[string]string, error) {
	out := map[string]string{}
	o, ok := raw.(*pyjson.Object)
	if !ok || o == nil {
		return out, nil
	}
	keys := o.Keys()
	sort.Strings(keys)
	for _, k := range keys {
		known := false
		for _, a := range b.Args {
			known = known || a.Name == k
		}
		if !known {
			return nil, &InvalidParams{"prompt " + b.Name + " has no argument " + k}
		}
		v, _ := o.Get(k)
		str, isStr := v.(string)
		if !isStr {
			return nil, &InvalidParams{"prompt argument " + k + " must be a string"}
		}
		out[k] = str
	}
	return out, nil
}

func (s *Server) readResource(params any) (any, error) {
	p, _ := params.(*pyjson.Object)
	uri, ok := lookup(p, "uri").(string)
	if !ok {
		return nil, &InvalidParams{"resources/read needs a string 'uri'"}
	}
	b := s.book(strings.TrimPrefix(uri, runbookScheme))
	if b == nil || !strings.HasPrefix(uri, runbookScheme) {
		return nil, &resourceNotFound{"resource not found: " + uri}
	}
	return obj("contents", []any{obj("uri", uri, "mimeType", "text/markdown", "text", b.Body)}), nil
}
