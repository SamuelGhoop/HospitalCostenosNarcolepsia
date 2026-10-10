package game_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
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

// §5.3: si el mapa está lleno (10) cuando toca una llegada, esa llegada se
// salta y se sortea otro intervalo.
func TestArrivals_SkippedWhileTheMapIsFull(t *testing.T) {
	g := newGame(t)
	for i := 0; i < 10; i++ { // Mild: en 30 s nadie se desploma ni se va
		g.ArriveForTest("Paciente", hospital.Mild)
	}

	tickFor(g, 30*time.Second) // el primer intervalo (18–25 s) se cumple con el mapa lleno
	snap := g.Snapshot()
	if len(snap.Patients) != 10 {
		t.Fatalf("con el mapa lleno hay %d pacientes; el máximo es 10", len(snap.Patients))
	}
	for i, p := range snap.Patients {
		if want := fmt.Sprintf("P-%03d", i+1); p.ID != want {
			t.Errorf("paciente %d tiene el ID %s; se esperaba %s (con el mapa lleno no llega nadie más)", i, p.ID, want)
		}
	}
}

// En un turno completo nunca hay más de 10 pacientes en el mapa, y los IDs
// van en orden de llegada.
func TestArrivals_NeverMoreThan10DuringTheShift(t *testing.T) {
	g := newGame(t)
	for s := 0; s < 299; s++ {
		tickFor(g, time.Second)
		snap := g.Snapshot()
		if len(snap.Patients) > 10 {
			t.Fatalf("a los %d s hay %d pacientes en el mapa; el máximo es 10", s+1, len(snap.Patients))
		}
		for i := 1; i < len(snap.Patients); i++ {
			if snap.Patients[i-1].ID >= snap.Patients[i].ID {
				t.Fatalf("a los %d s los IDs no van en orden: %s antes de %s", s+1, snap.Patients[i-1].ID, snap.Patients[i].ID)
			}
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
