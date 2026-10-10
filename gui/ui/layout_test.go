// Tests de caja blanca (package ui): prueban las funciones internas que
// convierten la maqueta en posiciones de pantalla. El dibujo en sí no se
// prueba aquí porque necesita la ventana de Ebitengine.
package ui

import (
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

func TestFromMockup_ScalesTheWholeWorldToTheScreen(t *testing.T) {
	got := fromMockup(0, 0, mockupWidth, mockupHeight)

	if want := image.Rect(0, 0, ScreenWidth, ScreenHeight); got != want {
		t.Errorf("fromMockup(mundo completo) = %v; se esperaba %v", got, want)
	}
}

// La pantalla lógica es de 1280×720, como el mapa de fondo (§10.7).
func TestScreen_IsTheSizeOfTheBackgroundMap(t *testing.T) {
	if ScreenWidth != 1280 || ScreenHeight != 720 {
		t.Errorf("pantalla lógica = %d×%d; se esperaba 1280×720", ScreenWidth, ScreenHeight)
	}
}

func TestFromMockup_KeepsTheProportionsOfTheMockup(t *testing.T) {
	// Habitación 101 en la maqueta: x=8, y=18, 96×50 → ×1280/464 ≈ 2,759.
	got := fromMockup(8, 18, 96, 50)

	if want := image.Rect(22, 50, 287, 188); got != want {
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

// Las zonas donde alguien se queda quieto en la demo tienen los puestos que
// hacen falta: de pie (los 5 pacientes en la recepción en los pasos 3 y 4,
// y el personal en su sala) y en el piso, con su ayudante al lado (donde
// se desploman Yeimy, Kevin, Ludys y Wilfrido).
func TestSpots_EnoughFreePlacesWhereTheDemoNeedsThem(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	standing := map[zone]int{zoneLobby: 5, zoneStaffRoom: 3}
	for z, n := range standing {
		s := spotsOf(z).standing
		if len(s) < n {
			t.Errorf("%s: hay %d puestos de pie libres; la demo necesita %d", zones[z].label, len(s), n)
		}
		for _, p := range s {
			if body := standingBody.Add(p); !body.In(zones[z].rect) || onFurniture(body) {
				t.Errorf("%s: el puesto %v deja el cuerpo fuera de la zona o sobre un mueble", zones[z].label, p)
			}
		}
	}
	for _, z := range []zone{zoneLobby, zoneCafeteria, zoneRadiology, zoneHallway2} {
		if len(spotsOf(z).floor) == 0 {
			t.Errorf("%s: no hay ningún puesto en el piso libre para un paciente acostado con su ayudante", zones[z].label)
		}
	}
}

// Pedido por Samuel: los rótulos de zona tampoco van sobre muebles.
func TestZoneLabels_AreNotOverFurniture(t *testing.T) {
	if err := loadFonts(); err != nil {
		t.Fatal(err)
	}
	for _, z := range zoneOrder {
		if box := zoneLabelBox(z); onFurniture(box) {
			t.Errorf("el rótulo %q (%v) queda sobre un mueble", zones[z].label, box)
		}
	}
}

// Pedido por Samuel: la cobija (bed_blanket.png, 56×48) va exactamente en
// (126, 88), (399, 88) y (672, 88), y esas posiciones salen de layout.go,
// desde las coordenadas de la maqueta.
func TestBlanketSpot_IsExactlyWhereTheMapHasTheBeds(t *testing.T) {
	want := map[zone]image.Point{zoneRoom101: {126, 88}, zoneRoom102: {399, 88}, zoneRoom103: {672, 88}}
	for room, p := range want {
		if got := blanketSpot(room); got != p {
			t.Errorf("%s: la cobija va en %v; se esperaba %v", zones[room].label, got, p)
		}
	}

	img, err := decodePNG(assets.Map, blanketPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.Bounds().Size(); got != image.Pt(56, 48) {
		t.Errorf("bed_blanket.png mide %v; se esperaba 56×48", got)
	}
}

// Acostado, lo que se ve del paciente cabe a lo ancho de la cama y queda
// con la mitad de abajo bajo la cobija.
func TestBedCell_HalfOfThePatientIsUnderTheBlanket(t *testing.T) {
	for _, room := range []zone{zoneRoom101, zoneRoom102, zoneRoom103} {
		cell, blanket, bed := bedCell(room), blanketSpot(room), bedRect(room)
		body := bedBody.Add(image.Pt(int(cell.x), int(cell.y)))
		if body.Min.Y+body.Dy()/2 != blanket.Y {
			t.Errorf("%s: el cuerpo va de y=%d a %d y la cobija empieza en y=%d; debía taparle la mitad", zones[room].label, body.Min.Y, body.Max.Y, blanket.Y)
		}
		if body.Min.X < bed.Min.X || body.Max.X > bed.Max.X {
			t.Errorf("%s: el cuerpo (%v) no cabe a lo ancho de la cama (%v)", zones[room].label, body, bed)
		}
	}
}
