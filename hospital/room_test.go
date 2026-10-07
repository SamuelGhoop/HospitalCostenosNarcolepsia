package hospital_test

import (
	"errors"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// newTestRoom crea una habitación válida o detiene el test si falla.
func newTestRoom(t *testing.T, number, capacity int) *hospital.Room {
	t.Helper()
	r, err := hospital.NewRoom(number, capacity)
	if err != nil {
		t.Fatalf("NewRoom(%d, %d) devolvió error: %v", number, capacity, err)
	}
	return r
}

// asleepPatient crea un paciente que ya sufrió un ataque (AsleepInHallway).
func asleepPatient(t *testing.T, id string) *hospital.Patient {
	t.Helper()
	p := newTestPatient(t, id, hospital.Moderate)
	if err := p.SufferSleepAttack("pasillo 2"); err != nil {
		t.Fatalf("SufferSleepAttack devolvió error: %v", err)
	}
	return p
}

func TestRoomState_String(t *testing.T) {
	tests := []struct {
		state hospital.RoomState
		want  string
	}{
		{hospital.Available, "disponible"},
		{hospital.Occupied, "ocupada"},
		{hospital.RoomState(5), "RoomState(5)"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("RoomState(%d).String() = %q; se esperaba %q", int(tt.state), got, tt.want)
		}
	}
}

func TestNewRoom_StartsEmptyAndAvailable(t *testing.T) {
	r := newTestRoom(t, 101, 1)

	if r.Number() != 101 || r.Capacity() != 1 {
		t.Errorf("habitación = (%d, cap %d); se esperaba (101, cap 1)", r.Number(), r.Capacity())
	}
	if r.State() != hospital.Available || !r.IsAvailable() {
		t.Errorf("State() = %v, IsAvailable() = %v; se esperaba disponible", r.State(), r.IsAvailable())
	}
	if len(r.Occupants()) != 0 {
		t.Errorf("Occupants() tiene %d; se esperaba 0", len(r.Occupants()))
	}
}

func TestNewRoom_RejectsInvalidCapacity(t *testing.T) {
	for _, capacity := range []int{0, -1} {
		r, err := hospital.NewRoom(101, capacity)
		if !errors.Is(err, hospital.ErrInvalidCapacity) {
			t.Errorf("NewRoom(101, %d): err = %v; se esperaba ErrInvalidCapacity", capacity, err)
		}
		if r != nil {
			t.Errorf("NewRoom(101, %d) devolvió una habitación; con error debe ser nil", capacity)
		}
	}
}

func TestRoom_OccupyPutsSleepingPatientInBed(t *testing.T) {
	r := newTestRoom(t, 101, 1)
	p := asleepPatient(t, "P-001")

	if err := r.Occupy(p); err != nil {
		t.Fatalf("Occupy devolvió error: %v", err)
	}

	if p.State() != hospital.AsleepInBed || p.Room() != r {
		t.Errorf("paciente = (%v, %v); se esperaba (dormido en cama, habitación 101)", p.State(), p.Room())
	}
	if p.Location() != "habitación 101" {
		t.Errorf("Location() = %q; se esperaba %q", p.Location(), "habitación 101")
	}
	if r.State() != hospital.Occupied || r.IsAvailable() {
		t.Errorf("habitación: State() = %v, IsAvailable() = %v; se esperaba ocupada", r.State(), r.IsAvailable())
	}
	checkBedInvariant(t, p)
}

func TestRoom_OccupyFullRoomFails(t *testing.T) {
	r := newTestRoom(t, 101, 1)
	first := asleepPatient(t, "P-001")
	second := asleepPatient(t, "P-002")
	if err := r.Occupy(first); err != nil {
		t.Fatal(err)
	}

	err := r.Occupy(second)

	if !errors.Is(err, hospital.ErrRoomFull) {
		t.Fatalf("err = %v; se esperaba ErrRoomFull", err)
	}
	if second.State() != hospital.AsleepInHallway || second.Room() != nil {
		t.Errorf("el segundo paciente cambió: (%v, %v); debe seguir en el pasillo", second.State(), second.Room())
	}
	if len(r.Occupants()) != 1 {
		t.Errorf("la habitación tiene %d ocupantes; se esperaba 1", len(r.Occupants()))
	}
	checkBedInvariant(t, second)
}

func TestRoom_OccupyRejectsInvalidPatients(t *testing.T) {
	inOtherRoom := asleepPatient(t, "P-003")
	if err := newTestRoom(t, 102, 1).Occupy(inOtherRoom); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		patient *hospital.Patient
		want    error
	}{
		{"paciente nil", nil, hospital.ErrNilPatient},
		{"paciente despierto", newTestPatient(t, "P-001", hospital.Mild), hospital.ErrNotAsleep},
		{"paciente ya en otra cama", inOtherRoom, hospital.ErrAlreadyInBed},
	}

	for _, tt := range tests {
		r := newTestRoom(t, 101, 1)
		if err := r.Occupy(tt.patient); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v; se esperaba %v", tt.name, err, tt.want)
		}
		if len(r.Occupants()) != 0 {
			t.Errorf("%s: la habitación quedó con %d ocupantes; se esperaba 0", tt.name, len(r.Occupants()))
		}
	}
}

