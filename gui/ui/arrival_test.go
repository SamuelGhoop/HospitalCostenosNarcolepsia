package ui

import (
	"math"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Paso 3 de la demo (pedido de Samuel, 2026-10-11): los 5 pacientes
// aparecen fuera de la pantalla, a la derecha, y entran caminando por la
// acera y por la puerta principal hasta su puesto de la recepción, uno
// cada 1,5 s. La coreografía (y con ella el avance automático) dura hasta
// que llega el último. La puerta se abre cuando alguien está entrando y se
// cierra cuando no viene nadie.
func TestDemo_PatientsArriveFromTheSidewalkOneByOne(t *testing.T) {
	needFonts(t)
	s := NewDemoScene(game.NewDemo())
	for step := 1; step <= 3; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		if step < 3 {
			s.finishChoreography()
		}
	}
	var ids []string
	for _, f := range s.figures {
		if f.patient {
			ids = append(ids, f.id)
			if s.pos[f.id].x < ScreenWidth {
				t.Errorf("%s aparece en %v, dentro de la pantalla; se esperaba fuera, a la derecha", f.id, s.pos[f.id])
			}
		}
	}
	if len(ids) != 5 {
		t.Fatalf("en el paso 3 hay %d pacientes; se esperaban 5", len(ids))
	}
	if s.door.open != 0 {
		t.Error("la puerta está abierta antes de que venga nadie")
	}

	started := map[string]int{}
	entered := map[string]bool{}
	door := doorways[len(doorways)-1].gap
	sawOpen := false
	tick := 0
	for ; tick < 6000 && s.choreo != nil; tick++ {
		before := map[string]vec{}
		for _, id := range ids {
			before[id] = s.pos[id]
		}
		s.animate()
		sawOpen = sawOpen || s.door.open == 1
		for _, id := range ids {
			a, b := before[id].add(feetOffset), s.pos[id].add(feetOffset)
			if _, ok := started[id]; !ok && a != b {
				started[id] = tick
			}
			if a.y >= float64(door.Max.Y) && b.y < float64(door.Max.Y) && b.x >= float64(door.Min.X) && b.x < float64(door.Max.X) {
				entered[id] = true // cruzó la fachada por el hueco de la puerta
			}
		}
	}
	for i, id := range ids {
		if i > 0 {
			if gap := started[id] - started[ids[i-1]]; gap != arrivalGap {
				t.Errorf("%s salió %d ticks después de %s; se esperaban %d (1,5 s)", id, gap, ids[i-1], arrivalGap)
			}
		}
		if !entered[id] {
			t.Errorf("%s no entró por la puerta principal", id)
		}
	}
	for _, f := range s.figures {
		if f.patient && s.pos[f.id] != f.target {
			t.Errorf("la coreografía terminó y %s no ha llegado a su puesto (%v, se esperaba %v)", f.id, s.pos[f.id], f.target)
		}
	}
	if !sawOpen {
		t.Error("la puerta nunca se abrió del todo mientras entraban")
	}
	for i := 0; i < doorCloseDelay+doorSlideTicks; i++ {
		s.animate()
	}
	if s.door.open != 0 {
		t.Errorf("la puerta sigue abierta (%.2f) cuando ya entraron todos", s.door.open)
	}
}

// La puerta principal se abre y se cierra suave (pedido de Samuel,
// 2026-10-11): las hojas se corren en 12 ticks, y se cierra 0,5 s después
// de que no haya nadie cerca. Si llega alguien mientras se cierra, se abre
// otra vez.
func TestMainDoor_SlidesOpenAndClosesHalfASecondLater(t *testing.T) {
	var d mainDoor
	if d.open != 0 {
		t.Fatalf("al empezar la puerta está abierta %.2f", d.open)
	}
	for tick := 1; tick <= doorSlideTicks; tick++ {
		d.step(true)
		if want := float64(tick) / doorSlideTicks; math.Abs(d.open-want) > 1e-9 {
			t.Fatalf("tick %d abriendo: %.3f; se esperaba %.3f", tick, d.open, want)
		}
	}
	for tick := 1; tick < doorCloseDelay; tick++ {
		d.step(false)
		if d.open != 1 {
			t.Fatalf("%d ticks sin nadie: la puerta ya se está cerrando (%.2f); se esperaba abierta medio segundo", tick, d.open)
		}
	}
	d.step(false)
	if want := 1 - 1.0/doorSlideTicks; math.Abs(d.open-want) > 1e-9 {
		t.Fatalf("pasado el medio segundo: %.3f; se esperaba que empezara a cerrarse (%.3f)", d.open, want)
	}
	d.step(true) // vuelve alguien
	if d.open != 1 {
		t.Errorf("volvió alguien mientras se cerraba: %.3f; se esperaba que se abriera otra vez", d.open)
	}
	for tick := 0; tick < doorCloseDelay+doorSlideTicks; tick++ {
		d.step(false)
	}
	if d.open != 0 {
		t.Errorf("sin nadie cerca, la puerta quedó abierta %.2f", d.open)
	}
}
