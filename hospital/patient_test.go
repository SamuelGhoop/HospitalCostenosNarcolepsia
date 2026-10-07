package hospital_test

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

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
