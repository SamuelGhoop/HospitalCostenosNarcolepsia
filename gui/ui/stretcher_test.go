package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
)

// La dirección de la marcha es el eje en el que más se mueve.
func TestDirOf_DominantAxis(t *testing.T) {
	for _, c := range []struct {
		dx, dy float64
		want   dir
		ok     bool
	}{
		{3, 1, dirRight, true}, {-3, 1, dirLeft, true}, {1, -3, dirUp, true}, {0, 2, dirDown, true}, {0, 0, 0, false},
	} {
		if got, ok := dirOf(c.dx, c.dy); got != c.want || ok != c.ok {
			t.Errorf("dirOf(%v, %v) = (%v, %v); se esperaba (%v, %v)", c.dx, c.dy, got, ok, c.want, c.ok)
		}
	}
}

// La camilla va DELANTE de quien la empuja, sin tocar sus pies:
// horizontal (larga en x) yendo a izquierda o derecha y vertical yendo
// arriba o abajo.
func TestStretcherFootprint_InFrontOfThePusher(t *testing.T) {
	feet := image.Pt(500, 300)
	fb := feetBox.Add(feet)
	for _, d := range []dir{dirRight, dirLeft, dirUp, dirDown} {
		fp := stretcherFootprint(feet, d)
		if fp.Overlaps(fb) {
			t.Errorf("%v: la camilla %v se monta sobre los pies %v", d, fp, fb)
		}
		horizontal := d == dirRight || d == dirLeft
		if horizontal && (fp.Dx() != stretcherLen || fp.Dy() != stretcherWid) || !horizontal && (fp.Dx() != stretcherWid || fp.Dy() != stretcherLen) {
			t.Errorf("%v: la camilla mide %v; se esperaba %s", d, fp.Size(), map[bool]string{true: "horizontal", false: "vertical"}[horizontal])
		}
		ahead := map[dir]bool{
			dirRight: fp.Min.X >= fb.Max.X, dirLeft: fp.Max.X <= fb.Min.X,
			dirUp: fp.Max.Y <= fb.Min.Y, dirDown: fp.Min.Y >= fb.Max.Y,
		}[d]
		if !ahead {
			t.Errorf("%v: la camilla %v no va delante de los pies %v", d, fp, fb)
		}
	}
}

// Mientras no haya sprite, la camilla se dibuja con código: marco gris
// #8a9099 con borde #141414, sábana blanca y dos ruedas. Es una imagen
// aparte, así se puede cambiar por un PNG sin tocar la lógica.
func TestStretcherArt_FrameSheetAndWheels(t *testing.T) {
	ink, frame, white := color.RGBA{0x14, 0x14, 0x14, 0xff}, color.RGBA{0x8a, 0x90, 0x99, 0xff}, color.RGBA{0xff, 0xff, 0xff, 0xff}
	for _, vertical := range []bool{false, true} {
		art := stretcherArt(vertical)
		b := art.Bounds()
		w, h := stretcherLen, stretcherWid
		if vertical {
			w, h = stretcherWid, stretcherLen
		}
		if b.Dx() != w || b.Dy() != h+stretcherLegs {
			t.Fatalf("vertical=%v: la imagen mide %v; se esperaba %d×%d (la camilla y sus patas)", vertical, b.Size(), w, h+stretcherLegs)
		}
		if got := art.RGBAAt(b.Min.X, b.Min.Y); got != ink {
			t.Errorf("vertical=%v: la esquina es %v; se esperaba el borde #141414", vertical, got)
		}
		if got := art.RGBAAt(b.Min.X+1, b.Min.Y+h/2); got != frame {
			t.Errorf("vertical=%v: el marco es %v; se esperaba #8a9099", vertical, got)
		}
		if got := art.RGBAAt(b.Min.X+w/2, b.Min.Y+h/2); got != white {
			t.Errorf("vertical=%v: el centro es %v; se esperaba la sábana blanca", vertical, got)
		}
		wheels := 0
		for x := b.Min.X; x < b.Max.X; x++ { // la fila de abajo: solo las ruedas
			if art.RGBAAt(x, b.Max.Y-1).A != 0 && (x == b.Min.X || art.RGBAAt(x-1, b.Max.Y-1).A == 0) {
				wheels++
			}
		}
		if wheels != 2 {
			t.Errorf("vertical=%v: abajo hay %d ruedas; se esperaban 2", vertical, wheels)
		}
	}
}

// checkCarry revisa una ruta empujando la camilla: en cada píxel de cada
// tramo, ni los pies de quien empuja ni la camilla (que va en la
// dirección del tramo) tocan una pared o un mueble.
func checkCarry(t *testing.T, what string, plan carryPlan) {
	t.Helper()
	prev := plan.pickup
	for i, p := range plan.path {
		a, b := prev.add(feetOffset), p.add(feetOffset)
		if !carrySegmentClear(a, b, plan.dirs[i], nil) {
			t.Errorf("%s: en el tramo %v → %v (%v) la camilla o los pies tocan algo", what, prev, p, plan.dirs[i])
		}
		prev = p
	}
}

