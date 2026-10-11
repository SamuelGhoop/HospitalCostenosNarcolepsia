package ui

import (
	"image"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Puestos sobre piso libre: dónde se para (o se acuesta) cada personaje en
// una zona sin quedar encima de un mueble del fondo ni de otro personaje.

// ─── Muebles ───────────────────────────────────────────────────────────

// sprite es lo que ocupa en la pantalla un mueble de hospital.dc.html. En
// la maqueta, put(make(w, h), x, y) dibuja un sprite de (w+2)×(h+2)
// píxeles (con su contorno de 1 px), cada píxel de sc/2 unidades del mundo,
// con la esquina en (x − sc/2, y − sc/2). sc es 3 si la maqueta no dice
// otra cosa.
func sprite(x, y, w, h float64) image.Rectangle { return spriteSc(x, y, w, h, 3) }

func spriteSc(x, y, w, h, sc float64) image.Rectangle {
	px := sc / 2
	return fromMockup(x-px, y-px, (w+2)*px, (h+2)*px)
}

// staffConsole es la consola de la sala del personal (abajo); su rótulo va
// justo encima, sobre piso libre.
var staffConsole = sprite(351, 161, 40, 10)

// obstacles son los muebles donde nadie se puede parar. Las camas cuentan:
// la excepción es el paciente acostado en la suya. Los tapetes y el tapete
// de la puerta son piso.
var obstacles = buildObstacles()

func buildObstacles() []image.Rectangle {
	var out []image.Rectangle
	for room, x0 := range roomMockupX { // las 3 habitaciones, iguales
		out = append(out,
			sprite(x0+1, 18, 3, 5), sprite(x0+21, 18, 3, 5), // cortinas
			sprite(x0+28, 19, 6, 6), // mesita de noche
			bedObstacle(room),       // cama
			sprite(x0+60, 19, 5, 9), // suero
			sprite(x0+83, 55, 7, 7), // silla de visitas
		)
	}
	out = append(out,
		sprite(338, 37, 56, 8), sprite(359, 22, 28, 28), sprite(309, 48, 9, 9), // radiología: camilla, escáner, letrero
		image.Rect(1215, 197, 1255, 245), // pasillo 1: el extintor (parche, ya en píxeles de pantalla)
		sprite(12, 101, 48, 14),          // cafetería: mostrador
	)
	for _, x := range []float64{18, 76} { // cafetería: mesas con sus 4 sillas
		out = append(out, sprite(x, 136, 16, 10),
			sprite(x+3, 127, 5, 5), sprite(x+13.5, 127, 5, 5), sprite(x+3, 152.5, 5, 5), sprite(x+13.5, 152.5, 5, 5))
	}
	out = append(out, sprite(267, 111, 10, 36), sprite(286.5, 120, 6, 6), spriteSc(132, 99, 30, 30, 2)) // recepción: mostrador, silla, matera
	for _, x := range []float64{164, 174, 228, 238} {                                                   // recepción: sillas de espera
		for _, y := range []float64{106, 116, 126, 136} {
			out = append(out, sprite(x, y, 6, 6))
		}
	}
	out = append(out, sprite(351, 100, 35, 12), sprite(432, 104, 14, 40), sprite(406, 122, 14, 22), staffConsole) // sala del personal
	// La acera: el paradero, su poste y la base de las 2 palmas (la copa no:
	// vista desde arriba, se camina por debajo).
	return append(out, sprite(120, 188, 38, 13), sprite(181, 188, 2, 12), sprite(292.5, 191.5, 10, 6), sprite(392.5, 191.5, 10, 6))
}

// bedObstacle es la cama de una habitación como mueble.
func bedObstacle(room zone) image.Rectangle { return sprite(roomMockupX[room]+39, 19, 12, 20) }

// onFurniture dice si r toca algún mueble.
func onFurniture(r image.Rectangle) bool {
	for _, o := range obstacles {
		if r.Overlaps(o) {
			return true
		}
	}
	return false
}

// ─── Cuerpos y rótulos ─────────────────────────────────────────────────

// scaleRect pasa un rectángulo de la hoja de sprites a la pantalla.
func scaleRect(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X*spriteScale, r.Min.Y*spriteScale, r.Max.X*spriteScale, r.Max.Y*spriteScale)
}

