package readtools

import (
	"fmt"
	"sync"
)

// runPool calls work(0) .. work(n-1), at most haWorkers at once, and returns when all returned.
// Each call writes only its own slot of the caller's results, so the caller merges them in index
// order and the outcome does not depend on which call finished first. A call that panics does not
// end the process (a pool goroutine has no request goroutine to recover it): its panic is that
// item's error, errs[i], and the other items still run.
func runPool(n int, work func(i int)) (errs []error) {
	errs = make([]error, n)
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < min(haWorkers, n); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				errs[i] = callRecovered(i, work)
			}
		}()
	}
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return errs
}

// callRecovered is work(i), with a panic returned as an error.
func callRecovered(i int, work func(i int)) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("the read panicked: %v", p)
		}
	}()
	work(i)
	return nil
}