// Desde cada puesto de piso de la demo se puede llevar al paciente en camilla hasta cualquier cama: la camilla
// cabe por todas las puertas y termina pegada al costado derecho de la
// cama, a su altura (de ahí el paciente pasa de lado).
func TestCarryPlan_FromEveryFloorPlaceToEveryBed(t *testing.T) {
	needFonts(t)
	for _, z := range []zone{zoneLobby, zoneCafeteria, zoneRadiology, zoneHallway2} {
		patient := toVec(spotsOf(z).floor[0].patient)
		for _, room := range []zone{zoneRoom101, zoneRoom102, zoneRoom103} {
			what := fmt.Sprintf("%s → %s", zones[z].label, zones[room].label)
			plan, ok := planCarry(patient, room, nil)
			if !ok {
				t.Errorf("%s: no hay ruta con la camilla", what)
				continue
			}
			checkCarry(t, what, plan)
			last := plan.dirs[len(plan.dirs)-1]
			fp := stretcherFootprint(image.Pt(int(plan.anchor().x+feetOffset.x), int(plan.anchor().y+feetOffset.y)), last)
			bed := bedRect(room)
			if cy := fp.Min.Y + fp.Dy()/2; fp.Min.X < bed.Max.X || fp.Min.X > bed.Max.X+navCell*2 || cy <= bed.Min.Y || cy >= bed.Max.Y {
				t.Errorf("%s: termina yendo %v con la camilla en %v; se esperaba pegada al costado derecho de la cama %v", what, last, fp, bed)
			}
		}
	}
}

// El pasillo 2 al camillero del paso 11: también cabe.
func TestCarryPlan_HallwayToRoom101(t *testing.T) {
	needFonts(t)
	plan, ok := planCarry(toVec(floorSpot(zoneHallway2, 0)), zoneRoom101, nil)
	if !ok {
		t.Fatal("no hay ruta con la camilla del pasillo 2 a la 101")
	}
	checkCarry(t, "pasillo 2 → 101", plan)
}

// El paciente va centrado a lo ancho de la camilla, en las 4 direcciones:
// lo que se ve de su cuerpo (en la hoja, el acostado en el piso ocupa de la
// fila 9 a la 24) queda con su centro sobre el centro de la camilla.
func TestRiderPlacement_CenteredOnTheStretcher(t *testing.T) {
	fp0 := image.Pt(500, 300)
	for _, d := range []dir{dirRight, dirLeft, dirUp, dirDown} {
		fp := stretcherFootprint(fp0, d)
		g := riderPlacement(fp, d)
		minX, maxX := 1e9, -1e9
		for _, c := range [][2]float64{{0, 9}, {16, 9}, {0, 24}, {16, 24}} {
			x, _ := g.Apply(c[0], c[1])
			minX, maxX = min(minX, x), max(maxX, x)
		}
		if got, want := (minX+maxX)/2, float64(fp.Min.X+fp.Dx()/2); got < want-2 || got > want+2 {
			t.Errorf("%v: el cuerpo queda centrado en x=%.0f; la camilla, en x=%.0f", d, got, want)
		}
	}
}

// Subiendo, el paciente sobre la camilla se ve por encima de la cabeza de
// quien empuja (en la vista 3/4, lo que va justo delante de sus pies queda
// detrás de su cuerpo): al menos la mitad de lo que se ve de su cuerpo
// asoma sobre ella.
func TestStretcherGoingUp_TheRiderShowsAboveThePusher(t *testing.T) {
	feet := image.Pt(500, 400)
	head := standingBody.Add(feet.Sub(feetOffsetPt)).Min.Y // donde empieza lo que se ve de quien empuja
	g := riderPlacement(stretcherFootprint(feet, dirUp), dirUp)
	minY, maxY := 1e9, -1e9
	for _, c := range [][2]float64{{0, 9}, {16, 9}, {0, 24}, {16, 24}} { // el cuerpo acostado, en la hoja
		_, y := g.Apply(c[0], c[1])
		minY, maxY = min(minY, y), max(maxY, y)
	}
	if above := float64(head) - minY; above < (maxY-minY)/2 {
		t.Errorf("subiendo, del paciente (y de %.0f a %.0f) asoman %.0f px sobre la cabeza de quien empuja (y=%d); se esperaba al menos la mitad", minY, maxY, above, head)
	}
}