// Lo que se VE del cuerpo dentro de su cuadro (medido en la hoja: lo demás
// del cuadro es transparente). Es lo que no puede tocar un mueble.
var (
	standingBody = scaleRect(image.Rect(1, 2, 15, 23)) // de pie: paciente (1,3)-(15,23) o personal (3,2)-(13,23)
	floorBody    = scaleRect(image.Rect(0, 9, 16, 24)) // acostado en el piso (horizontal)
	bedBody      = scaleRect(image.Rect(0, 0, 15, 16)) // acostado en la cama (ya rotado 90°)
)

// helperSpots: dónde puede pararse quien atiende a un paciente en el piso,
// en orden de preferencia: a su lado (un cuadro y media etiqueta más allá)
// y, si ahí no cabe, en diagonal arriba o abajo (lo que ocupa uno de pie
// con su punto y su etiqueta). Si tampoco, en el pasillo 1, debajo de la
// puerta de la zona (ver doorHelperSpots).
var helperSpots = []image.Point{
	{figWidth + 24, 0}, {-figWidth - 24, 0},
	{figWidth, -figHeight - 40}, {-figWidth, -figHeight - 40},
	{figWidth, figHeight}, {-figWidth, figHeight},
}

// doorHelperSpots son los puestos en el pasillo 1 junto a cada puerta de
// la zona z que da al pasillo (con los pies cerca del borde de abajo del
// pasillo): debajo de la puerta y a sus lados. De ahí se ve al paciente
// por la puerta.
func doorHelperSpots(z zone) []image.Point {
	r, h := zones[z].rect, zones[zoneHallway1].rect
	var out []image.Point
	for _, d := range doorways {
		if d.gap.Max.Y < r.Max.Y-navCell || d.gap.Min.Y > r.Max.Y+navCell || d.gap.Min.X < r.Min.X || d.gap.Max.X > r.Max.X {
			continue // no es una puerta de abajo de esta zona
		}
		for _, dx := range []int{0, figWidth + figWidth/2, -figWidth - figWidth/2, 2*figWidth + figWidth/2, -2*figWidth - figWidth/2} {
			feet := image.Pt(d.gap.Min.X+d.gap.Dx()/2+dx, h.Max.Y-navCell)
			out = append(out, feet.Sub(feetOffsetPt))
		}
	}
	return out
}

// shape es lo que ocupa un personaje en un puesto: el cuadro del cuerpo y,
// por separado, cada uno de sus rótulos (burbuja, rayitos, punto, etiqueta).
// Por separado y no en un solo rectángulo: la burbuja con sus rayitos es
// más ancha que el cuerpo, pero va arriba, así que al lado cabe alguien.
type shape struct {
	cell     image.Rectangle
	labels   []image.Rectangle // etiqueta y punto
	floating []image.Rectangle // burbuja y rayitos: flotan sobre el acostado
}

// rects son todos los rectángulos de la forma.
func (sh shape) rects() []image.Rectangle {
	return append(append([]image.Rectangle{sh.cell}, sh.labels...), sh.floating...)
}

// overlaps dice si dos formas se tocan en algún rectángulo.
func (sh shape) overlaps(other shape) bool {
	for _, a := range sh.rects() {
		for _, b := range other.rects() {
			if a.Overlaps(b) {
				return true
			}
		}
	}
	return false
}

// Personajes de muestra con los rótulos más grandes que lleva cada puesto:
// de pie, un paciente (etiqueta) y alguien del personal (punto y etiqueta);
// acostado, un paciente severo (burbuja, 3 rayitos y etiqueta).
var (
	sampleStanding = []figure{{id: "P-000", patient: true, state: hospital.Awake}, {id: "D-00", doctor: true}}
	sampleLying    = figure{id: "P-000", patient: true, state: hospital.AsleepInHallway, level: hospital.Severe}
)

