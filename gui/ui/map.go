package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// drawMap dibuja el hospital vacío siguiendo hospital.dc.html: arena,
// acera, calle, el edificio y sus zonas con su rótulo.
func drawMap(dst *ebiten.Image) {
	dst.Fill(colSand)

	// Acera y calle de dos carriles: línea amarilla discontinua (sin
	// pintar sobre el paso de cebra) y el paso de cebra frente a la puerta.
	fillRect(dst, sidewalkRect, colSidewalk)
	fillRect(dst, fromMockup(0, 210, mockupWidth, 2), colWhite) // bordillo
	fillRect(dst, fromMockup(0, 212, mockupWidth, 1), colInk)
	fillRect(dst, streetRect, colStreet)
	for x := 0.0; x < mockupWidth; x += 18 {
		if x+11 < 197 || x > 231 {
			fillRect(dst, fromMockup(x, 235, 11, 2), colYellow)
		}
	}
	for y := 215.0; y < 257; y += 6 {
		fillRect(dst, fromMockup(199, y, 30, 3), colWhite)
	}

	// Edificio y zonas.
	fillRect(dst, buildingRect, colWall)
	outline(dst, buildingRect, colInk)
	for _, z := range zoneOrder {
		info := zones[z]
		fillRect(dst, info.rect, info.floor)
		outline(dst, info.rect, colInk)
	}

	// Líneas guía: azul en el pasillo 1, verde pegada al borde del pasillo 2.
	fillRect(dst, fromMockup(8, 82, 448, 2), colBlue)
	fillRect(dst, fromMockup(339, 84, 2, 94), colGreen)

	// Puerta principal entre la recepción y la acera.
	fillRect(dst, fromMockup(199, 178, 30, 4), colMuted)

	// Rótulo de cada zona, en el sitio que calcula zoneLabelBox (el mismo
	// que revisa el test de rótulos sin solaparse).
	for _, z := range zoneOrder {
		box := zoneLabelBox(z)
		drawText(dst, zones[z].label, float64(box.Min.X+2), float64(box.Min.Y+1), smallFace, colInk)
	}
}

// La cama se dibuja en dos capas para que el paciente se vea arropado:
//
//	drawBeds (colchón y almohada) → el paciente → drawBlankets (la cobija)

// drawBeds dibuja la primera capa de cada cama: colchón, cabecera y almohada.
func drawBeds(dst *ebiten.Image, rooms []game.RoomView) {
	for _, r := range rooms {
		bed := bedRect(roomZone(r.Number))
		panel(dst, bed, colPaper)                                                                // colchón
		fillRect(dst, image.Rect(bed.Min.X, bed.Min.Y, bed.Max.X, bed.Min.Y+4), colMetal)        // cabecera
		fillRect(dst, image.Rect(bed.Min.X+2, bed.Min.Y+6, bed.Max.X-2, bed.Min.Y+12), colWhite) // almohada
	}
}

// drawBlankets dibuja la cobija ENCIMA del paciente. Si alguien está
// acostado en esa cama (lyingIn[número]), la cobija le tapa la mitad de
// abajo; si no, queda doblada al pie de la cama.
func drawBlankets(dst *ebiten.Image, rooms []game.RoomView, lyingIn map[int]bool) {
	for _, r := range rooms {
		bed := bedRect(roomZone(r.Number))
		top := bed.Min.Y + 24 // doblada al pie
		if lyingIn[r.Number] {
			top = bed.Min.Y + 12 // el paciente rotado ocupa de +4 a +20: le tapa de +12 hacia abajo
		}
		fillRect(dst, image.Rect(bed.Min.X+1, top, bed.Max.X-1, bed.Max.Y-1), colBlue)
		fillRect(dst, image.Rect(bed.Min.X+1, top, bed.Max.X-1, top+2), colLightBlue) // el doblez de la sábana
	}
}
