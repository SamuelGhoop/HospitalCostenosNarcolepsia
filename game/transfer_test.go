package game_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §5.6: Dispatch sobre un paciente del pasillo. El personal camina hasta él,
// lo carga y, al llegar, el juego llama AssignRoom. Si ya no hay cama, sale
// el aviso con el error del modelo (queda para la consulta 5.3), el paciente
// sigue en el pasillo y el personal queda libre.
func TestTransfer_WithoutABedTheModelErrorIsShown(t *testing.T) {
	g := fullHospital(t)            // 13 s: C-01 lleva a P-003 a la cama hasta los 14 s
	dispatch(t, g, "P-004", "D-01") // no hay camillero libre: puede ir el médico
	if s := staffByID(t, g.Snapshot(), "D-01"); s.Activity != game.GoingToTransfer || s.PatientID != "P-004" {
		t.Errorf("D-01 = (%v, %q); se esperaba (va a llevarlo a una cama, P-004)", s.Activity, s.PatientID)
	}

	tickFor(g, 5*time.Second) // 18 s: llega y AssignRoom falla
	snap := g.Snapshot()
	if p := patientByID(t, snap, "P-004"); p.Stage != game.InHallway || p.State != hospital.AsleepInHallway || p.Room != 0 {
		t.Errorf("P-004 = (%v, %v, hab. %d); se esperaba que siguiera en el pasillo", p.Stage, p.State, p.Room)
	}
	if s := staffByID(t, snap, "D-01"); s.Activity != game.Free {
		t.Errorf("D-01 está %v; sin cama debía quedar libre", s.Activity)
	}
	noBed := hospital.ErrNoRoomAvailable.Error()
	if n := len(snap.Notices); n == 0 || !strings.Contains(snap.Notices[n-1], "P-004") || !strings.Contains(snap.Notices[n-1], noBed) {
		t.Errorf("avisos = %q; se esperaba el error del modelo para P-004", snap.Notices)
	}
	if got := g.Report().LastAssignError; !strings.Contains(got, noBed) {
		t.Errorf("consulta 5.3: último error de AssignRoom = %q; se esperaba %q", got, noBed)
	}

	// La misma regla que para recoger: con C-01 libre, el médico no va.
	if err := g.Dispatch("P-004", "D-01"); !errors.Is(err, game.ErrOrderlyAvailable) {
		t.Errorf("con C-01 libre, mandar a D-01 al pasillo: err = %v; se esperaba ErrOrderlyAvailable", err)
	}
}

// §5.6: si hay cama cuando llega, AssignRoom se la da: el paciente queda
// esperando revisión y el camillero lo lleva 2 s. Ya no hay asignación
// automática: sin el despacho, el del pasillo se quedaría ahí.
func TestTransfer_ToABedThatWasFreed(t *testing.T) {
	g := fullHospital(t)                                // 13 s
	freed := patientByID(t, g.Snapshot(), "P-001").Room // la cama que va a quedar libre
	dispatch(t, g, "P-001", "D-01")                     // lo revisa a los 18 s (Mild: duerme 11,4 s)
	tickFor(g, 16400*time.Millisecond)                  // 29,4 s: se despierta, recibe el alta y libera la cama

	snap := g.Snapshot()
	if p := patientByID(t, snap, "P-001"); p.Stage != game.Leaving {
		t.Fatalf("P-001 está %v; se esperaba que saliera de alta", p.Stage)
	}
	if p := patientByID(t, snap, "P-004"); p.Stage != game.InHallway {
		t.Fatalf("P-004 está %v; sin despacho debía seguir en el pasillo (no hay asignación automática)", p.Stage)
	}

	dispatch(t, g, "P-004", "C-01")
	tickFor(g, 5*time.Second) // 34,4 s: llega y AssignRoom le da la cama libre
	snap = g.Snapshot()
	if p := patientByID(t, snap, "P-004"); p.Stage != game.AwaitingReview || p.State != hospital.AsleepInBed || p.Room != freed {
		t.Errorf("P-004 = (%v, %v, hab. %d); se esperaba (esperando revisión, dormido en cama, %d)", p.Stage, p.State, p.Room, freed)
	}
	if s := staffByID(t, snap, "C-01"); s.Activity != game.Carrying {
		t.Errorf("C-01 está %v; se esperaba que lo llevara a la cama", s.Activity)
	}
	if got := g.Report().LastAssignError; got != "" {
		t.Errorf("consulta 5.3: AssignRoom no falló, pero el último error es %q", got)
	}
}
