package ui

import (
	"fmt"
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Pedido por Samuel: en el estado final de CADA paso de la demo, ninguna
// etiqueta, burbuja, rayito ni punto se monta sobre otro ni sobre los
// letreros OCUPADA/DISPONIBLE de las habitaciones.
func TestOverlays_DoNotOverlapInAnyStepOfTheDemo(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	s := NewDemoScene(game.NewDemo())

	for step := 0; step <= game.DemoSteps; step++ {
		if step > 0 {
			if _, err := s.demo.Next(); err != nil {
				t.Fatal(err)
			}
			s.refresh()
		}

		var all []overlay
		var owner []string
		for _, b := range roomBadges(s.snap.Rooms) {
			all, owner = append(all, b), append(owner, "letrero "+b.text)
		}
		for _, f := range s.figures {
			// Estado final: cada uno en su puesto y, si duerme, con su burbuja.
			for _, o := range figureOverlays(f, f.target, f.state != hospital.Awake) {
				all, owner = append(all, o), append(owner, f.id)
			}
		}

		for i := range all {
			for j := i + 1; j < len(all); j++ {
				if all[i].rect.Overlaps(all[j].rect) {
					t.Errorf("paso %d: %q (%s, %v) se monta sobre %q (%s, %v)",
						step, all[i].text, owner[i], all[i].rect, all[j].text, owner[j], all[j].rect)
				}
			}
		}

		// Ni sobre los rótulos de las zonas ("CAFETERÍA", "PASILLO 2"…) ni
		// sobre las camas.
		for _, z := range zoneOrder {
			for i, o := range all {
				if o.rect.Overlaps(zoneLabelBox(z)) {
					t.Errorf("paso %d: %q (%s, %v) tapa el rótulo %q", step, o.text, owner[i], o.rect, zones[z].label)
				}
			}
		}
		for _, room := range s.snap.Rooms {
			bed := bedRect(roomZone(room.Number))
			for i, o := range all {
				if o.rect.Overlaps(bed) {
					t.Errorf("paso %d: %q (%s, %v) se monta sobre la cama de la %d", step, o.text, owner[i], o.rect, room.Number)
				}
			}
		}

		// Tampoco sobre el CUERPO de otro personaje (si no, lo tapa).
		for _, f := range s.figures {
			body := figureBody(f, f.target)
			for i, o := range all {
				if owner[i] != f.id && o.rect.Overlaps(body) {
					t.Errorf("paso %d: %q (%s, %v) queda debajo del cuerpo de %s (%v)",
						step, o.text, owner[i], o.rect, f.id, body)
				}
			}
		}
	}
}

// figureBody es lo que ocupa el cuerpo de un personaje en at: su cuadro de
// 32×48 en la pantalla, o el de 48×32 si está rotado en su cama.
func figureBody(f figure, at vec) image.Rectangle {
	if f.patient && f.room != 0 && at == bedCell(roomZone(f.room)) {
		cell := bedCell(roomZone(f.room))
		x, y := int(cell.x), int(cell.y)
		return image.Rect(x, y, x+figHeight, y+figWidth) // acostado (rotado): 48×32
	}
	x, y := int(at.x), int(at.y)
	return image.Rect(x, y, x+figWidth, y+figHeight)
}

// figureBox es todo lo que ocupa un personaje: su cuadro y sus rótulos.
func figureBox(f figure, at vec) image.Rectangle {
	x, y := int(at.x), int(at.y)
	box := image.Rect(x, y, x+figWidth, y+figHeight)
	for _, o := range figureOverlays(f, at, true) {
		box = box.Union(o.rect)
	}
	return box
}

// Pedido por Samuel: los puestos quedan BIEN dentro de su zona, con su
// burbuja, sus rayitos y su etiqueta.
func TestSpots_FigureAndItsLabelsStayInsideTheZone(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	sleeper := figure{id: "P-000", patient: true, state: hospital.AsleepInHallway, level: hospital.Severe}
	standing := figure{id: "P-000", patient: true, state: hospital.Awake}
	staff := figure{id: "D-00", doctor: true}

	check := func(what string, f figure, at vec, z zone) {
		t.Helper()
		if box := figureBox(f, at); !box.In(zones[z].rect) {
			t.Errorf("%s: %v se sale de %s %v", what, box, zones[z].label, zones[z].rect)
		}
	}

	// Zonas donde alguien se queda quieto en la demo (el pasillo 1 es solo de paso).
	for _, z := range []zone{zoneLobby, zoneCafeteria, zoneRadiology, zoneHallway2, zoneStaffRoom} {
		for k := 0; k < 2; k++ {
			check(fmt.Sprintf("de pie %d", k), standing, toVec(slot(z, k)), z)
			check(fmt.Sprintf("personal %d", k), staff, toVec(slot(z, k)), z)
		}
		floor := toVec(floorSpot(z, 0))
		check("dormido en el piso", sleeper, floor, z)
		check("el que lo atiende", staff, helperSpot(floor), z)
	}
	for _, number := range []int{101, 102, 103} {
		room := roomZone(number)
		inBed := figure{id: "P-000", patient: true, state: hospital.AsleepInBed, level: hospital.Severe, room: number}
		check(fmt.Sprintf("en la cama %d", number), inBed, bedCell(room), room)
		check(fmt.Sprintf("despierto al lado de la cama %d", number), standing, toVec(wakeSpot(room)), room)
	}
}
