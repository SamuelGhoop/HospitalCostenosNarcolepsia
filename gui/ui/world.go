package ui

import (
	"image"
	"math"
)

// allowOverlapPx: cuánto se pueden rozar dos cuerpos quietos sin que el
// test de que nadie se para encima de otro (overlap_test.go) lo cuente como
// un choque real. "Unos pocos píxeles",
// como pidió Samuel el 2026-10-17: ni cero (un roce al cruzarse no es un
// bug) ni demasiado (uno encima del otro sí lo es).
const allowOverlapPx = 4

// bodiesClash dice si a y b se solapan más de lo que se tolera.
func bodiesClash(a, b image.Rectangle) bool {
	return a.Inset(allowOverlapPx).Overlaps(b.Inset(allowOverlapPx))
}

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
	rest    map[string]restSpot     // dónde se queda quieto cada uno al terminar su ruta (ver settle)
}

// restSpot: para el destino dest que le pidió la escena, el sitio at donde
// de verdad se queda quieto. Casi siempre at == dest; es otro cuando dest
// quedaba encima de alguien que ya estaba quieto ahí (settle).
type restSpot struct{ dest, at vec }

func newWorld() world {
	return world{pos: map[string]vec{}, paths: map[string][]vec{}, pathTo: map[string]vec{},
		anims: map[string]*patientAnim{}, staff: map[string]*staffAnim{}, dodge: map[string]vec{},
		rest: map[string]restSpot{}}
}

// nextWaypoint es el próximo punto al que camina id para llegar a dest.
// Si dest cambió (otra fase de la coreografía, otro puesto), calcula la
// ruta de nuevo; cuando llega a un punto de paso, sigue con el siguiente.
//
// Regla de choques (pedido de Samuel, 2026-10-19): quien CAMINA no tiene
// cuerpo sólido para los demás; pasa de largo junto a otro (o a través de
// él) sin esperarlo. Antes cada caminante reservaba la celda de adelante y
// esperaba (o rodeaba) al que la ocupaba, pero en los pasillos angostos del
// mapa muchas veces no había otro camino y se quedaba trabado para siempre.
// Lo que sí es sólido es estar QUIETO: los puestos donde cada uno se queda
// (spotBook, spots.go) nunca se reparten encimados, así que dos quietos no
// terminan uno encima del otro (TestNobodyStandsOnAnother_*,
// overlap_test.go). La camilla de otro sí se esquiva al calcular la ruta
// (avoidFor): es un mueble que se mueve, no una persona.
func (w *world) nextWaypoint(id string, from, dest vec) vec {
	if math.Hypot(dest.x-from.x, dest.y-from.y) <= settleDistance {
		w.settle(id, dest)
	}
	to := w.restSpotFor(id, dest)
	if from == to {
		return to // ya llegó (por ejemplo, acostado en su cama: ahí no hay piso libre)
	}
	if w.pathTo[id] != to || len(w.paths[id]) == 0 {
		w.paths[id], w.pathTo[id] = routeAvoiding(from, to, w.avoidFor(id)), to
	}
	path := w.paths[id]
	for len(path) > 1 && from == path[0] {
		path = path[1:]
	}
	w.paths[id] = path
	return path[0]
}

// settleDistance: a cuántos píxeles de su destino decide cada uno dónde se
// queda quieto (settle). Cerca del final, para mirar a los que YA están
// quietos ahí cuando llega, no a los que estaban al salir.
const settleDistance = 3 * navCell

// restSpotFor es dónde se queda quieto id al terminar su ruta hacia dest:
// dest mismo, o el sitio libre que le buscó settle.
func (w *world) restSpotFor(id string, dest vec) vec {
	if r, ok := w.rest[id]; ok && r.dest == dest {
		return r.at
	}
	return dest
}

// arrived dice si id ya terminó su ruta hacia dest (llegó a dest o al sitio
// libre de al lado que le tocó, settle). Las escenas preguntan esto en vez
// de comparar la posición con dest a secas.
func (w *world) arrived(id string, dest vec) bool {
	return w.pos[id] == w.restSpotFor(id, dest)
}

