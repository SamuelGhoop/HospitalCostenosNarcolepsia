package ui

import (
	"image"
	"math"
)

// El traslado en camilla dentro de la escena: lo que manda la fase de la
// coreografía (carryOrder) hecho tick a tick, en cuatro etapas.
type carryStage int

const (
	carryToPickup carryStage = iota // quien empuja camina hasta donde arranca (cerca del paciente)
	carryHopOn                      // aparece la camilla y el paciente se pasa a ella
	carryPush                       // la empuja por los tramos de la ruta, con el paciente encima
	carryHopOff                     // el paciente pasa de la camilla a la cama
	carryDone                       // la camilla ya desapareció
)

// hopSpeed: lo que avanza por tick el paciente al pasar del piso a la
// camilla y de la camilla a la cama.
const hopSpeed = 2.0

// carrying es un traslado en curso.
type carrying struct {
	order *carryOrder
	stage carryStage
	seg   int             // tramo actual del empuje
	dir   dir             // hacia dónde apunta la camilla
	fp    image.Rectangle // huella de la camilla en el piso (vacía mientras no se ve; girando, la de después del giro)
	swing *swing          // el giro en curso (nil si no está girando)
}

// visible dice si la camilla está en el mapa.
func (c *carrying) visible() bool { return c.stage >= carryHopOn && c.stage < carryDone }

// riding dice si el paciente va sobre la camilla.
func (c *carrying) riding() bool { return c.stage == carryPush }

// footprintAt es la huella de la camilla delante de quien empuja, con su
// cuadro en corner.
func footprintAt(corner vec, d dir) image.Rectangle { return stretcherFootprint(feetPoint(corner), d) }

// feetPoint son los pies, en enteros, de alguien con el cuadro en corner.
func feetPoint(corner vec) image.Point {
	feet := corner.add(feetOffset)
	return image.Pt(int(math.Round(feet.x)), int(math.Round(feet.y)))
}

// stepCarry hace un tick del traslado: mueve a quien empuja y al paciente
// (los demás los mueve animate) y pasa de etapa cuando toca.
func (s *DemoScene) stepCarry() {
	c := s.carry
	o, plan := c.order, c.order.plan
	switch c.stage {
	case carryToPickup:
		at := s.pos[o.pusher]
		s.pos[o.pusher] = moveToward(at, s.nextWaypoint(o.pusher, at, plan.pickup), staffSpeed)
		if s.pos[o.pusher] == plan.pickup {
			c.stage, c.dir = carryHopOn, plan.dirs[0]
			c.fp = footprintAt(plan.pickup, c.dir) // aparece la camilla, delante de quien la va a empujar
		}
	case carryHopOn:
		to := riderPos(c.fp)
		s.pos[o.patient] = moveToward(s.pos[o.patient], to, hopSpeed)
		if s.pos[o.patient] == to {
			c.stage = carryPush
		}
	case carryPush:
		if w := c.swing; w != nil { // girando: quien empuja se queda quieto
			w.tick++
			s.pos[o.patient] = w.riderAt(w.progress())
			if w.done() {
				c.dir, c.swing = w.to, nil
			}
			return
		}
		goal := plan.path[c.seg]
		s.pos[o.pusher] = moveToward(s.pos[o.pusher], goal, staffSpeed)
		if s.pos[o.pusher] == goal {
			c.seg++
		}
		if c.seg == len(plan.path) {
			c.stage = carryHopOff
		} else if d := plan.dirs[c.seg]; d != c.dir {
			c.swing = newSwing(feetPoint(goal), c.dir, d) // gira en los próximos ticks
			c.fp = footprintAt(s.pos[o.pusher], d)
			s.pos[o.patient] = riderPos(footprintAt(s.pos[o.pusher], c.dir))
			return
		}
		c.fp = footprintAt(s.pos[o.pusher], c.dir)
		if c.stage == carryPush {
			s.pos[o.patient] = riderPos(c.fp)
		}
	case carryHopOff:
		bed := bedCell(o.room)
		s.pos[o.patient] = moveToward(s.pos[o.patient], bed, hopSpeed)
		if s.pos[o.patient] == bed {
			c.stage, c.fp = carryDone, image.Rectangle{} // la camilla desaparece
		}
	}
}

// inCarry dice si id es quien empuja o el paciente del traslado en curso.
func (s *DemoScene) inCarry(id string) bool {
	return s.carry != nil && (id == s.carry.order.pusher || id == s.carry.order.patient)
}

// avoidFor es lo que tiene que esquivar id al calcular su ruta: la camilla
// de otro (quien la empuja y el paciente no se la esquivan a sí mismos).
func (s *DemoScene) avoidFor(id string) []image.Rectangle {
	if s.carry == nil || !s.carry.visible() || s.inCarry(id) {
		return nil
	}
	return []image.Rectangle{s.carry.fp}
}

// ─── Hacerse a un lado ─────────────────────────────────────────────────

