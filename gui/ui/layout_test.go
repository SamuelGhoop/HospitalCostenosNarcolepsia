// Tests de caja blanca (package ui): prueban las funciones internas que
// convierten la maqueta en posiciones de pantalla. El dibujo en sí no se
// prueba aquí porque necesita la ventana de Ebitengine.
package ui

import (
	"image"
	"testing"
)

func TestFromMockup_ScalesTheWholeWorldToTheScreen(t *testing.T) {
	got := fromMockup(0, 0, mockupWidth, mockupHeight)

	if want := image.Rect(0, 0, ScreenWidth, ScreenHeight); got != want {
		t.Errorf("fromMockup(mundo completo) = %v; se esperaba %v", got, want)
	}
}

func TestFromMockup_KeepsTheProportionsOfTheMockup(t *testing.T) {
	// Habitación 101 en la maqueta: x=8, y=18, 96×50 → ×640/464 ≈ 1,379.
	got := fromMockup(8, 18, 96, 50)

	if want := image.Rect(11, 25, 143, 94); got != want {
		t.Errorf("habitación 101 = %v; se esperaba %v", got, want)
	}
}

func TestZones_DoNotOverlapAndStayInsideTheScreen(t *testing.T) {
	screen := image.Rect(0, 0, ScreenWidth, ScreenHeight)
	for i, a := range zoneOrder {
		ra := zones[a].rect
		if !ra.In(screen) || ra.Empty() {
			t.Errorf("la zona %s (%v) no está dentro de la pantalla", zones[a].label, ra)
		}
		for _, b := range zoneOrder[i+1:] {
			if ra.Overlaps(zones[b].rect) {
				t.Errorf("las zonas %s y %s se solapan", zones[a].label, zones[b].label)
			}
		}
	}
}

// Todas las ubicaciones que usa el guion de la demo (las mismas de main.go).
func TestZoneFor_MapsEveryLocationOfTheDemo(t *testing.T) {
	tests := []struct {
		location string
		want     zone
	}{
		{"recepción", zoneLobby},
		{"cafetería, después del sancocho", zoneCafeteria},
		{"fila de radiología", zoneRadiology},
		{"pista de champeta del patio", zoneLobby}, // decisión de Samuel: va en el Lobby
		{"pasillo 2, segundo piso", zoneHallway2},
		{"pasillo 1", zoneHallway1},
		{"habitación 101", zoneRoom101},
		{"habitación 102", zoneRoom102},
		{"habitación 103", zoneRoom103},
		{"un sitio que no está en el mapa", zoneLobby},
	}

	for _, tt := range tests {
		if got := zoneFor(tt.location); got != tt.want {
			t.Errorf("zoneFor(%q) = %s; se esperaba %s", tt.location, zones[got].label, zones[tt.want].label)
		}
	}
}

func TestSlot_StaysInsideItsZoneWithoutOverlapping(t *testing.T) {
	for _, z := range zoneOrder {
		seen := map[image.Point]bool{}
		for i := 0; i < 4; i++ {
			p := slot(z, i)
			cell := image.Rect(p.X, p.Y, p.X+cellWidth, p.Y+cellHeight)
			if !cell.In(zones[z].rect) {
				t.Errorf("%s: el puesto %d (%v) se sale de la zona %v", zones[z].label, i, cell, zones[z].rect)
			}
			if seen[p] {
				t.Errorf("%s: el puesto %d repite la posición %v", zones[z].label, i, p)
			}
			seen[p] = true
		}
	}
}
