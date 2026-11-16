package register

import "fmt"

// usageError is a bad command line or input: exit 2, after the usage text.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usageErrorf(format string, a ...any) error { return &usageError{fmt.Sprintf(format, a...)} }
