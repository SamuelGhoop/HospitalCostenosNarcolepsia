package ui

// staffAnim recuerda la animación de un médico: quieto, caminando o
// atendiendo. Como la del paciente, avanza tick a tick según cuánto se
// movió (step).
type staffAnim struct {
	anim     animation
	ticks    int  // ticks desde que empezó la animación actual
	flip     bool // caminando hacia la derecha
	attended bool // ya atendió en esta parada (se borra al volver a caminar)
}

// set arranca una animación desde su primer cuadro.
func (a *staffAnim) set(an animation) { a.anim, a.ticks = an, 0 }

// keep pone una animación que se repite, sin reiniciarla si ya iba en ella.
func (a *staffAnim) keep(an animation) {
	if a.anim != an {
		a.set(an)
	}
}

// step avanza un tick. dx, dy es cuánto se movió en este tick; attending
// dice si la coreografía lo tiene atendiendo a un paciente (solo médicos) y
// pushing si está empujando una camilla (solo camilleros; con un médico
// empujando porque no había camillero libre, sigue caminando: su hoja no
// tiene fila de empujar). Al parar junto al paciente hace atender una vez y
// después queda quieto.
func (a *staffAnim) step(dx, dy float64, attending, pushing bool) {
	a.ticks++
	moving := dx != 0 || dy != 0
	switch {
	case moving && pushing:
		a.attended = false
		a.keep(animPush)
		a.flip = dx > 0 // la fila de empujar es "hacia la izquierda"; a la derecha se voltea
	case moving:
		a.attended = false // en la próxima parada puede volver a atender
		an, flip := walkAnimation(dx, dy)
		a.keep(an)
		a.flip = flip
	case attending && !a.attended:
		a.attended = true
		a.flip = false
		a.set(animAttend)
	case a.anim == animAttend && !finished(animAttend, a.ticks):
		// que termine de atender
	default:
		a.flip = false
		a.keep(animIdle)
	}
}
