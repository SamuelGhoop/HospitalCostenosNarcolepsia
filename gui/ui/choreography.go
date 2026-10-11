package ui

import "image"

// helperHold: cuántos ticks se queda quien atiende al lado de un paciente
// que no consiguió cama, antes de volver a su sala (60 ticks = 1 s).
const helperHold = 60

// phase es un momento de la coreografía de un paso, como una escena de un
// guion: algunos personajes caminan a un sitio. La fase termina cuando
// todos llegaron, el paciente indicado ya está acostado (si se pidió) y
// pasaron hold ticks más.
type phase struct {
	moves     map[string]vec // a dónde camina cada personaje, por ID
	waitLying string         // ID del paciente que debe terminar de desplomarse ("" = no esperar)
	attend    string         // ID del médico que atiende al paciente en esta fase ("" = nadie)
	carry     *carryOrder    // traslado en camilla de esta fase (nil = ninguno; lo hace la escena)
	hold      int            // ticks de espera al terminar
	delays    map[string]int // ticks que espera cada uno antes de salir (la llegada escalonada)
	waitAt    map[string]vec // dónde espera mientras tanto
}

// carryOrder es un traslado en camilla dentro de una fase. La fase dice a
// dónde termina cada uno (quien empuja junto a la cama y el paciente en la
// cama); el cómo (ir por la camilla, subirlo, empujar, pasarlo a la cama)
// lo hace la escena (carry.go).
type carryOrder struct {
	pusher, patient string
	room            zone
	plan            carryPlan
}

// carryPhase arma la fase de un traslado en camilla del paciente acostado
// en patientAt hasta la cama de room, sin pisar los pies de los que están
// quietos (avoid). Si así no hay ruta, se planea sin ellos; y si tampoco
// (no debería pasar: lo revisa un test), el paciente va derecho a la cama.
func carryPhase(patient, pusher string, patientAt vec, room zone, avoid []image.Rectangle) phase {
	plan, ok := planCarry(patientAt, room, avoid)
	if !ok {
		plan, ok = planCarry(patientAt, room, nil)
	}
	if !ok {
		return phase{moves: map[string]vec{patient: bedCell(room), pusher: toVec(carrySpot(room))}}
	}
	return phase{
		moves: map[string]vec{patient: bedCell(room), pusher: plan.anchor()},
		carry: &carryOrder{pusher: pusher, patient: patient, room: room, plan: plan},
	}
}

// choreography es la lista de fases que se reproducen una tras otra.
//
// No decide nada del hospital: quién atiende y a qué cama va lo dijo el
// modelo (DemoStep.Episode). Aquí solo se decide CÓMO se ve.
type choreography struct {
	phases  []phase
	i       int // fase actual
	held    int // ticks esperados en la fase actual
	elapsed int // ticks desde que empezó la fase actual
}

// done dice si ya se reprodujeron todas las fases.
func (c *choreography) done() bool { return c.i >= len(c.phases) }

// target dice a dónde debe ir id según la coreografía: el sitio de la
// última fase (hasta la actual) que lo mueve. Así, cuando alguien llega, se
// queda ahí hasta que otra fase lo vuelva a mover. ok es false si la
// coreografía no lo ha movido todavía.
func (c *choreography) target(id string) (to vec, ok bool) {
	if !c.done() {
		if p := c.phases[c.i]; c.elapsed < p.delays[id] {
			return p.waitAt[id], true // todavía no le toca salir
		}
	}
	last := c.i
	if last >= len(c.phases) {
		last = len(c.phases) - 1
	}
	for i := last; i >= 0; i-- {
		if to, ok := c.phases[i].moves[id]; ok {
			return to, true
		}
	}
	return vec{}, false
}

// step avanza la coreografía un tick. arrived dice si un personaje ya llegó
// a su destino; lying, si un paciente ya está acostado.
func (c *choreography) step(arrived, lying func(id string) bool) {
	if c.done() {
		return
	}
	c.elapsed++
	p := c.phases[c.i]
	for id := range p.moves {
		if !arrived(id) {
			return
		}
	}
	if p.waitLying != "" && !lying(p.waitLying) {
		return
	}
	c.held++
	if c.held >= p.hold {
		c.i++
		c.held, c.elapsed = 0, 0
	}
}

// attendChoreography es la coreografía de un episodio (pasos 5–8):
//
//  1. el paciente camina hasta donde le dio el ataque y se desploma;
//  2. quien lo atendió (el que eligió el round-robin del modelo) camina
//     desde la sala del personal hasta él y lo atiende: attendHold ticks
//     a su lado (los de la animación de atender si es médico; 0 si es el
//     camillero, que todavía no tiene sprite);
//  3. con cama: lo lleva a la habitación en camilla;
//     sin cama: se queda a su lado un momento;
//  4. vuelve a la sala del personal.
//
// La fase 1 fija a TODOS los que participan: si no, la escena mandaría a
// cada uno directo a su puesto final del modelo (por ejemplo, el paciente
// a su cama antes de que lo atiendan).
func attendChoreography(patient, attender string, attackAt, staffHome vec, room zone, hasBed bool, attendHold int, avoid []image.Rectangle) *choreography {
	phases := []phase{
		{moves: map[string]vec{patient: attackAt, attender: staffHome}, waitLying: patient},
		{moves: map[string]vec{attender: helperSpot(attackAt)}, attend: attender, hold: attendHold},
	}
	if hasBed {
		phases = append(phases, carryPhase(patient, attender, attackAt, room, avoid))
	} else {
		phases = append(phases, phase{hold: helperHold})
	}
	phases = append(phases, phase{moves: map[string]vec{attender: staffHome}})
	return &choreography{phases: phases}
}

// carryChoreography es la del paso 11 (AssignRoom): el modelo no dice quién
// lleva al paciente, así que lo lleva el camillero (GAME_DESIGN §4.1). Va
// hasta él (el paciente espera quieto en el pasillo), lo lleva a la cama en
// camilla y vuelve.
func carryChoreography(patient, carrier string, patientAt, carrierHome vec, room zone, avoid []image.Rectangle) *choreography {
	return &choreography{phases: []phase{
		{moves: map[string]vec{carrier: helperSpot(patientAt), patient: patientAt}},
		carryPhase(patient, carrier, patientAt, room, avoid),
		{moves: map[string]vec{carrier: carrierHome}},
	}}
}

// carrying es el traslado en camilla de la fase actual (nil si no hay).
func (c *choreography) carrying() *carryOrder {
	if c.done() {
		return nil
	}
	return c.phases[c.i].carry
}

// attending es el ID de quien atiende a un paciente en la fase actual ("" si
// nadie): al parar a su lado hace la animación de atender.
func (c *choreography) attending() string {
	if c.done() {
		return ""
	}
	return c.phases[c.i].attend
}
