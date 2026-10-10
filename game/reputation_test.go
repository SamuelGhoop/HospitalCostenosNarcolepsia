package game_test

import (
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// reputationAt avanza hasta el instante at (desde el 0) y devuelve la
// reputación, en centésimas de estrella.
func reputationAt(g *game.Game, now *time.Duration, at time.Duration) int {
	tickFor(g, at-*now)
	*now = at
	return g.Snapshot().Reputation
}

// §5.9: la partida arranca con 3 ★ (300 centésimas). Esperando: −0,5 a los
// 20 s y −0,25 cada 10 s más; al irse enojado, −0,5 más.
func TestReputation_WaitingPenalties(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)

	var now time.Duration
	for _, c := range []struct {
		at   time.Duration
		want int
	}{
		{0, 300},
		{19900 * time.Millisecond, 300},
		{20 * time.Second, 250},
		{30 * time.Second, 225},
		{40 * time.Second, 200},
		{45 * time.Second, 150}, // se va enojado
	} {
		if got := reputationAt(g, &now, c.at); got != c.want {
			t.Errorf("a los %v: reputación = %d centésimas; se esperaban %d", c.at, got, c.want)
		}
	}
}

// La penalización sigue el cronómetro de espera: recogerlo NO la reinicia
// (esperando revisión también baja) y la revisión la detiene.
func TestReputation_PickUpDoesNotResetThePenaltyAndTheReviewStopsIt(t *testing.T) {
	g := newGame(t)
	reviewAt(t, g, hospital.Severe, 25*time.Second) // recogido a los 20 s, revisado a los 25 s
	if got := g.Snapshot().Reputation; got != 250 {
		t.Errorf("revisado a los 25 s: reputación = %d; se esperaba 250 (−50 a los 20 s)", got)
	}
	tickFor(g, 20*time.Second) // 45 s: revisado, ya no espera ni se va
	if got := g.Snapshot().Reputation; got != 250 {
		t.Errorf("a los 45 s, ya revisado: reputación = %d; se esperaba 250", got)
	}
}

// §5.9: +0,1 si lo recogen en menos de 10 s desde el desplome.
func TestReputation_QuickPickUpBonus(t *testing.T) {
	quick := newGame(t)
	collapse(t, quick, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, quick, "P-001", "C-01")
	tickFor(quick, 5*time.Second) // recogido a los 5 s
	if got := quick.Snapshot().Reputation; got != 310 {
		t.Errorf("recogido a los 5 s: reputación = %d; se esperaba 310", got)
	}

	slow := newGame(t)
	collapse(t, slow, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	tickFor(slow, 5*time.Second)
	dispatch(t, slow, "P-001", "C-01")
	tickFor(slow, 5*time.Second) // recogido justo a los 10 s: ya no es "menos de 10 s"
	if got := slow.Snapshot().Reputation; got != 300 {
		t.Errorf("recogido a los 10 s: reputación = %d; se esperaba 300 (sin bono)", got)
	}
}

// §5.9: +0,1 por alta.
func TestReputation_DischargeBonus(t *testing.T) {
	g := newGame(t)
	reviewAt(t, g, hospital.Mild, 10*time.Second) // recogido a los 5 s: +10
	tickFor(g, 11400*time.Millisecond)            // alta: +10
	if got := g.Snapshot().Reputation; got != 320 {
		t.Errorf("después del alta: reputación = %d; se esperaba 300 + 10 + 10", got)
	}
}
