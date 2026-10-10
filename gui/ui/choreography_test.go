package ui

import "testing"

// runChoreo avanza una coreografía como lo hace la escena: cada personaje
// llega de una vez a su destino y el paciente indicado ya está acostado.
func runChoreo(c *choreography, pos map[string]vec, lying map[string]bool, maxTicks int) {
	for t := 0; t < maxTicks && !c.done(); t++ {
		for id := range pos {
			if to, ok := c.target(id); ok {
				pos[id] = to
			}
		}
		c.step(
			func(id string) bool { to, ok := c.target(id); return !ok || pos[id] == to },
			func(id string) bool { return lying[id] },
		)
	}
}

// Pasos 5–7: el paciente camina y se desploma; el que lo atendió camina
// hasta él; lo llevan a la cama; el que atendió vuelve a su sala.
func TestAttendChoreography_WithBedFollowsTheFourPhases(t *testing.T) {
	attack, home := vec{30, 160}, vec{480, 145}
	c := attendChoreography("P-001", "D-01", attack, home, zoneRoom101, true)

	// Fase 1: solo se mueve el paciente; el médico espera en su sala.
	if to, ok := c.target("P-001"); !ok || to != attack {
		t.Fatalf("fase 1: el paciente va a %v; se esperaba %v", to, attack)
	}
	if to, ok := c.target("D-01"); !ok || to != home {
		t.Fatalf("fase 1: el médico va a (%v, %v); debe esperar en su sala %v", to, ok, home)
	}

	// Mientras el paciente no esté acostado, la fase 1 no termina.
	c.step(func(string) bool { return true }, func(string) bool { return false })
	if c.i != 0 {
		t.Fatalf("la fase 1 terminó sin que el paciente se desplomara")
	}

	// Ya acostado: fase 2, el médico camina hasta él.
	c.step(func(string) bool { return true }, func(string) bool { return true })
	if to, _ := c.target("D-01"); to != helperSpot(attack) {
		t.Errorf("fase 2: el médico va a %v; se esperaba al lado del paciente", to)
	}
	if to, _ := c.target("P-001"); to != attack {
		t.Error("en la fase 2 el paciente sigue donde se desplomó")
	}

	// Fase 3: los dos van a la habitación; fase 4: el médico vuelve.
	c.step(func(string) bool { return true }, func(string) bool { return true })
	if to, _ := c.target("P-001"); to != bedCell(zoneRoom101) {
		t.Errorf("fase 3: el paciente va a %v; se esperaba la cama de la 101", to)
	}
	if to, _ := c.target("D-01"); to != toVec(carrySpot(zoneRoom101)) {
		t.Errorf("fase 3: el médico va a %v; se esperaba al lado de la cama", to)
	}
	c.step(func(string) bool { return true }, func(string) bool { return true })
	if to, _ := c.target("D-01"); to != home {
		t.Errorf("fase 4: el médico va a %v; se esperaba su sala %v", to, home)
	}
	c.step(func(string) bool { return true }, func(string) bool { return true })
	if !c.done() {
		t.Error("después de las 4 fases la coreografía debe terminar")
	}
}

// Paso 8: sin cama, el que atiende se queda a su lado un rato y vuelve.
func TestAttendChoreography_WithoutBedStaysBesideThenGoesBack(t *testing.T) {
	attack, home := vec{420, 210}, vec{480, 145}
	c := attendChoreography("P-004", "D-01", attack, home, 0, false)
	pos := map[string]vec{"P-004": {200, 140}, "D-01": home}

	runChoreo(c, pos, map[string]bool{"P-004": true}, 1000)

	if !c.done() {
		t.Fatal("la coreografía no terminó")
	}
	if pos["P-004"] != attack || pos["D-01"] != home {
		t.Errorf("al final: paciente en %v y médico en %v; se esperaba %v y %v", pos["P-004"], pos["D-01"], attack, home)
	}
}

func TestAttendChoreography_WithoutBedWaitsBesideThePatient(t *testing.T) {
	attack := vec{420, 210}
	c := attendChoreography("P-004", "D-01", attack, vec{480, 145}, 0, false)
	yes := func(string) bool { return true }
	c.step(yes, yes) // fase 1 terminada
	c.step(yes, yes) // fase 2 terminada: el médico llegó a su lado

	for i := 0; i < helperHold-1; i++ {
		c.step(yes, yes)
	}
	if to, _ := c.target("D-01"); to != helperSpot(attack) {
		t.Errorf("antes de %d ticks el médico ya se fue a %v", helperHold, to)
	}
}

// Paso 11: el camillero va hasta Wilfrido, lo lleva a su cama y vuelve.
func TestCarryChoreography_TheOrderlyTakesThePatientToBed(t *testing.T) {
	hallway, home := vec{420, 210}, vec{536, 145}
	c := carryChoreography("P-004", "C-01", hallway, home, zoneRoom101)

	// El paciente debe quedarse QUIETO en el pasillo hasta que llegue el
	// camillero (si la coreografía no lo fija, la escena lo mandaría solo a
	// su cama, que es su puesto final en el modelo).
	if to, ok := c.target("P-004"); !ok || to != hallway {
		t.Fatalf("fase 1: el paciente va a (%v, %v); debe quedarse en el pasillo %v", to, ok, hallway)
	}
	if to, _ := c.target("C-01"); to != helperSpot(hallway) {
		t.Errorf("fase 1: el camillero va a %v; se esperaba al lado del paciente", to)
	}

	pos := map[string]vec{"P-004": hallway, "C-01": home}
	runChoreo(c, pos, map[string]bool{"P-004": true}, 1000)
	if pos["P-004"] != bedCell(zoneRoom101) || pos["C-01"] != home {
		t.Errorf("al final: paciente en %v y camillero en %v", pos["P-004"], pos["C-01"])
	}
}
