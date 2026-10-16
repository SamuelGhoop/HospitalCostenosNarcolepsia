package game

// Test interno (package game): revisa candidate.female y roleLabel, que no
// se ven desde afuera del paquete (CandidateView solo expone RoleLabel).

import (
	"slices"
	"strings"
	"testing"
)

// §6, F4: la hoja del candidato dice "Médica"/"Camillera" cuando el nombre
// salió femenino, el mismo sorteo que ya le pone "Dra." al nombre.
func TestCandidateRoleLabel_MatchesTheNameGender(t *testing.T) {
	g := newInternalGame(t)
	for i := 0; i < 200; i++ {
		c := g.newCandidateLocked()
		first := firstNameOf(c)
		female := slices.Contains(femaleFirstNames, first)
		got := roleLabel(c.isDoctor, c.female)

		if female != c.female {
			t.Fatalf("%q: candidate.female = %v, pero el nombre es de la lista femenina = %v", c.name, c.female, female)
		}
		want := map[bool]map[bool]string{
			true:  {true: "Médica", false: "Médico"},
			false: {true: "Camillera", false: "Camillero"},
		}[c.isDoctor][female]
		if got != want {
			t.Errorf("%q (médico=%v, femenino=%v): RoleLabel = %q; se esperaba %q", c.name, c.isDoctor, female, got, want)
		}
	}
}

// firstNameOf saca el primer nombre de la ficha del candidato: en los
// médicos, el nombre trae el título ("Dra. Yeimy Polo") antes.
func firstNameOf(c *candidate) string {
	parts := strings.Fields(c.name)
	if c.isDoctor {
		return parts[1]
	}
	return parts[0]
}
