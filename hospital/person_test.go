// Los tests usan el paquete hospital_test (caja negra): solo ven lo exportado,
// igual que main. Si un test necesitara un campo en minúscula, no compilaría;
// eso demuestra que la encapsulación funciona.
package hospital_test

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

func TestNewPerson_GettersReturnIdentity(t *testing.T) {
	p := hospital.NewPerson("P-001", "Yeimy Padilla", 34)

	if got := p.ID(); got != "P-001" {
		t.Errorf("ID() = %q; se esperaba %q", got, "P-001")
	}
	if got := p.Name(); got != "Yeimy Padilla" {
		t.Errorf("Name() = %q; se esperaba %q", got, "Yeimy Padilla")
	}
	if got := p.Age(); got != 34 {
		t.Errorf("Age() = %d; se esperaba %d", got, 34)
	}
}

func TestPerson_StringShowsIDAndName(t *testing.T) {
	p := hospital.NewPerson("P-001", "Yeimy Padilla", 34)

	if got, want := p.String(), "P-001 Yeimy Padilla"; got != want {
		t.Errorf("String() = %q; se esperaba %q", got, want)
	}
}
