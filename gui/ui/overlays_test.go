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
// 48×72 en la pantalla o, acostado en su cama, lo que se ve del cuerpo.
func figureBody(f figure, at vec) image.Rectangle {
	if f.patient && f.room != 0 && at == bedCell(roomZone(f.room)) {
		cell := bedCell(roomZone(f.room))
		return bedBody.Add(image.Pt(int(cell.x), int(cell.y))) // acostado en la cama: lo que se ve del cuerpo rotado
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
		box := figureBox(f, at)
		if box.In(zones[z].rect) {
			return
		}
		// Se sale: solo vale si es un puesto del segundo intento del
		// buscador y sus rótulos no tapan otro rótulo, un letrero ni un mueble.
		if !relaxedSpot(z, image.Pt(int(at.x), int(at.y))) {
			t.Errorf("%s: %v se sale de %s %v (y no es un puesto del segundo intento)", what, box, zones[z].label, zones[z].rect)
			return
		}
		for _, o := range figureOverlays(f, at, true) {
			if onFurniture(o.rect) || coversALabel(o.rect) {
				t.Errorf("%s: el rótulo %q (%v) se sale de %s y tapa un mueble o un rótulo", what, o.text, o.rect, zones[z].label)
			}
		}
	}

	// Lo que usa la demo: los 5 pacientes de pie en la recepción, el
	// personal en su sala y un acostado (con quien lo atiende) donde se
	// desploman Yeimy, Kevin, Ludys y Wilfrido.
	for k := 0; k < 5; k++ {
		check(fmt.Sprintf("de pie %d", k), standing, toVec(slot(zoneLobby, k)), zoneLobby)
	}
	for k := 0; k < 3; k++ {
		check(fmt.Sprintf("personal %d", k), staff, toVec(slot(zoneStaffRoom, k)), zoneStaffRoom)
	}
	for _, z := range []zone{zoneLobby, zoneCafeteria, zoneRadiology, zoneHallway2} {
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
