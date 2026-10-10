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
	for _, x0 := range roomMockupX { // las 3 habitaciones, iguales
		out = append(out,
			sprite(x0+1, 18, 3, 5), sprite(x0+21, 18, 3, 5), // cortinas
			sprite(x0+28, 19, 6, 6),   // mesita de noche
			sprite(x0+39, 19, 12, 20), // cama
			sprite(x0+60, 19, 5, 9),   // suero
			sprite(x0+83, 55, 7, 7),   // silla de visitas
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
	return append(out, sprite(351, 100, 35, 12), sprite(432, 104, 14, 40), sprite(406, 122, 14, 22), staffConsole) // sala del personal
}

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
// en orden de preferencia: a su lado (a 60 px, más que una etiqueta), y si
// ahí no cabe, en diagonal arriba o abajo (90 px: lo que ocupa uno de pie
// con su punto y su etiqueta).
var helperSpots = []image.Point{{60, 0}, {-60, 0}, {44, -90}, {-44, -90}, {44, 90}, {-44, 90}}

// shape es lo que ocupa un personaje en un puesto: el cuadro del cuerpo y,
// por separado, cada uno de sus rótulos (burbuja, rayitos, punto, etiqueta).
// Por separado y no en un solo rectángulo: la burbuja con sus rayitos es
// más ancha que el cuerpo, pero va arriba, así que al lado cabe alguien.
type shape struct {
	cell   image.Rectangle
	labels []image.Rectangle
}

// rects son todos los rectángulos de la forma.
func (sh shape) rects() []image.Rectangle { return append([]image.Rectangle{sh.cell}, sh.labels...) }

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
// rótulos que dibuja figureOverlays.
func shapeAt(at image.Point, figs []figure, bubble bool) shape {
	sh := shape{cell: image.Rectangle{at, at.Add(image.Pt(figWidth, figHeight))}}
	for _, f := range figs {
		for _, o := range figureOverlays(f, toVec(at), bubble) {
			sh.labels = append(sh.labels, o.rect)
		}
	}
	return sh
}

func standingShape(at image.Point) shape { return shapeAt(at, sampleStanding, false) }
func lyingShape(at image.Point) shape    { return shapeAt(at, []figure{sampleLying}, true) }

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

// lyingZones son las zonas donde un paciente se puede desplomar en el piso:
// por donde deambula (GAME_DESIGN §5.4) y el pasillo 1. En las habitaciones
// se acuesta en la cama y a la sala del personal no entra.
var lyingZones = map[zone]bool{zoneLobby: true, zoneCafeteria: true, zoneRadiology: true, zoneHallway1: true, zoneHallway2: true}

// spotsOf calcula los puestos de una zona. Necesita las fuentes cargadas:
// los rótulos se miden con su fuente.
//
// Se recorre la zona en una cuadrícula de 2 px. Un puesto sirve si:
//   - el cuerpo visible queda dentro de la zona y no toca ningún mueble;
//   - el cuadro y los rótulos quedan dentro de la zona, sin tapar el rótulo
//     de la zona ni a los puestos que ya se eligieron.
//
// Si así no cabe (o no caben todos), hay un segundo intento en el que los
// rótulos pueden salirse unos píxeles de la zona, siempre que no tapen
// otro rótulo de zona, un letrero de habitación ni un mueble (decisión de
// Samuel, 2026-10-10).
//
// Los puestos de piso se buscan desde abajo (lejos de los que están de pie)
// y llevan cerca el de quien atiende (ver helperSpots).
// Los de pie se buscan desde arriba y no se montan sobre los dos primeros
// de piso.
func spotsOf(z zone) *zoneSpots {
	if s, ok := spotCache[z]; ok {
		return s
	}
	r := zones[z].rect
	label := zoneLabelBox(z)
	s := &zoneSpots{}
	var taken []shape // formas ya ocupadas

	strict := true // el primer intento exige todo dentro de la zona
	free := func(body image.Rectangle, sh shape) bool {
		if !body.In(r) || onFurniture(body) {
			return false
		}
		for _, rr := range sh.rects() {
			if rr.Overlaps(label) || (strict && !rr.In(r)) {
				return false
			}
		}
		if !strict {
			for _, l := range sh.labels {
				if onFurniture(l) || coversALabel(l) {
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

	const step = 2
	findFloor := func() {
		for y := r.Max.Y - figHeight; y >= r.Min.Y; y -= step { // de piso: desde abajo
			for x := r.Min.X; x+figWidth <= r.Max.X; x += step {
				at := image.Pt(x, y)
				sh := lyingShape(at)
				if !free(floorBody.Add(at), sh) {
					continue
				}
				for _, d := range helperSpots {
					h := at.Add(d)
					hs := standingShape(h)
					if !hs.overlaps(sh) && free(standingBody.Add(h), hs) {
						s.floor = append(s.floor, floorPlace{at, h, !strict})
						taken = append(taken, sh, hs)
						break
					}
				}
			}
		}
	}
	findStanding := func() {
		for y := r.Min.Y; y+figHeight <= r.Max.Y; y += step { // de pie: desde arriba
			for x := r.Min.X; x+figWidth <= r.Max.X; x += step {
				at := image.Pt(x, y)
				if sh := standingShape(at); free(standingBody.Add(at), sh) {
					s.standing = append(s.standing, at)
					taken = append(taken, sh)
				}
			}
		}
	}

	if lyingZones[z] {
		findFloor()
		if len(s.floor) == 0 { // no cupo con todo dentro: se deja que los rótulos se salgan
			strict = false
			findFloor()
			strict = true
		}
	}
	if len(taken) > 4 {
		taken = taken[:4] // de pie se respetan solo los 2 primeros puestos de piso (con su ayudante)
	}
	findStanding()
	s.strictStanding = len(s.standing)
	strict = false // los que no caben con todo dentro van después
	findStanding()
	spotCache[z] = s
	return s
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
