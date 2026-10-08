package ui

import "math"

// Las animaciones avanzan por ticks de Update (60 por segundo), nunca con
// time.Sleep: así la ventana nunca se congela esperando.

// autoAdvanceTicks: 3 s a 60 ticks por segundo (tecla A de la demo).
const autoAdvanceTicks = 180

// autoAdvance cuenta ticks para el avance automático de la demo.
type autoAdvance struct {
	on    bool
	ticks int
}

// toggle prende o apaga el avance automático y reinicia la cuenta.
func (a *autoAdvance) toggle() {
	a.on = !a.on
	a.ticks = 0
}

// reset vuelve a contar desde cero (por ejemplo, si se avanzó a mano).
func (a *autoAdvance) reset() { a.ticks = 0 }

// tick cuenta un tick y dice si ya toca avanzar al siguiente paso.
func (a *autoAdvance) tick() bool {
	if !a.on {
		return false
	}
	a.ticks++
	if a.ticks >= autoAdvanceTicks {
		a.ticks = 0
		return true
	}
	return false
}

// vec es una posición en la pantalla con decimales, para que un personaje
// se pueda deslizar de a poco entre dos puestos.
type vec struct{ x, y float64 }

// moveToward acerca p a target como máximo speed píxeles, en línea recta.
// Si ya está más cerca que eso, llega exactamente a target.
func moveToward(p, target vec, speed float64) vec {
	dx, dy := target.x-p.x, target.y-p.y
	dist := math.Hypot(dx, dy)
	if dist <= speed {
		return target
	}
	return vec{p.x + dx/dist*speed, p.y + dy/dist*speed}
}