// La camilla gira de a poco (pedido de Samuel, 2026-10-11): en turnTicks
// ticks (~0,3 s) pasa de una dirección a la otra alrededor de los pies de
// quien empuja; el paciente encima arranca y termina sobre la camilla y
// recorre lo mismo en cada tick, sin saltos.
func TestStretcherSwing_TurnsGraduallyAroundThePusher(t *testing.T) {
	pivot := image.Pt(500, 400)
	for _, from := range []dir{dirRight, dirLeft, dirUp, dirDown} {
		for _, to := range []dir{dirRight, dirLeft, dirUp, dirDown} {
			if to == from || to == from.opposite() {
				continue
			}
			w := newSwing(pivot, from, to)
			if len(w.ts) != turnTicks || turnTicks < 15 || turnTicks > 21 {
				t.Errorf("%v→%v: el giro dura %d ticks; se esperaban unos 0,3 s (%d)", from, to, len(w.ts), turnTicks)
			}
			if got, want := w.riderAt(0), riderPos(stretcherFootprint(pivot, from)); got != want {
				t.Errorf("%v→%v: al empezar el paciente está en %v; la camilla lo pide en %v", from, to, got, want)
			}
			if got, want := w.riderAt(1), riderPos(stretcherFootprint(pivot, to)); math.Hypot(got.x-want.x, got.y-want.y) > 1e-6 {
				t.Errorf("%v→%v: al terminar el paciente está en %v; la camilla lo pide en %v", from, to, got, want)
			}
			step := w.riderStep()
			prev := w.riderAt(0)
			for k, tt := range w.ts {
				p := w.riderAt(tt)
				if d := math.Hypot(p.x-prev.x, p.y-prev.y); math.Abs(d-step) > 0.1 || d > 6 {
					t.Errorf("%v→%v, tick %d: el paciente se movió %.2f px; se esperaban %.2f en cada tick (y nunca más de 6)", from, to, k+1, d, step)
				}
				prev = p
			}
			if a := w.angle(1); math.Abs(math.Abs(a)-math.Pi/2) > 1e-9 {
				t.Errorf("%v→%v: gira %.2f rad; se esperaba un cuarto de vuelta", from, to, a)
			}
		}
	}
}

// El A* de la camilla prefiere girar donde hay espacio (habitaciones,
// recepción): con el costo de más de los giros sin espacio, cada traslado
// de un puesto de piso a una cama hace a lo sumo los mismos giros sin
// espacio que sin ese costo, y alguno hace menos.
func TestCarryPlan_PrefersTurningWhereThereIsRoom(t *testing.T) {
	needFonts(t)
	tight := func(p carryPlan) int {
		n := 0
		for i := 1; i < len(p.path); i++ {
			if p.dirs[i] != p.dirs[i-1] {
				pivot := feetPoint(p.path[i-1])
				if !swingClear(pivot, p.dirs[i-1], p.dirs[i], nil) {
					n++
				}
			}
		}
		return n
	}
	fewer := false
	for z := range lyingZones {
		for _, room := range []zone{zoneRoom101, zoneRoom102, zoneRoom103} {
			at := toVec(floorSpot(z, 0))
			with, ok1 := planCarry(at, room, nil) // el que usa la escena
			without, ok2 := planCarryWith(at, room, nil, 0)
			if !ok1 || !ok2 {
				t.Fatalf("%s → %s: sin ruta", zones[z].label, zones[room].label)
			}
			if a, b := tight(with), tight(without); a > b {
				t.Errorf("%s → %s: %d giros sin espacio; sin el costo de más eran %d", zones[z].label, zones[room].label, a, b)
			} else if a < b {
				fewer = true
			}
		}
	}
	if !fewer {
		t.Error("el costo de los giros sin espacio no cambió ningún traslado")
	}
}

// Con la puerta de arriba de la cafetería alineada con la de la 101 (mapa
// del 2026-10-11), la camilla que recoge a alguien en la cafetería sube
// derecho por las dos puertas: pasa por la de la cafetería y no toca más
// zonas que la cafetería, el pasillo 1 y la 101.
func TestCarryPlan_FromTheCafeteriaToRoom101GoesStraightUp(t *testing.T) {
	needFonts(t)
	cafeDoor := doorways[4].gap // pasillo 1 ↔ cafetería
	places := floorOf(zoneCafeteria)
	if len(places) == 0 {
		t.Fatal("la cafetería no tiene puestos de piso")
	}
	for _, f := range places {
		plan, ok := planCarry(toVec(f.patient), zoneRoom101, nil)
		if !ok {
			t.Fatalf("del puesto %v a la 101: no hay ruta con la camilla", f.patient)
		}
		through := false
		for _, r := range carryLane(plan) {
			through = through || r.Overlaps(cafeDoor)
			for _, z := range []zone{zoneLobby, zoneHallway2, zoneRadiology, zoneRoom102} {
				if r.Overlaps(zones[z].rect) {
					t.Errorf("del puesto %v a la 101: la camilla pasa por %s (%v)", f.patient, zones[z].label, r)
				}
			}
		}
		if !through {
			t.Errorf("del puesto %v a la 101: la camilla no sube por la puerta de arriba de la cafetería", f.patient)
		}
	}
}
