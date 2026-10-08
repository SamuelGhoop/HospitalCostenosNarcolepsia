package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// figure es un personaje listo para dibujar, armado con las vistas de game.
type figure struct {
	id      string
	patient bool // false = personal
	doctor  bool // solo personal: médico (true) o camillero (false)
	state   hospital.PatientState
	level   hospital.NarcolepsyLevel
	shirt   color.RGBA
	target  vec // puesto al que tiene que llegar
}

// drawFigure dibuja un personaje en su cuadro de 16×24 con esquina en at.
//
// Rectángulos provisionales mientras llegan los sprites. Igual que los
// sprites (GAME_DESIGN §10.3), el paciente dormido se dibuja HORIZONTAL
// dentro del mismo cuadro de 16×24, sin rotar nada.
func drawFigure(dst *ebiten.Image, f figure, at vec) {
	x, y := int(math.Round(at.x)), int(math.Round(at.y))

	switch {
	case f.patient && f.state != hospital.Awake:
		drawLying(dst, x, y, f.shirt)
	case f.patient:
		drawStanding(dst, x, y, f.shirt, colPants)
	case f.doctor:
		drawStanding(dst, x, y, colPaper, colTeal) // bata blanca y pantalón turquesa
	default:
		drawStanding(dst, x, y, colGreen, colGreenDark) // uniforme verde del camillero
	}

	if f.patient {
		// Dormido: burbuja Zzz y "rayitos" de nivel (solo en dormidos).
		if txt, bg, ok := bubbleFor(f.state); ok {
			bubble := label(dst, txt, x+cellWidth/2, y, smallFace, colWhite, bg)
			c, n := levelStyle(f.level)
			for i := 0; i < n; i++ {
				bolt := image.Rect(bubble.Max.X+2+i*4, bubble.Min.Y+2, bubble.Max.X+5+i*4, bubble.Max.Y-2)
				panel(dst, bolt, c)
			}
		}
	} else {
		// Punto verde = libre. En la demo el personal nunca queda ocupado.
		panel(dst, image.Rect(x+6, y-5, x+10, y-1), colGreen)
	}

	// En la demo los IDs se muestran siempre, para seguir los subtítulos.
	label(dst, f.id, x+cellWidth/2, y+cellHeight+1, smallFace, colInk, colPaper)
}

// drawStanding: personaje de pie (cabeza, tronco y piernas) con contorno.
func drawStanding(dst *ebiten.Image, x, y int, body, legs color.RGBA) {
	fillRect(dst, image.Rect(x+3, y+2, x+13, y+23), colInk) // contorno
	fillRect(dst, image.Rect(x+4, y+3, x+12, y+9), colSkin) // cabeza
	fillRect(dst, image.Rect(x+4, y+10, x+12, y+17), body)  // tronco
	fillRect(dst, image.Rect(x+4, y+18, x+12, y+22), legs)  // piernas
}

// drawLying: personaje acostado, horizontal en la parte de abajo del cuadro.
func drawLying(dst *ebiten.Image, x, y int, shirt color.RGBA) {
	fillRect(dst, image.Rect(x, y+13, x+16, y+23), colInk)   // contorno
	fillRect(dst, image.Rect(x+1, y+14, x+6, y+22), colSkin) // cabeza
	fillRect(dst, image.Rect(x+7, y+14, x+15, y+22), shirt)  // cuerpo
}

// bedCell es el cuadro de 16×24 del paciente acostado en la cama de una
// habitación: centrado sobre la cama.
func bedCell(room zone) vec {
	bed := bedRect(room)
	return vec{float64(bed.Min.X + (bed.Dx()-cellWidth)/2), float64(bed.Min.Y + 4)}
}

// toVec convierte un punto entero de la pantalla en un vec.
func toVec(p image.Point) vec { return vec{float64(p.X), float64(p.Y)} }
