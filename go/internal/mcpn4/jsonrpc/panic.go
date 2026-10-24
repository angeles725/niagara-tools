package jsonrpc

import "fmt"

// Panic is the error of a tool that panicked. Python's _call answers any exception it did not
// expect with "internal error (<type>)" and the process lives on; a Go panic is reported the
// same way, as an Exception, its value never shown.
type Panic struct{ Value any }

func (p *Panic) Error() string { return fmt.Sprint(p.Value) }

// PyType is the class the reply names.
func (p *Panic) PyType() string { return "Exception" }

// Recover runs f and turns a panic into a *Panic error.
func Recover[T any](f func() (T, error)) (v T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			v, err = zero, &Panic{r}
		}
	}()
	return f()
}
