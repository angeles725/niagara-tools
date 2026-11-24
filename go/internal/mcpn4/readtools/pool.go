package readtools

import "sync"

// runPool calls work(0) .. work(n-1), at most haWorkers at once, and returns when all returned.
// Each call writes only its own slot of the caller's results, so the caller merges them in index
// order and the outcome does not depend on which call finished first.
func runPool(n int, work func(i int)) {
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < min(haWorkers, n); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				work(i)
			}
		}()
	}
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}
