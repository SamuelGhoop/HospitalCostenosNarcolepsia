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
	room    int // habitación con cama asignada en el modelo; 0 = ninguna
	target  vec // puesto final según la foto del modelo (sin coreografía)
}

// drawFigureBody dibuja el cuerpo de un personaje en su cuadro de 16×24
// con esquina en at. Los rótulos (burbuja, etiqueta…) van aparte, en
// figureOverlays, porque se dibujan encima de todo (incluida la cobija).
// anim es la animación del paciente (nil para el personal).
func drawFigureBody(dst *ebiten.Image, f figure, at vec, anim *patientAnim) {
	x, y := int(math.Round(at.x)), int(math.Round(at.y))
	switch {
	case f.patient:
		drawPatientSprite(dst, f, at, anim)
	case f.doctor:
		drawStanding(dst, x, y, colPaper, colTeal) // bata blanca y pantalón turquesa
	default:
		drawStanding(dst, x, y, colGreen, colGreenDark) // uniforme verde del camillero
	}
}

// drawPatientSprite dibuja el cuadro actual de la animación del paciente
// (hoja patient_body_skin1.png). spritePlacement decide si va rotado
// dentro de la cama o tal cual en su puesto.
func drawPatientSprite(dst *ebiten.Image, f figure, at vec, a *patientAnim) {
	frame := frameAt(a.anim, a.ticks)

	// ¿En qué cama se dibuja, si toca dibujarlo en una? En la que le asignó
	// el modelo o, en el primer cuadro de despertarse, en la que acaba de dejar.
	bed, hasBed := image.Rectangle{}, false
	switch {
	case f.room != 0:
		bed, hasBed = bedRect(roomZone(f.room)), true
	case a.anim == animWakeUp && a.room != 0:
		bed, hasBed = bedRect(roomZone(a.room)), true
	}

	g, _ := spritePlacement(a.anim, frame, bed, hasBed, at, a.flip)
	op := &ebiten.DrawImageOptions{GeoM: g}
	// SubImage recorta el cuadro de la hoja sin copiar píxeles.
	dst.DrawImage(patientSheet.SubImage(frameRect(a.anim, frame)).(*ebiten.Image), op)
}

// drawStanding: personaje de pie (cabeza, tronco y piernas) con contorno.
// Lo usa el personal mientras llegan sus sprites.
func drawStanding(dst *ebiten.Image, x, y int, body, legs color.RGBA) {
	fillRect(dst, image.Rect(x+3, y+2, x+13, y+23), colInk) // contorno
	fillRect(dst, image.Rect(x+4, y+3, x+12, y+9), colSkin) // cabeza
	fillRect(dst, image.Rect(x+4, y+10, x+12, y+17), body)  // tronco
	fillRect(dst, image.Rect(x+4, y+18, x+12, y+22), legs)  // piernas
}

// toVec convierte un punto entero de la pantalla en un vec.
func toVec(p image.Point) vec { return vec{float64(p.X), float64(p.Y)} }