// settle decide, una sola vez por destino, dónde se queda quieto id al
// terminar su ruta (corrección de Samuel, 2026-10-19): caminando nadie
// choca (nextWaypoint), pero al llegar nadie queda encima de otro. Si en
// dest su cuerpo se encimaría con el de alguien quieto (o con el sitio al
// que va otro que todavía camina), busca el sitio libre más cercano
// (dodgeSpot, carry.go); si no hay ninguno, se queda con dest.
//
// Quien empuja o va en camilla no se aparta (la camilla necesita el punto
// exacto): son los quietos de ahí los que se hacen a un lado
// (clearSpotFor). Tampoco aplica al que va a su cama (ahí sí puede estar
// quien lo revisa). En la acera, fuera de las zonas, hacen fila.
func (w *world) settle(id string, dest vec) {
	if r, ok := w.rest[id]; ok && r.dest == dest {
		return // ya se decidió para este destino
	}
	w.rest[id] = restSpot{dest, dest}
	me, _ := w.figure(id)
	if w.inCarry(id) {
		w.clearSpotFor(id, dest)
		return
	}
	if me.patient && me.room != 0 && dest == bedCell(roomZone(me.room)) {
		return
	}
	z, ok := zoneAt(dest.add(feetOffset))
	onSidewalk := !ok || z == outside
	var taken []image.Rectangle
	for _, f := range w.figures {
		if f.id == id || (f.patient && w.onBed(f)) {
			continue
		}
		at := w.pos[f.id]
		if len(w.paths[f.id]) > 0 && at != w.pathTo[f.id] {
			// Caminando: no estorba donde está ahora, pero sí donde va a
			// quedarse (por ejemplo, quien va a recoger una camilla, que
			// necesita su punto exacto).
			at = w.pathTo[f.id]
		}
		taken = append(taken, visibleBody(f, at))
	}
	mine := visibleBody(me, dest)
	clash := false
	for _, b := range taken {
		clash = clash || bodiesClash(mine, b)
	}
	if !clash {
		return
	}
	if onSidewalk {
		// En la acera (por ejemplo, varios que se van por la misma puerta):
		// no hay zonas ni muebles; hacen fila hacia la derecha, hacia donde
		// después se van caminando.
		for k := 1; k <= 10; k++ {
			spot := dest.add(vec{float64(k) * arrivalQueueGap, 0})
			if !overlapsAny(visibleBody(me, spot), taken) {
				w.rest[id] = restSpot{dest, spot}
				return
			}
		}
		return
	}
	if spot, ok := dodgeSpot(dest, nil, taken); ok {
		w.rest[id] = restSpot{dest, spot}
	}
}

// clearSpotFor: quien va en un traslado (empujando o encima) necesita
// quedarse justo en dest, así que los que ya estaban quietos ahí, con el
// cuerpo encimado, se hacen al sitio libre más cercano (dodgeSpot), igual
// que cuando pasa la camilla. Siguen teniendo el mismo destino de la
// escena; solo cambia el sitio donde se quedan (rest).
func (w *world) clearSpotFor(id string, dest vec) {
	me, _ := w.figure(id)
	mine := visibleBody(me, dest)
	for _, f := range w.figures {
		if f.id == id || w.inCarry(f.id) || (f.patient && w.onBed(f)) {
			continue
		}
		at := w.pos[f.id]
		if len(w.paths[f.id]) > 0 && at != w.pathTo[f.id] {
			continue // caminando: su propio settle lo resuelve al llegar
		}
		if !bodiesClash(mine, visibleBody(f, at)) {
			continue
		}
		var taken []image.Rectangle
		for _, o := range w.figures {
			if o.id != f.id && !(o.patient && w.onBed(o)) {
				taken = append(taken, visibleBody(o, w.pos[o.id]))
			}
		}
		taken = append(taken, mine)
		spot, ok := dodgeSpot(at, nil, taken)
		if !ok {
			continue
		}
		sceneDest := at
		if r, ok := w.rest[f.id]; ok {
			sceneDest = r.dest
		}
		w.rest[f.id] = restSpot{sceneDest, spot}
	}
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
