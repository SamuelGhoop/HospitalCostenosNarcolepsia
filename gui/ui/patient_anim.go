package ui

import "github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"

// patientAnim recuerda qué animación está haciendo un paciente y desde
// cuándo. Hace falta porque la foto del modelo solo dice "está despierto",
// no "se acaba de despertar": las transiciones se detectan comparando la
// foto nueva con la anterior (observe) y se animan tick a tick (step).
type patientAnim struct {
	anim            animation
	ticks           int                   // ticks desde que empezó la animación actual
	flip            bool                  // caminando hacia la derecha
	state           hospital.PatientState // estado en la última foto
	room            int                   // última cama en la que durmió (para el cuadro 1 de despertarse)
	pendingCollapse bool                  // se durmió fuera de la cama: primero camina hasta el sitio
}

// showsBubble dice si toca dibujar la burbuja Zzz: solo cuando ya está
// acostado. El modelo lo marca dormido desde que le da el ataque, pero en
// pantalla primero camina al sitio y se desploma.
func (a *patientAnim) showsBubble() bool {
	return a.anim == animSleepFloor || a.anim == animSleepBed
}

// set arranca una animación desde su primer cuadro.
func (a *patientAnim) set(an animation) {
	a.anim, a.ticks = an, 0
}

// keep pone una animación que se repite, sin reiniciarla si ya iba en ella.
func (a *patientAnim) keep(an animation) {
	if a.anim != an {
		a.set(an)
	}
}

// observe compara el estado nuevo del modelo con el anterior y arranca la
// animación de transición que toque. Devuelve true si el paciente se acaba
// de levantar de la cama: la escena lo pone de una vez al lado de la cama.
func (a *patientAnim) observe(state hospital.PatientState, room int) (leftBed bool) {
	was := a.state
	a.state = state
	if room != 0 {
		a.room = room
	}

	switch {
	case was == hospital.Awake && state == hospital.AsleepInBed:
		a.set(animCollapse) // se desploma y lo llevan cargado a su cama
	case was == hospital.Awake && state == hospital.AsleepInHallway:
		a.pendingCollapse = true // camina hasta donde le dio el ataque y ahí se desploma
	case was != hospital.Awake && state == hospital.Awake:
		a.set(animWakeUp)
		leftBed = was == hospital.AsleepInBed
	}
	return leftBed
}

// step avanza un tick. dx, dy es cuánto se movió el paciente en este tick;
// inBed dice si el modelo le tiene asignada una cama.
func (a *patientAnim) step(dx, dy float64, inBed bool) {
	a.ticks++
	moving := dx != 0 || dy != 0

	// Las animaciones que no se repiten se dejan terminar.
	switch a.anim {
	case animWakeUp:
		if finished(a.anim, a.ticks) {
			a.set(animWave) // en la demo: se despierta y saluda una vez
		}
		return
	case animWave:
		if finished(a.anim, a.ticks) {
			a.set(animIdle) // y se queda quieto al lado de la cama
		}
		return
	case animCollapse:
		if !finished(a.anim, a.ticks) {
			return
		}
		// Terminó de desplomarse: sigue con la animación de dormido, abajo.
	}

	// Se durmió fuera de la cama: al llegar al sitio, se desploma.
	if a.pendingCollapse && !moving {
		a.pendingCollapse = false
		a.set(animCollapse)
		return
	}

	switch {
	case a.state != hospital.Awake && !a.pendingCollapse:
		a.flip = false
		if inBed && !moving {
			a.keep(animSleepBed)
		} else {
			a.keep(animSleepFloor) // en el piso, o mientras lo cargan a la cama
		}
	case moving:
		an, flip := walkAnimation(dx, dy)
		a.keep(an)
		a.flip = flip
	default:
		a.flip = false
		a.keep(animIdle)
	}
}
