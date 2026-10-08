package ui

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
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

	// Rótulos: abajo a la izquierda de cada zona (donde no estorban las
	// camas ni los personajes), salvo el pasillo 1, que es muy bajito.
	for _, z := range zoneOrder {
		info := zones[z]
		y := info.rect.Max.Y - 10
		if z == zoneHallway1 {
			y = info.rect.Min.Y + 2
		}
		drawText(dst, info.label, float64(info.rect.Min.X+3), float64(y), smallFace, colInk)
	}
}

// drawRooms dibuja la cama de cada habitación y su letrero de estado,
// centrado sobre el borde de abajo como en la maqueta. El texto del
// letrero sale del String() del modelo ("ocupada", "disponible").
func drawRooms(dst *ebiten.Image, rooms []game.RoomView) {
	for _, r := range rooms {
		z := roomZone(r.Number)
		bed := bedRect(z)

		panel(dst, bed, colPaper)                                                                // colchón
		fillRect(dst, image.Rect(bed.Min.X, bed.Min.Y, bed.Max.X, bed.Min.Y+4), colMetal)        // cabecera
		fillRect(dst, image.Rect(bed.Min.X+2, bed.Min.Y+6, bed.Max.X-2, bed.Min.Y+12), colWhite) // almohada
		blanketTop := bed.Min.Y + 24
		if len(r.Occupants) > 0 {
			blanketTop = bed.Min.Y + 14 // ocupada: la cobija sube
		}
		fillRect(dst, image.Rect(bed.Min.X+1, blanketTop, bed.Max.X-1, bed.Max.Y-1), colBlue)

		bg := colBadgeAvailable
		if r.State == hospital.Occupied {
			bg = colBadgeOccupied
		}
		room := zones[z].rect
		label(dst, strings.ToUpper(r.State.String()), room.Min.X+room.Dx()/2, room.Max.Y-5, smallFace, colWhite, bg)
	}
}
