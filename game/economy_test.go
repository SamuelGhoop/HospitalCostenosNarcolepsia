package game_test

import (
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// reviewAt deja a P-001 (desplomado en el instante 0) revisado por D-01
// exactamente a los at segundos: C-01 lo recoge a los at − 5 s y D-01,
// despachado en ese momento, camina 5 s.
func reviewAt(t *testing.T, g *game.Game, level hospital.NarcolepsyLevel, at time.Duration) {
	t.Helper()
	collapse(t, g, "P-001", "Yeimy Padilla", level, game.Cafeteria)
	tickFor(g, at-10*time.Second)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second) // lo recoge
	dispatch(t, g, "P-001", "D-01")
	tickFor(g, 5*time.Second) // lo revisa
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.InBed || p.Waited != at {
		t.Fatalf("preparando la revisión: P-001 = (%v, Waited %v); se esperaba (en cama, %v)", p.Stage, p.Waited, at)
	}
}

// §5.2 y §5.8: la partida arranca con $2.000 y cada episodio paga una vez,
// en la revisión, según cuánto esperó desde el desplome.
func TestMoney_TheReviewPaysAccordingToTheWait(t *testing.T) {
	if got := newGame(t).Snapshot().Money; got != 2000 {
		t.Fatalf("plata inicial = $%d; se esperaban $2.000", got)
	}

	fast := newGame(t)
	reviewAt(t, fast, hospital.Severe, 10*time.Second)
	if got := fast.Snapshot().Money; got != 2450 {
		t.Errorf("revisado a los 10 s: plata = $%d; se esperaban $2.000 + $450", got)
	}

	late := newGame(t)
	reviewAt(t, late, hospital.Severe, 25*time.Second)
	if got := late.Snapshot().Money; got != 2300 {
		t.Errorf("revisado a los 25 s: plata = $%d; se esperaban $2.000 + $300", got)
	}
}

// §5.8: el alta paga $200; el que se va enojado no paga nada.
func TestMoney_DischargePaysAndTheAngryDoesNot(t *testing.T) {
	g := newGame(t)
	reviewAt(t, g, hospital.Mild, 10*time.Second) // +$450
	tickFor(g, 11400*time.Millisecond)            // duerme 11,4 s y recibe el alta
	if got := g.Snapshot().Money; got != 2650 {
		t.Errorf("después del alta: plata = $%d; se esperaban $2.000 + $450 + $200", got)
	}

	angry := newGame(t)
	collapse(t, angry, "P-001", "Kevin Mercado", hospital.Severe, game.Radiology)
	tickFor(angry, 45*time.Second)
	if got := angry.Snapshot().Money; got != 2000 {
		t.Errorf("se fue enojado: plata = $%d; no debía cambiar", got)
	}
}
