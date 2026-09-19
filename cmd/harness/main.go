// Единая точка входа для автоматической проверки.
//
// Правила протокола, общие для всех работ курса:
//   - каждая команда печатает РОВНО ОДНУ строку JSON в stdout;
//   - диагностика идёт в stderr и на разбор не влияет;
//   - успех — код возврата 0, ошибка разбора аргументов — 2.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"pp/lab/memmodel"
)

func usage() {
	fmt.Fprint(os.Stderr, `Использование: harness <команда> [аргументы]

  info
  counter <T> <N> <relaxed|seqcst> [--cas]
  litmus <N> <relaxed|seqcst>
  spsc <capacity> <items>
  bench-counter <T> <N> <relaxed|seqcst> [--cas]
  bench-spsc <capacity> <items>
`)
}

func printJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func parseUint(s, name string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("аргумент %s: ожидалось целое число, получено %q", name, s)
	}
	return v, nil
}

func parseOrdering(name, s string) (memmodel.Ordering, error) {
	ordering, ok := memmodel.ParseOrdering(s)
	if !ok {
		return "", fmt.Errorf("%s: неизвестный порядок %q, ожидается relaxed или seqcst", name, s)
	}
	return ordering, nil
}

type infoOutput struct {
	Lab       int    `json:"lab"`
	Language  string `json:"language"`
	CacheLine int    `json:"cache_line"`
}

type counterOutput struct {
	Total    uint64 `json:"total"`
	Expected uint64 `json:"expected"`
	Ordering string `json:"ordering"`
	CAS      bool   `json:"cas"`
}

type litmusOutput struct {
	Iterations uint64 `json:"iterations"`
	BothZero   uint64 `json:"both_zero"`
	R1Only     uint64 `json:"r1_only"`
	R2Only     uint64 `json:"r2_only"`
	BothOne    uint64 `json:"both_one"`
}

type spscOutput struct {
	Popped   uint64  `json:"popped"`
	Sum      uint64  `json:"sum"`
	Ordered  bool    `json:"ordered"`
	Capacity uint64  `json:"capacity"`
	Seconds  float64 `json:"seconds"`
}

type benchCounterOutput struct {
	Ops       uint64  `json:"ops"`
	Seconds   float64 `json:"seconds"`
	OpsPerSec float64 `json:"ops_per_sec"`
}

type benchSPSCOutput struct {
	Items       uint64  `json:"items"`
	Seconds     float64 `json:"seconds"`
	ItemsPerSec float64 `json:"items_per_sec"`
}

func cmdCounter(rest []string) error {
	if len(rest) < 3 {
		usage()
		return fmt.Errorf("counter: нужно T N ordering")
	}
	threads, err := parseUint(rest[0], "T")
	if err != nil {
		return err
	}
	n, err := parseUint(rest[1], "N")
	if err != nil {
		return err
	}
	ordering, err := parseOrdering("counter", rest[2])
	if err != nil {
		return err
	}
	cas := len(rest) > 3 && rest[3] == "--cas"

	var total uint64
	if cas {
		total = memmodel.CounterCAS(threads, n, ordering)
	} else {
		total = memmodel.CounterFetchAdd(threads, n, ordering)
	}
	return printJSON(counterOutput{
		Total:    total,
		Expected: threads * n,
		Ordering: string(ordering),
		CAS:      cas,
	})
}

func cmdLitmus(rest []string) error {
	if len(rest) < 2 {
		usage()
		return fmt.Errorf("litmus: нужно N ordering")
	}
	n, err := parseUint(rest[0], "N")
	if err != nil {
		return err
	}
	ordering, err := parseOrdering("litmus", rest[1])
	if err != nil {
		return err
	}

	var counts memmodel.LitmusCounts
	if ordering == memmodel.SeqCst {
		counts = memmodel.LitmusSeqCst(n)
	} else {
		counts = memmodel.LitmusRelaxed(n)
	}
	return printJSON(litmusOutput{
		Iterations: n,
		BothZero:   counts.BothZero,
		R1Only:     counts.R1Only,
		R2Only:     counts.R2Only,
		BothOne:    counts.BothOne,
	})
}

// runSPSC гоняет producer в отдельной горутине, consumer — в текущей, пока
// не заберёт items элементов, и возвращает попавшуюся сумму и порядок.
func runSPSC(q *memmodel.SPSC, items uint64) (popped, sum uint64, ordered bool) {
	ordered = true
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := uint64(1); i <= items; i++ {
			for !q.TryPush(i) {
			}
		}
	}()

	expect := uint64(1)
	for popped < items {
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
	return
}

func cmdSPSC(rest []string) error {
	if len(rest) < 2 {
		usage()
		return fmt.Errorf("spsc: нужно capacity items")
	}
	capacity, err := parseUint(rest[0], "capacity")
	if err != nil {
		return err
	}
	items, err := parseUint(rest[1], "items")
	if err != nil {
		return err
	}

	q := memmodel.NewSPSC(capacity)
	start := time.Now()
	popped, sum, ordered := runSPSC(q, items)
	elapsed := time.Since(start).Seconds()

	return printJSON(spscOutput{
		Popped:   popped,
		Sum:      sum,
		Ordered:  ordered,
		Capacity: q.Capacity(),
		Seconds:  elapsed,
	})
}

func cmdBenchCounter(rest []string) error {
	if len(rest) < 3 {
		usage()
		return fmt.Errorf("bench-counter: нужно T N ordering")
	}
	threads, err := parseUint(rest[0], "T")
	if err != nil {
		return err
	}
	n, err := parseUint(rest[1], "N")
	if err != nil {
		return err
	}
	ordering, err := parseOrdering("bench-counter", rest[2])
	if err != nil {
		return err
	}
	cas := len(rest) > 3 && rest[3] == "--cas"

	start := time.Now()
	if cas {
		memmodel.CounterCAS(threads, n, ordering)
	} else {
		memmodel.CounterFetchAdd(threads, n, ordering)
	}
	elapsed := time.Since(start).Seconds()
	ops := threads * n
	var perSec float64
	if elapsed > 0 {
		perSec = float64(ops) / elapsed
	}
	return printJSON(benchCounterOutput{Ops: ops, Seconds: elapsed, OpsPerSec: perSec})
}

func cmdBenchSPSC(rest []string) error {
	if len(rest) < 2 {
		usage()
		return fmt.Errorf("bench-spsc: нужно capacity items")
	}
	capacity, err := parseUint(rest[0], "capacity")
	if err != nil {
		return err
	}
	items, err := parseUint(rest[1], "items")
	if err != nil {
		return err
	}

	q := memmodel.NewSPSC(capacity)
	start := time.Now()
	runSPSC(q, items)
	elapsed := time.Since(start).Seconds()
	var perSec float64
	if elapsed > 0 {
		perSec = float64(items) / elapsed
	}
	return printJSON(benchSPSCOutput{Items: items, Seconds: elapsed, ItemsPerSec: perSec})
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("не указана команда")
	}
	command, rest := args[0], args[1:]

	switch command {
	case "info":
		return printJSON(infoOutput{Lab: 1, Language: "Go", CacheLine: memmodel.CacheLineSize})
	case "counter":
		return cmdCounter(rest)
	case "litmus":
		return cmdLitmus(rest)
	case "spsc":
		return cmdSPSC(rest)
	case "bench-counter":
		return cmdBenchCounter(rest)
	case "bench-spsc":
		return cmdBenchSPSC(rest)
	default:
		usage()
		return fmt.Errorf("неизвестная команда: %s", command)
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
		os.Exit(2)
	}
}
