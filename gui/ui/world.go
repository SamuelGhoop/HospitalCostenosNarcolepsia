package ui

import "image"

// world es lo que tienen en común la demo y el modo juego: dónde está cada
// personaje, por dónde camina, sus animaciones y el traslado en camilla en
// curso. Cada escena lo incorpora (embebido) y decide A DÓNDE va cada uno:
// la demo, con su coreografía; el juego, con la foto de game/.
type world struct {
	figures []figure
	pos     map[string]vec          // posición dibujada de cada personaje, por ID
	paths   map[string][]vec        // lo que le falta de ruta a cada personaje (ver routes.go)
	pathTo  map[string]vec          // el destino para el que se calculó esa ruta
	anims   map[string]*patientAnim // animación de cada paciente, por ID
	staff   map[string]*staffAnim   // animación de cada médico, por ID
	carry   *carrying               // traslado en camilla en curso (nil si no hay)
	others  []*carrying             // en el juego, los demás traslados en curso (puede haber varios)
	dodge   map[string]vec          // los que se hicieron a un lado para dejar pasar la camilla
}

func newWorld() world {
	return world{pos: map[string]vec{}, paths: map[string][]vec{}, pathTo: map[string]vec{},
		anims: map[string]*patientAnim{}, staff: map[string]*staffAnim{}}
}

// nextWaypoint es el próximo punto al que camina id para llegar a dest.
// Si dest cambió (otra fase de la coreografía, otro puesto), calcula la
// ruta de nuevo; cuando llega a un punto de paso, sigue con el siguiente.
func (w *world) nextWaypoint(id string, from, dest vec) vec {
	if from == dest {
		return dest // ya llegó (por ejemplo, acostado en su cama: ahí no hay piso libre)
	}
	if w.pathTo[id] != dest || len(w.paths[id]) == 0 {
		w.paths[id], w.pathTo[id] = routeAvoiding(from, dest, w.avoidFor(id)), dest
	}
	path := w.paths[id]
	for len(path) > 1 && from == path[0] {
		path = path[1:]
	}
	w.paths[id] = path
	return path[0]
}

// onBed dice si un paciente ya está SOBRE su cama (no solo asignada: la
// cama se asigna en el modelo antes de que lo terminen de llevar).
func (w *world) onBed(f figure) bool {
	return f.room != 0 && w.pos[f.id] == bedCell(roomZone(f.room))
}

// figure busca un personaje por ID.
func (w *world) figure(id string) (figure, bool) {
	for _, f := range w.figures {
		if f.id == id {
			return f, true
		}
	}
	return figure{}, false
}

// allCarries son los traslados en curso: el de la demo y los del juego.
func (w *world) allCarries() []*carrying {
	if w.carry == nil {
		return w.others
	}
	return append([]*carrying{w.carry}, w.others...)
}

// carryOf es el traslado en el que va id (empujando o encima), o nil.
func (w *world) carryOf(id string) *carrying {
	for _, c := range w.allCarries() {
		if id == c.order.pusher || id == c.order.patient {
			return c
		}
	}
	return nil
}

// inCarry dice si id es quien empuja o el paciente de un traslado en curso.
func (w *world) inCarry(id string) bool { return w.carryOf(id) != nil }

// avoidFor es lo que tiene que esquivar id al calcular su ruta: las camillas
// de los otros (quien la empuja y el paciente no se la esquivan a sí mismos).
func (w *world) avoidFor(id string) []image.Rectangle {
	var out []image.Rectangle
	for _, c := range w.allCarries() {
		if c.visible() && id != c.order.pusher && id != c.order.patient {
			out = append(out, c.fp)
		}
	}
	return out
}
