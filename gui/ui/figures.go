package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// figure es un personaje listo para dibujar, armado con las vistas de game.
type figure struct {
	id      string
	patient bool // false = personal
	doctor  bool // solo personal: médico (true) o camillero (false)
	state   hospital.PatientState
	level   hospital.NarcolepsyLevel
	room    int             // habitación con cama asignada en el modelo; 0 = ninguna
	look    game.Appearance // cómo se ve (el camillero todavía no la usa: va como rectángulo)
	target  vec             // puesto final según la foto del modelo (sin coreografía)
}

// drawFigureBody dibuja el cuerpo de un personaje en su cuadro de 32×48
// con esquina en at. Los rótulos (burbuja, etiqueta…) van aparte, en
// figureOverlays, porque se dibujan encima de todo (incluida la cobija).
// pa es la animación del paciente y sa la del médico (nil si no aplica).
func drawFigureBody(dst *ebiten.Image, f figure, at vec, pa *patientAnim, sa *staffAnim) {
	x, y := int(math.Round(at.x)), int(math.Round(at.y))
	switch {
	case f.patient:
		drawPatientSprite(dst, f, at, pa)
	case f.doctor:
		drawDoctorSprite(dst, f, at, sa)
	default:
		drawStanding(dst, x, y, colGreen, colGreenDark) // uniforme verde del camillero
	}
}

// drawPatientSprite dibuja el cuadro actual de la animación del paciente,
// capa por capa según su apariencia. spritePlacement decide si va rotado
// dentro de la cama o tal cual en su puesto.
func drawPatientSprite(dst *ebiten.Image, f figure, at vec, a *patientAnim) {
	frame := frameAt(a.anim, a.ticks)

	// ¿En qué cama se dibuja, si toca dibujarlo en una? En la que le asignó
	// el modelo o, en el primer cuadro de despertarse, en la que acaba de dejar.
	bedAt, hasBed := vec{}, false
	switch {
	case f.room != 0:
		bedAt, hasBed = bedCell(roomZone(f.room)), true
	case a.anim == animWakeUp && a.room != 0:
		bedAt, hasBed = bedCell(roomZone(a.room)), true
	}

	g, _ := spritePlacement(a.anim, frame, bedAt, hasBed, at, a.flip)
	drawLayers(dst, patientLayers(f.look, a.anim, frame), g)
}

// drawRider dibuja al paciente acostado sobre la camilla fp (yendo en d),
// con sus capas y la animación de dormido en el piso.
func drawRider(dst *ebiten.Image, f figure, a *patientAnim, fp image.Rectangle, d dir) {
	frame := frameAt(a.anim, a.ticks)
	drawLayers(dst, patientLayers(f.look, a.anim, frame), riderPlacement(fp, d))
}

// drawDoctorSprite dibuja al médico (su hoja y el pelo encima) en el cuadro
// actual de su animación. Nunca va en una cama: solo se voltea si camina a
// la derecha.
func drawDoctorSprite(dst *ebiten.Image, f figure, at vec, a *staffAnim) {
	if a == nil {
		a = &staffAnim{} // sin animación todavía: quieto
	}
	frame := frameAt(a.anim, a.ticks)
	g, _ := spritePlacement(a.anim, frame, vec{}, false, at, a.flip)
	drawLayers(dst, doctorLayers(f.look, a.anim, frame), g)
}

// drawStanding: personaje de pie (cabeza, tronco y piernas) con contorno.
// Las medidas están en píxeles de la hoja (un cuadro de 16×24) y se
// escalan como los sprites, así el personal mide lo mismo que los pacientes.
// Lo usa el camillero mientras llega su sprite.
func drawStanding(dst *ebiten.Image, x, y int, body, legs color.RGBA) {
	at := image.Pt(x, y)
	fillRect(dst, scaleRect(image.Rect(3, 2, 13, 23)).Add(at), colInk) // contorno
	fillRect(dst, scaleRect(image.Rect(4, 3, 12, 9)).Add(at), colSkin) // cabeza
	fillRect(dst, scaleRect(image.Rect(4, 10, 12, 17)).Add(at), body)  // tronco
	fillRect(dst, scaleRect(image.Rect(4, 18, 12, 22)).Add(at), legs)  // piernas
}

// toVec convierte un punto entero de la pantalla en un vec.
func toVec(p image.Point) vec { return vec{float64(p.X), float64(p.Y)} }
