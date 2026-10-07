package hospital_test

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Polimorfismo: un Doctor y un Orderly son tipos distintos, pero los dos
// cumplen Attender, así que caben en el mismo []Attender y se usan igual.
// El for no sabe ni pregunta qué tipo concreto tiene cada uno.
func TestAttender_DoctorAndOrderlyWorkThroughTheSameInterface(t *testing.T) {
	staff := []hospital.Attender{
		hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño"),
		hospital.NewOrderly("C-01", "Wilmer Camargo", 30),
	}

	for _, a := range staff {
		p := asleepPatient(t, "P-"+a.ID())

		rec, err := a.Attend(p, "pasillo 2")
		if err != nil {
			t.Fatalf("%s (%s) no pudo atender: %v", a.Name(), a.ID(), err)
		}
		if rec.AttendedBy() != a {
			t.Errorf("el registro dice que atendió %v; se esperaba %v", rec.AttendedBy(), a)
		}
		if !a.IsAvailable() {
			t.Errorf("%s debería seguir disponible", a.Name())
		}
	}
}
