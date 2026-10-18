package game_test

import (
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// reviewPatientAt es como reviewAt (economy_test.go), pero sirve para
// cualquier ID, no solo P-001: encadena varias veces para armar una racha
// (§5.8, §12.8). C-01 lo recoge a los at−5 s y D-01 lo revisa 5 s después,
// así que queda esperando exactamente at.
func reviewPatientAt(t *testing.T, g *game.Game, id, name string, level hospital.NarcolepsyLevel, zone game.Zone, at time.Duration) {
	t.Helper()
	collapse(t, g, id, name, level, zone)
	tickFor(g, at-10*time.Second)
	dispatch(t, g, id, "C-01")
	tickFor(g, 5*time.Second)
	dispatch(t, g, id, "D-01")
	tickFor(g, 5*time.Second)
}

// §5.8 y §12.8: cada revisión rápida (menos de firstWaitPenaltyAt = 30 s de
// espera) sube la racha, y el pago de la SIGUIENTE revisión se multiplica
// según la racha que ya traía. newQuietGame: sin llegadas por la calle, para
// que solo paguen los cinco episodios de este test.
//
// Usa Severe (3 sueños para el alta) a propósito: así ninguno recibe el
// alta durante la ronda de 50 s del test, y el pago de cada revisión no se
// mezcla con el +$200 de un alta que caiga a mitad de la ronda siguiente.
func TestStreak_FastReviewsIncreaseItAndMultiplyThePay(t *testing.T) {
	g := newQuietGame(t)
	ids := []string{"P-001", "P-002", "P-003", "P-004", "P-005"}
	// fee base (reviewFee a los 10 s) = $450; el multiplicador usa la racha
	// ANTERIOR a cada revisión: 100, 125, 150, 175, 200 %.
	wantFees := []int{450, 563, 675, 788, 900}
	wantStreakAfter := []int{1, 2, 3, 4, 5}
	wantMultiplierAfter := []int{125, 150, 175, 200, 200} // tope a los 4 escalones

	money := g.Snapshot().Money
	for i, id := range ids {
		reviewPatientAt(t, g, id, "Paciente "+id, hospital.Severe, game.Cafeteria, 10*time.Second)
		snap := g.Snapshot()
		paid := snap.Money - money
		if paid != wantFees[i] {
			t.Errorf("revisión %d (racha anterior %d): pagó $%d; se esperaban $%d", i+1, i, paid, wantFees[i])
		}
		money = snap.Money
		if snap.Streak != wantStreakAfter[i] {
			t.Errorf("revisión %d: racha = %d; se esperaba %d", i+1, snap.Streak, wantStreakAfter[i])
		}
		if snap.StreakMultiplier != wantMultiplierAfter[i] {
			t.Errorf("revisión %d: multiplicador = %d; se esperaba %d", i+1, snap.StreakMultiplier, wantMultiplierAfter[i])
		}
	}
}

// §5.8: el DayReport guarda la racha más alta del día (BestStreak), aunque
// algo la haya roto antes de que cierre el turno. Usa Mild (1 sueño para el
// alta) y deja un hueco entre revisiones para que cada alta salga del mapa
// antes de la siguiente: así el día cierra sin pacientes sin atender.
func TestDayReport_BestStreakSurvivesAStreakReset(t *testing.T) {
	g := newQuietGame(t)
	reviewPatientAt(t, g, "P-001", "Yeimy Padilla", hospital.Mild, game.Cafeteria, 10*time.Second)
	tickFor(g, 20*time.Second) // de sobra para que P-001 duerma (11,4 s), reciba el alta y salga (4 s)

	reviewPatientAt(t, g, "P-002", "Kevin Mercado", hospital.Mild, game.Radiology, 10*time.Second)
	if got := g.Snapshot().Streak; got != 2 {
		t.Fatalf("preparando: racha = %d; se esperaba 2", got)
	}
	tickFor(g, 20*time.Second) // P-002 también sale de alta

	// P-003 se desploma y nadie lo revisa hasta que cae la penalización de
	// los 30 s: rompe la racha (que llegó a 2) antes de cerrar el día.
	collapse(t, g, "P-003", "Breiner Julio", hospital.Mild, game.Lobby)
	tickFor(g, 30*time.Second)
	if got := g.Snapshot().Streak; got != 0 {
		t.Fatalf("P-003 lleva 30 s esperando: racha = %d; se esperaba 0", got)
	}
	dispatch(t, g, "P-003", "C-01")
	tickFor(g, 5*time.Second)
	dispatch(t, g, "P-003", "D-01")
	tickFor(g, 5*time.Second) // revisado: se va a dormir y, con el tiempo, a recibir el alta

	tickFor(g, 300*time.Second) // de sobra para que el reloj llegue a las 20:00
	if got := g.DayReport().BestStreak; got != 2 {
		t.Errorf("BestStreak del día = %d; se esperaba 2 (la racha más alta, aunque terminó en 0)", got)
	}
}

// §5.8: una penalización de espera (30 s sin revisión) rompe la racha; la
// siguiente revisión rápida vuelve a pagar al ×1 (sin bono).
func TestStreak_ResetsWhenAWaitPenaltyFalls(t *testing.T) {
	g := newQuietGame(t)
	reviewPatientAt(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria, 10*time.Second)
	if got := g.Snapshot().Streak; got != 1 {
		t.Fatalf("preparando: racha = %d; se esperaba 1", got)
	}

	// P-002 se desploma y nadie lo revisa hasta que cae la penalización de
	// los 30 s: eso rompe la racha.
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Severe, game.Radiology)
	tickFor(g, 30*time.Second)
	if got := g.Snapshot().Streak; got != 0 {
		t.Fatalf("a los 30 s de espera: racha = %d; se esperaba 0 (la penalización la rompe)", got)
	}

	money := g.Snapshot().Money
	dispatch(t, g, "P-002", "C-01")
	tickFor(g, 5*time.Second) // recogido a los 35 s
	dispatch(t, g, "P-002", "D-01")
	tickFor(g, 5*time.Second) // revisado a los 40 s: tarde, pero el punto es el multiplicador

	// reviewFee a los 40 s: $450 − $15 × 25 s de más = $75, con el piso de
	// $100 (§5.8). Racha en 0: sin multiplicador.
	paid := g.Snapshot().Money - money
	if paid != 100 {
		t.Fatalf("pagó $%d; se esperaban $100 (la racha ya estaba en 0, sin multiplicador)", paid)
	}
}

