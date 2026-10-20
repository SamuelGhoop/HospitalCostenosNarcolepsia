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
	waveOnArrival   bool                  // se acaba de despertar: saluda cuando llegue a su sitio
	drowsy          bool                  // cabecea: en el juego, los 2 s antes del ataque (Stage Drowsy)
}

// holdsStill dice si el paciente no se puede mover todavía: mientras se
// despierta (en la cama, se sienta y se para) se queda donde estaba.
func (a *patientAnim) holdsStill() bool { return a.anim == animWakeUp }

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
// animación de transición que toque.
func (a *patientAnim) observe(state hospital.PatientState, room int) {
	was := a.state
	a.state = state
	if room != 0 {
		a.room = room
	}

	switch {
	case was == hospital.Awake && state != hospital.Awake:
		// Camina hasta donde le dio el ataque y ahí se desploma. Si le tocó
		// cama, después lo llevan cargado (lo dirige la coreografía).
		a.pendingCollapse = true
	case was != hospital.Awake && state == hospital.Awake:
		a.set(animWakeUp) // se despierta donde está; después camina a su sitio
	}
}

// settle deja al paciente de una vez en la animación de reposo de su estado.
// Se usa cuando el usuario adelanta la coreografía. Si se estaba despertando
// o saludando, lo deja terminar.
func (a *patientAnim) settle(inBed bool) {
	a.pendingCollapse = false
	a.flip = false
	switch {
	case a.state != hospital.Awake && inBed:
		a.set(animSleepBed)
	case a.state != hospital.Awake:
		a.set(animSleepFloor)
	case a.anim == animWakeUp || a.anim == animWave:
		// que termine de despertarse y de saludar
	default:
		a.set(animIdle)
	}
}

// step avanza un tick. dx, dy es cuánto se movió el paciente en este tick;
// arrived dice si YA está en su destino de verdad (no solo quieto este
// tick): quedarse quieto un tick a mitad de camino no es haber llegado al
// sitio del ataque, y desplomarse ahí podía dejarlo encima de un mueble
// (TestDemo_NobodyWalksThroughWallsOrFurniture). inBed dice si ya está
// sobre su cama (no solo si el modelo se la asignó: mientras lo cargan
// hacia ella todavía no lo está).
func (a *patientAnim) step(dx, dy float64, arrived, inBed bool) {
	a.ticks++
	moving := dx != 0 || dy != 0

	// Las animaciones que no se repiten se dejan terminar.
	switch a.anim {
	case animWakeUp:
		if finished(a.anim, a.ticks) {
			a.set(animIdle) // ya de pie: camina a su sitio (el costado de la cama)
			a.waveOnArrival = true
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
	if a.pendingCollapse && !moving && arrived {
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
	case a.drowsy:
		a.flip = false
		a.keep(animDrowsy) // quieto, cabeceando
	case a.waveOnArrival:
		a.waveOnArrival = false
		a.flip = false
		a.set(animWave) // en la demo: al llegar a su sitio saluda una vez
	default:
		a.flip = false
		a.keep(animIdle)
	}
}
