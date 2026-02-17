package parallel_test

import (
	"sync/atomic"
	"testing"

	"github.com/tmuxpack/tpack/internal/parallel"
)

func TestDo(t *testing.T) {
	var count atomic.Int64
	items := []int{1, 2, 3, 4, 5}

	parallel.Do(items, 2, func(v int) {
		count.Add(int64(v))
	})

	if got := count.Load(); got != 15 {
		t.Errorf("sum = %d, want 15", got)
	}
}

func TestDoEmpty(t *testing.T) {
	parallel.Do([]int{}, 2, func(_ int) {
		t.Fatal("should not be called")
	})
}

func TestDoSingleConcurrency(t *testing.T) {
	var maxConcurrent atomic.Int32
	var current atomic.Int32

	items := make([]int, 10)
	for i := range items {
		items[i] = i
	}

	parallel.Do(items, 1, func(_ int) {
		cur := current.Add(1)
		if cur > maxConcurrent.Load() {
			maxConcurrent.Store(cur)
		}
		current.Add(-1)
	})

	if got := maxConcurrent.Load(); got > 1 {
		t.Errorf("max concurrent = %d, want <= 1", got)
	}
}
