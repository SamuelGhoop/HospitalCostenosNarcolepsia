package ui

import (
	"image"
	"math"
)

// Rutas por puntos de paso (GAME_DESIGN §10.7): para ir de una zona a otra,
// los personajes salen por las puertas y caminan por los pasillos, en vez
// de ir en línea recta atravesando paredes.
//
// Las rutas son para los PIES del personaje: el punto de abajo al centro de
// su cuadro de 32×48. En una vista desde arriba lo que tiene que pasar por
// el hueco de la puerta son los pies; el cuerpo se dibuja encima.

// feetOffset va de la esquina del cuadro (donde se guarda la posición del
// personaje) a sus pies. figHeight-1: la última fila del cuadro, que sigue
// dentro de la zona.
var feetOffset = vec{figWidth / 2, figHeight - 1}

// add suma dos vectores.
func (v vec) add(w vec) vec { return vec{v.x + w.x, v.y + w.y} }

// doorway es una puerta de la maqueta: el hueco en la pared y un punto de
// paso a cada lado, de frente al hueco. Cruzar la puerta es ir derecho de
// un punto al otro.
type doorway struct {
	gap   image.Rectangle // el hueco en la pared, en la pantalla
	sides [2]vec          // un punto de paso a cada lado (para los pies), en la pantalla
}

// door arma una puerta con coordenadas de la maqueta: el hueco (x, y, w, h)
// y los dos puntos de paso (ax, ay) y (bx, by), 3 unidades adentro de
// cada zona.
func door(x, y, w, h, ax, ay, bx, by float64) doorway {
	return doorway{gap: fromMockup(x, y, w, h), sides: [2]vec{mockupPoint(ax, ay), mockupPoint(bx, by)}}
}

// mockupPoint convierte un punto de la maqueta a la pantalla.
func mockupPoint(x, y float64) vec { return toVec(fromMockup(x, y, 0, 0).Min) }

