// Pool micro-benchmarks - isolate each operation in GetPerPState/PutPerPState
package pool

import (
	"sync/atomic"
	"testing"
	"unsafe"
)

// Benchmark just the getPID calculation
func BenchmarkPool_GetPID(b *testing.B) {
	p := NewExtractedPool()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = p.getPID()
	}
}

// Benchmark just CAS operation on a slot
func BenchmarkPool_SingleCAS(b *testing.B) {
	var inUse atomic.Bool
	inUse.Store(false)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		inUse.CompareAndSwap(false, true)
		inUse.Store(false) // Reset for next iteration
	}
}

// Benchmark just atomic store (release)
func BenchmarkPool_AtomicStore(b *testing.B) {
	var inUse atomic.Bool

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		inUse.Store(false)
	}
}

// Benchmark stats increment
func BenchmarkPool_StatsIncrement(b *testing.B) {
	var counter atomic.Uint64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		counter.Add(1)
	}
}

// Benchmark array index calculation
func BenchmarkPool_ArrayIndex(b *testing.B) {
	p := NewExtractedPool()
	numP := p.numP
	slotsPerP := p.slotsPerP

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pID := uint(i % 20)
		baseIndex := int(pID%uint(numP)) * slotsPerP
		_ = baseIndex
	}
}

// Benchmark shard access
func BenchmarkPool_ShardAccess(b *testing.B) {
	p := NewExtractedPool()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		slot := &p.shards[i%len(p.shards)]
		_ = slot
	}
}

// Benchmark full GetPerPState (current impl)
func BenchmarkPool_GetPerPState_Current(b *testing.B) {
	p := NewExtractedPool()
	p.Warmup(32)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		state := p.GetPerPState()
		p.PutPerPState(state)
	}
}

// Benchmark: What if we skip the stats atomic?
func BenchmarkPool_GetPerPState_NoStats(b *testing.B) {
	p := NewExtractedPool()
	p.Warmup(32)
	numP := p.numP
	slotsPerP := p.slotsPerP

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Inline GetPerPState without stats
		var x int
		sp := uintptr(unsafe.Pointer(&x))
		pID := uint((sp >> 12) ^ (sp >> 20))
		baseIndex := int(pID%uint(numP)) * slotsPerP

		var state *ExtractedPerPState
		for j := 0; j < slotsPerP; j++ {
			slot := &p.shards[baseIndex+j]
			if slot.inUse.CompareAndSwap(false, true) {
				state = slot
				break
			}
		}
		if state != nil {
			state.inUse.Store(false)
		}
	}
}

// Benchmark: What if we use simple round-robin instead of P-ID?
func BenchmarkPool_GetPerPState_RoundRobin(b *testing.B) {
	p := NewExtractedPool()
	p.Warmup(32)
	var counter atomic.Uint64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simple round-robin
		idx := int(counter.Add(1) % uint64(len(p.shards)))
		slot := &p.shards[idx]
		if slot.inUse.CompareAndSwap(false, true) {
			slot.inUse.Store(false)
		}
	}
}

// Benchmark: What if we just use the iteration index (single-threaded)?
func BenchmarkPool_GetPerPState_DirectIndex(b *testing.B) {
	p := NewExtractedPool()
	p.Warmup(32)
	totalSlots := len(p.shards)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		idx := i % totalSlots
		slot := &p.shards[idx]
		slot.inUse.Store(true)
		slot.inUse.Store(false)
	}
}

// Baseline: What's the absolute minimum - just return a pre-acquired state
func BenchmarkPool_Baseline_NoPooling(b *testing.B) {
	p := NewExtractedPool()
	state := &p.shards[0]

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = state
	}
}
