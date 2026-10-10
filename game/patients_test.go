package game_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// arrive pone a un paciente nuevo en la calle, como una llegada real pero
// sin esperar el intervalo (ver export_test.go).
func arrive(t *testing.T, g *game.Game, wantID, name string, level hospital.NarcolepsyLevel) {
	t.Helper()
	id := g.ArriveForTest(name, level)
	if id != wantID {
		t.Fatalf("ArriveForTest le puso %s a %s; el test esperaba %s", id, name, wantID)
	}
}

// §5.3: camina 4 s por la acera y al cruzar la puerta queda en la recepción.
// La puerta se abre sola el último segundo antes de que la cruce.
func TestArrival_EntersTheLobbyWhenCrossingTheDoor(t *testing.T) {
	g := newGame(t)
	arrive(t, g, "P-001", "Yeimy Padilla", hospital.Mild)

	tickFor(g, 2900*time.Millisecond)
	snap := g.Snapshot()
	if p := patientByID(t, snap, "P-001"); p.Stage != game.Arriving {
		t.Fatalf("a los 2,9 s P-001 está %v; todavía va por la acera", p.Stage)
	}
	if snap.DoorOpen {
		t.Error("a 1,1 s de la puerta ya está abierta; se abre en el último segundo")
	}

	g.Tick(tick) // 3 s: le falta 1 s
	if !g.Snapshot().DoorOpen {
		t.Error("a 1 s de la puerta debería estar abierta")
	}

	tickFor(g, time.Second) // 4 s: cruza
	snap = g.Snapshot()
	if p := patientByID(t, snap, "P-001"); p.Stage != game.Wandering || p.Zone != game.Lobby {
		t.Errorf("después de cruzar, P-001 = (%v, %v); se esperaba (deambulando, recepción)", p.Stage, p.Zone)
	}
	if snap.DoorOpen {
		t.Error("ya cruzó: la puerta debería cerrarse")
	}
}

// Ajuste de Samuel: el tiempo despierto arranca al cruzar la puerta, nunca
// en la acera. Un Severe aguanta despierto de 15 a 30 s (§5.4), así que el
// cabeceo llega entre los 4 + 15 y los 4 + 30 s desde que aparece en la calle,
// y mientras va por la calle nunca se desploma.
//
// Se prueban 20 pacientes (2 partidas de 10) para que alguno saque un tiempo
// despierto corto: si el reloj corriera en la acera, ese cabecearía antes de
// los 19 s.
func TestAttack_NeverOnTheStreetAndOnlyAfterTheAwakeTime(t *testing.T) {
	for seed := int64(1); seed <= 2; seed++ {
		g, err := game.New(rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		var ids []string // los 10 aparecen en la calle en el instante 0
		for i := 0; i < 10; i++ {
			ids = append(ids, g.ArriveForTest("Paciente", hospital.Severe))
		}

		drowsyAt := map[string]time.Duration{}
		for elapsed := tick; elapsed <= 35*time.Second; elapsed += tick {
			g.Tick(tick)
			for _, id := range ids {
				p := patientByID(t, g.Snapshot(), id)
				if elapsed <= 4*time.Second && p.Stage != game.Arriving && p.Stage != game.Wandering {
					t.Fatalf("semilla %d: %s está %v a los %v, todavía en la calle o recién entrando", seed, id, p.Stage, elapsed)
				}
				if _, seen := drowsyAt[id]; !seen && p.Stage == game.Drowsy {
					drowsyAt[id] = elapsed
				}
			}
		}
		for _, id := range ids {
			at, ok := drowsyAt[id]
			if !ok || at < 19*time.Second || at > 34*time.Second {
				t.Errorf("semilla %d: %s cabeceó a los %v (¿llegó a cabecear? %v); debía ser entre 19 y 34 s", seed, id, at, ok)
			}
		}
	}
}

// §5.4: el cabeceo dura 2 s y después se desploma. En el modelo sigue
// despierto: el hospital no se entera hasta que alguien lo recoge.
func TestAttack_DrowsyFor2sThenCollapsed(t *testing.T) {
	g := newGame(t)
	arrive(t, g, "P-001", "Yeimy Padilla", hospital.Severe)

	var drowsySince time.Duration
	for elapsed := tick; elapsed <= 40*time.Second; elapsed += tick {
		g.Tick(tick)
		p := patientByID(t, g.Snapshot(), "P-001")
		switch {
		case p.Stage == game.Drowsy && drowsySince == 0:
			drowsySince = elapsed
		case p.Stage == game.Collapsed:
			if drowsySince == 0 || elapsed-drowsySince != 2*time.Second {
				t.Fatalf("se desplomó a los %v; empezó a cabecear a los %v (el cabeceo dura 2 s)", elapsed, drowsySince)
			}
			if p.State != hospital.Awake {
				t.Errorf("desplomado sin que nadie lo recoja, el modelo lo tiene %v; debía seguir despierto", p.State)
			}
			return
		}
	}
	t.Fatal("en 40 s P-001 (Severe) nunca se desplomó")
}

// §5.4: despierto, cambia de zona cada 10 a 20 s, siempre a una distinta.
func TestWandering_ChangesZoneEvery10To20s(t *testing.T) {
	g := newGame(t)
	arrive(t, g, "P-001", "Yeimy Padilla", hospital.Mild) // aguanta despierto 60 s o más
	tickFor(g, 4*time.Second)                             // cruza la puerta: queda en la recepción

	zone, since := game.Lobby, time.Duration(0)
	changes := 0
	for elapsed := tick; elapsed <= 55*time.Second; elapsed += tick {
		g.Tick(tick)
		p := patientByID(t, g.Snapshot(), "P-001")
		if p.Stage != game.Wandering {
			t.Fatalf("a los %v P-001 está %v; un Mild sigue despierto al menos 60 s", elapsed, p.Stage)
		}
		if p.Zone == zone {
			continue
		}
		if stayed := elapsed - since; stayed < 10*time.Second || stayed > 20*time.Second {
			t.Errorf("se quedó %v en %v; debía quedarse de 10 a 20 s", stayed, zone)
		}
		zone, since = p.Zone, elapsed
		changes++
	}
	if changes < 2 {
		t.Errorf("en 55 s cambió de zona %d veces; se esperaban por lo menos 2", changes)
	}
}

// §5.4: un solo cronómetro de espera por episodio. Arranca con el
// desplome, NO vuelve a empezar cuando lo recogen y se detiene con la
// revisión del médico.
func TestWaiting_OneTimerFromTheCollapseUntilTheReview(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)

	tickFor(g, 3*time.Second)
	if w := patientByID(t, g.Snapshot(), "P-001").Waited; w != 3*time.Second {
		t.Errorf("3 s desplomado: Waited = %v", w)
	}

	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second) // C-01 lo recoge: esperando revisión
	if w := patientByID(t, g.Snapshot(), "P-001").Waited; w != 8*time.Second {
		t.Errorf("recogido a los 8 s: Waited = %v; no debía volver a empezar", w)
	}

	dispatch(t, g, "P-001", "D-01")
	tickFor(g, 5*time.Second) // D-01 lo revisa a los 13 s
	tickFor(g, 5*time.Second) // y el cronómetro ya no corre
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.InBed || p.Waited != 13*time.Second {
		t.Errorf("P-001 = (%v, Waited %v); se esperaba (en cama, 13 s): la revisión detiene el cronómetro", p.Stage, p.Waited)
	}
}

