package ui

import "testing"

// El médico camina con la animación de su dirección (a la derecha, la de
// la izquierda volteada) y quieto respira.
func TestStaffAnim_WalksAndStands(t *testing.T) {
	a := &staffAnim{}
	for _, c := range []struct {
		dx, dy float64
		want   animation
		flip   bool
	}{
		{-3, 1, animWalkLeft, false},
		{3, -1, animWalkLeft, true},
		{1, -3, animWalkUp, false},
		{0, 3, animWalkDown, false},
		{0, 0, animIdle, false},
	} {
		a.step(c.dx, c.dy, false)
		if a.anim != c.want || a.flip != c.flip {
			t.Errorf("moviéndose (%v, %v): %v volteado=%v; se esperaba %v volteado=%v", c.dx, c.dy, a.anim, a.flip, c.want, c.flip)
		}
	}
}

// Al llegar junto al paciente que atiende, hace atender UNA vez (sus 4
// cuadros) y queda quieto aunque siga a su lado; si camina y vuelve a
// parar junto a otro, atiende otra vez.
func TestStaffAnim_AttendsOnceWhenItStopsBesideThePatient(t *testing.T) {
	a := &staffAnim{}
	a.step(2, 0, true) // todavía caminando hacia él: no atiende
	if a.anim == animAttend {
		t.Fatal("atendió caminando")
	}
	a.step(0, 0, true)
	if a.anim != animAttend {
		t.Fatalf("al parar junto al paciente quedó en %v; se esperaba atender", a.anim)
	}
	for i := 1; i < attendTicks; i++ {
		a.step(0, 0, true)
		if a.anim != animAttend {
			t.Fatalf("dejó de atender a los %d ticks; dura %d", i, attendTicks)
		}
	}
	for i := 0; i < 3*attendTicks; i++ {
		a.step(0, 0, true)
	}
	if a.anim != animIdle {
		t.Errorf("después de atender quedó en %v; se esperaba quieto (atiende una sola vez)", a.anim)
	}

	a.step(0, 2, false) // se va
	a.step(0, 0, true)  // y para junto a otro paciente
	if a.anim != animAttend {
		t.Errorf("en la siguiente parada quedó en %v; se esperaba atender otra vez", a.anim)
	}
}

// Fase 2 de un episodio que atiende un médico: al llegar junto al
// paciente, la coreografía lo marca como el que atiende y espera a que
// termine de atender antes de llevarlo a la cama.
func TestAttendChoreography_TheDoctorAttendsBeforeCarrying(t *testing.T) {
	attack := vec{30, 160}
	c := attendChoreography("P-001", "D-01", attack, vec{480, 145}, zoneRoom101, true, attendTicks, nil)
	yes := func(string) bool { return true }
	if c.attending() != "" {
		t.Errorf("en la fase 1 ya atiende %q", c.attending())
	}
	c.step(yes, yes) // fase 1 terminada: el médico va hacia el paciente
	if c.attending() != "D-01" {
		t.Fatalf("en la fase 2 atiende %q; se esperaba D-01", c.attending())
	}
	for i := 0; i < attendTicks-1; i++ {
		c.step(yes, yes)
	}
	if to, _ := c.target("P-001"); to != attack {
		t.Fatalf("antes de %d ticks ya se lo llevaron a %v", attendTicks, to)
	}
	c.step(yes, yes)
	if to, _ := c.target("P-001"); to != bedCell(zoneRoom101) {
		t.Errorf("después de atender el paciente va a %v; se esperaba la cama", to)
	}
	if c.attending() != "" {
		t.Errorf("llevándolo a la cama sigue atendiendo %q", c.attending())
	}
}