// §5.8: alguien que se va enojado también rompe la racha.
func TestStreak_ResetsWhenSomeoneLeavesAngry(t *testing.T) {
	g := newQuietGame(t)
	reviewPatientAt(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria, 10*time.Second)
	if got := g.Snapshot().Streak; got != 1 {
		t.Fatalf("preparando: racha = %d; se esperaba 1", got)
	}

	collapse(t, g, "P-002", "Kevin Mercado", hospital.Severe, game.Radiology)
	tickFor(g, 45*time.Second) // nadie lo revisa: se va enojado
	if got := g.Snapshot().Streak; got != 0 {
		t.Errorf("P-002 se fue enojado: racha = %d; se esperaba 0", got)
	}
}

// §5.8: la racha también se reinicia al terminar el día.
func TestStreak_ResetsWhenTheDayEnds(t *testing.T) {
	g := newQuietGame(t)
	reviewPatientAt(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria, 10*time.Second)
	if got := g.Snapshot().Streak; got != 1 {
		t.Fatalf("preparando: racha = %d; se esperaba 1", got)
	}

	tickFor(g, 300*time.Second) // pasa las 20:00
	if got := g.Snapshot().Streak; got != 0 {
		t.Errorf("al terminar el día: racha = %d; se esperaba 0", got)
	}
}
