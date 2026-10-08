package game_test

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Report y las vistas son SOLO VALORES: si la interfaz modifica lo que
// recibe, la demo no cambia. (Si devolvieran slices o punteros del modelo,
// la interfaz podría tocar el hospital sin pasar por la demo.)
func TestReportAndSnapshotAreCopies(t *testing.T) {
	d := game.NewDemo()
	if err := d.Skip(); err != nil {
		t.Fatal(err)
	}

	r := d.Report()
	r.Rooms[0].Occupants[0] = "X-999"
	r.Severe[0].Episodes = 99
	r.Episodes[0].Episodes[0].PatientID = "X-999"

	snap := d.Snapshot()
	snap.Patients[0].Name = "Otro"
	snap.Rooms[0].Occupants[0] = "X-999"

	again := d.Report()
	if again.Rooms[0].Occupants[0] == "X-999" || again.Severe[0].Episodes == 99 ||
		again.Episodes[0].Episodes[0].PatientID == "X-999" {
		t.Error("modificar el Report cambió la demo; Report() debe devolver copias")
	}
	if d.Snapshot().Patients[0].Name == "Otro" || d.Snapshot().Rooms[0].Occupants[0] == "X-999" {
		t.Error("modificar el DemoSnapshot cambió la demo; Snapshot() debe devolver copias")
	}
}

// En la demo la hora de cada episodio es la hora REAL (At()), en formato HH:MM.
func TestReport_EpisodeLinesCarryTheRealTime(t *testing.T) {
	d := game.NewDemo()
	if err := d.Skip(); err != nil {
		t.Fatal(err)
	}

	for _, de := range d.Report().Episodes {
		for _, e := range de.Episodes {
			if len(e.Time) != len("15:04") || e.Time[2] != ':' {
				t.Errorf("episodio %s: Time = %q; se esperaba HH:MM", e.ID, e.Time)
			}
			if e.AttendedBy == "" || e.Location == "" {
				t.Errorf("episodio %s incompleto: %+v", e.ID, e)
			}
		}
	}
}
