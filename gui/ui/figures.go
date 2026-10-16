package ui

import (
	"image"

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
	look    game.Appearance // cómo se ve
	target  vec             // puesto final según la foto del modelo (sin coreografía)
}

// drawFigureBody dibuja el cuerpo de un personaje en su cuadro de 32×48
// con esquina en at. Los rótulos (burbuja, etiqueta…) van aparte, en
// figureOverlays, porque se dibujan encima de todo (incluida la cobija).
// pa es la animación del paciente y sa la del médico (nil si no aplica).
func drawFigureBody(dst *ebiten.Image, f figure, at vec, pa *patientAnim, sa *staffAnim) {
	switch {
	case f.patient:
		drawPatientSprite(dst, f, at, pa)
	case f.doctor:
		drawDoctorSprite(dst, f, at, sa)
	default:
		drawOrderlySprite(dst, f, at, sa)
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
	drawRiderWith(dst, f, a, riderPlacement(fp, d))
}

// drawRiderWith dibuja al paciente acostado con la GeoM g (la de la camilla
// girando, por ejemplo).
func drawRiderWith(dst *ebiten.Image, f figure, a *patientAnim, g ebiten.GeoM) {
	frame := frameAt(a.anim, a.ticks)
	drawLayers(dst, patientLayers(f.look, a.anim, frame), g)
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

// drawOrderlySprite dibuja al camillero (su hoja y el pelo encima) en el
// cuadro actual de su animación: quieto, caminando o empujando la camilla
// (fila 5 de su hoja). Igual que el médico, solo se voltea caminando a la
// derecha o empujando hacia la derecha.
func drawOrderlySprite(dst *ebiten.Image, f figure, at vec, a *staffAnim) {
	if a == nil {
		a = &staffAnim{} // sin animación todavía: quieto
	}
	frame := frameAt(a.anim, a.ticks)
	g, _ := spritePlacement(a.anim, frame, vec{}, false, at, a.flip)
	drawLayers(dst, orderlyLayers(f.look, a.anim, frame), g)
}

// toVec convierte un punto entero de la pantalla en un vec.
func toVec(p image.Point) vec { return vec{float64(p.X), float64(p.Y)} }
