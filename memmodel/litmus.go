package memmodel

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LitmusCounts — распределение исходов litmus-теста «буфер записи» (store
// buffering) за N независимых раундов:
//
//	начало: x = 0, y = 0
//	поток 1:  x = 1;  r1 = y
//	поток 2:  y = 1;  r2 = x
type LitmusCounts struct {
	BothZero uint64 // r1==0 && r2==0 — обе записи ещё лежали в буферах
	R1Only   uint64 // r1==0 && r2==1
	R2Only   uint64 // r1==1 && r2==0
	BothOne  uint64 // r1==1 && r2==1
}

func (c *LitmusCounts) record(r1, r2 uint32) {
	switch {
	case r1 == 0 && r2 == 0:
		c.BothZero++
	case r1 == 0 && r2 == 1:
		c.R1Only++
	case r1 == 1 && r2 == 0:
		c.R2Only++
	default:
		c.BothOne++
	}
}

// paddedFlag — булев флаг хендшейка в собственной строке кэша. Общий
// барьер для старта раунда не годится: потоки выходят из него с разбросом
// в сотни наносекунд, и один всегда успевает целиком раньше другого — тогда
// критические участки просто не пересекутся. Поэтому у каждого потока свой
// флаг, и flip чередует true/false от раунда к раунду.
type paddedFlag struct {
	v atomic.Bool
	_ [CacheLineSize - 1]byte
}

func (f *paddedFlag) waitFor(want bool) {
	for f.v.Load() != want {
		// Раунд занимает десятки наносекунд — планировщик здесь дороже
		// самого ожидания, поэтому спин, а не что-то блокирующее.
	}
}

var spinSink atomic.Uint64 // наблюдаемый побочный эффект, не даёт свернуть спин в ничто

// spin — случайная задержка перед критическим участком, геометрически
// распределённая (в среднем ~9 холостых итераций). Без неё сдвиг между
// потоками остаётся одним и тем же от раунда к раунду, и оба нуля никогда
// не совпадут (см. REPORT.md и теоретическую справку §6, §11, §16).
func spin(rng *rand.Rand) {
	acc := uint64(0)
	for rng.Intn(10) != 0 {
		acc++
	}
	spinSink.Add(acc)
}

// runLitmusRounds — общий каркас синхронизации N раундов. threadA/threadB —
// тело каждого потока для одного раунда (сама гонка — внутри них, каркас
// сам по себе всегда синхронизирован атомарными флагами). reset обнуляет
// x и y перед раундом.
func runLitmusRounds(iterations uint64, threadA, threadB func(*rand.Rand) uint32, reset func()) LitmusCounts {
	var goA, doneA, goB, doneB paddedFlag
	var r1, r2 uint32
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rng := rand.New(rand.NewSource(time.Now().UnixNano() ^ 0x51a17e51))
		for round := uint64(0); round < iterations; round++ {
			expect := round%2 == 0
			goA.waitFor(expect)
			r1 = threadA(rng)
			doneA.v.Store(expect)
		}
	}()
	go func() {
		defer wg.Done()
		rng := rand.New(rand.NewSource(time.Now().UnixNano() ^ 0x0ff1ce))
		for round := uint64(0); round < iterations; round++ {
			expect := round%2 == 0
			goB.waitFor(expect)
			r2 = threadB(rng)
			doneB.v.Store(expect)
		}
	}()

	var counts LitmusCounts
	for round := uint64(0); round < iterations; round++ {
		expect := round%2 == 0
		reset()
		goA.v.Store(expect)
		goB.v.Store(expect)
		doneA.waitFor(expect)
		doneB.waitFor(expect)
		counts.record(r1, r2)
	}
	wg.Wait()
	return counts
}

// LitmusSeqCst прогоняет тест на настоящих атомарных операциях. Go не
// предлагает более слабого порядка, чем seq_cst (см. Ordering), поэтому
// both_zero обязан быть нулём при любом N — это и проверяется тестами.
func LitmusSeqCst(iterations uint64) LitmusCounts {
	var x, y atomic.Uint32
	return runLitmusRounds(iterations,
		func(rng *rand.Rand) uint32 {
			spin(rng)
			x.Store(1)
			return y.Load()
		},
		func(rng *rand.Rand) uint32 {
			spin(rng)
			y.Store(1)
			return x.Load()
		},
		func() { x.Store(0); y.Store(0) },
	)
}

// LitmusRelaxed — та же схема, но x и y здесь обычные переменные без
// синхронизации. Это НАМЕРЕННАЯ гонка данных — она и есть предмет
// изучения части B задания, а не ошибка. Раунд-каркас (флаги goA/doneA и
// т.д.) остаётся полностью синхронизированным, гонка изолирована ровно в
// доступах к x и y. Нельзя гонять под -race: детектор корректно найдёт эту
// гонку и завалит прогон, поэтому соответствующий тест пропускается при
// RaceDetectorEnabled (см. race_on.go / race_off.go).
func LitmusRelaxed(iterations uint64) LitmusCounts {
	var x, y uint32
	return runLitmusRounds(iterations,
		func(rng *rand.Rand) uint32 {
			spin(rng)
			x = 1
			return y
		},
		func(rng *rand.Rand) uint32 {
			spin(rng)
			y = 1
			return x
		},
		func() { x, y = 0, 0 },
	)
}
