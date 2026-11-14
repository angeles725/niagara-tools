package envfile

// Option changes how a file is read. Without one the read is what the package comment says.
type Option func(*reader)

// reader is the options of one read.
type reader struct {
	crLines *[]int // TrimCR: where the numbers of the stripped lines go
}

// TrimCR strips one carriage return from the end of every line that has one (a CRLF file) before the line
// is read, and appends the numbers of those lines to lines so the caller can say so. It is opt-in: a reader
// that does not ask keeps the carriage return in the value.
func TrimCR(lines *[]int) Option { return func(r *reader) { r.crLines = lines } }
