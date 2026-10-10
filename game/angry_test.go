package game_test

import (
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// fullHospital deja la partida a los 13 s con las 3 camas ocupadas por
// pacientes que esperan revisión (P-001 Mild, P-002 Moderate y P-003 Severe)
// y P-004 (Severe) dormido en el pasillo. Los cuatro se desplomaron en el
// instante 0.
func fullHospital(t *testing.T) *game.Game {
	t.Helper()
	g := newGame(t)
	collapse(t, g, "P-001", "Ludys Arrieta", hospital.Mild, game.Cafeteria)
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Moderate, game.Radiology)
	collapse(t, g, "P-003", "Breiner Julio", hospital.Severe, game.Lobby)
	collapse(t, g, "P-004", "Wilfrido Berrío", hospital.Severe, game.Hallway2)

	dispatch(t, g, "P-001", "C-01")
	dispatch(t, g, "P-002", "D-01") // C-01 ya va ocupado
	tickFor(g, 7*time.Second)       // 101 y 102 ocupadas; los dos quedan libres
	dispatch(t, g, "P-003", "C-01")
	tickFor(g, time.Second)
	dispatch(t, g, "P-004", "D-01") // C-01 ya va ocupado
	tickFor(g, 5*time.Second)       // 12 s: P-003 a la 103 · 13 s: P-004 sin cama

	snap := g.Snapshot()
	for id, want := range map[string]game.Stage{"P-001": game.AwaitingReview, "P-002": game.AwaitingReview, "P-003": game.AwaitingReview, "P-004": game.InHallway} {
		if p := patientByID(t, snap, id); p.Stage != want {
			t.Fatalf("preparando el hospital lleno: %s está %v; se esperaba %v", id, p.Stage, want)
		}
	}
	return g
}

// walksOut revisa que el paciente salga durante 4 s sin volver a tener
// ataque (ajuste de Samuel) y que después desaparezca del mapa.
func walksOut(t *testing.T, g *game.Game, id string) {
	t.Helper()
	for elapsed := tick; elapsed <= 4*time.Second; elapsed += tick {
		g.Tick(tick)
		p, onMap := findPatient(g.Snapshot(), id)
		if elapsed < 4*time.Second && (!onMap || p.Stage != game.Leaving) {
			t.Fatalf("a los %v de salir %s está %v (¿en el mapa? %v); debía seguir saliendo", elapsed, id, p.Stage, onMap)
		}
		if elapsed == 4*time.Second && onMap {
			t.Errorf("a los 4 s %s ya cruzó la puerta, pero sigue en el mapa (%v)", id, p.Stage)
		}
	}
}

// §5.4: a los 45 s del desplome sin revisión se despierta solo y se va
// enojado. Si nadie lo había recogido, el modelo nunca se enteró: el juego
// no llama nada del modelo y no queda episodio.
func TestAngry_NobodyPickedHimUpSoTheModelNeverKnew(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)

	tickFor(g, 44900*time.Millisecond)
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.Collapsed {
		t.Fatalf("a los 44,9 s P-001 está %v; todavía espera", p.Stage)
	}

	g.Tick(tick) // 45 s
	snap := g.Snapshot()
	if p := patientByID(t, snap, "P-001"); p.Stage != game.Leaving || p.State != hospital.Awake {
		t.Errorf("a los 45 s P-001 = (%v, %v); se esperaba (saliendo, despierto)", p.Stage, p.State)
	}
	// Un solo aviso: si el juego hubiera llamado WakePatient, el modelo
	// habría devuelto un error (ya estaba despierto) y saldría otro aviso.
	if len(snap.Notices) != 1 || !strings.Contains(snap.Notices[0], "P-001") || !strings.Contains(snap.Notices[0], "se fue enojado") {
		t.Errorf("avisos = %q; se esperaba solo el de P-001 enojado", snap.Notices)
	}
	// La consulta 5.4 lista a TODOS los Severe admitidos, también con 0
	// episodios: P-001 sale con 0 (nadie lo recogió) y marcado.
	if line, ok := severeLine(g.Report(), "P-001"); !ok || line.Episodes != 0 || line.Note != "(se fue enojado)" {
		t.Errorf("consulta 5.4 de P-001 = %+v (¿está? %v); se esperaban 0 episodios y \"(se fue enojado)\"", line, ok)
	}

	walksOut(t, g, "P-001")
}

