package dedup

import (
	"sync"
	"testing"
)

func TestIsNew_FirstTime(t *testing.T) {
	d := New()
	if !d.IsNew("asset-1") {
		t.Fatal("first call should return true")
	}
}

func TestIsNew_Duplicate(t *testing.T) {
	d := New()
	d.IsNew("asset-1")
	if d.IsNew("asset-1") {
		t.Fatal("second call should return false")
	}
}

func TestIsNew_DifferentValues(t *testing.T) {
	d := New()
	if !d.IsNew("a") {
		t.Fatal("a should be new")
	}
	if !d.IsNew("b") {
		t.Fatal("b should be new")
	}
	if d.IsNew("a") {
		t.Fatal("a should not be new again")
	}
}

func TestMarkSeen(t *testing.T) {
	d := New()
	d.MarkSeen("pre-seen")
	if d.IsNew("pre-seen") {
		t.Fatal("MarkSeen should prevent IsNew from returning true")
	}
}

func TestCount(t *testing.T) {
	d := New()
	if d.Count() != 0 {
		t.Fatalf("expected 0, got %d", d.Count())
	}
	d.IsNew("a")
	d.IsNew("b")
	d.IsNew("a") // duplicate
	if d.Count() != 2 {
		t.Fatalf("expected 2, got %d", d.Count())
	}
}

func TestConcurrentIsNew(t *testing.T) {
	d := New()
	const n = 1000
	var wg sync.WaitGroup
	results := make([]bool, n)

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = d.IsNew("shared-key")
		}(i)
	}
	wg.Wait()

	// Exactly one goroutine should see it as new
	trueCount := 0
	for _, r := range results {
		if r {
			trueCount++
		}
	}
	if trueCount != 1 {
		t.Fatalf("expected exactly 1 true, got %d", trueCount)
	}
}

func TestConcurrentDifferentKeys(t *testing.T) {
	d := New()
	const n = 100
	var wg sync.WaitGroup

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			key := string(rune('a' + idx%26)) + string(rune('0'+idx/26))
			d.IsNew(key)
		}(i)
	}
	wg.Wait()

	if d.Count() == 0 {
		t.Fatal("expected some unique keys")
	}
}
