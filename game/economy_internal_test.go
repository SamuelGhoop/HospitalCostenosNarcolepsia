package game

import (
	"math/rand"
	"testing"
	"time"
)

// §5.8: $450 hasta 15 s; después −$15 por cada segundo completo; mínimo $100.
func TestReviewFee_DependsOnTheWait(t *testing.T) {
	cases := []struct {
		waited time.Duration
		want   int
	}{
		{10 * time.Second, 450},
		{15 * time.Second, 450},
		{15900 * time.Millisecond, 450}, // todavía no se completa el segundo 16
		{16 * time.Second, 435},
		{25 * time.Second, 300},
		{38 * time.Second, 105},
		{39 * time.Second, 100},
		{60 * time.Second, 100},
	}
	for _, c := range cases {
		if got := reviewFee(c.waited); got != c.want {
			t.Errorf("reviewFee(%v) = $%d; se esperaban $%d", c.waited, got, c.want)
		}
	}
}

// §5.8: la fórmula de la nómina (y del costo de contratar, que es un día de
// salario adelantado): médico 250 + 50 × (rapidez + pericia); camillero
// 120 + 60 × rapidez.
func TestSalary_Formulas(t *testing.T) {
	doctorCases := []struct {
		speed, skill int
		want         int
	}{
		{1, 1, 350}, // 250 + 50×2
		{3, 3, 550}, // el médico inicial (§5.2): 250 + 50×6
		{5, 5, 750}, // 250 + 50×10
		{1, 5, 550}, // 250 + 50×6, igual que (3,3): la fórmula suma los dos
	}
	for _, c := range doctorCases {
		if got := doctorSalary(c.speed, c.skill); got != c.want {
			t.Errorf("doctorSalary(%d, %d) = $%d; se esperaban $%d", c.speed, c.skill, got, c.want)
		}
	}

	orderlyCases := []struct {
		speed int
		want  int
	}{
		{1, 180}, // 120 + 60×1
		{3, 300}, // el camillero inicial (§5.2): 120 + 60×3
		{5, 420}, // 120 + 60×5
	}
	for _, c := range orderlyCases {
		if got := orderlySalary(c.speed); got != c.want {
			t.Errorf("orderlySalary(%d) = $%d; se esperaban $%d", c.speed, got, c.want)
		}
	}
}

// §6: weightedStat solo devuelve valores de 1 a maxStat, y con muchas
// muestras salen más 2 y 3 que 1 o 5 (statWeights les da más peso).
func TestWeightedStat_RangeAndDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	counts := map[int]int{}
	const samples = 20000
	for i := 0; i < samples; i++ {
		v := weightedStat(rng)
		if v < 1 || v > maxStat {
			t.Fatalf("weightedStat() = %d; fuera de 1..%d", v, maxStat)
		}
		counts[v]++
	}
	if counts[2] <= counts[1] || counts[3] <= counts[1] {
		t.Errorf("con %d muestras, 2 y 3 deben salir más que 1: %v", samples, counts)
	}
	if counts[2] <= counts[5] || counts[3] <= counts[5] {
		t.Errorf("con %d muestras, 2 y 3 deben salir más que 5: %v", samples, counts)
	}
}
