// Alternative pool strategies for sub-10ns performance
package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/go-gen-ecosystem/halolog/types"
)

// Strategy 1: Thread-local via sync.Pool (let Go runtime handle P-locality)
type SyncPoolStrategy struct {
	pool sync.Pool
}

func NewSyncPoolStrategy() *SyncPoolStrategy {
	return &SyncPoolStrategy{
		pool: sync.Pool{
			New: func() interface{} {
				return &ExtractedPerPState{}
			},
		},
	}
}

func (s *SyncPoolStrategy) Get() *ExtractedPerPState {
	return s.pool.Get().(*ExtractedPerPState)
}

func (s *SyncPoolStrategy) Put(state *ExtractedPerPState) {
	s.pool.Put(state)
}

// Strategy 2: Per-goroutine cache (reuse same state in hot loop)
type CachedStrategy struct {
	lastState *ExtractedPerPState
	pool      sync.Pool
}

func NewCachedStrategy() *CachedStrategy {
	return &CachedStrategy{
		pool: sync.Pool{
			New: func() interface{} {
				return &ExtractedPerPState{}
			},
		},
	}
}

func (s *CachedStrategy) Get() *ExtractedPerPState {
	// In single-threaded scenario, reuse last state
	if s.lastState != nil {
		return s.lastState
	}
	s.lastState = s.pool.Get().(*ExtractedPerPState)
	return s.lastState
}

func (s *CachedStrategy) Put(state *ExtractedPerPState) {
	// Don't put back - keep for reuse
}

// Strategy 3: Lock-free with TryLock pattern (optimistic)
type OptimisticPool struct {
	states    []ExtractedPerPState
	counter   atomic.Uint64
	numStates int
}

func NewOptimisticPool(n int) *OptimisticPool {
	return &OptimisticPool{
		states:    make([]ExtractedPerPState, n),
		numStates: n,
	}
}

func (p *OptimisticPool) Get() *ExtractedPerPState {
	// Optimistic: try current slot first (likely uncontended)
	idx := int(p.counter.Add(1)) % p.numStates
	return &p.states[idx]
}

func (p *OptimisticPool) Put(state *ExtractedPerPState) {
	// No-op for pre-allocated states
}

// Strategy 4: Static pre-allocated (no pooling at all for simple logging)
type StaticEntry struct {
	entry types.LogEntry
}

var globalStaticEntry = &StaticEntry{}

// Benchmarks
func BenchmarkStrategy_SyncPool(b *testing.B) {
	s := NewSyncPoolStrategy()
	// Warmup
	for i := 0; i < 1000; i++ {
		state := s.Get()
		s.Put(state)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state := s.Get()
		s.Put(state)
	}
}

func BenchmarkStrategy_Cached(b *testing.B) {
	s := NewCachedStrategy()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state := s.Get()
		s.Put(state)
	}
}

func BenchmarkStrategy_Optimistic(b *testing.B) {
	p := NewOptimisticPool(64)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state := p.Get()
		p.Put(state)
	}
}

func BenchmarkStrategy_Static(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry := &globalStaticEntry.entry
		_ = entry
	}
}

// Strategy 5: Inline everything - no pool abstraction
func BenchmarkStrategy_Inline(b *testing.B) {
	// Pre-allocate a single entry like in dispatchSimple
	var entry types.LogEntry

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		entry.Level = types.InfoLevel
		entry.Message = "test"
		entry.StaticFieldCount = 0
		_ = &entry
	}
}

// Real comparison: sync.Pool vs Current ExtractedPool
func BenchmarkComparison_SyncPool_vs_ExtractedPool(b *testing.B) {
	b.Run("SyncPool", func(b *testing.B) {
		pool := sync.Pool{
			New: func() interface{} {
				return &ExtractedPerPState{}
			},
		}
		// Warmup
		for i := 0; i < 1000; i++ {
			state := pool.Get().(*ExtractedPerPState)
			pool.Put(state)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			state := pool.Get().(*ExtractedPerPState)
			pool.Put(state)
		}
	})

	b.Run("ExtractedPool", func(b *testing.B) {
		p := NewExtractedPool()
		p.Warmup(32)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			state := p.GetPerPState()
			p.PutPerPState(state)
		}
	})
}

// What Go's sync.Pool actually costs
func BenchmarkPure_SyncPool_TypeAssert(b *testing.B) {
	pool := sync.Pool{
		New: func() interface{} {
			return &ExtractedPerPState{}
		},
	}
	// Warmup
	for i := 0; i < 1000; i++ {
		pool.Put(&ExtractedPerPState{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state := pool.Get().(*ExtractedPerPState)
		pool.Put(state)
	}
}

// sync.Pool without type assertion (using unsafe)
func BenchmarkPure_SyncPool_Unsafe(b *testing.B) {
	pool := sync.Pool{
		New: func() interface{} {
			return &ExtractedPerPState{}
		},
	}
	// Warmup
	for i := 0; i < 1000; i++ {
		pool.Put(&ExtractedPerPState{})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ptr := pool.Get()
		state := (*ExtractedPerPState)((*[2]unsafe.Pointer)(unsafe.Pointer(&ptr))[1])
		_ = state
		pool.Put(ptr)
	}
}
