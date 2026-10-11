package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Tamaños de pantalla (GAME_DESIGN §10.7): resolución lógica de 1280×720,
// la misma del mapa de fondo y de la ventana.
const (
	ScreenWidth  = 1280
	ScreenHeight = 720
)

// La maqueta hospital.dc.html dibuja un mundo de 464×261 (también 16:9).
// Aquí se copian sus coordenadas tal cual, para poder compararlas con la
// maqueta, y fromMockup las escala a la pantalla de 1280×720.
const (
	mockupWidth  = 464
	mockupHeight = 261
)

// Cada personaje es un cuadro de 16×24 px en su hoja de sprites y se
// dibuja ×3 (escalado entero, sin suavizar): en la pantalla mide 48×72
// (decisión de Samuel, 2026-10-11). Es más alto que el pasillo 1 y más ancho
// que el hueco de las puertas, así que las reglas valen para los pies: en
// el pasillo 1 la cabeza se monta sobre la pared de arriba (vista 3/4) y al
// cruzar una puerta el cuerpo roza el marco.
const (
	cellWidth   = 16
	cellHeight  = 24
	spriteScale = 3
	figWidth    = cellWidth * spriteScale  // 48
	figHeight   = cellHeight * spriteScale // 72
)

// fromMockup convierte un rectángulo de la maqueta (x, y, ancho, alto) a
// la pantalla, multiplicando por 1280/464 ≈ 2,759 y redondeando cada borde,
// igual que se renderizó el mapa de fondo.
func fromMockup(x, y, w, h float64) image.Rectangle {
	const k = float64(ScreenWidth) / mockupWidth
	round := func(v float64) int { return int(math.Round(v * k)) }
	return image.Rect(round(x), round(y), round(x+w), round(y+h))
}

// zone es una zona del mapa del hospital.
type zone int

// Zonas del mapa. zoneLobby va primero: es el valor cero, y por eso
// también el destino por defecto de una ubicación desconocida.
const (
	zoneLobby zone = iota
	zoneCafeteria
	zoneRadiology
	zoneHallway1
	zoneHallway2
	zoneStaffRoom
	zoneRoom101
	zoneRoom102
	zoneRoom103
)

// zoneInfo es lo que se necesita para dibujar una zona.
type zoneInfo struct {
	label string          // rótulo en español
	rect  image.Rectangle // en la pantalla de 1280×720
	floor color.RGBA      // color del piso
}

// zones: los rectángulos son los del objeto F de hospital.dc.html.
var zones = map[zone]zoneInfo{
	// "HAB." abreviado: el rótulo largo quedaba tapado por el letrero OCUPADA.
	zoneRoom101:   {"HAB. 101", fromMockup(8, 18, 96, 50), colRoomFloor},
	zoneRoom102:   {"HAB. 102", fromMockup(107, 18, 96, 50), colRoomFloor},
	zoneRoom103:   {"HAB. 103", fromMockup(206, 18, 96, 50), colRoomFloor},
	zoneRadiology: {"RADIOLOGÍA", fromMockup(305, 18, 151, 50), colRadiologyFloor},
	zoneHallway1:  {"PASILLO 1", fromMockup(8, 71, 448, 24), colHallFloor},
	zoneCafeteria: {"CAFETERÍA", fromMockup(8, 98, 120, 80), colCafeFloor},
	zoneLobby:     {"RECEPCIÓN", fromMockup(131, 98, 166, 80), colLobbyFloor},
	zoneHallway2:  {"PASILLO 2", fromMockup(300, 98, 44, 80), colHallFloor},
	zoneStaffRoom: {"SALA DEL PERSONAL", fromMockup(347, 98, 109, 80), colStaffFloor},
}

// zoneOrder es el orden en que se dibujan las zonas (un map no tiene orden).
var zoneOrder = []zone{
	zoneRoom101, zoneRoom102, zoneRoom103, zoneRadiology,
	zoneHallway1, zoneCafeteria, zoneLobby, zoneHallway2, zoneStaffRoom,
}

// Otras partes del mapa de la maqueta.
var (
	buildingRect = fromMockup(4, 14, 456, 168)
	sidewalkRect = fromMockup(0, 182, 464, 28)
	streetRect   = fromMockup(0, 213, 464, 46)
)

// roomMockupX es la x de cada habitación en la maqueta, para ubicar su cama.
var roomMockupX = map[zone]float64{zoneRoom101: 8, zoneRoom102: 107, zoneRoom103: 206}

// bedRect es la cama de una habitación: en la maqueta, put(bed, x0+39, 19)
// con un sprite de 12×20 a escala 1,5 → 18×30 en el mundo de la maqueta.
func bedRect(room zone) image.Rectangle {
	return fromMockup(roomMockupX[room]+39, 19, 18, 30)
}

// zoneKeywords relaciona palabras de las ubicaciones del modelo con zonas
// del mapa. El orden importa: se usa la primera que aparezca.
var zoneKeywords = []struct {
	keyword string
	zone    zone
}{
	{"habitación 101", zoneRoom101},
	{"habitación 102", zoneRoom102},
	{"habitación 103", zoneRoom103},
	{"cafetería", zoneCafeteria},
	{"radiología", zoneRadiology},
	{"pasillo 1", zoneHallway1},
	{"pasillo 2", zoneHallway2},
	{"champeta", zoneLobby},  // "pista de champeta del patio" → Lobby (decisión de Samuel)
	{"recepción", zoneLobby}, // ubicación con la que llega todo paciente
}

