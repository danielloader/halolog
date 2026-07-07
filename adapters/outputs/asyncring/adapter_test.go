// @author Admilson B. F. Cossa

package asyncring

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	jsonfmt "github.com/go-gen-ecosystem/halolog/adapters/formatters/json"
	"github.com/go-gen-ecosystem/halolog/types"
)

func newTestAdapter(t *testing.T, capacity int, onFull OnFull, w interface {
	Write([]byte) (int, error)
}) *RingAdapter {
	t.Helper()
	a, err := New(Options{Writer: w, Formatter: jsonfmt.NewJsonFormatter(), Capacity: capacity, OnFull: onFull, BatchSize: 64})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func entryWithField(key, val string) *types.LogEntry {
	return &types.LogEntry{
		Level: types.InfoLevel, Message: "m", TimestampUnix: 1_700_000_000_000_000_000,
		StaticFields:     []types.TypedFieldData{{Key: key, Val: types.StringValue(val)}},
		StaticFieldCount: 1,
	}
}

func countLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

func TestRingAdapter_OptionsValidation(t *testing.T) {
	if _, err := New(Options{Formatter: jsonfmt.NewJsonFormatter()}); err != ErrNoWriter {
		t.Errorf("missing Writer: got %v, want ErrNoWriter", err)
	}
	if _, err := New(Options{Writer: &bytes.Buffer{}}); err != ErrNoFormatter {
		t.Errorf("missing Formatter: got %v, want ErrNoFormatter", err)
	}
}

// TestRingAdapter_UseAfterRecycleSafe is the core regression: after WriteZero
// returns, mutating (recycling) the source entry must not change what gets
// serialized, because the adapter copied the record into ring-owned storage.
func TestRingAdapter_UseAfterRecycleSafe(t *testing.T) {
	var buf bytes.Buffer
	a := newTestAdapter(t, 16, Drop, &buf)

	e := entryWithField("user", "alice")
	_ = a.WriteZero(e)
	// Simulate the pooled entry being recycled/overwritten immediately.
	e.StaticFields[0].Val = types.StringValue("MALLORY")
	e.Message = "overwritten"
	e.StaticFieldCount = 0

	_ = a.Close()
	out := buf.String()
	if !strings.Contains(out, `"user":"alice"`) || strings.Contains(out, "MALLORY") {
		t.Fatalf("record was corrupted by source recycle: %s", out)
	}
}

// TestRingAdapter_NoLossUnderCapacity: everything written below capacity is
// serialized, in order, and drained by Close.
func TestRingAdapter_NoLossUnderCapacity(t *testing.T) {
	var buf bytes.Buffer
	const n = 500
	a := newTestAdapter(t, 1024, Block, &buf)
	for i := 0; i < n; i++ {
		_ = a.WriteZero(entryWithField("i", strings.Repeat("x", 0)+itoa(i)))
	}
	_ = a.Close()

	if got := countLines(buf.String()); got != n {
		t.Fatalf("wrote %d records, serialized %d", n, got)
	}
	if d := a.Dropped(); d != 0 {
		t.Fatalf("Block policy under capacity dropped %d", d)
	}
	// Order preserved (single consumer).
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for i, l := range lines {
		if !strings.Contains(l, `"i":"`+itoa(i)+`"`) {
			t.Fatalf("out of order at %d: %s", i, l)
			break
		}
	}
}

// gatedWriter blocks the first Write until released, and signals when entered, so
// a test can deterministically fill the ring while the consumer is stalled.
type gatedWriter struct {
	buf     bytes.Buffer
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
}

func (g *gatedWriter) Write(p []byte) (int, error) {
	g.once.Do(func() {
		close(g.entered)
		<-g.release
	})
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.buf.Write(p)
}

// TestRingAdapter_DropAccounting proves the no-silent-loss invariant: with a
// stalled consumer and a full ring under OnFull=Drop, every record is either
// serialized or counted in Dropped() — received + dropped == total.
func TestRingAdapter_DropAccounting(t *testing.T) {
	g := &gatedWriter{entered: make(chan struct{}), release: make(chan struct{})}
	a, err := New(Options{Writer: g, Formatter: jsonfmt.NewJsonFormatter(), Capacity: 4, OnFull: Drop, BatchSize: 1})
	if err != nil {
		t.Fatal(err)
	}

	const total = 200
	_ = a.WriteZero(entryWithField("k", "0")) // pull the writer into Write and stall it
	<-g.entered
	for i := 1; i < total; i++ {
		_ = a.WriteZero(entryWithField("k", itoa(i)))
	}
	close(g.release)
	_ = a.Close()

	g.mu.Lock()
	received := countLines(g.buf.String())
	g.mu.Unlock()
	dropped := int(a.Dropped())
	if dropped == 0 {
		t.Fatalf("expected some drops with a stalled consumer and cap=4")
	}
	if received+dropped != total {
		t.Fatalf("silent loss: received=%d dropped=%d total=%d", received, dropped, total)
	}
}

// TestRingAdapter_ZeroAllocProducer guards the producer hot path.
func TestRingAdapter_ZeroAllocProducer(t *testing.T) {
	a := newTestAdapter(t, 4096, Drop, &bytes.Buffer{})
	defer func() { _ = a.Close() }()
	e := entryWithField("user", "alice")
	if allocs := testing.AllocsPerRun(1000, func() { _ = a.WriteZero(e) }); allocs != 0 {
		t.Fatalf("producer must be 0 allocs/op, got %.2f", allocs)
	}
}

// TestRingAdapter_ConcurrentProducers exercises many producers under -race and
// asserts no silent loss.
func TestRingAdapter_ConcurrentProducers(t *testing.T) {
	var buf bytes.Buffer
	a := newTestAdapter(t, 4096, Block, &buf)

	const producers = 8
	const each = 1000
	var wg sync.WaitGroup
	wg.Add(producers)
	for p := 0; p < producers; p++ {
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				_ = a.WriteZero(entryWithField("k", "v"))
			}
		}()
	}
	wg.Wait()
	_ = a.Close()

	received := countLines(buf.String())
	if received+int(a.Dropped()) != producers*each {
		t.Fatalf("loss: received=%d dropped=%d total=%d", received, a.Dropped(), producers*each)
	}
	if a.Dropped() != 0 {
		t.Fatalf("Block policy dropped %d", a.Dropped())
	}
}

