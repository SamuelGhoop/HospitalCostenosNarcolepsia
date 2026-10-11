package ui

import (
	"image"
	"math"
)

// Las zonas, las puertas y las paredes del mapa. Las rutas (nav.go) son
// para los PIES del personaje: el punto de abajo al centro de su cuadro.
// En una vista desde arriba lo que tiene que pasar por el hueco de la
// puerta son los pies; el cuerpo se dibuja encima.

// feetOffset va de la esquina del cuadro (donde se guarda la posición del
// personaje) a sus pies. figHeight-1: la última fila del cuadro, que sigue
// dentro de la zona.
var feetOffset = vec{figWidth / 2, figHeight - 1}

// add suma dos vectores.
func (v vec) add(w vec) vec { return vec{v.x + w.x, v.y + w.y} }

// doorway es una puerta de la maqueta: el hueco en la pared.
type doorway struct {
	gap image.Rectangle // el hueco en la pared, en la pantalla
}

// door arma una puerta con el hueco (x, y, w, h) en coordenadas de la maqueta.
func door(x, y, w, h float64) doorway { return doorway{gap: fromMockup(x, y, w, h)} }

// doorways son las puertas de hospital.dc.html: hDoor(x, y, w) es un hueco
// de w×3 en una pared horizontal y vDoor(x, y, h), uno de 3×h en una
// vertical. La puerta principal es el rectángulo (199, 178, 30, 4) y va
// de última. Las medidas son las del mapa de Samuel del 2026-10-11.
var doorways = []doorway{
	door(70, 68, 20, 3),   // habitación 101 ↔ pasillo 1
	door(169, 68, 20, 3),  // habitación 102 ↔ pasillo 1
	door(268, 68, 20, 3),  // habitación 103 ↔ pasillo 1
	door(318, 68, 20, 3),  // radiología ↔ pasillo 1
	door(68, 95, 30, 3),   // pasillo 1 ↔ cafetería (alineada con la de la 101)
	door(184, 95, 60, 3),  // pasillo 1 ↔ recepción
	door(300, 95, 44, 3),  // pasillo 1 ↔ pasillo 2
	door(128, 147, 3, 20), // cafetería ↔ recepción
	door(344, 147, 3, 20), // pasillo 2 ↔ sala del personal
	door(297, 147, 3, 20), // recepción ↔ pasillo 2
	door(199, 178, 30, 4), // recepción ↔ acera (puerta principal)
}

// outside es la "zona" de afuera del edificio: la acera y la calle.
const outside zone = -1

// zoneAt dice en qué zona está un punto de la pantalla: una zona del mapa o
// outside. ok es false si está sobre una pared o en el hueco de una puerta.
func zoneAt(p vec) (z zone, ok bool) {
	pt := image.Pt(int(math.Floor(p.x)), int(math.Floor(p.y)))
	for _, z := range zoneOrder {
		if pt.In(zones[z].rect) {
			return z, true
		}
	}
	if !pt.In(buildingRect) && pt.In(image.Rect(0, 0, ScreenWidth, ScreenHeight)) {
		return outside, true
	}
	return 0, false
}
