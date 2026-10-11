package ui

import (
	"image"
	"maps"
	"math"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Pedido por Samuel (2026-10-11): en toda la demo, tick a tick, nadie
// atraviesa paredes ni muebles. Lo que pisa el suelo no los toca:
//   - de pie o caminando, los pies (la huella feetBox);
//   - acostado en el piso, lo que se ve del cuerpo;
//   - la camilla, su huella (con el paciente encima).
//
// No cuentan el que está en su cama (la cama es un mueble, pero ahí sí va)
// ni el paciente mientras pasa del piso a la camilla o de la camilla a la
// cama. Y los que están quietos no quedan uno encima del otro.
//
// Los muebles son los de la maqueta (obstacles) y las paredes, lo que no
// es piso: la cuadrícula de las rutas se arma con ellos, pero el test los
// mira directo, así que si a la cuadrícula le faltara un mueble, el test
// lo nota.
func TestDemo_NobodyWalksThroughWallsOrFurniture(t *testing.T) {
	needFonts(t)
	s := NewDemoScene(game.NewDemo())
	for step := 1; step <= game.DemoSteps; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 8000; tick++ {
			s.animate()
			checkTick(t, s, step, tick)
			if s.choreo == nil && s.everyoneArrived() {
				break
			}
		}
	}
}

// onFurnitureBut dice si r toca un mueble de la maqueta que no sea except.
func onFurnitureBut(r, except image.Rectangle) bool {
	for _, o := range obstacles {
		if o != except && r.Overlaps(o) {
			return true
		}
	}
	return false
}

// touchesWallOrFurniture dice si r toca una pared (lo que no es piso) o un
// mueble de la maqueta.
func touchesWallOrFurniture(r image.Rectangle) bool {
	navMaps()
	return count(wallSum, r) > 0 || onFurniture(r)
}

// everyoneArrived dice si todos los personajes ya llegaron a su destino.
func (s *DemoScene) everyoneArrived() bool {
	for _, f := range s.figures {
		if s.pos[f.id] != s.destination(f) {
			return false
		}
	}
	return true
}

func checkTick(t *testing.T, s *DemoScene, step, tick int) {
	t.Helper()
	if s.carry != nil && s.carry.visible() && touchesWallOrFurniture(s.carry.fp) {
		t.Fatalf("paso %d, tick %d: la camilla (%v) toca una pared o un mueble", step, tick, s.carry.fp)
	}
	var still []figure
	for _, f := range s.figures {
		at := s.pos[f.id]
		p := image.Pt(int(math.Round(at.x)), int(math.Round(at.y)))
		hopping := s.inCarry(f.id) && f.id == s.carry.order.patient && s.carry.stage >= carryHopOn && s.carry.stage <= carryHopOff
		a := s.anims[f.id]
		switch {
		case f.patient && s.onBed(f), hopping:
			// en su cama, o pasando del piso a la camilla y de la camilla a la cama
		case a != nil && a.state == hospital.Awake && a.room != 0 && feetBox.Add(p.Add(feetOffsetPt)).Overlaps(bedObstacle(roomZone(a.room))):
			// se levantó de su cama y se baja: sus pies pueden ir sobre ella, no sobre nada más
			if feet := feetBox.Add(p.Add(feetOffsetPt)); count(wallSum, feet) > 0 || onFurnitureBut(feet, bedObstacle(roomZone(a.room))) {
				t.Fatalf("paso %d, tick %d: al bajarse de la cama, los pies de %s (%v) tocan una pared u otro mueble", step, tick, f.id, feet)
			}
		case a != nil && (a.showsBubble() || a.anim == animCollapse):
			if body := floorBody.Add(p); touchesWallOrFurniture(body) {
				t.Fatalf("paso %d, tick %d: %s está acostado sobre una pared o un mueble (%v)", step, tick, f.id, body)
			}
		default:
			if feet := feetBox.Add(p.Add(feetOffsetPt)); touchesWallOrFurniture(feet) {
				t.Fatalf("paso %d, tick %d: los pies de %s (%v) tocan una pared o un mueble", step, tick, f.id, feet)
			}
		}
		if at == s.destination(f) && !hopping && p.X < ScreenWidth { // fuera de la pantalla esperan los que van a llegar
			still = append(still, f)
		}
	}
	for i, a := range still {
		for _, b := range still[i+1:] {
			if figureBody(a, s.pos[a.id]).Overlaps(figureBody(b, s.pos[b.id])) {
				t.Fatalf("paso %d, tick %d: %s y %s están quietos uno encima del otro (%v, %v)",
					step, tick, a.id, b.id, s.pos[a.id], s.pos[b.id])
			}
		}
	}
}

// Pedido por Samuel (2026-10-11): nada de teletransporte. En toda la demo,
// tick a tick, nadie se mueve en un tick más que su velocidad, más un píxel
// de redondeo (la camilla y el paciente encima van al píxel). La excepción
// es aparecer: los pacientes nuevos aparecen fuera de la pantalla. Al
// paciente que llevan en camilla lo mueve quien la empuja: va a la
// velocidad del personal.
//
// Cuando la camilla gira (de a poco, en turnTicks ticks; decisión de Samuel
// del 2026-10-11), el paciente encima va a la velocidad del giro: recorre
// lo mismo en cada tick, lo que sale de repartir el arco en esos ticks.
func TestDemo_NobodyJumps(t *testing.T) {
	needFonts(t)
	s := NewDemoScene(game.NewDemo())
	const rounding = 1.0
	before := maps.Clone(s.pos)
	for step := 1; step <= game.DemoSteps; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		for _, f := range s.figures {
			if _, seen := before[f.id]; !seen {
				if f.patient && s.pos[f.id].x < ScreenWidth {
					t.Errorf("paso %d: %s aparece dentro de la pantalla, en %v", step, f.id, s.pos[f.id])
				}
				before[f.id] = s.pos[f.id]
			}
		}
		for tick := 0; tick < 8000; tick++ {
			var turning *swing // el giro de este tick, si lo hay (puede terminar en este tick)
			if s.carry != nil {
				turning = s.carry.swing
			}
			s.animate()
			c := s.carry
			if c != nil && c.swing != nil {
				turning = c.swing
			}
			for _, f := range s.figures {
				was, now := before[f.id], s.pos[f.id]
				limit := speedOf(f)
				if c != nil && c.visible() && f.id == c.order.patient {
					limit = staffSpeed
					if turning != nil {
						limit = math.Max(limit, turning.riderStep())
					}
				}
				if d := math.Hypot(now.x-was.x, now.y-was.y); d > limit+rounding {
					t.Errorf("paso %d, tick %d: %s se movió %.1f px en un tick (de %v a %v); su velocidad es %.1f", step, tick, f.id, d, was, now, limit)
				}
			}
			before = maps.Clone(s.pos)
			if s.choreo == nil && s.everyoneArrived() {
				break
			}
		}
	}
}
