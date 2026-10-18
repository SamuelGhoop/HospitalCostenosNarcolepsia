package ui

import (
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Pedido de Samuel (2026-10-17): "evita que los personajes en general se
// puedan poner en el mismo lugar al mismo tiempo". Este test revisa, tick a
// tick, que ningún par de cuerpos (ni en la demo ni en una partida con
// varios empleados) quede encimado más de unos pocos píxeles.
//
// Exclusiones legítimas (son un solo grupo o no cuentan):
//   - quien empuja la camilla y el paciente que va sobre ella;
//   - un paciente acostado en su cama (el que lo revisa se le acerca).
//
// bodiesClash y visibleBody (world.go, spots.go) son las mismas que usa la
// reserva de celdas al caminar: el test y la implementación miran la misma
// forma.

// clashCounts lleva la cuenta de los solapes encontrados, por tipo, para el
// diagnóstico que pide Samuel ("cuántos solapes... y de qué tipo").
type clashCounts struct {
	total, bothWalking, atSameSpot, helperAtBed, staffFree, patientWander, sidewalk int
}

// classify adivina de qué tipo es el choque entre a y b, para el reporte:
// no cambia el veredicto (ya se decidió con clash), solo lo describe.
func classify(c *clashCounts, wa, wb *world, a, b figure) {
	c.total++
	atA, atB := wa.pos[a.id], wb.pos[b.id]
	stillA, stillB := atA == a.target, atB == b.target
	switch {
	case !stillA && !stillB:
		c.bothWalking++
	case stillA && stillB:
		c.atSameSpot++
	}
	if (!a.patient && a.target == b.target) || (!b.patient && b.target == a.target) {
		c.helperAtBed++
	}
	if (!a.patient && stillA) || (!b.patient && stillB) {
		c.staffFree++
	}
	if (a.patient && stillA && a.room == 0) || (b.patient && stillB && b.room == 0) {
		c.patientWander++
	}
	if atA.x > ScreenWidth-120 || atB.x > ScreenWidth-120 {
		c.sidewalk++
	}
}

// checkNoClashes revisa el tick actual de w: ningún par de cuerpos (menos
// las excepciones legítimas) se encima más de lo tolerado. Cuenta lo que
// encuentra en c (nil: no cuenta, solo falla si corresponde).
func checkNoClashes(t *testing.T, w *world, label string, c *clashCounts) {
	t.Helper()
	type body struct {
		f figure
		r image.Rectangle
	}
	var bodies []body
	for _, f := range w.figures {
		at := w.pos[f.id]
		if f.patient && f.room != 0 && at == bedCell(roomZone(f.room)) {
			continue // acostado en su cama: afuera del chequeo
		}
		bodies = append(bodies, body{f, visibleBody(f, at)})
	}
	for i, a := range bodies {
		for _, b := range bodies[i+1:] {
			if cc := w.carryOf(a.f.id); cc != nil && (b.f.id == cc.order.pusher || b.f.id == cc.order.patient) {
				continue // el grupo de la camilla
			}
			if bodiesClash(a.r, b.r) {
				if c != nil {
					classify(c, w, w, a.f, b.f)
				} else {
					t.Errorf("%s: %s y %s se encimaron (%v, %v)", label, a.f.id, b.f.id, a.r, b.r)
				}
			}
		}
	}
}

// TestNobodySharesABody_Demo recorre la demo completa, tick a tick.
func TestNobodySharesABody_Demo(t *testing.T) {
	needFonts(t)
	s := NewDemoScene(game.NewDemo())
	for step := 1; step <= game.DemoSteps; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 8000; tick++ {
			s.animate()
			checkNoClashes(t, &s.world, "demo", nil)
			if s.choreo == nil && s.everyoneArrived() {
				break
			}
		}
	}
}

// TestNobodySharesABody_Game juega varios miles de ticks despachando a
// quien se pueda, con el personal por defecto, y revisa que nadie quede
// encimado con otro.
func TestNobodySharesABody_Game(t *testing.T) {
	s := newTestGameScene(t)
	for tick := 0; tick < 6000; tick++ {
		if tick%60 == 0 {
			for _, p := range s.snap.Patients {
				if !needsSomeone(p.Stage) {
					continue
				}
				for _, st := range s.snap.Staff {
					if st.Activity == game.Free && s.g.Dispatch(p.ID, st.ID) == nil {
						break
					}
				}
			}
		}
		play(s, 1)
		checkNoClashes(t, &s.world, "juego", nil)
	}
}