// §3.4 y §5.4: si estaba en el pasillo o esperando revisión, el juego llama
// WakePatient: el modelo lo despierta y libera la cama si tenía. En el Shift
// Report sale marcado "(se fue enojado)".
func TestAngry_FromTheHallwayOrAwaitingReviewCallsWakePatient(t *testing.T) {
	g := fullHospital(t)
	tickFor(g, 32*time.Second) // 45 s desde el desplome de los cuatro

	snap := g.Snapshot()
	for _, id := range []string{"P-001", "P-002", "P-003", "P-004"} {
		if p := patientByID(t, snap, id); p.Stage != game.Leaving || p.State != hospital.Awake || p.Room != 0 {
			t.Errorf("%s = (%v, %v, hab. %d); se esperaba (saliendo, despierto, sin cama)", id, p.Stage, p.State, p.Room)
		}
	}
	for _, r := range snap.Rooms {
		if r.State != hospital.Available {
			t.Errorf("la %d quedó %v; WakePatient debía liberarla", r.Number, r.State)
		}
	}

	rep := g.Report()
	if len(rep.Hallway) != 0 {
		t.Errorf("consulta 5.1 = %+v; el del pasillo ya se fue", rep.Hallway)
	}
	for _, id := range []string{"P-003", "P-004"} { // los Severe del hospital lleno: 1 episodio cada uno
		if line, ok := severeLine(rep, id); !ok || line.Episodes != 1 || line.Note != "(se fue enojado)" {
			t.Errorf("consulta 5.4 de %s = %+v (¿está? %v); se esperaba 1 episodio y \"(se fue enojado)\"", id, line, ok)
		}
	}
}

// severeLine busca la línea de un paciente en la consulta 5.4.
func severeLine(rep game.Report, id string) (game.SevereLine, bool) {
	for _, line := range rep.Severe {
		if line.PatientID == id {
			return line, true
		}
	}
	return game.SevereLine{}, false
}

// Si el médico lo revisa a los 44,9 s, el cronómetro de espera se detiene y
// ya no se va.
func TestAngry_ReviewedJustInTimeStays(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 39900*time.Millisecond) // C-01 lo dejó en la 101 a los 7 s
	dispatch(t, g, "P-001", "D-01")    // llega a los 44,9 s

	tickFor(g, 10*time.Second)
	if p := patientByID(t, g.Snapshot(), "P-001"); p.Stage != game.InBed || p.Waited != 44900*time.Millisecond {
		t.Errorf("P-001 = (%v, Waited %v); revisado a los 44,9 s debía quedarse en cama", p.Stage, p.Waited)
	}
}

// §5.5: si se va enojado mientras alguien va en camino hacia él, el despacho
// se cancela SIN llamar al modelo y esa persona queda libre.
func TestAngry_CancelsWhoeverWasOnTheWay(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Severe, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01") // lo recoge a los 5 s: queda esperando revisión
	tickFor(g, 2*time.Second)
	collapse(t, g, "P-002", "Kevin Mercado", hospital.Severe, game.Radiology) // se va a los 47 s

	tickFor(g, 39*time.Second)      // 41 s
	dispatch(t, g, "P-001", "D-01") // llegaría a revisarlo a los 46 s
	tickFor(g, 2*time.Second)       // 43 s
	dispatch(t, g, "P-002", "C-01") // llegaría a recogerlo a los 48 s

	tickFor(g, 2*time.Second) // 45 s: P-001 se va
	if s := staffByID(t, g.Snapshot(), "D-01"); s.Activity != game.Free {
		t.Errorf("P-001 se fue y D-01 sigue %v hacia él; el despacho debía cancelarse", s.Activity)
	}

	tickFor(g, 2*time.Second) // 47 s: P-002 se va
	if s := staffByID(t, g.Snapshot(), "C-01"); s.Activity != game.Free {
		t.Errorf("P-002 se fue y C-01 sigue %v hacia él; el despacho debía cancelarse", s.Activity)
	}

	tickFor(g, 3*time.Second) // 50 s: ya habrían llegado los dos
	if line, _ := severeLine(g.Report(), "P-002"); line.Episodes != 0 {
		t.Errorf("a P-002 nadie lo alcanzó a recoger, pero el modelo tiene %d episodios suyos", line.Episodes)
	}
	if p, onMap := findPatient(g.Snapshot(), "P-001"); onMap && p.Stage == game.InBed {
		t.Error("D-01 revisó a P-001 después de que se fue")
	}
}
