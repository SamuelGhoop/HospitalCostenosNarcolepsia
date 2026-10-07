package hospital_test

import (
	"strings"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

func TestEpisodeRecord_CapturesTheMomentOfTheEpisode(t *testing.T) {
	treating := newTestDoctor(t, "D-02")
	attender := newTestDoctor(t, "D-01")
	r := newTestRoom(t, 101, 1)
	p := newTestPatient(t, "P-001", hospital.Severe)
	if err := treating.DiagnosePatient(p); err != nil {
		t.Fatal(err)
	}
	if err := p.SufferSleepAttack("cafetería"); err != nil {
		t.Fatal(err)
	}
	if err := r.Occupy(p); err != nil {
		t.Fatal(err)
	}

	rec, err := attender.Attend(p, "cafetería")
	if err != nil {
		t.Fatal(err)
	}

	if rec.Patient() != p || rec.Location() != "cafetería" || rec.Room() != r {
		t.Errorf("registro = (%v, %q, %v)", rec.Patient(), rec.Location(), rec.Room())
	}
	if rec.AttendedBy() != attender { // se compara la interfaz con el puntero concreto
		t.Errorf("AttendedBy() = %v; se esperaba %v", rec.AttendedBy(), attender)
	}
	if rec.TreatingDoctor() != treating {
		t.Errorf("TreatingDoctor() = %v; se esperaba %v", rec.TreatingDoctor(), treating)
	}
	if !rec.At().Equal(p.AsleepSince()) {
		t.Errorf("At() = %v; se esperaba la hora del ataque %v", rec.At(), p.AsleepSince())
	}

	// El registro es una foto del momento: si el paciente se despierta,
	// el registro sigue diciendo que estuvo en la habitación 101.
	if err := p.WakeUp(); err != nil {
		t.Fatal(err)
	}
	if rec.Room() != r {
		t.Error("el registro cambió después de que el paciente se despertó; debe ser inmutable")
	}
}

func TestEpisodeRecord_IDsAreUnique(t *testing.T) {
	d := newTestDoctor(t, "D-01")
	first, err := d.Attend(asleepPatient(t, "P-001"), "cafetería")
	if err != nil {
		t.Fatal(err)
	}
	second, err := d.Attend(asleepPatient(t, "P-002"), "parqueadero")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(first.ID(), "EP-") || first.ID() == second.ID() {
		t.Errorf("IDs = %q y %q; se esperaban dos IDs distintos con prefijo EP-", first.ID(), second.ID())
	}
}

func TestEpisodeRecord_SummaryShowsWhoAttendedAndTheTreatingDoctor(t *testing.T) {
	treating := hospital.NewDoctor("D-02", "Dr. Rafa Escalona", 50, "Medicina interna")
	attender := hospital.NewOrderly("C-01", "Wilmer Camargo", 30)
	r := newTestRoom(t, 101, 1)
	p := hospital.NewPatient("P-001", "Yeimy Padilla", 34, hospital.Severe)
	if err := treating.DiagnosePatient(p); err != nil {
		t.Fatal(err)
	}
	if err := p.SufferSleepAttack("cafetería"); err != nil {
		t.Fatal(err)
	}
	if err := r.Occupy(p); err != nil {
		t.Fatal(err)
	}

	rec, err := attender.Attend(p, "cafetería")
	if err != nil {
		t.Fatal(err)
	}
	summary := rec.Summary()

	for _, want := range []string{
		rec.ID(), "P-001 Yeimy Padilla", "cafetería", "habitación 101",
		"atendió: Wilmer Camargo", "tratante: Dr. Rafa Escalona",
	} {
		if !strings.Contains(summary, want) {
			t.Errorf("Summary() = %q; falta %q", summary, want)
		}
	}
}

func TestEpisodeRecord_SummaryWithoutBedNorTreatingDoctor(t *testing.T) {
	attender := hospital.NewOrderly("C-01", "Wilmer Camargo", 30)

	rec, err := attender.Attend(asleepPatient(t, "P-004"), "pasillo 2")
	if err != nil {
		t.Fatal(err)
	}
	summary := rec.Summary()

	for _, want := range []string{"pasillo (sin cama)", "sin tratante"} {
		if !strings.Contains(summary, want) {
			t.Errorf("Summary() = %q; falta %q", summary, want)
		}
	}
}
