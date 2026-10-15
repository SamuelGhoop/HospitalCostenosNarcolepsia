package ui

import (
	"fmt"
	"image"
	"math"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// La huella de los pies son las piernas en las 3 filas de abajo del cuadro
// (x de 3 a 13 de la hoja), alrededor del punto de los pies.
func TestFeetBox_IsTheLegsOnTheLastRows(t *testing.T) {
	if feetBox.Dx() != 10*spriteScale || feetBox.Dy() != 3*spriteScale {
		t.Errorf("la huella mide %v; se esperaba %d×%d", feetBox.Size(), 10*spriteScale, 3*spriteScale)
	}
	if feetBox.Max.Y != 1 || feetBox.Min.X+feetBox.Max.X != 0 {
		t.Errorf("la huella %v no está centrada sobre el punto de los pies (que es su última fila)", feetBox)
	}
}

// Los puestos de pie solo van donde se llega caminando: ninguno queda en
// un rincón encerrado por los muebles.
func TestSpots_AreReachableFromTheMainDoor(t *testing.T) {
	needFonts(t)
	for _, z := range zoneOrder {
		for _, p := range spotsOf(z).standing {
			if !standable(p.Add(feetOffsetPt)) {
				t.Errorf("%s: al puesto de pie %v no se llega caminando desde la puerta principal", zones[z].label, p)
			}
		}
		for _, f := range spotsOf(z).floor {
			if !standable(f.patient.Add(feetOffsetPt)) || !standable(f.helper.Add(feetOffsetPt)) {
				t.Errorf("%s: al puesto de piso %v (o a su ayudante %v) no se llega caminando", zones[z].label, f.patient, f.helper)
			}
		}
	}
}

// La cuadrícula ve las paredes y los muebles: el centro de una mesa de la
// cafetería y una pared entre dos habitaciones están bloqueados; el centro
// del pasillo 1 y el hueco de la puerta principal, no.
func TestNavGrid_BlocksWallsAndFurniture(t *testing.T) {
	table := sprite(18, 136, 16, 10)
	wall := image.Pt(zones[zoneRoom101].rect.Max.X+2, zones[zoneRoom101].rect.Min.Y+40)
	h1 := zones[zoneHallway1].rect
	door := doorways[len(doorways)-1].gap
	for _, c := range []struct {
		what    string
		feet    image.Point
		blocked bool
	}{
		{"la mesa de la cafetería", table.Min.Add(table.Max).Div(2), true},
		{"la pared entre la 101 y la 102", wall, true},
		{"el centro del pasillo 1", h1.Min.Add(h1.Max).Div(2), false},
		{"el hueco de la puerta principal", image.Pt(door.Min.X+door.Dx()/2, door.Max.Y-1), false},
		{"la calle", image.Pt(400, streetRect.Min.Y+20), true},
		{"la cama de la 101", bedRect(zoneRoom101).Min.Add(bedRect(zoneRoom101).Max).Div(2), true},
		// La ronda de mapa del 2026-10-11: acera más ancha, un carril y la playa;
		// el mostrador de la cafetería más corto y el de la recepción horizontal.
		{"la parte nueva de la acera", fromMockup(100, 212, 0, 0).Min, false},
		{"el carril", fromMockup(100, 230, 0, 0).Min, true},
		{"la playa", fromMockup(100, 252, 0, 0).Min, true},
		{"donde estaba la bandeja del mostrador de la cafetería", fromMockup(78, 109, 0, 0).Min, false},
		{"el mostrador nuevo de la recepción", fromMockup(272, 115, 0, 0).Min, true},
		{"donde estaba el mostrador vertical de la recepción", fromMockup(274, 158, 0, 0).Min, false},
		{"el paradero de la acera", sprite(120, 184, 38, 13).Min.Add(sprite(120, 184, 38, 13).Max).Div(2), true},
		{"la base de una palma", sprite(292.5, 186, 10, 6).Min.Add(sprite(292.5, 186, 10, 6).Max).Div(2), true},
		// Desde el 2026-10-11 el paradero y las palmas están 4 y 5,5 px más arriba:
		// debajo, donde llegaban antes, ya es acera libre.
		{"debajo del paradero (donde llegaba antes)", fromMockup(139, 209, 0, 0).Min, false},
		{"debajo de la base de una palma (donde llegaba antes)", fromMockup(297.5, 201, 0, 0).Min, false},
	} {
		if got := feetBlocked(c.feet); got != c.blocked {
			t.Errorf("%s (%v): bloqueado = %v; se esperaba %v", c.what, c.feet, got, c.blocked)
		}
	}
}

// needFonts carga las fuentes: los puestos dependen de los rótulos.
func needFonts(t *testing.T) {
	t.Helper()
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
}

// checkRoute revisa una ruta (en esquinas de cuadro): termina en el destino
// y en cada tramo, píxel a píxel, la huella de los pies no toca una pared ni
// un mueble.
func checkRoute(t *testing.T, what string, from, to vec, avoid ...image.Rectangle) []vec {
	t.Helper()
	path := routeAvoiding(from, to, avoid)
	if len(path) == 0 || path[len(path)-1] != to {
		t.Fatalf("%s: la ruta %v no termina en %v", what, path, to)
	}
	prev := from
	for _, p := range path {
		if !segmentClear(prev.add(feetOffset), p.add(feetOffset), avoid) {
			t.Errorf("%s: en el tramo %v → %v los pies tocan una pared o un mueble", what, prev, p)
		}
		prev = p
	}
	return path
}

// Todas las zonas quedan conectadas entre sí y con la acera, y ningún tramo
// toca paredes ni muebles. Se usa el piso alcanzable más cercano al centro
// de cada zona (en radiología, el centro es el escáner y detrás queda un
// rincón al que no se llega).
func TestRoute_ConnectsEveryPairOfZones(t *testing.T) {
	needFonts(t)
	spots := map[string]vec{"acera": arrivalSpawn()}
	for _, z := range zoneOrder {
		feet, ok := reachableFeetNear(zones[z].rect.Min.Add(zones[z].rect.Max).Div(2))
		if !ok {
			t.Fatalf("%s: no hay piso libre cerca del centro", zones[z].label)
		}
		spots[zones[z].label] = toVec(feet).add(vec{-feetOffset.x, -feetOffset.y})
	}
	for a, from := range spots {
		for b, to := range spots {
			checkRoute(t, fmt.Sprintf("%s → %s", a, b), from, to)
		}
	}
}

// De la 101 a la 102 se sale al pasillo 1 (no se atraviesa la pared).
func TestRoute_FromRoom101ToRoom102GoesThroughHallway1(t *testing.T) {
	needFonts(t)
	path := checkRoute(t, "101 → 102", toVec(slot(zoneRoom101, 0)), toVec(slot(zoneRoom102, 0)))
	for _, p := range path {
		if z, ok := zoneAt(p.add(feetOffset)); ok && z == zoneHallway1 {
			return
		}
	}
	t.Errorf("la ruta de la 101 a la 102 (%v) no pasa por el pasillo 1", path)
}

// El suavizado quita la escalera: la ruta tiene muchos menos puntos que
// celdas recorrió el A*, y sigue sin tocar nada.
func TestRoute_SmoothingRemovesTheStairs(t *testing.T) {
	needFonts(t)
	from, to := toVec(slot(zoneLobby, 0)), toVec(slot(zoneStaffRoom, 0))
	sc, _ := nearestFreeCell(feetCellOf(from.add(feetOffset)), nil)
	gc, _ := nearestFreeCell(feetCellOf(to.add(feetOffset)), nil)
	cells := astar(sc, gc, nil)
	path := checkRoute(t, "recepción → sala del personal", from, to)
	if len(cells) < 20 || len(path) > len(cells)/4 {
		t.Errorf("el A* recorrió %d celdas y la ruta suavizada tiene %d puntos; se esperaba menos de la cuarta parte", len(cells), len(path))
	}
}

// Alrededor de un mueble: en la cafetería, de un lado al otro de una mesa,
// la línea recta la atraviesa y la ruta la rodea.
func TestRoute_GoesAroundFurniture(t *testing.T) {
	table := sprite(18, 136, 16, 10)
	y := table.Min.Y + table.Dy()/2
	a, okA := freeFeetNear(image.Pt(table.Min.X-30, y))
	b, okB := freeFeetNear(image.Pt(table.Max.X+30, y))
	if !okA || !okB {
		t.Fatal("no hay piso libre a los lados de la mesa")
	}
	from, to := toVec(a).add(vec{-feetOffset.x, -feetOffset.y}), toVec(b).add(vec{-feetOffset.x, -feetOffset.y})
	if segmentClear(toVec(a), toVec(b), nil) {
		t.Fatal("la línea recta no toca la mesa: el test no prueba nada")
	}
	checkRoute(t, "alrededor de la mesa", from, to)
}

// Una camilla en medio del pasillo 1 es un obstáculo: la ruta de los demás
// la rodea.
func TestRoute_AvoidsAStretcher(t *testing.T) {
	h1 := zones[zoneHallway1].rect
	stretcher := image.Rect(600, h1.Max.Y-30, 660, h1.Max.Y-6)
	a, _ := freeFeetNear(image.Pt(400, h1.Max.Y-12))
	b, _ := freeFeetNear(image.Pt(860, h1.Max.Y-12))
	from, to := toVec(a).add(vec{-feetOffset.x, -feetOffset.y}), toVec(b).add(vec{-feetOffset.x, -feetOffset.y})
	if segmentClear(toVec(a), toVec(b), []image.Rectangle{stretcher}) {
		t.Fatal("la línea recta no toca la camilla: el test no prueba nada")
	}
	checkRoute(t, "por el pasillo 1 con una camilla", from, to, stretcher)
}

// Desde fuera de la pantalla, por la derecha, por la franja baja de la
// acera (el cuerpo no se monta sobre la fachada), hasta la recepción por la
// puerta principal.
func TestRoute_FromOffscreenRightEntersThroughTheMainDoor(t *testing.T) {
	needFonts(t)
	from := arrivalSpawn()
	if from.x < ScreenWidth {
		t.Fatalf("el punto de salida %v no está fuera de la pantalla", from)
	}
	path := checkRoute(t, "acera → recepción", from, toVec(slot(zoneLobby, 0)))
	door := doorways[len(doorways)-1].gap
	crossed := false
	prev := from
	for _, p := range path {
		a, b := prev.add(feetOffset), p.add(feetOffset)
		if (a.y >= float64(door.Max.Y)) != (b.y >= float64(door.Max.Y)) { // cruza la línea de la fachada
			x := a.x + (b.x-a.x)*(float64(door.Max.Y)-a.y)/(b.y-a.y)
			crossed = crossed || (x >= float64(door.Min.X) && x < float64(door.Max.X))
		}
		// A lo largo del tramo, cada 4 px: en la acera, lejos de la puerta, el
		// cuerpo no se monta sobre la fachada.
		steps := int(math.Ceil(math.Hypot(b.x-a.x, b.y-a.y) / 4))
		for k := 0; k <= steps; k++ {
			tt := 1.0
			if steps > 0 {
				tt = float64(k) / float64(steps)
			}
			f := vec{a.x + (b.x-a.x)*tt, a.y + (b.y-a.y)*tt}
			if f.y < float64(buildingRect.Max.Y) || (f.x >= float64(door.Min.X)-figWidth && f.x <= float64(door.Max.X)+figWidth) {
				continue
			}
			corner := image.Pt(int(f.x-feetOffset.x), int(f.y-feetOffset.y))
			if body := standingBody.Add(corner); body.Min.Y < buildingRect.Max.Y {
				t.Fatalf("en la acera, lejos de la puerta, el cuerpo (%v) se monta sobre la fachada", body)
			}
		}
		prev = p
	}
	if !crossed {
		t.Errorf("la ruta %v no cruza la fachada por el hueco de la puerta principal %v", path, door)
	}
}

// Una ruta de un punto a sí mismo no se mueve, aunque el punto esté sobre
// un mueble (el paciente acostado en su cama: ahí no hay piso libre y el
// A* arrancaría en la celda libre más cercana, sacándolo de la cama).
func TestRoute_ToItselfStaysPut(t *testing.T) {
	bed := bedCell(zoneRoom101)
	if path := route(bed, bed); len(path) != 1 || path[0] != bed {
		t.Errorf("la ruta de la cama a la cama es %v; se esperaba quedarse en %v", path, bed)
	}
}

// Las puertas del código son los huecos del mapa (Samuel ensanchó el PNG el
// 2026-10-11): en la línea del medio de la pared, el hueco del PNG (lo que
// no es pared, ni el contorno negro ni el gris del muro) va de un borde al
// otro de la puerta, con 1 px de redondeo. Menos la puerta principal, que
// en el mapa es un tapete.
func TestDoorways_MatchTheMapImage(t *testing.T) {
	img, err := decodePNG(assets.Map, backgroundPath)
	if err != nil {
		t.Fatal(err)
	}
	wall := func(p image.Point) bool {
		r, g, b, _ := img.At(p.X, p.Y).RGBA()
		c := [3]uint32{r >> 8, g >> 8, b >> 8}
		return c == [3]uint32{20, 20, 20} || c == [3]uint32{185, 191, 200}
	}
	for i, d := range doorways[:len(doorways)-1] {
		g := d.gap
		step, along := image.Pt(1, 0), func(p image.Point) int { return p.X }
		lo, hi := g.Min.X, g.Max.X
		if g.Dy() > g.Dx() { // puerta en una pared vertical
			step, along = image.Pt(0, 1), func(p image.Point) int { return p.Y }
			lo, hi = g.Min.Y, g.Max.Y
		}
		c := g.Min.Add(g.Max).Div(2)
		first, last := c, c
		for n := 0; n < 200 && !wall(first.Sub(step)); n++ {
			first = first.Sub(step)
		}
		for n := 0; n < 200 && !wall(last.Add(step)); n++ {
			last = last.Add(step)
		}
		if a, b := along(first), along(last)+1; abs(a-lo) > 1 || abs(b-hi) > 1 {
			t.Errorf("puerta %d: en el mapa el hueco va de %d a %d; en el código, de %d a %d", i, a, b, lo, hi)
		}
	}
}

// Con las puertas anchas (2026-10-11), quien va caminando de la cafetería a
// la 101 sube por la puerta de arriba de la cafetería: no da la vuelta por
// la recepción. (La camilla no cabe por ahí: ver GAME_DESIGN §10.8.)
func TestRoute_FromTheCafeteriaToRoom101GoesUpThroughTheCafeteriaDoor(t *testing.T) {
	needFonts(t)
	cafeDoor := doorways[4].gap // pasillo 1 ↔ cafetería
	froms := []vec{toVec(floorSpot(zoneCafeteria, 0)), toVec(slot(zoneCafeteria, 0)), toVec(slot(zoneCafeteria, 1))}
	tos := []vec{toVec(carrySpot(zoneRoom101)), toVec(wakeSpot(zoneRoom101))}
	for _, from := range froms {
		for _, to := range tos {
			through, lobby := false, false
			prev := from
			for _, p := range route(from, to) {
				steps := int(math.Ceil(math.Hypot(p.x-prev.x, p.y-prev.y) / 2))
				for k := 0; k <= steps; k++ {
					f := 1.0
					if steps > 0 {
						f = float64(k) / float64(steps)
					}
					feet := image.Pt(int(math.Round(prev.x+(p.x-prev.x)*f+feetOffset.x)), int(math.Round(prev.y+(p.y-prev.y)*f+feetOffset.y)))
					through = through || feet.In(cafeDoor)
					lobby = lobby || feet.In(zones[zoneLobby].rect)
				}
				prev = p
			}
			if !through || lobby {
				t.Errorf("de %v a %v: pasa por la puerta de arriba de la cafetería=%v, por la recepción=%v; se esperaba (true, false)", from, to, through, lobby)
			}
		}
	}
}

// La puerta nueva entre la recepción y el pasillo 2 (2026-10-11): quien va
// de la recepción al pasillo 2 pasa por ella, sin subir al pasillo 1.
func TestRoute_FromTheLobbyToHallway2UsesTheSideDoor(t *testing.T) {
	needFonts(t)
	side := doorways[len(doorways)-2].gap // recepción ↔ pasillo 2
	from := toVec(slot(zoneLobby, 0))
	to := toVec(slot(zoneHallway2, 0))
	through, hallway1 := false, false
	prev := from
	for _, p := range route(from, to) {
		steps := int(math.Ceil(math.Hypot(p.x-prev.x, p.y-prev.y) / 2))
		for k := 0; k <= steps; k++ {
			f := 1.0
			if steps > 0 {
				f = float64(k) / float64(steps)
			}
			feet := image.Pt(int(math.Round(prev.x+(p.x-prev.x)*f+feetOffset.x)), int(math.Round(prev.y+(p.y-prev.y)*f+feetOffset.y)))
			through = through || feet.In(side)
			hallway1 = hallway1 || feet.In(zones[zoneHallway1].rect)
		}
		prev = p
	}
	if !through || hallway1 {
		t.Errorf("de la recepción al pasillo 2: pasa por la puerta del costado=%v, por el pasillo 1=%v; se esperaba (true, false)", through, hallway1)
	}
}

// Si la camilla de otro tapa el único paso, la ruta no se va en línea recta
// por las paredes: pasa como si la camilla no estuviera (en el juego puede
// haber varias camillas a la vez).
func TestRoute_WhenAStretcherBlocksTheWayItIgnoresIt(t *testing.T) {
	needFonts(t)
	door := doorways[0].gap // la puerta de la 101: la camilla la tapa entera
	block := image.Rect(door.Min.X-40, door.Min.Y-40, door.Max.X+40, door.Max.Y+60)
	from, to := toVec(wakeSpot(zoneRoom101)), toVec(slot(zoneStaffRoom, 0))
	path := routeAvoiding(from, to, []image.Rectangle{block})
	if len(path) < 3 {
		t.Fatalf("con la puerta tapada la ruta es %v: en línea recta por las paredes", path)
	}
	if want := route(from, to); len(path) != len(want) {
		t.Errorf("con la puerta tapada la ruta es %v; se esperaba la de sin camilla %v", path, want)
	}
}
