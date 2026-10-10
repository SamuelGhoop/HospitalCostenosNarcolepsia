package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
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
// dibuja ×2 (escalado entero, sin suavizar): en la pantalla mide 32×48.
// (Se probó ×3 y se descartó: los personajes quedaban más altos que el
// pasillo 1 y más anchos que el hueco de las puertas.)
const (
	cellWidth   = 16
	cellHeight  = 24
	spriteScale = 2
	figWidth    = cellWidth * spriteScale  // 32
	figHeight   = cellHeight * spriteScale // 48
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
	// entrance: en la acera, frente a la puerta de la recepción. Ahí
	// aparece cada paciente nuevo antes de caminar hacia adentro.
	entrance = fromMockup(208, 186, 0, 0).Min
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

// wakeSpot: donde queda de pie el que se despierta, a la IZQUIERDA de la
// cama (la derecha es para los rótulos del que esté acostado): con el
// cuerpo justo debajo de la mesita de noche y su etiqueta sin tocar la cama.
func wakeSpot(room zone) image.Point {
	bed := bedRect(room)
	nightstand := sprite(roomMockupX[room]+28, 19, 6, 6)
	return image.Pt(bed.Min.X-figWidth/2-idLabelHalfWidth, nightstand.Max.Y-standingBody.Min.Y)
}

// carrySpot: donde se para, a la DERECHA de la cama, quien trae a un
// paciente: con el cuerpo justo debajo del suero. Es de paso: después
// vuelve a su sala.
func carrySpot(room zone) image.Point {
	bed := bedRect(room)
	iv := sprite(roomMockupX[room]+60, 19, 5, 9)
	return image.Pt(bed.Max.X+4, iv.Max.Y-standingBody.Min.Y)
}
