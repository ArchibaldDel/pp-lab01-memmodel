package memmodel

import (
	"sync"
	"testing"
)

func TestSPSCCapacityRoundsUpToPowerOfTwo(t *testing.T) {
	cases := map[uint64]uint64{0: 1, 1: 1, 2: 2, 3: 4, 5: 8, 16: 16, 17: 32}
	for in, want := range cases {
		if got := NewSPSC(in).Capacity(); got != want {
			t.Errorf("NewSPSC(%d).Capacity() = %d, хотим %d", in, got, want)
		}
	}
}

func TestSPSCCapacityOne(t *testing.T) {
	q := NewSPSC(1)
	if q.Capacity() != 1 {
		t.Fatalf("capacity = %d, ожидалось 1", q.Capacity())
	}
	if !q.TryPush(7) {
		t.Fatal("push в пустую очередь ёмкости 1 обязан пройти")
	}
	if q.TryPush(8) {
		t.Fatal("очередь ёмкости 1 уже полна, push обязан вернуть false")
	}
	if v, ok := q.TryPop(); !ok || v != 7 {
		t.Fatalf("pop = (%d,%v), ожидалось (7,true)", v, ok)
	}
	if _, ok := q.TryPop(); ok {
		t.Fatal("очередь пуста, pop обязан вернуть false")
	}
}

func TestSPSCFIFOSingleThreaded(t *testing.T) {
	q := NewSPSC(4)
	for i := uint64(1); i <= 4; i++ {
		if !q.TryPush(i) {
			t.Fatalf("push %d должен пройти, очередь ещё не полна", i)
		}
	}
	if q.TryPush(99) {
		t.Fatal("очередь полна, push обязан вернуть false")
	}
	for i := uint64(1); i <= 4; i++ {
		v, ok := q.TryPop()
		if !ok || v != i {
			t.Fatalf("pop #%d: получено (%d,%v), ожидалось (%d,true)", i, v, ok, i)
		}
	}
	if _, ok := q.TryPop(); ok {
		t.Fatal("очередь пуста, pop обязан вернуть false")
	}
}

func TestSPSCConcurrentProducerConsumerPreservesOrderAndSum(t *testing.T) {
	const capacity = 8
	const items = 200000
	q := NewSPSC(capacity)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := uint64(1); i <= items; i++ {
			for !q.TryPush(i) {
				// очередь полна — крутимся, ждём потребителя
			}
		}
	}()

	var sum uint64
	ordered := true
	expect := uint64(1)
	for popped := uint64(0); popped < items; {
		v, ok := q.TryPop()
		if !ok {
			continue
		}
		sum += v
		if v != expect {
			ordered = false
		}
		expect++
		popped++
	}
	wg.Wait()

	if want := uint64(items) * (items + 1) / 2; sum != want {
		t.Fatalf("сумма %d, ожидалось %d", sum, want)
	}
	if !ordered {
		t.Fatal("порядок FIFO нарушен")
	}
}
