package memmodel

import (
	"sync"
	"sync/atomic"
)

// Ordering — метка порядка памяти, которую по протоколу harness принимают
// все три языка курса. В Go она не меняет кодогенерацию: sync/atomic весь
// целиком последовательно согласован (seq_cst), ослабленных порядков в
// языке нет (см. REPORT.md, часть A). Параметр сохранён ради единого
// интерфейса командной строки и явно эхуется в JSON-ответе.
type Ordering string

const (
	Relaxed Ordering = "relaxed"
	SeqCst  Ordering = "seqcst"
)

// ParseOrdering разбирает строковый аргумент harness.
func ParseOrdering(s string) (Ordering, bool) {
	switch Ordering(s) {
	case Relaxed, SeqCst:
		return Ordering(s), true
	default:
		return "", false
	}
}

// runOnThreads запускает threads горутин, каждая выполняет body, и ждёт
// завершения всех. При threads == 0 body не вызывается ни разу.
func runOnThreads(threads uint64, body func()) {
	var wg sync.WaitGroup
	wg.Add(int(threads))
	for i := uint64(0); i < threads; i++ {
		go func() {
			defer wg.Done()
			body()
		}()
	}
	wg.Wait()
}

// CounterFetchAdd — threads горутин делают по incrementsPerThread
// инкрементов готовой атомарной операцией (atomic.Uint64.Add — аналог
// fetch_add). ordering принят только ради протокола (см. тип Ordering).
func CounterFetchAdd(threads, incrementsPerThread uint64, _ Ordering) uint64 {
	var counter atomic.Uint64
	runOnThreads(threads, func() {
		for i := uint64(0); i < incrementsPerThread; i++ {
			counter.Add(1)
		}
	})
	return counter.Load()
}

// CounterCAS — тот же счётчик, но вручную, циклом compare-and-swap
// (аналог compare_exchange_weak: при неудаче старое значение уже
// обновлено load'ом внутри CompareAndSwap, повторяем попытку без
// дополнительного чтения).
func CounterCAS(threads, incrementsPerThread uint64, _ Ordering) uint64 {
	var counter atomic.Uint64
	runOnThreads(threads, func() {
		for i := uint64(0); i < incrementsPerThread; i++ {
			for {
				old := counter.Load()
				if counter.CompareAndSwap(old, old+1) {
					break
				}
			}
		}
	})
	return counter.Load()
}
