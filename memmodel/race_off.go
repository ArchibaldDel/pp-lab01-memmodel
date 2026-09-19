//go:build !race

package memmodel

// RaceDetectorEnabled — true при сборке/тесте с -race (Go сам выставляет
// build-тег "race" в этом режиме). Используется только затем, чтобы
// пропустить тест на LitmusRelaxed: он содержит намеренную гонку данных
// (см. litmus.go), и раннер гонок совершенно правильно на неё пожалуется —
// это предмет изучения задания, а не дефект.
const RaceDetectorEnabled = false
