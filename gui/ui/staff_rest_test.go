package ui

import (
	"image"
	"math/rand"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// §10.7 punto 7 (pedido de Samuel, 2026-10-17): 6 puestos para el personal,
// sin que dos se encimen. staffHomeSpot reparte los de la sala y, si no
// alcanzan, el desborde del pasillo 2.
func TestStaffHomeSpot_SixDistinctSpotsWithoutOverlap(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	var pts []image.Point
	var shapes []shape
	for i := 0; i < 6; i++ {
		p := staffHomeSpot(i)
		for j, q := range pts {
			if p == q {
				t.Errorf("el puesto %d (%v) coincide con el %d", i, p, j)
			}
		}
		sh := standingShape(p)
		for j, other := range shapes {
			if sh.overlaps(other) {
				t.Errorf("el puesto %d (%v) se encima con el %d (%v)", i, p, j, pts[j])
			}
		}
		pts, shapes = append(pts, p), append(shapes, sh)
	}
}

// Los primeros puestos son los de verdad de la sala (los que calcula
// spotsOf): el desborde del pasillo 2 es solo para el que no cupo ahí.
func TestStaffHomeSpot_FirstOnesAreTheStaffRoomsOwnSpots(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	room := spotsOf(zoneStaffRoom).standing
	for i, want := range room {
		if got := staffHomeSpot(i); got != want {
			t.Errorf("staffHomeSpot(%d) = %v; se esperaba el puesto de la sala %v", i, got, want)
		}
	}
	if got := staffHomeSpot(len(room)); got == room[len(room)-1] {
		t.Errorf("staffHomeSpot(%d) repitió el último de la sala; se esperaba un puesto nuevo en el pasillo 2", len(room))
	}
}

// El desborde del pasillo 2 no toca muebles, no se sale del pasillo y
// ninguno de sus puestos se encima con otro (incluye el cupo general de la
// zona, el que usan los pacientes que deambulan ahí: va primero en la
// lista).
func TestStaffOverflowSpots_InsideHallway2WithoutFurnitureOrOverlap(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	pool := spotsOf(zoneHallway2).standing
	out := staffOverflowSpots()
	for _, want := range pool { // el cupo general también sirve de desborde
		found := false
		for _, got := range out {
			found = found || got == want
		}
		if !found {
			t.Errorf("staffOverflowSpots() = %v; le falta el cupo general %v", out, want)
		}
	}
	var shapes []shape
	for i, p := range out {
		body := standingBody.Add(p)
		if !body.In(zones[zoneHallway2].rect) {
			t.Errorf("el desborde %v se sale del pasillo 2", p)
		}
		if onFurniture(body) {
			t.Errorf("el desborde %v queda sobre un mueble", p)
		}
		sh := standingShape(p)
		for j, other := range shapes {
			if sh.overlaps(other) {
				t.Errorf("el desborde %d (%v) se encima con el %d", i, p, j)
			}
		}
		shapes = append(shapes, sh)
	}
}

func TestRestWanderTicks_Between8And15Seconds(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		got := restWanderTicks(rng)
		if got < 8*60 || got > 15*60 {
			t.Fatalf("restWanderTicks() = %d; se esperaba entre %d y %d", got, 8*60, 15*60)
		}
	}
}

func TestRestZonesFor_DoctorsAndOrderliesGoWhereSamuelSaid(t *testing.T) {
	has := func(zs []zone, z zone) bool {
		for _, x := range zs {
			if x == z {
				return true
			}
		}
		return false
	}
	doc := restZonesFor(true)
	for _, want := range []zone{zoneStaffRoom, zoneCafeteria, zoneHallway2} {
		if !has(doc, want) {
			t.Errorf("restZonesFor(médico) = %v; le falta %s", doc, zones[want].label)
		}
	}
	ord := restZonesFor(false)
	for _, want := range []zone{zoneStaffRoom, zoneLobby} {
		if !has(ord, want) {
			t.Errorf("restZonesFor(camillero) = %v; le falta %s", ord, zones[want].label)
		}
	}
	if has(ord, zoneCafeteria) || has(ord, zoneHallway2) {
		t.Errorf("restZonesFor(camillero) = %v; el camillero deambula entre la sala y la recepción, no por ahí", ord)
	}
}

// El personal libre no se queda siempre en la sala: con el tiempo (más de
// 15 s de Update, el peor caso de restWanderTicks) termina pisando otra de
// sus zonas, y nunca comparte el puesto exacto con otro miembro del
// personal (pedido de Samuel, 2026-10-17).
func TestFreeStaff_WandersAwayFromTheStaffRoomOverTime(t *testing.T) {
	s := newTestGameScene(t)
	seenZone := map[string]map[zone]bool{"D-01": {}, "C-01": {}}
	for i := 0; i < 20*60; i++ { // 20 s de sobra para los 15 s más largos de restWanderTicks
		play(s, 1)
		positions := map[vec][]string{}
		for _, id := range []string{"D-01", "C-01"} {
			st := staffNow(s, id)
			if st.Activity != game.Free {
				continue // ocupado: no cuenta para el deambular
			}
			p := s.pos[id]
			seenZone[id][zoneOfFeet(image.Pt(int(p.x), int(p.y)))] = true
			positions[p] = append(positions[p], id)
		}
		for p, ids := range positions {
			if len(ids) > 1 {
				t.Fatalf("tick %d: %v comparten el mismo puesto %v", i, ids, p)
			}
		}
	}
	for id, zs := range seenZone {
		if len(zs) < 2 {
			t.Errorf("%s nunca cambió de zona en 20 s libre: %v", id, zs)
		}
	}
}
