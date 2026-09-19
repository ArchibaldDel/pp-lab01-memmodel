# Лабораторная работа №1. Модель памяти и атомарные операции

Решение на Go. Три части, каждая — отдельный файл в пакете `memmodel`:

| Часть | Файл | Что внутри |
|---|---|---|
| A. Атомарный счётчик | `memmodel/counter.go` | `fetch_add` (`atomic.Uint64.Add`) и ручной `compare_exchange` (CAS-цикл) |
| B. Litmus-тест «буфер записи» | `memmodel/litmus.go` | `seq_cst` (реальные атомики) и `relaxed` (намеренная гонка данных на обычных переменных) |
| C. Кольцевой буфер SPSC | `memmodel/spsc.go` | без блокировок, ёмкость — степень двойки, `head`/`tail` в разных строках кэша |

Ни одной блокировки (`sync.Mutex`, `sync.RWMutex`, `sync.Map`) и ни одного
канала — только `sync/atomic`, как требует задание. Обоснования решений —
в [REPORT.md](REPORT.md).

## Сборка и запуск

```bash
docker build -t pp-lab-go .
docker run --rm pp-lab-go info
```

Локально:

```bash
go build -o harness ./cmd/harness
./harness info
```

## Протокол harness

Первый аргумент — команда, на стандартный вывод печатается ровно одна
строка JSON, диагностика — в stderr. Успех — код возврата 0, неизвестная
команда или неверные аргументы — 2.

| Команда | Пример | Вывод |
|---|---|---|
| `info` | `harness info` | `{"lab":1,"language":"Go","cache_line":64}` |
| `counter <T> <N> <relaxed\|seqcst> [--cas]` | `harness counter 8 50000 seqcst` | `{"total":…,"expected":…,"ordering":"…","cas":…}` |
| `litmus <N> <relaxed\|seqcst>` | `harness litmus 20000 seqcst` | `{"iterations":…,"both_zero":…,"r1_only":…,"r2_only":…,"both_one":…}` |
| `spsc <capacity> <items>` | `harness spsc 5 1000` | `{"popped":…,"sum":…,"ordered":…,"capacity":…,"seconds":…}` |
| `bench-counter <T> <N> <relaxed\|seqcst> [--cas]` | `harness bench-counter 8 200000 seqcst` | `{"ops":…,"seconds":…,"ops_per_sec":…}` |
| `bench-spsc <capacity> <items>` | `harness bench-spsc 1024 2000000` | `{"items":…,"seconds":…,"items_per_sec":…}` |

`ordering` в Go не меняет кодогенерацию (см. REPORT.md) — параметр принят
и эхуется в JSON только ради общего протокола курса.

## Тесты

```bash
go test ./...          # функциональные тесты
go test ./... -race    # детектор гонок; relaxed-вариант litmus пропускается намеренно
```

## Самопроверка

```bash
python3 selfcheck.py .
```
