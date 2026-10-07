package hospital_test

import (
	"errors"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// newTestPatient crea un paciente de prueba. Los helpers de test reciben
// *testing.T y llaman t.Helper() para que, si algo falla, el error apunte
// a la línea del test y no a la del helper.
func newTestPatient(t *testing.T, id string, level hospital.NarcolepsyLevel) *hospital.Patient {
	t.Helper()
	return hospital.NewPatient(id, "Paciente "+id, 40, level)
}

// checkBedInvariant verifica la invariante central del modelo:
// el paciente está AsleepInBed  ⇔  tiene habitación  ⇔  aparece entre sus ocupantes.
func checkBedInvariant(t *testing.T, p *hospital.Patient) {
	t.Helper()
	inBed := p.State() == hospital.AsleepInBed
	hasRoom := p.Room() != nil
	if inBed != hasRoom {
		t.Fatalf("invariante rota: estado %q pero Room() = %v", p.State(), p.Room())
	}
	if !hasRoom {
		return
	}
	for _, o := range p.Room().Occupants() {
		if o == p {
			return
		}
	}
	t.Fatalf("invariante rota: %s dice estar en la habitación %d pero no está entre sus ocupantes",
		p, p.Room().Number())
}

func TestNewPatient_StartsAwakeAtReception(t *testing.T) {
	p := hospital.NewPatient("P-001", "Yeimy Padilla", 34, hospital.Severe)

	// ID, Name, Age y String vienen de Person: son métodos PROMOVIDOS por el embedding.
	if p.ID() != "P-001" || p.Name() != "Yeimy Padilla" || p.Age() != 34 {
		t.Errorf("identidad = (%q, %q, %d); se esperaba (P-001, Yeimy Padilla, 34)", p.ID(), p.Name(), p.Age())
	}
	if got := p.String(); got != "P-001 Yeimy Padilla" {
		t.Errorf("String() = %q; se esperaba %q", got, "P-001 Yeimy Padilla")
	}
	if p.Level() != hospital.Severe {
		t.Errorf("Level() = %v; se esperaba %v", p.Level(), hospital.Severe)
	}
	if p.State() != hospital.Awake {
		t.Errorf("State() = %v; se esperaba %v", p.State(), hospital.Awake)
	}
	if p.Location() != "recepción" {
		t.Errorf("Location() = %q; se esperaba %q", p.Location(), "recepción")
	}
	if p.Room() != nil {
		t.Errorf("Room() = %v; un paciente nuevo no tiene habitación", p.Room())
	}
	checkBedInvariant(t, p)
}

func TestPatient_SufferSleepAttackLeavesHimAsleepWhereHeWas(t *testing.T) {
	p := newTestPatient(t, "P-001", hospital.Mild)

	if err := p.SufferSleepAttack("cafetería"); err != nil {
		t.Fatalf("SufferSleepAttack devolvió error: %v", err)
	}

	if p.State() != hospital.AsleepInHallway {
		t.Errorf("State() = %v; se esperaba %v", p.State(), hospital.AsleepInHallway)
	}
	if p.Location() != "cafetería" {
		t.Errorf("Location() = %q; se esperaba %q", p.Location(), "cafetería")
	}
	if p.AsleepSince().IsZero() {
		t.Error("AsleepSince() está vacío; debe guardar la hora del ataque")
	}
	checkBedInvariant(t, p)
}

func TestPatient_SufferSleepAttackWhileAsleepFails(t *testing.T) {
	p := newTestPatient(t, "P-001", hospital.Mild)
	if err := p.SufferSleepAttack("cafetería"); err != nil {
		t.Fatal(err)
	}

	err := p.SufferSleepAttack("parqueadero")

	if !errors.Is(err, hospital.ErrAlreadyAsleep) {
		t.Fatalf("err = %v; se esperaba ErrAlreadyAsleep", err)
	}
	if p.Location() != "cafetería" {
		t.Errorf("Location() = %q; un ataque fallido no debe moverlo", p.Location())
	}
}

func TestPatient_WakeUpFromHallway(t *testing.T) {
	p := newTestPatient(t, "P-001", hospital.Mild)
	if err := p.SufferSleepAttack("cafetería"); err != nil {
		t.Fatal(err)
	}

	if err := p.WakeUp(); err != nil {
		t.Fatalf("WakeUp devolvió error: %v", err)
	}

	if p.State() != hospital.Awake {
		t.Errorf("State() = %v; se esperaba %v", p.State(), hospital.Awake)
	}
	checkBedInvariant(t, p)
}

func TestPatient_WakeUpWhileAwakeFails(t *testing.T) {
	p := newTestPatient(t, "P-001", hospital.Mild)

	if err := p.WakeUp(); !errors.Is(err, hospital.ErrAlreadyAwake) {
		t.Fatalf("err = %v; se esperaba ErrAlreadyAwake", err)
	}
}

func TestPatient_WakeUpReleasesHisRoom(t *testing.T) {
	p := newTestPatient(t, "P-001", hospital.Mild)
	r := newTestRoom(t, 101, 1)
	if err := p.SufferSleepAttack("cafetería"); err != nil {
		t.Fatal(err)
	}
	if err := r.Occupy(p); err != nil {
		t.Fatal(err)
	}

	if err := p.WakeUp(); err != nil {
		t.Fatalf("WakeUp devolvió error: %v", err)
	}

	if p.State() != hospital.Awake || p.Room() != nil {
		t.Errorf("paciente = (%v, %v); se esperaba (despierto, sin habitación)", p.State(), p.Room())
	}
	if !r.IsAvailable() || len(r.Occupants()) != 0 {
		t.Errorf("la habitación 101 no quedó libre: estado %v, ocupantes %d", r.State(), len(r.Occupants()))
	}
	checkBedInvariant(t, p)
}

// Test "table-driven": una tabla de casos y un solo for que los recorre.
// Es la forma idiomática de probar muchos valores en Go.
func TestPatientState_String(t *testing.T) {
	tests := []struct {
		state hospital.PatientState
		want  string
	}{
		{hospital.Awake, "despierto"},
		{hospital.AsleepInHallway, "dormido en el pasillo"},
		{hospital.AsleepInBed, "dormido en cama"},
		{hospital.PatientState(99), "PatientState(99)"}, // valor inválido: no debe explotar
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("PatientState(%d).String() = %q; se esperaba %q", int(tt.state), got, tt.want)
		}
	}
}