// shapeAt arma la forma de los personajes de muestra en at, con los mismos
// rótulos que dibuja figureOverlays. body es lo que ocupa el cuerpo: el
// cuadro entero de pie; acostado, solo lo que se ve (arriba del cuadro
// queda vacío, y ahí va la burbuja).
func shapeAt(at image.Point, figs []figure, bubble bool, body image.Rectangle) shape {
	sh := shape{cell: body.Add(at)}
	for _, f := range figs {
		for _, o := range figureOverlays(f, toVec(at), bubble) {
			if o.floating {
				sh.floating = append(sh.floating, o.rect)
			} else {
				sh.labels = append(sh.labels, o.rect)
			}
		}
	}
	return sh
}

func standingShape(at image.Point) shape {
	return shapeAt(at, sampleStanding, false, image.Rect(0, 0, figWidth, figHeight))
}
func lyingShape(at image.Point) shape { return shapeAt(at, []figure{sampleLying}, true, floorBody) }

// ─── Puestos ───────────────────────────────────────────────────────────

// floorPlace es un puesto en el piso: el paciente acostado y, a su lado,
// quien lo atiende. relaxed dice si salió del segundo intento (ver spotsOf).
type floorPlace struct {
	patient, helper image.Point
	relaxed         bool
}

// zoneSpots son los puestos de una zona, calculados una sola vez. Los de
// pie del primer intento van antes que los del segundo: strictStanding
// dice cuántos son del primero.
type zoneSpots struct {
	standing       []image.Point
	strictStanding int
	floor          []floorPlace
}

var spotCache = map[zone]*zoneSpots{}

// standingClearOfFloor: cuántos puestos de pie (los primeros) dejan libre
// el primer puesto de piso de la zona. Los demás solo se usan cuando hay
// mucha gente de pie: en la demo, los 5 pacientes recién llegados a la
// recepción (pasos 3 y 4); cuando Ludys se desploma ahí (paso 7) ya solo
// quedan 2 de pie. A ×3 la recepción no da para más.
const standingClearOfFloor = 2

// lyingZones son las zonas donde un paciente se puede desplomar en el piso:
// por donde deambula (GAME_DESIGN §5.4) y el pasillo 1. En las habitaciones
// se acuesta en la cama y a la sala del personal no entra.
var lyingZones = map[zone]bool{zoneLobby: true, zoneCafeteria: true, zoneRadiology: true, zoneHallway1: true, zoneHallway2: true}

