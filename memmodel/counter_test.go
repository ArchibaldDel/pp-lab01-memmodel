package memmodel

import "testing"

func TestParseOrdering(t *testing.T) {
	if _, ok := ParseOrdering("relaxed"); !ok {
		t.Fatal("relaxed обязан приниматься")
	}
	if _, ok := ParseOrdering("seqcst"); !ok {
		t.Fatal("seqcst обязан приниматься")
	}
	if _, ok := ParseOrdering("bogus"); ok {
		t.Fatal("неизвестный порядок обязан отвергаться")
	}
}

func TestCounterFetchAddIsExact(t *testing.T) {
	const n = 20000
	for _, threads := range []uint64{0, 1, 2, 8, 16, 33} {
		got := CounterFetchAdd(threads, n, SeqCst)
		want := threads * n
		if got != want {
			t.Errorf("threads=%d: получено %d, ожидалось %d", threads, got, want)
		}
	}
}

func TestCounterCASIsExact(t *testing.T) {
	const n = 20000
	for _, threads := range []uint64{0, 1, 2, 8, 16, 33} {
		got := CounterCAS(threads, n, SeqCst)
		want := threads * n
		if got != want {
			t.Errorf("threads=%d: получено %d, ожидалось %d", threads, got, want)
		}
	}
}

func TestCounterZeroThreadsGivesZero(t *testing.T) {
	if got := CounterFetchAdd(0, 1_000_000, SeqCst); got != 0 {
		t.Fatalf("0 потоков: получено %d, ожидалось 0", got)
	}
	if got := CounterCAS(0, 1_000_000, SeqCst); got != 0 {
		t.Fatalf("0 потоков (CAS): получено %d, ожидалось 0", got)
	}
}
