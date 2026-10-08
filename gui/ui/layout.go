package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
)

// Tamaños de pantalla (GAME_DESIGN §10.1): resolución lógica de 640×360,
// que Ebitengine escala a la ventana de 1280×720 (×2) sin suavizar.
const (
	ScreenWidth  = 640
	ScreenHeight = 360
)

// La maqueta hospital.dc.html dibuja un mundo de 464×261 (también 16:9).
// Aquí se copian sus coordenadas tal cual, para poder compararlas con la
// maqueta, y fromMockup las escala a la pantalla de 640×360.
const (
	mockupWidth  = 464
	mockupHeight = 261
)

// Cada personaje ocupa un cuadro de 16×24 px (GAME_DESIGN §10.1).
const (
	cellWidth  = 16
	cellHeight = 24
)

// fromMockup convierte un rectángulo de la maqueta (x, y, ancho, alto) a
// la pantalla, multiplicando por 640/464 ≈ 1,379 y redondeando cada borde.
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
	rect  image.Rectangle // en la pantalla de 640×360
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

// Separación entre los puestos de los personajes dentro de una zona.
const (
	slotPad   = 4  // margen desde la esquina de la zona
	slotStepX = 28 // distancia horizontal: lo que mide la etiqueta "P-001"
	slotStepY = 36 // distancia vertical: el cuadro de 24 px más su etiqueta
)

// slot devuelve la esquina superior izquierda del cuadro de 16×24 del
// i-ésimo personaje de una zona: se llenan filas de izquierda a derecha.
func slot(z zone, i int) image.Point {
	r := zones[z].rect
	perRow := (r.Dx() - slotPad) / slotStepX
	if perRow < 1 {
		perRow = 1
	}
	return image.Pt(r.Min.X+slotPad+(i%perRow)*slotStepX, r.Min.Y+slotPad+(i/perRow)*slotStepY)
}
