package memmodel

import "testing"

func TestLitmusSeqCstNeverBothZero(t *testing.T) {
	const n = 20000
	c := LitmusSeqCst(n)
	if c.BothZero != 0 {
		t.Fatalf("seq_cst обязан давать both_zero=0, получено %d", c.BothZero)
	}
	sum := c.BothZero + c.R1Only + c.R2Only + c.BothOne
	if sum != n {
		t.Fatalf("сумма исходов %d, ожидалось %d раундов", sum, n)
	}
}

// TestLitmusRelaxedCanReorder — намеренная гонка данных, предмет изучения
// части B (см. litmus.go). Нельзя гонять под -race: детектор корректно
// найдёт эту гонку и завалит прогон, поэтому пропускаем именно этот тест.
func TestLitmusRelaxedCanReorder(t *testing.T) {
	if RaceDetectorEnabled {
		t.Skip("relaxed-вариант содержит намеренную гонку данных; пропуск под -race")
	}
	const n = 300000
	c := LitmusRelaxed(n)
	sum := c.BothZero + c.R1Only + c.R2Only + c.BothOne
	if sum != n {
		t.Fatalf("сумма исходов %d, ожидалось %d раундов", sum, n)
	}
	if c.BothZero == 0 {
		t.Skipf("both_zero=0 за %d раундов на этом железе/планировщике — эффект аппаратно- и "+
			"нагрузочно-зависим (см. REPORT.md), не считаем это провалом теста", n)
	}
}