// spotFree dice si un personaje cabe en at, con los pies en la zona z.
// body es lo que se ve de su cuerpo y sh su forma con los rótulos; taken,
// las formas de los puestos ya elegidos. Un puesto sirve si:
//   - se llega caminando desde la puerta principal (standable): ningún
//     puesto queda en un rincón encerrado por los muebles;
//   - no queda frente a la puerta principal (por ahí entran todos, y la
//     puerta se cierra sola cuando no hay nadie cerca);
//   - el cuerpo visible queda dentro de la zona y no toca ningún mueble;
//   - el cuadro y los rótulos quedan dentro de la zona, sin tapar el rótulo
//     de la zona ni a los puestos que ya se eligieron.
//
// En el pasillo 1, más bajito que un personaje, "dentro de la zona" vale
// para los pies: lo de arriba (la cabeza, el punto) se monta sobre la pared
// (vista 3/4; decisión de Samuel, 2026-10-11).
//
// Si no es strict (el segundo intento), los rótulos pueden salirse de la
// zona, siempre que no tapen otro rótulo de zona, un letrero de habitación
// ni un mueble (decisión de Samuel, 2026-10-10). La burbuja y los rayitos
// de un acostado sí pueden quedar sobre un mueble: flotan encima de él (a
// ×3, en radiología no cabe de otra forma; pendiente del OK de Samuel).
func spotFree(z zone, at image.Point, body image.Rectangle, sh shape, strict bool, taken []shape) bool {
	r := zones[z].rect
	inside := func(x image.Rectangle) bool {
		if z == zoneHallway1 {
			x.Min.Y = max(x.Min.Y, r.Min.Y) // la cabeza puede quedar sobre la pared de arriba
		}
		return x.In(r)
	}
	if !inside(body) || onFurniture(body) || !standable(at.Add(feetOffsetPt)) {
		return false
	}
	if distToRect(toVec(at.Add(feetOffsetPt)), doorways[len(doorways)-1].gap) <= doorReach {
		return false // frente a la puerta principal no se para nadie: por ahí entran y la puerta se cierra sola
	}

	for _, rr := range sh.rects() {
		if coversALabel(rr) || (strict && !inside(rr)) { // ni el rótulo de una zona ni el letrero de una habitación
			return false
		}
	}
	if !strict {
		for _, l := range sh.labels {
			if onFurniture(l) || coversALabel(l) {
				return false
			}
		}
		for _, l := range sh.floating {
			if coversALabel(l) {
				return false
			}
		}
	}
	for _, t := range taken {
		if sh.overlaps(t) {
			return false
		}
	}
	return true
}

// zoneOfFeet es la zona donde están los pies de alguien parado en at.
func zoneOfFeet(at image.Point) zone {
	z, _ := zoneAt(toVec(at.Add(feetOffsetPt)))
	return z
}

// spotsOf calcula los puestos de una zona. Necesita las fuentes cargadas:
// los rótulos se miden con su fuente.
//
// Se recorre la zona en una cuadrícula de 2 px y se toman los puestos que
// cumplen spotFree, primero con todo dentro de la zona y, si no caben (o
// no caben todos), en un segundo intento en el que los rótulos se pueden
// salir.
//
// Los puestos de piso (floorOf) van primero. Los de pie se buscan desde
// arriba, y los primeros standingClearOfFloor no se montan sobre el primer
// puesto de piso (con su ayudante).
func spotsOf(z zone) *zoneSpots {
	if s, ok := spotCache[z]; ok {
		return s
	}
	r := zones[z].rect
	s := &zoneSpots{floor: floorOf(z)}
	var floorShapes, standingShapes []shape
	if len(s.floor) > 0 {
		f := s.floor[0]
		floorShapes = []shape{lyingShape(f.patient), standingShape(f.helper)}
	}
	strict := true // el primer intento exige todo dentro de la zona
	findStanding := func() {
		top := r.Min.Y
		if z == zoneHallway1 {
			top = r.Max.Y - figHeight - 20 // los pies dentro; la cabeza puede quedar sobre la pared
		}
		for y := top; y+figHeight <= r.Max.Y; y += spotStep { // de pie: desde arriba
			for x := r.Min.X; x+figWidth <= r.Max.X; x += spotStep {
				at := image.Pt(x, y)
				avoid := standingShapes
				if len(s.standing) < standingClearOfFloor {
					avoid = append(append([]shape(nil), standingShapes...), floorShapes...)
				}
				if sh := standingShape(at); spotFree(z, at, standingBody.Add(at), sh, strict, avoid) {
					s.standing = append(s.standing, at)
					standingShapes = append(standingShapes, sh)
				}
			}
		}
	}
	findStanding()
	s.strictStanding = len(s.standing)
	strict = false // los que no caben con todo dentro van después
	findStanding()
	spotCache[z] = s
	return s
}

// spotStep: el paso de la búsqueda de puestos, en px.
const spotStep = 2

// floorCache guarda los puestos de piso de cada zona (ver floorOf).
var floorCache = map[zone][]floorPlace{}

