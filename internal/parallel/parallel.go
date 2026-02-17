// Package parallel provides lightweight helpers for bounded concurrent work.
package parallel

import "sync"

// Do runs fn for each item in items with at most maxConcurrent goroutines.
// It blocks until all work is complete.
func Do[T any](items []T, maxConcurrent int, fn func(T)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrent)
	for _, item := range items {
		wg.Add(1)
		go func(v T) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fn(v)
		}(item)
	}
	wg.Wait()
}
