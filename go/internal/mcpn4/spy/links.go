package spy

// closeLink files the open link: a breadcrumb inside the navigation table, else a child link.
func (p *parser) closeLink() {
	if p.link == nil {
		return
	}
	name, long := clip(oneLine(p.link.String()), MaxNameRunes)
	href := p.href
	p.link = nil
	if long {
		p.cut(CutName)
	}
	if b := p.top(); b != nil && b.nav {
		p.crumbs = append(p.crumbs, name)
		return
	}
	path, ok := linkPath(href)
	switch {
	case !ok:
		p.page.LinksSkipped++
	case p.seen[path]:
	case len(p.page.Links) >= MaxLinks:
		p.cut(CutLinks)
	default:
		p.seen[path] = true
		p.page.Links = append(p.page.Links, Link{Name: name, Path: path})
	}
}

// addAction names the page a POST form posts to, once, as a clean spy path (see actionPath);
// an action that is not one is left out.
func (p *parser) addAction(action string) {
	path, ok := actionPath(action)
	switch {
	case !ok || p.acted[path]:
	case len(p.page.Actions) >= MaxActions:
		p.cut(CutActions)
	default:
		p.acted[path] = true
		p.page.Actions = append(p.page.Actions, path)
	}
}
