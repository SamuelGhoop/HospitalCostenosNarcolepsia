package game_test

import (
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §5.10: +100 por revisión (la atención completa), +250 por alta y +500
// cuando el reloj llega a las 20:00. Sin llegadas por la calle, para que
// nadie más espere y la partida llegue entera al final del turno.
func TestScore_ReviewDischargeAndCompletedDay(t *testing.T) {
	g := newQuietGame(t)
	reviewAt(t, g, hospital.Mild, 10*time.Second)
	if got := g.Snapshot().Score; got != 100 {
		t.Errorf("después de la revisión: puntaje %d; se esperaba 100", got)
	}

	tickFor(g, 11400*time.Millisecond) // alta
	if got := g.Snapshot().Score; got != 350 {
		t.Errorf("después del alta: puntaje %d; se esperaba 100 + 250", got)
	}

	tickFor(g, 300*time.Second) // pasa las 20:00
	snap := g.Snapshot()
	if !snap.DayOver || snap.Score != 850 {
		t.Errorf("al terminar el día: (terminado=%v, puntaje %d); se esperaba (true, 350 + 500)", snap.DayOver, snap.Score)
	}
	tickFor(g, 10*time.Second)
	if got := g.Snapshot().Score; got != 850 {
		t.Errorf("después de las 20:00 el puntaje siguió subiendo: %d", got)
	}
}

// §5.10: al perder, el puntaje final suma la plata ÷ 10 y las estrellas × 200.
func TestScore_FinalScoreWhenTheGameIsLost(t *testing.T) {
	g := newGame(t)
	reviewAt(t, g, hospital.Severe, 10*time.Second) // puntaje 100, $2.450, 3,10 ★
	if got := g.Snapshot().FinalScore; got != 0 {
		t.Errorf("con la partida en curso FinalScore = %d; se esperaba 0", got)
	}

	for _, id := range []string{"P-002", "P-003", "P-004", "P-005", "P-006"} {
		collapse(t, g, id, "Paciente "+id, hospital.Mild, game.Cafeteria)
	}
	g.Tick(tick) // 5 desplomados: colapsa
	snap := g.Snapshot()
	if snap.GameOver != game.HospitalCollapsed {
		t.Fatalf("GameOver = %v; se esperaba el colapso", snap.GameOver)
	}
	// 100 + 2.450 ÷ 10 + 3,10 × 200 = 100 + 245 + 620
	if snap.FinalScore != 965 {
		t.Errorf("puntaje final = %d; se esperaba 100 + 245 + 620 = 965", snap.FinalScore)
	}
}
