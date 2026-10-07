package hospital_test

import (
	"errors"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

func TestOrderly_AttendPatientInBedCountsATransfer(t *testing.T) {
	o := hospital.NewOrderly("C-01", "Wilmer Camargo", 30)
	r := newTestRoom(t, 101, 1)
	p := asleepPatient(t, "P-001")
	if err := r.Occupy(p); err != nil {
		t.Fatal(err)
	}

	rec, err := o.Attend(p, "pasillo 2")
	if err != nil {
		t.Fatalf("Attend devolvió error: %v", err)
	}

	if o.Transfers() != 1 {
		t.Errorf("Transfers() = %d; se esperaba 1", o.Transfers())
	}
	if rec.Room() != r {
		t.Errorf("el registro dice habitación %v; se esperaba la 101", rec.Room())
	}
}

// Caso pedido en la revisión del plan: si no hay cama, el camillero no
// traslada a nadie. Lo deja vigilado en el pasillo y el registro queda sin habitación.
func TestOrderly_AttendPatientWithoutBedRecordsNoRoomAndNoTransfer(t *testing.T) {
	o := hospital.NewOrderly("C-01", "Wilmer Camargo", 30)
	p := asleepPatient(t, "P-004")

	rec, err := o.Attend(p, "pasillo 2")
	if err != nil {
		t.Fatalf("Attend devolvió error: %v", err)
	}

	if rec.Room() != nil {
		t.Errorf("rec.Room() = %v; sin cama debe ser nil", rec.Room())
	}
	if o.Transfers() != 0 {
		t.Errorf("Transfers() = %d; sin cama no hay traslado", o.Transfers())
	}
	if p.State() != hospital.AsleepInHallway {
		t.Errorf("State() = %v; el paciente debe seguir en el pasillo", p.State())
	}
}

func TestOrderly_AttendAwakePatientFails(t *testing.T) {
	o := hospital.NewOrderly("C-01", "Wilmer Camargo", 30)

	if _, err := o.Attend(newTestPatient(t, "P-001", hospital.Mild), "cafetería"); !errors.Is(err, hospital.ErrNotAsleep) {
		t.Errorf("paciente despierto: err = %v; se esperaba ErrNotAsleep", err)
	}
	if _, err := o.Attend(nil, "cafetería"); !errors.Is(err, hospital.ErrNilPatient) {
		t.Errorf("paciente nil: err = %v; se esperaba ErrNilPatient", err)
	}
	if o.Transfers() != 0 {
		t.Error("una atención fallida no debe contar como traslado")
	}
}

func TestOrderly_IsAlwaysAvailable(t *testing.T) {
	if !hospital.NewOrderly("C-01", "Wilmer Camargo", 30).IsAvailable() {
		t.Error("el camillero siempre está de turno")
	}
}
