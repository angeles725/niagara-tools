package javasrc

// File is one Java source file as the lint engine hands it to every rule: its path, raw lines and the
// scanner results, each computed once on first use and shared by every rule that reads the file.
// It holds scanner-level data only; lint logic belongs in internal/lint.
type File struct {
	Path string   // the path as listed by the walker (rows print it unchanged)
	Raw  []string // the lines as Lines splits them ("\r" kept)

	stripped []string
	end      EndState
	methods  []Method
	scopes   []Method
	scanned  bool // stripped and end are set
	parsed   bool // methods is set
	scoped   bool // scopes is set
}

// NewFile wraps raw lines read from path.
func NewFile(path string, raw []string) *File { return &File{Path: path, Raw: raw} }

func (f *File) strip() {
	if !f.scanned {
		f.stripped, f.end = Strip(f.Raw)
		f.scanned = true
	}
}

// Stripped is Strip(Raw): comments blanked, literal text kept, line numbers unchanged.
func (f *File) Stripped() []string { f.strip(); return f.stripped }

// End is the lexical state Strip ended in.
func (f *File) End() EndState { f.strip(); return f.end }

// Methods is Methods(Stripped()).
func (f *File) Methods() []Method {
	if !f.parsed {
		f.methods = Methods(f.Stripped())
		f.parsed = true
	}
	return f.methods
}

// Scopes is Scopes(Stripped()).
func (f *File) Scopes() []Method {
	if !f.scoped {
		f.scopes = Scopes(f.Stripped())
		f.scoped = true
	}
	return f.scopes
}