// carryLane es por dónde pasa un traslado: los pies de quien empuja y la
// huella de la camilla, cada 4 px de la ruta, y lo que barre al girar.
func carryLane(plan carryPlan) []image.Rectangle {
	var out []image.Rectangle
	prev := plan.pickup
	for i, p := range plan.path {
		if i > 0 && plan.dirs[i] != plan.dirs[i-1] { // gira donde terminó el tramo anterior
			w := swing{pivot: feetPoint(prev), from: plan.dirs[i-1], to: plan.dirs[i]}
			for k := 0; k <= swingSteps; k++ {
				out = append(out, w.bounds(float64(k)/swingSteps))
			}
		}
		steps := int(math.Ceil(math.Hypot(p.x-prev.x, p.y-prev.y) / 4))
		for k := 0; k <= steps; k++ {
			t := 1.0
			if steps > 0 {
				t = float64(k) / float64(steps)
			}
			at := vec{prev.x + (p.x-prev.x)*t, prev.y + (p.y-prev.y)*t}
			feet := at.add(feetOffset)
			out = append(out, footprintAt(at, plan.dirs[i]), feetBox.Add(image.Pt(int(math.Round(feet.x)), int(math.Round(feet.y)))))
		}
		prev = p
	}
	return out
}

func overlapsAny(r image.Rectangle, rs []image.Rectangle) bool {
	for _, x := range rs {
		if r.Overlaps(x) {
			return true
		}
	}
	return false
}

// dodgesFor: los que están quietos en el camino de la camilla se hacen a
// un lado mientras pasa (y después vuelven a su puesto): a cada uno se le
// busca el sitio libre más cercano fuera del camino, sin encimarse con
// nadie. Así la recepción, donde a ×3 no cabe un pasillo libre con los 5
// pacientes esperando, no se queda sin puestos.
func (s *DemoScene) dodgesFor(o *carryOrder) map[string]vec {
	lane := carryLane(o.plan)
	var bodies []image.Rectangle // dónde están (o van a estar) los demás
	for _, f := range s.figures {
		if f.id != o.pusher && f.id != o.patient {
			bodies = append(bodies, figureBodyAt(f, s.destination(f)))
		}
	}
	dodge := map[string]vec{}
	for _, f := range s.figures {
		at := s.destination(f)
		if f.id == o.pusher || f.id == o.patient || (f.patient && f.room != 0 && at == bedCell(roomZone(f.room))) {
			continue
		}
		if !overlapsAny(feetRectAt(at), lane) {
			continue
		}
		own := figureBodyAt(f, at)
		var others []image.Rectangle
		for _, b := range bodies {
			if b != own {
				others = append(others, b)
			}
		}
		if spot, ok := dodgeSpot(at, lane, others); ok {
			dodge[f.id] = spot
			bodies = append(bodies, figureBodyAt(f, spot))
		}
	}
	return dodge
}

// feetRectAt es la huella de los pies de alguien con el cuadro en at.
func feetRectAt(at vec) image.Rectangle {
	feet := at.add(feetOffset)
	return feetBox.Add(image.Pt(int(math.Round(feet.x)), int(math.Round(feet.y))))
}

// figureBodyAt es el cuadro de un personaje de pie con la esquina en at.
func figureBodyAt(f figure, at vec) image.Rectangle {
	x, y := int(math.Round(at.x)), int(math.Round(at.y))
	return image.Rect(x, y, x+figWidth, y+figHeight)
}

// dodgeSpot es el sitio libre más cercano a from (esquina del cuadro) donde
// alguien puede esperar de pie fuera del camino lane, sin encimarse con
// bodies: los pies en piso libre al que se llega caminando, el cuerpo sin
// tocar muebles y dentro de una zona.
func dodgeSpot(from vec, lane, bodies []image.Rectangle) (vec, bool) {
	k0 := feetCellOf(from.add(feetOffset))
	for radius := 1; radius <= 16; radius++ {
		best, bestD, found := vec{}, math.Inf(1), false
		for dc := -radius; dc <= radius; dc++ {
			for dr := -radius; dr <= radius; dr++ {
				k := cell{k0.c + dc, k0.r + dr}
				if max(abs(dc), abs(dr)) != radius || !reachableCell(k) {
					continue
				}
				corner := toVec(k.center()).add(vec{-feetOffset.x, -feetOffset.y})
				body := image.Rect(int(corner.x), int(corner.y), int(corner.x)+figWidth, int(corner.y)+figHeight)
				if z, ok := zoneAt(toVec(k.center())); !ok || z == outside {
					continue
				}
				if overlapsAny(feetBox.Add(k.center()).Inset(-4), lane) || overlapsAny(body, bodies) || onFurniture(standingBody.Add(body.Min)) {
					continue
				}
				if d := math.Hypot(float64(dc), float64(dr)); d < bestD {
					best, bestD, found = corner, d, true
				}
			}
		}
		if found {
			return best, true
		}
	}
	return vec{}, false
}
