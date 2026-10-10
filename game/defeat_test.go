package game_test

import (
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §5.10: con 5 o más pacientes desplomados o en el pasillo a la vez, el
// hospital colapsa.
func TestDefeat_FivePatientsDownAtOnceCollapseTheHospital(t *testing.T) {
	g := newGame(t)
	for _, id := range []string{"P-001", "P-002", "P-003", "P-004"} {
		collapse(t, g, id, "Paciente "+id, hospital.Mild, game.Cafeteria)
	}
	g.Tick(tick)
	if over := g.Snapshot().GameOver; over != game.NotOver {
		t.Fatalf("con 4 desplomados la partida terminó: %v", over)
	}

	collapse(t, g, "P-005", "Paciente P-005", hospital.Mild, game.Cafeteria)
	g.Tick(tick)
	if over := g.Snapshot().GameOver; over != game.HospitalCollapsed || over.String() != "¡HOSPITAL COLAPSADO!" {
		t.Errorf("con 5 desplomados: GameOver = %v; se esperaba ¡HOSPITAL COLAPSADO!", over)
	}
}

// Para el colapso NO cuentan los que esperan revisión (§5.9).
func TestDefeat_AwaitingReviewDoesNotCountForTheCollapse(t *testing.T) {
	g := newGame(t)
	collapse(t, g, "P-001", "Yeimy Padilla", hospital.Mild, game.Cafeteria)
	dispatch(t, g, "P-001", "C-01")
	tickFor(g, 5*time.Second) // esperando revisión
	for _, id := range []string{"P-002", "P-003", "P-004", "P-005"} {
		collapse(t, g, id, "Paciente "+id, hospital.Mild, game.Cafeteria)
	}
	g.Tick(tick)
	if over := g.Snapshot().GameOver; over != game.NotOver {
		t.Errorf("4 desplomados + 1 esperando revisión: la partida terminó (%v); no debía", over)
	}
}