// floorOf calcula los puestos de piso de una zona: el paciente acostado y
// quien lo atiende cerca, a su lado, en diagonal (helperSpots) o en el
// pasillo 1 junto a la puerta (doorHelperSpots). Se buscan desde abajo
// (lejos de los que están de pie). Solo en lyingZones.
func floorOf(z zone) []floorPlace {
	if f, ok := floorCache[z]; ok {
		return f
	}
	var out []floorPlace
	if lyingZones[z] {
		r := zones[z].rect
		var taken []shape
		find := func(strict bool) {
			for y := r.Max.Y - figHeight; y >= r.Min.Y-floorBody.Min.Y; y -= spotStep {
				for x := r.Min.X; x+figWidth <= r.Max.X; x += spotStep {
					at := image.Pt(x, y)
					sh := lyingShape(at)
					if !spotFree(z, at, floorBody.Add(at), sh, strict, taken) {
						continue
					}
					var helpers []image.Point
					for _, d := range helperSpots {
						helpers = append(helpers, at.Add(d))
					}
					for _, h := range append(helpers, doorHelperSpots(z)...) {
						hs := standingShape(h)
						if !hs.overlaps(sh) && spotFree(zoneOfFeet(h), h, standingBody.Add(h), hs, strict, append(taken, sh)) {
							out = append(out, floorPlace{at, h, !strict})
							taken = append(taken, sh, hs)
							break
						}
					}
				}
			}
		}
		find(true)
		if len(out) == 0 { // no cupo con todo dentro: se deja que los rótulos se salgan
			find(false)
		}
	}
	floorCache[z] = out
	return out
}

// coversALabel dice si r tapa el rótulo de alguna zona o el letrero de
// alguna habitación (medido con el más ancho, DISPONIBLE).
func coversALabel(r image.Rectangle) bool {
	for _, z := range zoneOrder {
		if r.Overlaps(zoneLabelBox(z)) {
			return true
		}
	}
	for _, room := range []zone{zoneRoom101, zoneRoom102, zoneRoom103} {
		rr := zones[room].rect
		if r.Overlaps(centeredBox("DISPONIBLE", rr.Min.X+rr.Dx()/2, rr.Max.Y-10, smallFace)) {
			return true
		}
	}
	return false
}

// relaxedSpot dice si el puesto at (de pie o de piso, o el ayudante de uno
// de piso) de la zona z salió del segundo intento del buscador.
func relaxedSpot(z zone, at image.Point) bool {
	if wakeRelaxed[at] {
		return true
	}
	s := spotsOf(z)
	for i, p := range s.standing {
		if p == at {
			return i >= s.strictStanding
		}
	}
	for _, f := range s.floor {
		if f.patient == at || f.helper == at {
			return f.relaxed
		}
	}
	return false
}

// slot es el i-ésimo puesto para estar DE PIE en una zona. Si la zona no
// tiene tantos (o ninguno), devuelve el último que tenga, o la esquina de
// la zona: el test de la demo avisa si alguien queda encima de otro.
func slot(z zone, i int) image.Point {
	s := spotsOf(z).standing
	switch {
	case i < len(s):
		return s[i]
	case len(s) > 0:
		return s[len(s)-1]
	default:
		return zones[z].rect.Min
	}
}

// floorSpot es el k-ésimo puesto en el PISO de una zona, para los dormidos
// sin cama (con el mismo respaldo que slot).
func floorSpot(z zone, k int) image.Point {
	f := spotsOf(z).floor
	switch {
	case k < len(f):
		return f[k].patient
	case len(f) > 0:
		return f[len(f)-1].patient
	default:
		return zones[z].rect.Min
	}
}

// helperSpot es donde se para quien atiende al paciente acostado en p: el
// ayudante de ese puesto de piso; si p no es un puesto de piso, a su derecha.
func helperSpot(p vec) vec {
	for _, s := range spotCache {
		for _, f := range s.floor {
			if toVec(f.patient) == p {
				return toVec(f.helper)
			}
		}
	}
	return p.add(toVec(helperSpots[0]))
}
