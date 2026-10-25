package batserrexit

import "strings"

// The parser builds the part of the bash grammar errexit depends on: a body is a sequence of and-or lists
// (separated by newline, ; or &), a list is pipelines joined by && or ||, a pipeline is commands joined by
// | with an optional leading !, and a command is a simple command (its words) or a compound one: if,
// while/until, for/select, case, a { } group, a ( ) subshell or a function definition, each holding the
// bodies of its conditions and branches. It never fails: a token it cannot place is skipped, so a file
// it misreads yields fewer rows, never an error.

// cmd is one command: words for a simple command, or the condition and branch bodies of a compound one.
type cmd struct {
	words    []string // a simple command's words
	compound string   // "", "if", "for", "while", "until", "case", "group", "subshell", "func"
	conds    [][]*list
	bodies   [][]*list
	line     int
}

// pipeline is commands joined by |; bang is a leading !.
type pipeline struct {
	bang bool
	cmds []*cmd
}

// list is an and-or list: pipes joined by ops (each "&&" or "||"), from its first token's line.
type list struct {
	pipes      []*pipeline
	ops        []string
	line       int
	start, end int
}

// parser reads a token slice from index i.
type parser struct {
	toks []token
	i    int
}

// cur is the current token, nil at the end.
func (p *parser) cur() *token {
	if p.i < len(p.toks) {
		return &p.toks[p.i]
	}
	return nil
}

func (p *parser) isWord(s string) bool {
	t := p.cur()
	return t != nil && t.kind == tWord && t.text == s
}

func (p *parser) isOp(s string) bool {
	t := p.cur()
	return t != nil && t.kind == tOp && t.text == s
}

// skipNL skips newlines and the ; and & separators.
func (p *parser) skipNL() {
	for t := p.cur(); t != nil && (t.kind == tNL || t.kind == tOp && (t.text == ";" || t.text == "&")); t = p.cur() {
		p.i++
	}
}

// statements parses lists until one of the stop words (at command start) or stop ops.
func (p *parser) statements(stopWords []string, stopOps []string) []*list {
	var out []*list
	for {
		p.skipNL()
		t := p.cur()
		if t == nil {
			return out
		}
		if t.kind == tWord && contains(stopWords, t.text) || t.kind == tOp && contains(stopOps, t.text) {
			return out
		}
		l := p.list(stopWords, stopOps)
		if l == nil {
			p.i++ // unparseable token, move on
			continue
		}
		out = append(out, l)
	}
}

// contains reports whether s is in a.
func contains(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}
	return false
}

// list parses one and-or list; nil when the next token starts none (a stop word, a closing token).
func (p *parser) list(stopWords, stopOps []string) *list {
	t := p.cur()
	l := &list{line: t.line, start: t.start}
	for {
		pl := p.pipeline(stopWords, stopOps)
		if pl == nil {
			break
		}
		l.pipes = append(l.pipes, pl)
		l.end = p.toks[p.i-1].end
		if p.isOp("&&") || p.isOp("||") {
			l.ops = append(l.ops, p.cur().text)
			p.i++
			for p.cur() != nil && p.cur().kind == tNL {
				p.i++
			}
			continue
		}
		break
	}
	if len(l.pipes) == 0 {
		return nil
	}
	return l
}

// pipeline parses one pipeline, its leading ! included.
func (p *parser) pipeline(stopWords, stopOps []string) *pipeline {
	pl := &pipeline{}
	if p.isWord("!") {
		pl.bang = true
		p.i++
	}
	for {
		c := p.command(stopWords, stopOps)
		if c == nil {
			break
		}
		pl.cmds = append(pl.cmds, c)
		if p.isOp("|") || p.isOp("|&") {
			p.i++
			for p.cur() != nil && p.cur().kind == tNL {
				p.i++
			}
			continue
		}
		break
	}
	if len(pl.cmds) == 0 {
		return nil
	}
	return pl
}

// reserved are the words that end a command list without starting a command.
var reserved = []string{"then", "elif", "else", "fi", "do", "done", "esac", "}", ";;"}