// doorways son las puertas de hospital.dc.html: hDoor(x, y, w) es un hueco
// de w×3 en una pared horizontal y vDoor(x, y, h), uno de 3×h en una
// vertical. La puerta principal es el rectángulo (199, 178, 30, 4).
var doorways = []doorway{
	door(76, 68, 14, 3, 83, 65, 83, 74),       // habitación 101 ↔ pasillo 1
	door(175, 68, 14, 3, 182, 65, 182, 74),    // habitación 102 ↔ pasillo 1
	door(274, 68, 14, 3, 281, 65, 281, 74),    // habitación 103 ↔ pasillo 1
	door(320, 68, 16, 3, 328, 65, 328, 74),    // radiología ↔ pasillo 1
	door(90, 95, 14, 3, 97, 92, 97, 101),      // pasillo 1 ↔ cafetería
	door(184, 95, 60, 3, 214, 92, 214, 101),   // pasillo 1 ↔ recepción
	door(300, 95, 44, 3, 322, 92, 322, 101),   // pasillo 1 ↔ pasillo 2
	door(128, 150, 3, 14, 125, 157, 134, 157), // cafetería ↔ recepción
	door(344, 150, 3, 14, 341, 157, 350, 157), // pasillo 2 ↔ sala del personal
	door(199, 178, 30, 4, 214, 175, 214, 190), // recepción ↔ acera (puerta principal)
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

// walkable dice si se puede pisar un punto: el piso de una zona, afuera
// del edificio o el hueco de una puerta. Lo demás del edificio es pared.
func walkable(p vec) bool {
	if _, ok := zoneAt(p); ok {
		return true
	}
	pt := image.Pt(int(math.Floor(p.x)), int(math.Floor(p.y)))
	for _, d := range doorways {
		if pt.In(d.gap) {
			return true
		}
	}
	return false
}

// walkableSegment revisa un tramo recto punto por punto, cada píxel.
func walkableSegment(a, b vec) bool {
	steps := int(math.Ceil(math.Hypot(b.x-a.x, b.y-a.y)))
	for i := 0; i <= steps; i++ {
		t := 1.0
		if steps > 0 {
			t = float64(i) / float64(steps)
		}
		if !walkable(vec{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t}) {
			return false
		}
	}
	return true
}

// waypoints son todos los puntos de paso: los dos lados de cada puerta.
func waypoints() []vec {
	var out []vec
	for _, d := range doorways {
		out = append(out, d.sides[0], d.sides[1])
	}
	return out
}

// neighbors dice con qué puntos se conecta el punto i: con el otro lado de
// su puerta, y con todos los que están en su misma zona (cada zona es un
// rectángulo, así que entre dos puntos de la misma zona hay línea recta).
func neighbors(nodes []vec, i int) []int {
	var out []int
	zi, oki := zoneAt(nodes[i])
	for j := range nodes {
		if j == i {
			continue
		}
		sameDoor := i/2 == j/2 // waypoints() pone los dos lados de cada puerta seguidos
		zj, okj := zoneAt(nodes[j])
		if sameDoor || (oki && okj && zi == zj) {
			out = append(out, j)
		}
	}
	return out
}

// nodesIn devuelve los índices de los puntos que están en la zona z.
func nodesIn(nodes []vec, z zone) []int {
	var out []int
	for i, n := range nodes {
		if nz, ok := zoneAt(n); ok && nz == z {
			out = append(out, i)
		}
	}
	return out
}

// route calcula la ruta más corta de from a to (posiciones de cuadro,
// esquina superior izquierda). Devuelve los puntos por donde pasa, sin
// from y terminando en to. Si los dos están en la misma zona, va derecho.
//
// Es el algoritmo de Dijkstra sobre los puntos de paso, más dos puntos
// extra: el inicio (conectado con los puntos de paso de su zona) y el final.
func route(from, to vec) []vec {
	start, end := from.add(feetOffset), to.add(feetOffset)
	zs, okS := zoneAt(start)
	ze, okE := zoneAt(end)
	if !okS || !okE || zs == ze {
		return []vec{to} // misma zona (o fuera del mapa): en línea recta
	}

	nodes := append(waypoints(), start, end)
	s, e := len(nodes)-2, len(nodes)-1
	edges := func(i int) []int {
		if i == s { // del inicio, a los puntos de paso de su zona
			return nodesIn(nodes[:s], zs)
		}
		out := neighbors(nodes[:s], i)
		if z, ok := zoneAt(nodes[i]); ok && z == ze {
			out = append(out, e) // desde la zona final se llega derecho al destino
		}
		return out
	}

	// Dijkstra: dist[i] es lo menos que cuesta llegar a i desde el inicio.
	dist := make([]float64, len(nodes))
	prev := make([]int, len(nodes))
	done := make([]bool, len(nodes))
	for i := range dist {
		dist[i], prev[i] = math.Inf(1), -1
	}
	dist[s] = 0
	for {
		u := -1
		for i := range nodes {
			if !done[i] && (u == -1 || dist[i] < dist[u]) {
				u = i
			}
		}
		if u == -1 || math.IsInf(dist[u], 1) || u == e {
			break
		}
		done[u] = true
		for _, v := range edges(u) {
			if d := dist[u] + math.Hypot(nodes[v].x-nodes[u].x, nodes[v].y-nodes[u].y); d < dist[v] {
				dist[v], prev[v] = d, u
			}
		}
	}
	if math.IsInf(dist[e], 1) {
		return []vec{to} // no hay camino (no debería pasar): en línea recta
	}

	// Se arma de atrás para adelante y se pasa de pies a esquina de cuadro.
	var path []vec
	for i := prev[e]; i != s; i = prev[i] {
		n := nodes[i]
		path = append([]vec{{n.x - feetOffset.x, n.y - feetOffset.y}}, path...)
	}
	return append(path, to)
}
