// @author Admilson B. F. Cossa

package asyncring

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRoundUpPow2(t *testing.T) {
	cases := map[int]int{0: 2, 1: 2, 2: 2, 3: 4, 5: 8, 8: 8, 100: 128, 1024: 1024, 1025: 2048}
	for in, want := range cases {
		if got := roundUpPow2(in); got != want {
			t.Errorf("roundUpPow2(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestRing_FIFOSingleProducer(t *testing.T) {
	r := newRing[int](8)
	for i := 0; i < r.Cap(); i++ {
		if !r.enqueue(func(p *int) { *p = i }) {
			t.Fatalf("enqueue %d failed on a non-full ring", i)
		}
	}
	for i := 0; i < r.Cap(); i++ {
		var got int
		if !r.dequeue(func(p *int) { got = *p }) {
			t.Fatalf("dequeue %d failed on a non-empty ring", i)
		}
		if got != i {
			t.Fatalf("FIFO violated: got %d, want %d", got, i)
		}
	}
}

func TestRing_FullAndEmpty(t *testing.T) {
	r := newRing[int](4)
	var sink int
	if r.dequeue(func(p *int) { sink = *p }) {
		t.Fatal("dequeue on empty ring returned true")
	}
	for i := 0; i < r.Cap(); i++ {
		if !r.enqueue(func(p *int) { *p = i }) {
			t.Fatalf("enqueue %d failed below capacity", i)
		}
	}
	if r.enqueue(func(p *int) { *p = 99 }) {
		t.Fatal("enqueue on full ring returned true")
	}
	if !r.dequeue(func(p *int) { sink = *p }) {
		t.Fatal("dequeue on full ring failed")
	}
	_ = sink
	if !r.enqueue(func(p *int) { *p = 42 }) {
		t.Fatal("enqueue after making room failed")
	}
}

// TestRing_ConcurrentMPSC drives many producers and one consumer under -race,
// asserting every enqueued value is dequeued exactly once (no loss, no dup).
func TestRing_ConcurrentMPSC(t *testing.T) {
	const producers = 16
	const perProducer = 5000
	total := producers * perProducer

	r := newRing[int](1024)
	seen := make([]int32, total) // seen[v] must end at exactly 1
	var received int64
	done := make(chan struct{})

	// Single consumer.
	go func() {
		for atomic.LoadInt64(&received) < int64(total) {
			r.dequeue(func(p *int) {
				atomic.AddInt32(&seen[*p], 1)
				atomic.AddInt64(&received, 1)
			})
		}
		close(done)
	}()

	var wg sync.WaitGroup
	wg.Add(producers)
	for p := 0; p < producers; p++ {
		go func(base int) {
			defer wg.Done()
			for i := 0; i < perProducer; i++ {
				v := base + i
				for !r.enqueue(func(pp *int) { *pp = v }) {
					// ring full: let the consumer drain, then retry.
				}
			}
		}(p * perProducer)
	}
	wg.Wait()
	<-done

	for v := 0; v < total; v++ {
		if seen[v] != 1 {
			t.Fatalf("value %d dequeued %d times, want exactly 1", v, seen[v])
		}
	}
}
