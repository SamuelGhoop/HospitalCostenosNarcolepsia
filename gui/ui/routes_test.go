package ui

import (
	"fmt"
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Pedido por Samuel: ningún tramo entre dos puntos de paso conectados cruza
// una pared. Las paredes salen de los mismos rectángulos de la maqueta: lo
// caminable es el piso de las zonas, los huecos de las puertas y lo de
// afuera del edificio.
func TestRoutes_NoSegmentCrossesAWall(t *testing.T) {
	nodes := waypoints()
	for i, a := range nodes {
		for _, j := range neighbors(nodes, i) {
			if b := nodes[j]; !walkableSegment(a, b) {
				t.Errorf("el tramo %v → %v cruza una pared", a, b)
			}
		}
	}
}

// El detector de paredes funciona: el tramo directo de la 101 a la 102 (lo
// que hacía el camillero en la captura s11c) atraviesa la pared entre las dos.
func TestWalkableSegment_SeesTheWallBetweenTwoRooms(t *testing.T) {
	a := toVec(slot(zoneRoom101, 0)).add(feetOffset)
	b := toVec(slot(zoneRoom102, 0)).add(feetOffset)
	if walkableSegment(a, b) {
		t.Error("el tramo directo de la 101 a la 102 no debería ser caminable: hay una pared")
	}
}

// Una ruta (en coordenadas de cuadro) recorre tramos caminables por los pies
// y termina en el destino.
func checkRoute(t *testing.T, what string, from, to vec) []vec {
	t.Helper()
	path := route(from, to)
	if len(path) == 0 || path[len(path)-1] != to {
		t.Fatalf("%s: la ruta %v no termina en %v", what, path, to)
	}
	prev := from
	for _, p := range path {
		if !walkableSegment(prev.add(feetOffset), p.add(feetOffset)) {
			t.Errorf("%s: el tramo %v → %v cruza una pared", what, prev, p)
		}
		prev = p
	}
	return path
}

// El caso de la captura s11c: de la 101 a la 102 se sale al pasillo 1.
func TestRoute_FromRoom101ToRoom102GoesThroughHallway1(t *testing.T) {
	path := checkRoute(t, "101 → 102", toVec(slot(zoneRoom101, 0)), toVec(slot(zoneRoom102, 0)))
	for _, p := range path {
		if z, ok := zoneAt(p.add(feetOffset)); ok && z == zoneHallway1 {
			return
		}
	}
	t.Errorf("la ruta de la 101 a la 102 (%v) no pasa por el pasillo 1", path)
}

// De la acera a la sala del personal se entra por la puerta principal: la
// ruta pasa por el lado de afuera de la puerta y enseguida por el de adentro.
func TestRoute_FromTheSidewalkEntersThroughTheMainDoor(t *testing.T) {
	path := checkRoute(t, "acera → sala del personal", toVec(entrance), toVec(slot(zoneStaffRoom, 0)))
	main := doorways[len(doorways)-1] // la puerta principal es la última de la lista
	out, in := main.sides[1].add(vec{-feetOffset.x, -feetOffset.y}), main.sides[0].add(vec{-feetOffset.x, -feetOffset.y})
	for i := 0; i+1 < len(path); i++ {
		if path[i] == out && path[i+1] == in {
			return
		}
	}
	t.Errorf("la ruta desde la acera (%v) no cruza la puerta principal (%v → %v)", path, out, in)
}

// Todas las zonas quedan conectadas entre sí, y con la acera. Se usa el
// centro de cada zona (con los pies ahí), no un puesto de pie: en el
// pasillo 1 nadie se para.
func TestRoute_ConnectsEveryPairOfZones(t *testing.T) {
	spots := map[string]vec{"acera": toVec(entrance)}
	for _, z := range zoneOrder {
		c := zones[z].rect.Min.Add(zones[z].rect.Max).Div(2)
		spots[zones[z].label] = vec{float64(c.X) - feetOffset.x, float64(c.Y) - feetOffset.y}
	}
	for a, from := range spots {
		for b, to := range spots {
			checkRoute(t, fmt.Sprintf("%s → %s", a, b), from, to)
		}
	}
}

// En toda la demo, en cada tick: los pies de cada personaje están sobre algo
// caminable (nadie atraviesa una pared), y los que están quietos no quedan
// uno encima del otro ni sobre un mueble del fondo.
func TestDemo_NobodyWalksThroughWallsOrStandsOnSomeoneElse(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	s := NewDemoScene(game.NewDemo())
	for step := 1; step <= game.DemoSteps; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		for tick := 0; tick < 3000; tick++ {
			s.animate()
			checkTick(t, s, step, tick)
			if s.choreo == nil && s.everyoneArrived() {
				break
			}
		}
	}
}

// visibleBody es lo que se ve del cuerpo de un personaje en su puesto. ok
// es false para el que duerme en su cama: la cama es un mueble, pero ahí sí
// va.
func visibleBody(s *DemoScene, f figure) (image.Rectangle, bool) {
	at := s.pos[f.id]
	p := image.Pt(int(at.x), int(at.y))
	switch {
	case f.patient && s.onBed(f):
		return image.Rectangle{}, false
	case f.patient && f.state != hospital.Awake:
		return floorBody.Add(p), true // acostado en el piso
	default:
		return standingBody.Add(p), true
	}
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
	var still []figure
	for _, f := range s.figures {
		at := s.pos[f.id]
		if !walkable(at.add(feetOffset)) {
			t.Fatalf("paso %d, tick %d: %s tiene los pies en una pared, en %v", step, tick, f.id, at)
		}
		if at == s.destination(f) {
			still = append(still, f)
		}
	}
	for _, f := range still {
		if body, ok := visibleBody(s, f); ok && onFurniture(body) {
			t.Fatalf("paso %d, tick %d: %s está quieto sobre un mueble (%v)", step, tick, f.id, body)
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