// zoneFor ubica en el mapa una ubicación escrita por el modelo (por
// ejemplo "pasillo 2, segundo piso"). Si no la reconoce, va a la recepción.
func zoneFor(location string) zone {
	location = strings.ToLower(location)
	for _, k := range zoneKeywords {
		if strings.Contains(location, k.keyword) {
			return k.zone
		}
	}
	return zoneLobby
}

// roomNumber es el número de la habitación de una zona (0 si no es una).
func roomNumber(z zone) int {
	for _, n := range []int{101, 102, 103} {
		if roomZone(n) == z {
			return n
		}
	}
	return 0
}

// roomZone es la zona de la habitación con ese número.
func roomZone(number int) zone {
	return zoneFor(fmt.Sprintf("habitación %d", number))
}

// blanketSpot es la esquina de la cobija (bed_blanket.png, 56×48) de la cama
// de una habitación. Es el recorte que se sacó de la maqueta: un píxel del
// sprite de la cama (1,5 en el mundo) a la izquierda de la cama y 13 abajo
// de su borde de arriba. Da (126, 88), (399, 88) y (672, 88).
func blanketSpot(room zone) image.Point {
	return fromMockup(roomMockupX[room]+37.5, 32, 0, 0).Min
}

// bedCell es donde va el cuadro (rotado) del paciente acostado en su cama:
// el cuerpo que se ve (bedBody) centrado a lo ancho de la cama y con la
// mitad de abajo bajo la cobija, para que se vea arropado.
func bedCell(room zone) vec {
	bed := bedRect(room)
	x := bed.Min.X + bed.Dx()/2 - (bedBody.Min.X+bedBody.Max.X)/2
	y := blanketSpot(room).Y - bedBody.Dy()/2
	return vec{float64(x), float64(y)}
}

// idLabelHalfWidth: la mitad de la etiqueta "P-001" (unos 50 px), con aire.
const idLabelHalfWidth = 27

// wakeSpot: donde queda de pie el que se despierta, al lado de su cama. Se
// busca el puesto libre (spotFree) más cercano al costado DERECHO de la
// cama, donde se para quien trae a un paciente, y si no, al IZQUIERDO,
// debajo de la mesita de noche: primero con todo dentro de la habitación y,
// solo si así no cabe en ningún lado, con los rótulos afuera (el segundo
// intento). A ×3, a la derecha no queda sitio con todo dentro (el suero, la
// silla de visitas y el letrero de la habitación), así que queda a la
// izquierda. Se calcula una vez por habitación.
func wakeSpot(room zone) image.Point {
	if p, ok := wakeCache[room]; ok {
		return p
	}
	bed, r := bedRect(room), zones[room].rect
	nightstand := sprite(roomMockupX[room]+28, 19, 6, 6)
	right := carrySpot(room)
	left := image.Pt(bed.Min.X-figWidth/2-idLabelHalfWidth, nightstand.Max.Y-standingBody.Min.Y)
	// Lo que ocupa un paciente acostado en esa cama (el cuerpo y sus rótulos,
	// a la derecha de la cama): puede llegar otro mientras este sigue ahí.
	sleeper := figure{id: "P-000", patient: true, state: hospital.AsleepInBed, level: hospital.Severe, room: roomNumber(room)}
	sh := shape{cell: bedBody.Add(image.Pt(int(bedCell(room).x), int(bedCell(room).y)))}
	for _, o := range figureOverlays(sleeper, bedCell(room), true) {
		sh.labels = append(sh.labels, o.rect)
	}
	inBed := []shape{sh}
	for _, strict := range []bool{true, false} {
		best := map[bool]image.Point{}
		bestD := map[bool]float64{true: math.Inf(1), false: math.Inf(1)}
		for y := r.Min.Y; y+figHeight <= r.Max.Y+figHeight; y += 2 {
			for x := r.Min.X; x+figWidth <= r.Max.X; x += 2 {
				at := image.Pt(x, y)
				body := standingBody.Add(at)
				isRight := body.Min.X >= bed.Max.X
				if !isRight && body.Max.X > bed.Min.X {
					continue // ni a un lado ni al otro: encima de la cama
				}
				anchor := left
				if isRight {
					anchor = right
				}
				d := math.Hypot(float64(x-anchor.X), float64(y-anchor.Y))
				if d < bestD[isRight] && spotFree(room, at, body, standingShape(at), strict, inBed) {
					best[isRight], bestD[isRight] = at, d
				}
			}
		}
		for _, side := range []bool{true, false} {
			if !math.IsInf(bestD[side], 1) {
				wakeCache[room] = best[side]
				wakeRelaxed[best[side]] = !strict
				return best[side]
			}
		}
	}
	wakeCache[room] = left
	return left
}

// wakeRelaxed dice qué puestos de despertar salieron del segundo intento.
var wakeRelaxed = map[image.Point]bool{}

var wakeCache = map[zone]image.Point{}

// carrySpot: donde se para, a la DERECHA de la cama, quien trae a un
// paciente: con el cuerpo justo debajo del suero. Es de paso: después
// vuelve a su sala.
func carrySpot(room zone) image.Point {
	bed := bedRect(room)
	iv := sprite(roomMockupX[room]+60, 19, 5, 9)
	return image.Pt(bed.Max.X+4, iv.Max.Y-standingBody.Min.Y)
}
