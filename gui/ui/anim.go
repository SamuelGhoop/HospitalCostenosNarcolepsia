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

// ─── Portapapeles animado (pausa, bolsa de empleo, fin de día, Game Over) ──
//
// Pedido de Samuel el 2026-10-17: "hazle así animados los menús como el de
// contratar que me gustó mucho" (la bolsa de empleo, hiring_scene.go). Las
// fórmulas de abrir (ease-out) y cerrar (ease-in) y la transformación de
// escala y centro son las mismas que ya usaba la bolsa; se factorizan acá
// para que los demás menús (pausa, fin de día, Game Over) las reusen en vez
// de copiarlas.

// easeOutOpenness: 1 − (1 − t)³, con t de 0 a 1 según los ticks que ya
// pasaron de total. Usada para ABRIR (arranca rápido y frena al final).
func easeOutOpenness(ticks, total int) float64 {
	t := float64(ticks) / float64(total)
	return 1 - math.Pow(1-t, 3)
}

// easeInOpenness: 1 − t³. Usada para CERRAR (arranca lento y acelera).
func easeInOpenness(ticks, total int) float64 {
	t := float64(ticks) / float64(total)
	return 1 - math.Pow(t, 3)
}

// clipboardTransform da la escala y el centro de un panel con esa openness
// o (0 cerrado, del tamaño y en el sitio de from; 1 abierto, a su tamaño,
// centrado en to): interpola linealmente el centro entre from y to y la
// escala entre minScale y 1.
func clipboardTransform(o, minScale float64, from, to vec) (scale float64, center vec) {
	scale = minScale + (1-minScale)*o
	center = vec{from.x + (to.x-from.x)*o, from.y + (to.y-from.y)*o}
	return
}

// panelPhase es en qué momento de abrir/cerrar está un panel animado.
type panelPhase int

const (
	panelOpening panelPhase = iota // apareciendo desde from
	panelOpen                      // quieto, en to: acepta entrada
	panelClosing                   // volviendo a from
)

// animPanel es la animación de abrir/cerrar de un panel tipo
// portapapeles: mientras no está en panelOpen, no acepta entrada (igual
// que la bolsa de empleo). from y to son los centros cerrado y abierto, en
// las unidades que use quien dibuja (normalmente píxeles de pantalla).
type animPanel struct {
	phase                 panelPhase
	ticks                 int
	openTicks, closeTicks int
	from, to              vec
	minScale              float64
}

// newAnimPanel arranca abriendo desde from hasta to.
func newAnimPanel(openTicks, closeTicks int, from, to vec, minScale float64) *animPanel {
	return &animPanel{phase: panelOpening, openTicks: openTicks, closeTicks: closeTicks, from: from, to: to, minScale: minScale}
}

// step avanza un tick la animación (abriendo o cerrando; en panelOpen no
// hace nada). Devuelve true en el tick en que termina de cerrar: ahí es
// cuando quien la usa hace la acción que estaba esperando (volver al
// menú, empezar partida nueva, pasar al día siguiente...).
func (a *animPanel) step() (finishedClosing bool) {
	switch a.phase {
	case panelOpening:
		a.ticks++
		if a.ticks >= a.openTicks {
			a.phase, a.ticks = panelOpen, 0
		}
	case panelClosing:
		a.ticks++
		if a.ticks >= a.closeTicks {
			return true
		}
	}
	return false
}

// startClose arranca el cierre (si no estaba ya cerrando).
func (a *animPanel) startClose() {
	if a.phase != panelClosing {
		a.phase, a.ticks = panelClosing, 0
	}
}

// acceptsInput: solo quieto y abierto se le hace caso al teclado o al mouse.
func (a *animPanel) acceptsInput() bool { return a.phase == panelOpen }

// openness: 0 cerrado, 1 abierto (ease-out al abrir, ease-in al cerrar).
func (a *animPanel) openness() float64 {
	switch a.phase {
	case panelOpening:
		return easeOutOpenness(a.ticks, a.openTicks)
	case panelClosing:
		return easeInOpenness(a.ticks, a.closeTicks)
	default:
		return 1
	}
}

// transform da la escala y el centro actuales (clipboardTransform con la
// openness de ahora).
func (a *animPanel) transform() (scale float64, center vec) {
	return clipboardTransform(a.openness(), a.minScale, a.from, a.to)
}

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
