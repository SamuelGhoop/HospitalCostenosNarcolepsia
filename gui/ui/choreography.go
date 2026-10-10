package ui

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
	hold      int            // ticks de espera al terminar
}

// choreography es la lista de fases que se reproducen una tras otra.
//
// No decide nada del hospital: quién atiende y a qué cama va lo dijo el
// modelo (DemoStep.Episode). Aquí solo se decide CÓMO se ve.
type choreography struct {
	phases []phase
	i      int // fase actual
	held   int // ticks esperados en la fase actual
}

// done dice si ya se reprodujeron todas las fases.
func (c *choreography) done() bool { return c.i >= len(c.phases) }

// target dice a dónde debe ir id según la coreografía: el sitio de la
// última fase (hasta la actual) que lo mueve. Así, cuando alguien llega, se
// queda ahí hasta que otra fase lo vuelva a mover. ok es false si la
// coreografía no lo ha movido todavía.
func (c *choreography) target(id string) (to vec, ok bool) {
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
		c.held = 0
	}
}

// attendChoreography es la coreografía de un episodio (pasos 5–8):
//
//  1. el paciente camina hasta donde le dio el ataque y se desploma;
//  2. quien lo atendió (el que eligió el round-robin del modelo) camina
//     desde la sala del personal hasta él;
//  3. con cama: lo lleva a la habitación (el paciente va acostado);
//     sin cama: se queda a su lado un momento;
//  4. vuelve a la sala del personal.
//
// La fase 1 fija a TODOS los que participan: si no, la escena mandaría a
// cada uno directo a su puesto final del modelo (por ejemplo, el paciente
// a su cama antes de que lo atiendan).
func attendChoreography(patient, attender string, attackAt, staffHome vec, room zone, hasBed bool) *choreography {
	phases := []phase{
		{moves: map[string]vec{patient: attackAt, attender: staffHome}, waitLying: patient},
		{moves: map[string]vec{attender: helperSpot(attackAt)}},
	}
	if hasBed {
		phases = append(phases, phase{moves: map[string]vec{
			patient:  bedCell(room),
			attender: toVec(carrySpot(room)),
		}})
	} else {
		phases = append(phases, phase{hold: helperHold})
	}
	phases = append(phases, phase{moves: map[string]vec{attender: staffHome}})
	return &choreography{phases: phases}
}

// carryChoreography es la del paso 11 (AssignRoom): el modelo no dice quién
// lleva al paciente, así que lo lleva el camillero (GAME_DESIGN §4.1). Va
// hasta él (el paciente espera quieto en el pasillo), lo lleva a la cama y
// vuelve.
func carryChoreography(patient, carrier string, patientAt, carrierHome vec, room zone) *choreography {
	return &choreography{phases: []phase{
		{moves: map[string]vec{carrier: helperSpot(patientAt), patient: patientAt}},
		{moves: map[string]vec{patient: bedCell(room), carrier: toVec(carrySpot(room))}},
		{moves: map[string]vec{carrier: carrierHome}},
	}}
}
