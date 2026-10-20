package ui

import (
	"fmt"
	"image"
	"math/rand"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Pedido de Samuel (2026-10-17): "evita que los personajes en general se
// puedan poner en el mismo lugar al mismo tiempo"; afinado el 2026-10-19:
// quien camina no choca con nadie (pasa de largo, ver nextWaypoint en
// world.go), pero quien está QUIETO sí tiene cuerpo sólido. Este test
// revisa, tick a tick, que ningún par de cuerpos quietos (ni en la demo ni
// en una partida con varios empleados) quede encimado más de unos pocos
// píxeles. Quieto = no se movió en este tick.
//
// Exclusiones legítimas (son un solo grupo o no cuentan):
//   - quien empuja la camilla y el paciente que va sobre ella;
//   - un paciente acostado en su cama (el que lo revisa se le acerca).
//
// bodiesClash y visibleBody están en world.go y spots.go.

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

// checkNoClashes revisa el tick actual de w: ningún par de cuerpos quietos
// (los que no se movieron desde before; menos las excepciones legítimas) se
// encima más de lo tolerado. Cuenta lo que encuentra en c (nil: no cuenta,
// solo falla si corresponde).
func checkNoClashes(t *testing.T, w *world, before map[string]vec, label string, c *clashCounts) {
	t.Helper()
	type body struct {
		f figure
		r image.Rectangle
	}
	var bodies []body
	for _, f := range w.figures {
		at := w.pos[f.id]
		if prev, ok := before[f.id]; !ok || prev != at {
			continue // caminando (o recién aparecido): pasa de largo, no choca
		}
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

// TestNobodyStandsOnAnother_Demo recorre la demo completa, tick a tick.
func TestNobodyStandsOnAnother_Demo(t *testing.T) {
	needFonts(t)
	s := NewDemoScene(game.NewDemo())
	for step := 1; step <= game.DemoSteps; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 8000; tick++ {
			before := positions(&s.world)
			s.animate()
			checkNoClashes(t, &s.world, before, "demo", nil)
			if s.choreo == nil && s.everyoneArrived() {
				break
			}
		}
	}
}

// TestNobodyStandsOnAnother_Game juega varios miles de ticks despachando a
// quien se pueda, con el personal por defecto, y revisa que nadie quede
// quieto encima de otro.
func TestNobodyStandsOnAnother_Game(t *testing.T) {
	playChecking(t, newTestGameScene(t), 6000)
}

// TestNobodyStandsOnAnother_GameSeeds: semillas que encontraron casos al
// terminar la ruta (2026-10-19): varios que salen por la misma puerta y se
// quedaban en el mismo punto de la acera (semilla 1) y alguien que se había
// hecho a un lado de una camilla justo donde quien la empuja se queda quieto
// al llegar a la cama (semillas 5 y 6).
func TestNobodyStandsOnAnother_GameSeeds(t *testing.T) {
	for _, seed := range []int64{1, 5, 6} {
		seed := seed
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			needFonts(t)
			old := seedSource
			seedSource = func() int64 { return seed }
			t.Cleanup(func() { seedSource = old })
			g, err := game.New(rand.New(rand.NewSource(seed)), rand.New(rand.NewSource(seed+1)), rand.New(rand.NewSource(seed+2)))
			if err != nil {
				t.Fatal(err)
			}
			playChecking(t, newGameScene(g), 20000)
		})
	}
}

// playChecking juega ticks ticks despachando a quien se pueda (cada
// segundo, al primero libre) y revisa en cada uno que nadie quede quieto
// encima de otro.
func playChecking(t *testing.T, s *GameScene, ticks int) {
	t.Helper()
	for tick := 0; tick < ticks; tick++ {
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
		before := positions(&s.world)
		play(s, 1)
		checkNoClashes(t, &s.world, before, fmt.Sprintf("juego, tick %d", tick), nil)
		if t.Failed() {
			return
		}
	}
}

// positions copia dónde está cada uno ahora (para saber, después del tick,
// quién se quedó quieto).
func positions(w *world) map[string]vec {
	out := map[string]vec{}
	for id, p := range w.pos {
		out[id] = p
	}
	return out
}

// TestWalkers_PassThroughEachOtherWithoutWaiting: dos que caminan en
// sentidos opuestos por el mismo pasillo no se esperan ni se traban (pedido
// de Samuel, 2026-10-19: antes, con el camino tapado, muchas veces no había
// otra ruta y se quedaban quietos para siempre). Cada tick los dos avanzan.
func TestWalkers_PassThroughEachOtherWithoutWaiting(t *testing.T) {
	w := newWorld()
	w.figures = []figure{{id: "D-01", doctor: true}, {id: "D-02", doctor: true}}
	h := zones[zoneHallway1].rect
	y := float64(h.Min.Y+h.Max.Y)/2 - feetOffset.y
	left, right := vec{float64(h.Min.X) + 60, y}, vec{float64(h.Max.X) - 120, y}
	w.pos["D-01"], w.pos["D-02"] = left, right
	for tick := 0; tick < 4000 && (w.pos["D-01"] != right || w.pos["D-02"] != left); tick++ {
		for _, f := range w.figures {
			dest := right
			if f.id == "D-02" {
				dest = left
			}
			old := w.pos[f.id]
			if old == dest {
				continue
			}
			w.pos[f.id] = moveToward(old, w.nextWaypoint(f.id, old, dest), 2)
			if w.pos[f.id] == old {
				t.Fatalf("tick %d: %s se quedó quieto en %v camino a %v (esperando al otro)", tick, f.id, old, dest)
			}
		}
	}
	if w.pos["D-01"] != right || w.pos["D-02"] != left {
		t.Errorf("no llegaron: D-01=%v (quería %v), D-02=%v (quería %v)", w.pos["D-01"], right, w.pos["D-02"], left)
	}
}

// TestWalker_DoesNotStopOnTopOfSomeoneStill: corrección de Samuel
// (2026-10-19): caminando nadie choca, pero al TERMINAR la ruta nadie queda
// encima de otro. D-02 camina hacia el puesto exacto donde ya está parado
// D-01: llega a un sitio libre cerca, no encima de él, y cuenta como llegado.
func TestWalker_DoesNotStopOnTopOfSomeoneStill(t *testing.T) {
	w := newWorld()
	d1, d2 := figure{id: "D-01", doctor: true}, figure{id: "D-02", doctor: true}
	w.figures = []figure{d1, d2}
	h := zones[zoneHallway1].rect
	y := float64(h.Min.Y+h.Max.Y)/2 - feetOffset.y
	spot := vec{float64(h.Min.X+h.Max.X) / 2, y}
	w.pos["D-01"], w.pos["D-02"] = spot, vec{float64(h.Min.X) + 60, y}
	for tick := 0; tick < 4000 && !w.arrived("D-02", spot); tick++ {
		old := w.pos["D-02"]
		w.pos["D-02"] = moveToward(old, w.nextWaypoint("D-02", old, spot), 2)
	}
	if !w.arrived("D-02", spot) {
		t.Fatalf("D-02 nunca terminó su ruta (está en %v)", w.pos["D-02"])
	}
	if bodiesClash(visibleBody(d1, w.pos["D-01"]), visibleBody(d2, w.pos["D-02"])) {
		t.Errorf("D-02 terminó encima de D-01: %v y %v", w.pos["D-02"], w.pos["D-01"])
	}
}
