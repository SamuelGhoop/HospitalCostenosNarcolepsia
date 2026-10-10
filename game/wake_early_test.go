package game_test

import (
	"errors"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §5.7: DESPERTAR a un paciente revisado libera su cama al instante y cuesta
// −0,25 ★. No cuenta como sueño completo (un Mild no recibe el alta) y vuelve
// a dormirse en la mitad del tiempo normal.
func TestWakeEarly_FreesTheBedAndHalvesTheNextAwakeTime(t *testing.T) {
	g := newGame(t)
	reviewAt(t, g, hospital.Mild, 10*time.Second) // recogido a los 5 s: 310 centésimas
	bed := patientByID(t, g.Snapshot(), "P-001").Room

	if err := g.WakeEarly("P-001"); err != nil {
		t.Fatalf("WakeEarly: %v", err)
	}
	snap := g.Snapshot()
	if p := patientByID(t, snap, "P-001"); p.Stage != game.Wandering || p.State != hospital.Awake || p.Room != 0 {
		t.Errorf("P-001 = (%v, %v, hab. %d); se esperaba (deambulando, despierto, sin cama): no recibe el alta", p.Stage, p.State, p.Room)
	}
	for _, r := range snap.Rooms {
		if r.Number == bed && r.State != hospital.Available {
			t.Errorf("la %d quedó %v; WakePatient debía liberarla al instante", r.Number, r.State)
		}
	}
	if snap.Reputation != 285 {
		t.Errorf("reputación = %d; se esperaba 310 − 25", snap.Reputation)
	}

	// Un Mild aguanta despierto de 60 a 90 s; después de DESPERTAR, la mitad.
	for elapsed := tick; elapsed <= 50*time.Second; elapsed += tick {
		g.Tick(tick)
		if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage == game.Drowsy {
			if elapsed < 30*time.Second || elapsed > 45*time.Second {
				t.Errorf("volvió a cabecear a los %v; debía ser entre 30 y 45 s (la mitad de 60–90 s)", elapsed)
			}
			return
		}
	}
	t.Error("en 50 s no volvió a cabecear; con la mitad del tiempo despierto debía hacerlo antes de los 45 s")
}

// Solo se despierta antes de tiempo a un paciente revisado que duerme en
// cama: el sueño corre desde la revisión.
func TestWakeEarly_OnlyForReviewedPatientsInBed(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Severe, game.Radiology)
	dispatch(t, g, "P-002", "C-01")
	tickFor(g, 5*time.Second) // P-002 recogido: esperando revisión

	for _, id := range []string{"P-001", "P-002"} {
		if err := g.WakeEarly(id); !errors.Is(err, game.ErrNotInBed) {
			t.Errorf("WakeEarly(%s) = %v; se esperaba ErrNotInBed", id, err)
		}
	}
	if err := g.WakeEarly("P-999"); !errors.Is(err, game.ErrUnknownPatient) {
		t.Errorf("WakeEarly(P-999) = %v; se esperaba ErrUnknownPatient", err)
	}
	if got := g.Snapshot().Reputation; got != 310 {
		t.Errorf("los WakeEarly rechazados cambiaron la reputación: %d; se esperaba 310", got)
	}
}

// waitForStage avanza hasta que el paciente llega a stage (máximo limit).
func waitForStage(t *testing.T, g *game.Game, id string, stage game.Stage, limit time.Duration) {
	t.Helper()
	for elapsed := time.Duration(0); elapsed < limit; elapsed += tick {
		if patientByID(t, g.Snapshot(), id).Stage == stage {
			return
		}
		g.Tick(tick)
	}
	t.Fatalf("en %v %s nunca llegó a %v", limit, id, stage)
}

// Despertado antes de tiempo, el sueño NO cuenta para el alta: un Moderate
// necesita 2 sueños completos, así que después de DESPERTAR y un sueño
// completo vuelve a deambular en vez de irse de alta.
func TestWakeEarly_DoesNotCountAsACompletedSleep(t *testing.T) {
	g := newQuietGame(t)
	reviewAt(t, g, hospital.Moderate, 10*time.Second)
	if err := g.WakeEarly("P-001"); err != nil {
		t.Fatalf("WakeEarly: %v", err)
	}

	waitForStage(t, g, "P-001", game.Collapsed, 35*time.Second) // vuelve a dormirse (la mitad de 35–55 s, + 2 s)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second)
	dispatch(t, g, "P-001", "D-01")
	tickFor(g, 5*time.Second)                                   // revisado otra vez
	waitForStage(t, g, "P-001", game.Wandering, 20*time.Second) // y un sueño completo

	// Si DESPERTAR hubiera contado, ya serían 2 sueños y estaría saliendo de alta.
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.Wandering {
		t.Errorf("después de 1 sueño completo, P-001 (Moderate) está %v; debía seguir deambulando", p.Stage)
	}
}
