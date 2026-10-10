package game_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// §5.3: el primer paciente aparece en la calle entre los 18 y los 25 s.
func TestArrivals_FirstPatientAppearsOnTheStreetAfter18To25s(t *testing.T) {
	g := newGame(t)

	tickFor(g, 17900*time.Millisecond)
	if n := len(g.Snapshot().Patients); n != 0 {
		t.Fatalf("a los 17,9 s ya hay %d pacientes; el primero llega desde los 18 s", n)
	}

	tickFor(g, 7100*time.Millisecond) // 25 s
	snap := g.Snapshot()
	if len(snap.Patients) != 1 {
		t.Fatalf("a los 25 s hay %d pacientes; se esperaba 1", len(snap.Patients))
	}
	if p := snap.Patients[0]; p.ID != "P-001" || p.Name == "" || p.Stage != game.Arriving {
		t.Errorf("primer paciente = (%s, %q, %v); se esperaba P-001, con nombre, llegando por la calle", p.ID, p.Name, p.Stage)
	}
}

// §5.3: nunca hay más de 10 pacientes en el mapa; cuando está lleno, la
// llegada se salta. Los IDs son consecutivos.
func TestArrivals_NeverMoreThan10OnTheMap(t *testing.T) {
	g := newGame(t)
	for s := 0; s < 299; s++ { // casi todo el turno (en F1.3a nadie se va solo)
		tickFor(g, time.Second)
		if n := len(g.Snapshot().Patients); n > 10 {
			t.Fatalf("a los %d s hay %d pacientes en el mapa; el máximo es 10", s+1, n)
		}
	}
	snap := g.Snapshot()
	if len(snap.Patients) != 10 {
		t.Fatalf("al final del turno hay %d pacientes; con llegadas cada 18–25 s se esperaba el mapa lleno (10)", len(snap.Patients))
	}
	for i, p := range snap.Patients {
		if want := fmt.Sprintf("P-%03d", i+1); p.ID != want {
			t.Errorf("paciente %d tiene el ID %s; se esperaba %s", i, p.ID, want)
		}
	}
}

// La pausa también congela las llegadas.
func TestArrivals_PauseStopsThem(t *testing.T) {
	g := newGame(t)
	g.Pause()
	tickFor(g, 60*time.Second)
	if n := len(g.Snapshot().Patients); n != 0 {
		t.Errorf("en pausa llegaron %d pacientes", n)
	}
}