// TestRingAdapter_CloseNoLossWithConcurrentProducers closes the adapter WHILE
// producers are still writing (unlike the other tests, which wait first). This
// exercises the finalDrain shutdown ordering: every record must be either
// serialized or counted as dropped — never silently lost. Runs many iterations
// to hit the narrow in-flight-at-close window.
func TestRingAdapter_CloseNoLossWithConcurrentProducers(t *testing.T) {
	const iters = 300
	const producers = 4
	const each = 25
	for it := 0; it < iters; it++ {
		g := &syncBuf{}
		a := newTestAdapter(t, 64, Block, g)

		var wg sync.WaitGroup
		wg.Add(producers)
		for p := 0; p < producers; p++ {
			go func() {
				defer wg.Done()
				for i := 0; i < each; i++ {
					_ = a.WriteZero(entryWithField("k", "v"))
				}
			}()
		}
		// Close concurrently with the producers (do NOT wait first).
		_ = a.Close()
		wg.Wait()

		received := countLines(g.String())
		if received+int(a.Dropped()) != producers*each {
			t.Fatalf("iter %d: silent loss — received=%d dropped=%d total=%d",
				it, received, a.Dropped(), producers*each)
		}
	}
}

// syncBuf is a mutex-guarded buffer for tests that read while the writer runs.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// TestRingAdapter_Flush verifies Flush drains everything already accepted before
// it returns.
func TestRingAdapter_Flush(t *testing.T) {
	g := &syncBuf{}
	a := newTestAdapter(t, 1024, Block, g)
	defer func() { _ = a.Close() }()

	const n = 100
	for i := 0; i < n; i++ {
		_ = a.WriteZero(entryWithField("k", "v"))
	}
	if err := a.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if got := countLines(g.String()); got != n {
		t.Fatalf("after Flush: serialized %d, want %d", got, n)
	}
}

// TestRingAdapter_InterfaceAndClose exercises the interface glue and close
// idempotency/post-close behavior.
func TestRingAdapter_InterfaceAndClose(t *testing.T) {
	var buf bytes.Buffer
	a := newTestAdapter(t, 8, Drop, &buf)
	if a.Name() != "asyncring" {
		t.Errorf("Name = %q", a.Name())
	}
	a.SetFormatter(nil) // no-op by contract
	if a.Health() != nil {
		t.Errorf("Health should be nil")
	}
	_ = a.Write(entryWithField("k", "v")) // Write (not WriteZero)
	_ = a.WriteZero(nil)                  // nil entry is a no-op
	_ = a.Flush()
	_ = a.WriteErrors()

	_ = a.Close()
	_ = a.Close() // idempotent
	// A write after close is dropped, not a panic.
	if err := a.WriteZero(entryWithField("k", "v")); err != nil {
		t.Errorf("post-close write returned %v", err)
	}
	if a.Dropped() == 0 {
		t.Errorf("post-close write should be counted as dropped")
	}
}

// TestRingAdapter_BlockNeverDrops floods a tiny Block-policy ring; the producer
// spins while full and a fast writer drains, so nothing is dropped.
func TestRingAdapter_BlockNeverDrops(t *testing.T) {
	g := &syncBuf{}
	a, err := New(Options{Writer: g, Formatter: jsonfmt.NewJsonFormatter(), Capacity: 2, OnFull: Block, BatchSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	const n = 2000
	for i := 0; i < n; i++ {
		_ = a.WriteZero(entryWithField("k", "v"))
	}
	_ = a.Close()
	if got := countLines(g.String()); got != n {
		t.Fatalf("Block dropped records: serialized %d, want %d", got, n)
	}
	if a.Dropped() != 0 {
		t.Fatalf("Block dropped %d", a.Dropped())
	}
}

// itoa is a tiny allocation-free-enough helper for test labels.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}
