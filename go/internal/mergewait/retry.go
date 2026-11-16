package mergewait

import (
	"errors"
	"fmt"
)

// printedError is a gh error withRetries has already printed: the caller reports it again only as an
// outcome, never as a second line.
type printedError struct{ error }

func (e printedError) Unwrap() error { return e.error }

// withRetries prints every gh error and sends a call that failed on a network error (or hit its deadline)
// again, at most c.Retries more times, c.Interval apart. Any other error is returned at once. The error it
// returns is marked printed.
func (c Config) withRetries(gh Gh) Gh {
	return func(args ...string) (string, error) {
		for attempt := 0; ; attempt++ {
			out, err := gh(args...)
			if err == nil {
				return out, nil
			}
			fmt.Fprintf(c.Err, msgGhError, err)
			if !isNetworkError(err) || attempt >= c.Retries {
				return out, printedError{err}
			}
			fmt.Fprintf(c.Err, msgGhRetry, attempt+1, c.Retries, c.Interval)
			c.Sleep(c.Interval)
		}
	}
}

// report prints err unless withRetries already did: every error reaches stderr exactly once.
func (c Config) report(err error) {
	if !wasPrinted(err) {
		fmt.Fprintf(c.Err, msgGhError, err)
	}
}

func wasPrinted(err error) bool {
	var p printedError
	return errors.As(err, &p)
}

// ghFailed reports err and is the outcome of a gh call or reply that failed.
func (c Config) ghFailed(err error) int {
	c.report(err)
	return ExitGh
}
