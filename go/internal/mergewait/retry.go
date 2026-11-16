package mergewait

import "fmt"

// withRetries prints every gh error and sends a call that failed on a network error again, at most
// c.Retries more times, c.Interval apart. Any other error is returned at once.
func (c Config) withRetries(gh Gh) Gh {
	return func(args ...string) (string, error) {
		for attempt := 0; ; attempt++ {
			out, err := gh(args...)
			if err == nil {
				return out, nil
			}
			fmt.Fprintf(c.Err, msgGhError, err)
			if !isNetworkError(err) || attempt >= c.Retries {
				return out, err
			}
			fmt.Fprintf(c.Err, msgGhRetry, attempt+1, c.Retries, c.Interval)
			c.Sleep(c.Interval)
		}
	}
}
