package memmodel

import "sync/atomic"

// SPSC — очередь фиксированной ёмкости для одного производителя и одного
// потребителя (single producer, single consumer). У каждого индекса ровно
// один писатель (tail — producer, head — consumer), поэтому конкуренции за
// запись нет вовсе и блокировка не нужна (см. REPORT.md, §9 теорсправки).
type SPSC struct {
	buf  []uint64
	mask uint64
	cap  uint64

	_    [CacheLineSize]byte
	tail atomic.Uint64 // пишет только producer; читает consumer (acquire)
	_    [CacheLineSize - 8]byte
	head atomic.Uint64 // пишет только consumer; читает producer (acquire)
	_    [CacheLineSize - 8]byte

	// Собственное состояние каждой стороны плюс локальная копия чужого
	// индекса: не перечитывать чужую строку кэша на каждой операции, а
	// заглядывать в неё только тогда, когда по локальной копии кажется,
	// что места (или данных) больше нет.
	producerTail      uint64
	producerHeadCache uint64

	consumerHead      uint64
	consumerTailCache uint64
}

// NewSPSC создаёт очередь ёмкостью capacity, округлённой вверх до степени
// двойки (взятие остатка потом заменяется маской — одна инструкция вместо
// деления, и маска остаётся верной даже после переполнения счётчика).
func NewSPSC(capacity uint64) *SPSC {
	cap := nextPow2(capacity)
	return &SPSC{
		buf:  make([]uint64, cap),
		mask: cap - 1,
		cap:  cap,
	}
}

// Capacity возвращает фактическую ёмкость после округления.
func (q *SPSC) Capacity() uint64 { return q.cap }

// TryPush не блокируется: если очередь полна, возвращает false.
// Единственный писатель — вызывающая горутина; звать из двух горутин
// одновременно нельзя (SPSC).
func (q *SPSC) TryPush(v uint64) bool {
	if q.producerTail-q.producerHeadCache >= q.cap {
		q.producerHeadCache = q.head.Load() // acquire: видит все pop'ы consumer'а
		if q.producerTail-q.producerHeadCache >= q.cap {
			return false // очередь полна
		}
	}
	q.buf[q.producerTail&q.mask] = v
	q.producerTail++
	q.tail.Store(q.producerTail) // release: публикует и данные, и индекс
	return true
}

// TryPop не блокируется: если очередь пуста, возвращает (0, false).
// Единственный читатель — вызывающая горутина.
func (q *SPSC) TryPop() (uint64, bool) {
	if q.consumerHead == q.consumerTailCache {
		q.consumerTailCache = q.tail.Load() // acquire: видит данные, опубликованные producer'ом
		if q.consumerHead == q.consumerTailCache {
			return 0, false // очередь пуста
		}
	}
	v := q.buf[q.consumerHead&q.mask]
	q.consumerHead++
	q.head.Store(q.consumerHead) // release: сообщает producer'у, что место освободилось
	return v, true
}