// command parses one simple or compound command, with the redirections after a compound one.
func (p *parser) command(stopWords, stopOps []string) *cmd {
	t := p.cur()
	if t == nil || t.kind == tNL {
		return nil
	}
	if t.kind == tOp {
		if t.text == "(" {
			p.i++
			c := &cmd{compound: "subshell", line: t.line}
			c.bodies = append(c.bodies, p.statements(nil, []string{")"}))
			if p.isOp(")") {
				p.i++
			}
			p.redirs()
			return c
		}
		return nil
	}
	if contains(stopWords, t.text) || contains(reserved, t.text) {
		return nil
	}
	switch t.text {
	case "if":
		p.i++
		c := &cmd{compound: "if", line: t.line}
		c.conds = append(c.conds, p.statements([]string{"then"}, nil))
		p.i++ // then
		for {
			c.bodies = append(c.bodies, p.statements([]string{"elif", "else", "fi"}, nil))
			if p.isWord("elif") {
				p.i++
				c.conds = append(c.conds, p.statements([]string{"then"}, nil))
				p.i++
				continue
			}
			if p.isWord("else") {
				p.i++
				continue
			}
			break
		}
		p.i++ // fi
		p.redirs()
		return c
	case "while", "until":
		p.i++
		c := &cmd{compound: t.text, line: t.line}
		c.conds = append(c.conds, p.statements([]string{"do"}, nil))
		p.i++
		c.bodies = append(c.bodies, p.statements([]string{"done"}, nil))
		p.i++
		p.redirs()
		return c
	case "for", "select":
		p.i++
		c := &cmd{compound: "for", line: t.line}
		for p.cur() != nil && !p.isWord("do") {
			p.i++
		}
		p.i++
		c.bodies = append(c.bodies, p.statements([]string{"done"}, nil))
		p.i++
		p.redirs()
		return c
	case "case":
		p.i++
		c := &cmd{compound: "case", line: t.line}
		for p.cur() != nil && !p.isWord("in") {
			p.i++
		}
		p.i++
		for {
			p.skipNL()
			if p.cur() == nil || p.isWord("esac") {
				break
			}
			for p.cur() != nil && !p.isOp(")") {
				p.i++
			}
			p.i++
			c.bodies = append(c.bodies, p.statements([]string{"esac"}, []string{";;"}))
			if p.isOp(";;") {
				p.i++
			}
		}
		p.i++
		p.redirs()
		return c
	case "{":
		p.i++
		c := &cmd{compound: "group", line: t.line}
		c.bodies = append(c.bodies, p.statements([]string{"}"}, nil))
		p.i++
		p.redirs()
		return c
	case "function":
		p.i += 2
		if p.isOp("(") {
			p.i += 2
		}
		p.skipNL()
		body := p.command(nil, nil)
		return &cmd{compound: "func", line: t.line, bodies: bodyOf(body)}
	}
	// name() compound
	if p.i+2 < len(p.toks) && p.toks[p.i+1].kind == tOp && p.toks[p.i+1].text == "(" &&
		p.toks[p.i+2].kind == tOp && p.toks[p.i+2].text == ")" {
		p.i += 3
		p.skipNL()
		body := p.command(nil, nil)
		return &cmd{compound: "func", words: []string{t.text}, line: t.line, bodies: bodyOf(body)}
	}
	c := &cmd{line: t.line}
	for t := p.cur(); t != nil && t.kind == tWord; t = p.cur() {
		c.words = append(c.words, t.text)
		p.i++
	}
	return c
}

// bodyOf is a function definition's body: the bodies of the compound command it defines.
func bodyOf(c *cmd) [][]*list {
	if c == nil {
		return nil
	}
	return c.bodies
}

// redirs skips the redirections after a compound command (`done < file`, `} 2>&1`).
func (p *parser) redirs() {
	for t := p.cur(); t != nil && t.kind == tWord && (strings.ContainsAny(t.text[:1], "<>0123456789&")); t = p.cur() {
		if !strings.ContainsAny(t.text, "<>") {
			return
		}
		p.i++
		if (strings.HasSuffix(t.text, ">") || strings.HasSuffix(t.text, "<")) && p.cur() != nil && p.cur().kind == tWord {
			p.i++
		}
	}
}