func TestNarcolepsyLevel_String(t *testing.T) {
	tests := []struct {
		level hospital.NarcolepsyLevel
		want  string
	}{
		{hospital.Mild, "leve"},
		{hospital.Moderate, "moderada"},
		{hospital.Severe, "severa"},
		{hospital.NarcolepsyLevel(7), "NarcolepsyLevel(7)"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("NarcolepsyLevel(%d).String() = %q; se esperaba %q", int(tt.level), got, tt.want)
		}
	}
}

// El valor cero de un tipo (lo que vale una variable declarada sin asignar)
// es el primer valor del iota. Por eso el orden de las constantes importa:
// un paciente recién creado arranca despierto sin tener que decirlo.
func TestStates_ZeroValueIsTheFirstConstant(t *testing.T) {
	var s hospital.PatientState
	if s != hospital.Awake {
		t.Errorf("el valor cero de PatientState es %v; se esperaba %v", s, hospital.Awake)
	}

	var l hospital.NarcolepsyLevel
	if l != hospital.Mild {
		t.Errorf("el valor cero de NarcolepsyLevel es %v; se esperaba %v", l, hospital.Mild)
	}

	var r hospital.RoomState
	if r != hospital.Available {
		t.Errorf("el valor cero de RoomState es %v; se esperaba %v", r, hospital.Available)
	}
}