func TestRoom_RoomWithCapacityTwoFillsUpOnSecondPatient(t *testing.T) {
	r := newTestRoom(t, 201, 2)

	if err := r.Occupy(asleepPatient(t, "P-001")); err != nil {
		t.Fatal(err)
	}
	if !r.IsAvailable() {
		t.Error("con 1 de 2 camas ocupadas la habitación debe seguir disponible")
	}

	if err := r.Occupy(asleepPatient(t, "P-002")); err != nil {
		t.Fatal(err)
	}
	if r.IsAvailable() || r.State() != hospital.Occupied {
		t.Error("con 2 de 2 camas ocupadas la habitación debe quedar ocupada")
	}
}

func TestRoom_ReleaseFreesBedAndLeavesPatientAsleepInHallway(t *testing.T) {
	r := newTestRoom(t, 101, 1)
	p := asleepPatient(t, "P-001")
	if err := r.Occupy(p); err != nil {
		t.Fatal(err)
	}

	if err := r.Release(p); err != nil {
		t.Fatalf("Release devolvió error: %v", err)
	}

	if !r.IsAvailable() || r.State() != hospital.Available || len(r.Occupants()) != 0 {
		t.Errorf("habitación tras Release: estado %v, %d ocupantes; se esperaba libre", r.State(), len(r.Occupants()))
	}
	// Lo sacaron de la cama pero nadie lo despertó: sigue dormido, ahora sin cama.
	if p.State() != hospital.AsleepInHallway || p.Room() != nil {
		t.Errorf("paciente = (%v, %v); se esperaba (dormido en el pasillo, sin habitación)", p.State(), p.Room())
	}
	checkBedInvariant(t, p)
}

func TestRoom_ReleasePatientNotInRoomFails(t *testing.T) {
	r := newTestRoom(t, 101, 1)
	p := asleepPatient(t, "P-001")

	if err := r.Release(p); !errors.Is(err, hospital.ErrNotInRoom) {
		t.Fatalf("err = %v; se esperaba ErrNotInRoom", err)
	}
}

func TestRoom_OccupantsReturnsACopy(t *testing.T) {
	r := newTestRoom(t, 101, 1)
	p := asleepPatient(t, "P-001")
	if err := r.Occupy(p); err != nil {
		t.Fatal(err)
	}

	list := r.Occupants()
	list[0] = nil // alguien de afuera intenta "sacar" al paciente

	if r.Occupants()[0] != p {
		t.Error("modificar el slice devuelto cambió la habitación; Occupants() debe devolver una copia")
	}
}
