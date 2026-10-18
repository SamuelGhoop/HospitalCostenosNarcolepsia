package ui

import (
	"image"
	"math"
)

// allowOverlapPx: cuánto se pueden rozar dos cuerpos sin que la reserva de
// celdas (más abajo) ni el test de que nadie comparte cuerpo con otro
// (overlap_test.go) lo cuenten como un choque real. "Unos pocos píxeles",
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

	snapshotPos map[string]vec // posiciones al arrancar el tick (ver beginTick): la reserva de celdas no depende del orden en que se mueve cada uno
	stuckTicks  map[string]int // ticks que lleva cada uno esperando porque el próximo paso está ocupado
}

func newWorld() world {
	return world{pos: map[string]vec{}, paths: map[string][]vec{}, pathTo: map[string]vec{},
		anims: map[string]*patientAnim{}, staff: map[string]*staffAnim{}, dodge: map[string]vec{},
		snapshotPos: map[string]vec{}, stuckTicks: map[string]int{}}
}

// beginTick congela las posiciones de todos al arrancar el tick, antes de
// mover a nadie: la reserva de celdas (nextWaypoint) se calcula contra esta
// foto, así no importa en qué orden se recorra la lista de personajes (si
// no, el primero de la lista vería ya movido al segundo). Hay que llamarla
// una sola vez por tick, antes de animar a nadie.
func (w *world) beginTick() {
	for id := range w.snapshotPos {
		delete(w.snapshotPos, id)
	}
	for id, p := range w.pos {
		w.snapshotPos[id] = p
	}
}

// waitBeforeSidestep: cuántos ticks espera un personaje con el próximo paso
// ocupado antes de tratarlo como un obstáculo temporal y recalcular la
// ruta alrededor (unos 0,33 s a 60 ticks/s de Update), en vez de quedarse
// esperando para siempre (GAME_DESIGN §10.8, pedido de Samuel el
// 2026-10-17: "evita que los personajes... se puedan poner en el mismo
// lugar al mismo tiempo").
const waitBeforeSidestep = 20

// occupantOf dice quién reserva, al arrancar este tick, el lugar de next:
// su cuerpo (de pie, caminando) se vería encimado con el de otro personaje
// ahí (bodiesClash), parado en su sitio de ahora o camino al suyo (para que
// nadie entre a un lugar justo cuando otro va para allá). id nunca se
// cuenta a sí mismo. El que va en camilla no reserva por su cuenta (lo hace
// quien empuja, por el grupo); el que está acostado en su cama tampoco: ahí
// sí puede estar quien lo revisa. "" si el lugar está libre.
//
// Se mira el cuerpo entero (no solo la celda de los pies, más chica que el
// cuadro de 48×72 de un personaje) porque dos personajes en celdas
// distintas todavía pueden quedar con el cuerpo encimado al cruzarse en un
// pasillo angosto.
//
// occupancyMargin: de más, para frenar ANTES de tocarse de verdad. En
// teoría ayuda contra el caso en que dos se acercan el uno al otro (cada
// uno ve libre el lugar del otro porque mira su posición de ARRANQUE del
// tick, no la de llegada) y los dos siguen acercándose unos píxeles por
// tick hasta terminar encimados. Se probó con 3 y con 8 px: en los dos
// casos, en algún punto de la demo no quedaba ninguna celda libre para
// rodear al bloqueado (un pasillo angosto) y el respaldo de routeAvoiding
// ("no debería pasar") mandaba a alguien en línea recta sobre una pared.
// Por eso queda en 0 (sin margen de más): la salida es más segura que la
// entrada por el medio de un mueble. Pendiente: si Samuel quiere cerrar
// también ese caso (dos que se cruzan convergen un poco antes de que la
// reserva de celda los frene), hace falta que routeAvoiding no recurra a
// la línea recta cuando no hay camino ni esquivando.
const occupancyMargin = 0

func (w *world) occupantOf(id string, next vec) string {
	nextBody := standingBody.Add(image.Pt(int(math.Round(next.x)), int(math.Round(next.y)))).Inset(-occupancyMargin)
	clashesAt := func(f figure, at vec) bool { return nextBody.Overlaps(visibleBody(f, at).Inset(-occupancyMargin)) }
	for _, f := range w.figures {
		if f.id == id {
			continue
		}
		if c := w.carryOf(f.id); c != nil && c.order.patient == f.id {
			continue // en camilla: no reserva lugar por su cuenta
		}
		if f.patient && w.onBed(f) {
			continue // acostado en su cama: no reserva lugar
		}
		at, ok := w.snapshotPos[f.id]
		if !ok {
			at = w.pos[f.id]
		}
		if clashesAt(f, at) {
			return f.id
		}
		if len(w.paths[f.id]) > 0 && clashesAt(f, w.paths[f.id][0]) {
			return f.id // f.id ya va camino para allá: tampoco se le gana de mano
		}
	}
	return ""
}

// stillMoving dice si, al arrancar este tick, a id todavía le faltaba
// camino (no había llegado a su destino de entonces). Si ya llegó, no se
// va a apartar solo: hay que rodearlo.
func (w *world) stillMoving(id string) bool {
	at, ok := w.snapshotPos[id]
	return !ok || at != w.pathTo[id]
}

// nextWaypoint es el próximo punto al que camina id para llegar a dest.
// Si dest cambió (otra fase de la coreografía, otro puesto), calcula la
// ruta de nuevo; cuando llega a un punto de paso, sigue con el siguiente.
//
// Antes de entrar a una celda que ya ocupa otro personaje (o la camilla de
// otro), espera: nadie se pone en el mismo lugar que otro al mismo tiempo
// (GAME_DESIGN §10.8). Para que dos que se cruzan no se esperen el uno al
// otro para siempre, decide por ID: el que tiene prioridad (ID más chico)
// espera a que el otro se aparte solo; pasado un rato sin que se libere, el
// que no tiene prioridad la trata como un obstáculo temporal y recalcula la
// ruta por otro lado, como ya se hacía con la camilla de otro que tapaba el
// paso.
func (w *world) nextWaypoint(id string, from, dest vec) vec {
	if from == dest {
		w.stuckTicks[id] = 0
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
	next := path[0]

	blockedByStretcher := overlapsAny(feetRectAt(next), w.avoidFor(id))
	occ := ""
	if !blockedByStretcher {
		occ = w.occupantOf(id, next)
	}
	if occ == "" && !blockedByStretcher {
		w.stuckTicks[id] = 0
		return next
	}
	w.stuckTicks[id]++
	if occ != "" && id < occ && w.stillMoving(occ) {
		return from // tengo prioridad y el otro se puede apartar solo: espero
	}
	if w.stuckTicks[id] < waitBeforeSidestep {
		return from // unos ticks de margen: puede que se libere sola
	}

	// Sigue ocupada: la trata como un obstáculo temporal y recalcula la
	// ruta por otro lado (igual que con la camilla de otro, GAME_DESIGN
	// §10.8). Si ni rodeándola se puede (un pasillo tan angosto que no hay
	// otro camino), se queda esperando: nunca entra a una celda ocupada.
	w.stuckTicks[id] = 0
	avoid := append([]image.Rectangle(nil), w.avoidFor(id)...)
	if occ != "" {
		avoid = append(avoid, feetRectAt(w.snapshotPos[occ]))
	}
	around := routeAvoiding(from, dest, avoid)
	w.paths[id] = around
	next = around[0]
	if overlapsAny(feetRectAt(next), w.avoidFor(id)) || w.occupantOf(id, next) != "" {
		return from
	}
	return next
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