// findPatient busca a un paciente en la foto; ok = false si ya no está en el mapa.
func findPatient(snap game.Snapshot, id string) (game.GamePatientView, bool) {
	for _, p := range snap.Patients {
		if p.ID == id {
			return p, true
		}
	}
	return game.GamePatientView{}, false
}

// §5.4: el sueño arranca con la revisión y dura según el nivel, con −8 %
// por punto de pericia del médico que lo revisó. Al cumplirse se llama
// WakePatient, que libera la cama. Un Severe necesita 3 sueños para el alta,
// así que después del primero vuelve a deambular.
func TestSleep_LastsByLevelAndReviewersSkillThenWakesUp(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second) // C-01 lo recoge
	dispatch(t, g, "P-001", "D-01")
	tickFor(g, 5*time.Second) // D-01 (pericia 3) lo revisa a los 10 s

	// 30 s × (100 − 8 × 3) / 100 = 22,8 s de sueño.
	tickFor(g, 22700*time.Millisecond)
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.InBed {
		t.Fatalf("a los 22,7 s de sueño P-001 está %v; debía seguir dormido", p.Stage)
	}
	g.Tick(tick) // 22,8 s
	snap := g.Snapshot()
	p := patientByID(t, snap, "P-001")
	if p.Stage != game.Wandering || p.State != hospital.Awake || p.Room != 0 {
		t.Errorf("al despertar P-001 = (%v, %v, hab. %d); se esperaba (deambulando, despierto, sin cama)", p.Stage, p.State, p.Room)
	}
	if r := snap.Rooms[0]; r.Number != 101 || r.State != hospital.Available {
		t.Errorf("la 101 quedó %v; WakePatient debía liberarla", r.State)
	}
}

// §5.4: un Mild recibe el alta después de 1 sueño. Sale 4 s hacia la puerta
// (que se abre el último segundo), desaparece del mapa y en el Shift Report
// queda "(alta)". Ajuste de Samuel: mientras sale NO vuelve a tener ataque.
func TestDischarge_LeavesWithoutAnotherAttackAndIsMarkedInTheReport(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Mild, game.Cafeteria)
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Severe, game.Hallway2)
	dispatch(t, g, "P-002", "C-01") // ocupa al camillero…
	dispatch(t, g, "P-001", "D-01") // …para que a P-001 lo recoja el médico (sale en su 5.2)
	tickFor(g, 7*time.Second)       // D-01 lo recoge y lo deja en la cama
	dispatch(t, g, "P-001", "D-01")
	tickFor(g, 5*time.Second) // lo revisa a los 12 s

	tickFor(g, 11400*time.Millisecond) // 15 s × 0,76 = 11,4 s de sueño
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.Leaving {
		t.Fatalf("después de su primer sueño, P-001 (Mild) está %v; se esperaba que saliera", p.Stage)
	}

	doorOpened := false
	for elapsed := tick; elapsed <= 4*time.Second; elapsed += tick {
		g.Tick(tick)
		snap := g.Snapshot()
		doorOpened = doorOpened || snap.DoorOpen
		p, onMap := findPatient(snap, "P-001")
		if elapsed < 4*time.Second && (!onMap || p.Stage != game.Leaving) {
			t.Fatalf("a los %v de salir P-001 está %v (¿en el mapa? %v); debía seguir saliendo", elapsed, p.Stage, onMap)
		}
		if elapsed == 4*time.Second && onMap {
			t.Errorf("a los 4 s ya cruzó la puerta; P-001 sigue en el mapa (%v)", p.Stage)
		}
	}
	if !doorOpened {
		t.Error("la puerta nunca se abrió mientras P-001 salía")
	}

	rep := g.Report()
	if eps := rep.Episodes[0].Episodes; len(eps) != 1 || eps[0].PatientID != "P-001" || eps[0].PatientNote != "(alta)" {
		t.Errorf("en la 5.2 de D-01 se esperaba P-001 marcado \"(alta)\": %+v", eps)
	}
}
