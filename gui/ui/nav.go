package ui

import (
	"container/heap"
	"image"
	"math"
	"sync"
)

// Rutas por una cuadrícula de piso libre (GAME_DESIGN §10.7): la pantalla
// se parte en celdas de 8 px y cada celda dice si los PIES de un personaje
// caben ahí sin tocar una pared ni un mueble. El camino sale con A* y se
// suaviza para que no se vea en escalera.
//
// Las rutas son para los pies: lo que pisa el suelo. El cuerpo se dibuja
// encima y puede rozar el marco de una puerta o, en el pasillo 1, montarse
// con la cabeza sobre la pared de arriba (vista 3/4).
//
// Movimiento "cuadriculado" (pedido de Samuel, 2026-10-14; GAME_DESIGN §10.8,
// 2026-10-15): solo 4 direcciones, sin diagonales. El A* ya no da pasos
// diagonales y penaliza cada cambio de dirección (navTurnCost), así prefiere
// pocos tramos largos; el suavizado (smooth) solo quita puntos colineales,
// para no volver a introducir un tramo en ángulo. Como el punto de partida y
// el de llegada no siempre caen justo en el centro de una celda, axisAlign
// parte esos dos tramos en una escuadra (un tramo horizontal y uno vertical)
// en vez de dejarlos en diagonal.

// navCell: el lado de una celda de la cuadrícula, en px.
const navCell = 8

// Tamaño de la cuadrícula: toda la pantalla.
const (
	navCols = ScreenWidth / navCell
	navRows = ScreenHeight / navCell
)

// navPenalty: cuánto más cuesta pisar una celda donde el cuerpo quedaría
// encima de una pared (fuera del pasillo 1 y de las puertas). El A* la usa
// solo si no hay otro camino.
const navPenalty = 3

// feetBox es la huella de los pies alrededor del punto de los pies (abajo al
// centro del cuadro): las piernas, x de 3 a 13 de la hoja, en las 3 filas
// de abajo. A ×3 mide 30×9 px.
var feetBox = image.Rect(
	3*spriteScale-figWidth/2, 21*spriteScale-(figHeight-1),
	13*spriteScale-figWidth/2, 24*spriteScale-(figHeight-1),
)

// cell es una celda de la cuadrícula: columna y fila.
type cell struct{ c, r int }

// center es el punto de los pies cuando están en esa celda.
func (k cell) center() image.Point { return image.Pt(k.c*navCell+navCell/2, k.r*navCell+navCell/2) }

func (k cell) inGrid() bool { return k.c >= 0 && k.c < navCols && k.r >= 0 && k.r < navRows }

// feetCellOf es la celda donde caen unos pies (dentro de la cuadrícula).
func feetCellOf(v vec) cell {
	k := cell{int(math.Floor(v.x)) / navCell, int(math.Floor(v.y)) / navCell}
	k.c = max(0, min(navCols-1, k.c))
	k.r = max(0, min(navRows-1, k.r))
	return k
}

// ─── Mapa de lo que se puede pisar ─────────────────────────────────────

// Sumas acumuladas (integral) de dos mapas de píxeles: blocked (pared o
// mueble) y wall (solo pared). Con ellas, saber si un rectángulo toca algo
// cuesta 4 sumas, sin recorrer sus píxeles.
var (
	navOnce             sync.Once
	blockedSum, wallSum []int32
	sumWidth            = ScreenWidth + 1
)

// navMaps arma los dos mapas la primera vez que se usan. El piso es el de
// las zonas, los huecos de las puertas y la acera; lo demás (incluida la
// calle) es pared. Los muebles son obstacles.
func navMaps() {
	navOnce.Do(func() {
		floor := make([]bool, ScreenWidth*ScreenHeight)
		paint := func(r image.Rectangle, v bool, dst []bool) {
			r = r.Intersect(image.Rect(0, 0, ScreenWidth, ScreenHeight))
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					dst[y*ScreenWidth+x] = v
				}
			}
		}
		for _, z := range zoneOrder {
			paint(zones[z].rect, true, floor)
		}
		for _, d := range doorways {
			paint(d.gap, true, floor)
		}
		paint(sidewalkRect, true, floor)

		blocked := make([]bool, len(floor))
		for i, f := range floor {
			blocked[i] = !f
		}
		for _, o := range obstacles {
			paint(o, true, blocked)
		}
		blockedSum, wallSum = integral(blocked), integral(invert(floor))
	})
}

func invert(b []bool) []bool {
	out := make([]bool, len(b))
	for i, v := range b {
		out[i] = !v
	}
	return out
}

// integral es la tabla de sumas acumuladas: s[y][x] cuenta los píxeles
// marcados en [0,x)×[0,y).
func integral(b []bool) []int32 {
	s := make([]int32, sumWidth*(ScreenHeight+1))
	for y := 0; y < ScreenHeight; y++ {
		var row int32
		for x := 0; x < ScreenWidth; x++ {
			if b[y*ScreenWidth+x] {
				row++
			}
			s[(y+1)*sumWidth+x+1] = s[y*sumWidth+x+1] + row
		}
	}
	return s
}

// count dice cuántos píxeles marcados hay en r. Lo que queda fuera de la
// pantalla no cuenta: por ahí solo se camina fuera de cuadro, por la acera.
// Antes de leer blockedSum o wallSum hay que llamar navMaps().
func count(s []int32, r image.Rectangle) int32 {
	r = r.Intersect(image.Rect(0, 0, ScreenWidth, ScreenHeight))
	if r.Empty() {
		return 0
	}
	return s[r.Max.Y*sumWidth+r.Max.X] - s[r.Min.Y*sumWidth+r.Max.X] - s[r.Max.Y*sumWidth+r.Min.X] + s[r.Min.Y*sumWidth+r.Min.X]
}

// rectBlocked dice si r toca una pared, un mueble o alguno de avoid (por
// ejemplo, la camilla de otro).
func rectBlocked(r image.Rectangle, avoid []image.Rectangle) bool {
	navMaps()
	if count(blockedSum, r) > 0 {
		return true
	}
	for _, a := range avoid {
		if r.Overlaps(a) {
			return true
		}
	}
	return false
}

// feetBlocked dice si unos pies en feet tocan una pared o un mueble.
func feetBlocked(feet image.Point) bool { return rectBlocked(feetBox.Add(feet), nil) }

// cellFree dice si los pies caben en la celda k.
func cellFree(k cell, avoid []image.Rectangle) bool {
	return k.inGrid() && !rectBlocked(feetBox.Add(k.center()), avoid)
}

// feetOffsetPt es feetOffset en enteros: de la esquina del cuadro a los pies.
var feetOffsetPt = image.Pt(figWidth/2, figHeight-1)

// doorCorridor es la franja alrededor de una puerta donde el cuerpo puede
// rozar el marco al cruzarla: en una puerta de pared horizontal, desde el
// hueco hasta un cuerpo de alto más abajo (los pies ya pasaron y la cabeza
// todavía no); en una de pared vertical, medio cuerpo a cada lado.
func doorCorridor(d doorway) image.Rectangle {
	g := d.gap
	if g.Dx() >= g.Dy() {
		return image.Rect(g.Min.X-navCell, g.Min.Y-navCell, g.Max.X+navCell, g.Max.Y+figHeight)
	}
	return image.Rect(g.Min.X-figWidth/2-navCell, g.Min.Y-navCell, g.Max.X+figWidth/2+navCell, g.Max.Y+navCell)
}

// penaltyAt dice si con los pies en feet el cuerpo quedaría encima de una
// pared donde no debe: en el pasillo 1 se le perdona la cabeza (la pared de
// arriba), en las puertas el roce con el marco, y afuera el cuerpo no se
// monta sobre la fachada salvo frente a la puerta principal.
func penaltyAt(feet image.Point) bool {
	navMaps()
	for _, d := range doorways {
		if feet.In(doorCorridor(d)) {
			return false
		}
	}
	body := standingBody.Add(feet.Sub(feetOffsetPt))
	z, ok := zoneAt(toVec(feet))
	switch {
	case ok && z == outside:
		return body.Overlaps(buildingRect)
	case ok && z == zoneHallway1:
		body.Min.Y = max(body.Min.Y, zones[zoneHallway1].rect.Min.Y)
	}
	return count(wallSum, body) > 0
}

// ─── A* ────────────────────────────────────────────────────────────────

// navCost es lo que cuesta entrar a una celda (1 o, si el cuerpo quedaría
// sobre una pared, 1 + navPenalty). Se calcula una vez por celda.
var (
	costOnce sync.Once
	navCosts []float64
)

func cellCost(k cell) float64 {
	costOnce.Do(func() {
		navCosts = make([]float64, navCols*navRows)
		for r := 0; r < navRows; r++ {
			for c := 0; c < navCols; c++ {
				navCosts[r*navCols+c] = 1
				if penaltyAt(cell{c, r}.center()) {
					navCosts[r*navCols+c] = 1 + navPenalty
				}
			}
		}
	})
	return navCosts[k.r*navCols+k.c]
}

// navTurnCost: cuánto más cuesta entrar a una celda cambiando de dirección.
// Así el A* prefiere pocos tramos largos (menos vueltas) y no zigzaguea
// entre dos caminos igual de cortos (GAME_DESIGN §10.8, 2026-10-15).
const navTurnCost = 4

// navState es un estado del A*: la celda y la dirección con la que se
// entró a ella. Hace falta la dirección (y no solo la celda) porque entrar
// por un lado o por otro cuesta distinto: girar tiene el costo de más.
type navState struct {
	k cell
	d dir // dirNone al arrancar: el primer paso nunca paga el costo de girar
}

// navDirs son las 4 direcciones en las que puede dar un paso el A*.
var navDirs = [4]dir{dirRight, dirLeft, dirUp, dirDown}

// navStateIdx numera (celda, dirección) para los mapas del A*: 5 valores de
// dirección (dirNone y las 4 direcciones) por celda.
func navStateIdx(s navState) int { return (s.k.r*navCols+s.k.c)*5 + int(s.d) + 1 }

// navPQItem y navPQ: la cola de prioridad del A* (el estado con menos costo
// estimado sale primero).
type navPQItem struct {
	s navState
	f float64
}
type navPQ []navPQItem

func (q navPQ) Len() int           { return len(q) }
func (q navPQ) Less(i, j int) bool { return q[i].f < q[j].f }
func (q navPQ) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *navPQ) Push(x any)        { *q = append(*q, x.(navPQItem)) }
func (q *navPQ) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// manhattan es la distancia en la cuadrícula con solo pasos rectos (sin
// diagonales): la estimación del A*, que nunca se pasa de lo que cuesta de
// verdad.
func manhattan(a, b cell) float64 {
	return math.Abs(float64(a.c-b.c)) + math.Abs(float64(a.r-b.r))
}

// astar busca el camino más barato de start a goal por celdas libres, con
// pasos en las 4 direcciones (sin diagonales): el movimiento "cuadriculado"
// pedido por Samuel. Cambiar de dirección cuesta navTurnCost de más, para
// que prefiera tramos largos. Devuelve las celdas de start a goal, o nil si
// no hay camino.
func astar(start, goal cell, avoid []image.Rectangle) []cell {
	if !cellFree(start, avoid) || !cellFree(goal, avoid) {
		return nil
	}
	if start == goal {
		return []cell{start}
	}
	s0 := navState{start, dirNone}
	g := map[int]float64{navStateIdx(s0): 0}
	prev := map[int]navState{}
	closed := map[int]bool{}
	open := &navPQ{{s0, manhattan(start, goal)}}
	for open.Len() > 0 {
		cur := heap.Pop(open).(navPQItem).s
		if cur.k == goal {
			path := []cell{goal}
			for s := cur; s != s0; {
				p := prev[navStateIdx(s)]
				path = append([]cell{p.k}, path...)
				s = p
			}
			return path
		}
		if closed[navStateIdx(cur)] {
			continue
		}
		closed[navStateIdx(cur)] = true
		for _, m := range navDirs {
			next := cell{cur.k.c + m.step().c, cur.k.r + m.step().r}
			ns := navState{next, m}
			if closed[navStateIdx(ns)] || !cellFree(next, avoid) {
				continue
			}
			cost := g[navStateIdx(cur)] + cellCost(next)
			if cur.d != dirNone && cur.d != m {
				cost += navTurnCost
			}
			if old, seen := g[navStateIdx(ns)]; !seen || cost < old {
				g[navStateIdx(ns)] = cost
				prev[navStateIdx(ns)] = cur
				heap.Push(open, navPQItem{ns, cost + manhattan(next, goal)})
			}
		}
	}
	return nil
}

// nearestFreeCell es la celda libre más cercana a k (k misma si está
// libre), buscando en anillos cada vez más grandes.
func nearestFreeCell(k cell, avoid []image.Rectangle) (cell, bool) {
	for radius := 0; radius < 30; radius++ {
		best, found := cell{}, false
		bestD := math.Inf(1)
		for dc := -radius; dc <= radius; dc++ {
			for dr := -radius; dr <= radius; dr++ {
				if max(abs(dc), abs(dr)) != radius {
					continue
				}
				n := cell{k.c + dc, k.r + dr}
				if d := math.Hypot(float64(dc), float64(dr)); cellFree(n, avoid) && d < bestD {
					best, bestD, found = n, d, true
				}
			}
		}
		if found {
			return best, true
		}
	}
	return k, false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// freeFeetNear es el centro de la celda libre más cercana a p.
func freeFeetNear(p image.Point) (image.Point, bool) {
	k, ok := nearestFreeCell(feetCellOf(toVec(p)), nil)
	return k.center(), ok
}

// ─── A dónde se llega caminando ────────────────────────────────────────

// reachable marca las celdas a las que se llega caminando desde la puerta
// principal. Los muebles dejan rincones encerrados (por ejemplo, la mitad
// de radiología detrás del escáner): ahí no puede ir ningún puesto.
var (
	reachOnce sync.Once
	reachable []bool
)

// reachableCell dice si se llega caminando a la celda k desde la puerta
// principal.
func reachableCell(k cell) bool {
	reachOnce.Do(func() {
		reachable = make([]bool, navCols*navRows)
		door := doorways[len(doorways)-1].gap
		start, ok := nearestFreeCell(feetCellOf(vec{float64(door.Min.X + door.Dx()/2), float64(door.Min.Y - navCell)}), nil)
		if !ok {
			return
		}
		reachable[start.r*navCols+start.c] = true
		queue := []cell{start}
		for len(queue) > 0 {
			k := queue[0]
			queue = queue[1:]
			for _, d := range []cell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				n := cell{k.c + d.c, k.r + d.r}
				if cellFree(n, nil) && !reachable[n.r*navCols+n.c] {
					reachable[n.r*navCols+n.c] = true
					queue = append(queue, n)
				}
			}
		}
	})
	return k.inGrid() && reachable[k.r*navCols+k.c]
}

// standable dice si alguien puede quedarse con los pies en feet: la huella
// no toca nada y se llega ahí caminando desde la puerta principal.
func standable(feet image.Point) bool {
	if feetBlocked(feet) {
		return false
	}
	k, ok := nearestFreeCell(feetCellOf(toVec(feet)), nil)
	return ok && reachableCell(k) && segmentClear(toVec(k.center()), toVec(feet), nil)
}

// reachableFeetNear es el centro de la celda alcanzable más cercana a p.
func reachableFeetNear(p image.Point) (image.Point, bool) {
	k := feetCellOf(toVec(p))
	for radius := 0; radius < 60; radius++ {
		best, found, bestD := cell{}, false, math.Inf(1)
		for dc := -radius; dc <= radius; dc++ {
			for dr := -radius; dr <= radius; dr++ {
				n := cell{k.c + dc, k.r + dr}
				if max(abs(dc), abs(dr)) != radius || !reachableCell(n) {
					continue
				}
				if d := math.Hypot(float64(dc), float64(dr)); d < bestD {
					best, bestD, found = n, d, true
				}
			}
		}
		if found {
			return best.center(), true
		}
	}
	return p, false
}

// ─── Ruta suavizada ────────────────────────────────────────────────────

// segmentClear revisa un tramo recto de los pies, píxel a píxel: la huella
// no toca paredes, muebles ni nada de avoid.
func segmentClear(a, b vec, avoid []image.Rectangle) bool {
	steps := int(math.Ceil(math.Hypot(b.x-a.x, b.y-a.y)))
	for i := 0; i <= steps; i++ {
		t := 1.0
		if steps > 0 {
			t = float64(i) / float64(steps)
		}
		p := image.Pt(int(math.Round(a.x+(b.x-a.x)*t)), int(math.Round(a.y+(b.y-a.y)*t)))
		if rectBlocked(feetBox.Add(p), avoid) {
			return false
		}
	}
	return true
}

// segmentCost es el costo más alto de las celdas por las que pasa un tramo.
func segmentCost(a, b vec) float64 {
	worst := 0.0
	steps := int(math.Ceil(math.Hypot(b.x-a.x, b.y-a.y)))
	for i := 0; i <= steps; i++ {
		t := 1.0
		if steps > 0 {
			t = float64(i) / float64(steps)
		}
		worst = math.Max(worst, cellCost(feetCellOf(vec{a.x + (b.x-a.x)*t, a.y + (b.y-a.y)*t})))
	}
	return worst
}

// colinear dice si a y b caen en la misma fila o la misma columna: el tramo
// entre ellos es horizontal o vertical. Con el movimiento cuadriculado
// (2026-10-15), smooth solo junta puntos colineales: así nunca vuelve a
// armar un tramo en ángulo.
func colinear(a, b vec) bool { return a.x == b.x || a.y == b.y }

// smooth quita los puntos intermedios de un camino mientras el tramo que
// los salta siga siendo colineal con el anterior, siga libre y no pase por
// celdas más caras que las del camino original: así un pasillo recto de
// muchas celdas se ve como un solo tramo largo, en vez de en escalera.
func smooth(pts []vec, avoid []image.Rectangle) []vec {
	if len(pts) < 3 {
		return pts
	}
	out := []vec{pts[0]}
	for i := 0; i < len(pts)-1; {
		j := i + 1
		allowed := math.Max(segmentCost(pts[i], pts[i+1]), 1)
		for j+1 < len(pts) && colinear(pts[i], pts[j+1]) {
			allowed = math.Max(allowed, segmentCost(pts[j], pts[j+1]))
			if !segmentClear(pts[i], pts[j+1], avoid) || segmentCost(pts[i], pts[j+1]) > allowed {
				break
			}
			j++
		}
		out = append(out, pts[j])
		i = j
	}
	return out
}

// elbow parte el tramo de a a b en una escuadra (primero horizontal, si
// cabe; si no, vertical) cuando no son colineales: hace falta porque el
// punto de partida y el de llegada de una ruta son la posición real de los
// pies, no necesariamente el centro de una celda, así que el primer y el
// último tramo no siempre salen ya alineados con la cuadrícula.
func elbow(a, b vec, avoid []image.Rectangle) []vec {
	if colinear(a, b) {
		return []vec{b}
	}
	h := vec{b.x, a.y}
	if segmentClear(a, h, avoid) && segmentClear(h, b, avoid) {
		return []vec{h, b}
	}
	v := vec{a.x, b.y} // si tampoco cabe (no debería pasar), sigue igual
	return []vec{v, b}
}

// axisAlign pasa por todos los puntos de un camino y, donde dos seguidos no
// queden en la misma fila o columna, les mete la escuadra de elbow en el
// medio.
func axisAlign(pts []vec, avoid []image.Rectangle) []vec {
	out := []vec{pts[0]}
	for i := 1; i < len(pts); i++ {
		out = append(out, elbow(out[len(out)-1], pts[i], avoid)...)
	}
	return out
}

// route calcula la ruta de from a to (posiciones de cuadro, esquina
// superior izquierda). Devuelve los puntos por donde pasa, sin from y
// terminando exactamente en to.
func route(from, to vec) []vec { return routeAvoiding(from, to, nil) }

// routeAvoiding es route esquivando además los rectángulos de avoid (la
// camilla de otro, por ejemplo).
func routeAvoiding(from, to vec, avoid []image.Rectangle) []vec {
	if from == to {
		return []vec{to}
	}
	start, goal := from.add(feetOffset), to.add(feetOffset)
	sc, okS := nearestFreeCell(feetCellOf(start), avoid)
	gc, okG := nearestFreeCell(feetCellOf(goal), avoid)
	var cells []cell
	if okS && okG {
		cells = astar(sc, gc, avoid)
	}
	if cells == nil && len(avoid) > 0 {
		// La camilla de otro tapa el único paso (en el juego puede haber
		// varias): mejor pasar junto a ella que atravesar una pared.
		return routeAvoiding(from, to, nil)
	}
	if cells == nil {
		return []vec{to} // no hay camino (no debería pasar): en línea recta
	}
	pts := []vec{start}
	for _, k := range cells {
		pts = append(pts, toVec(k.center()))
	}
	// El primer y el último tramo pueden no caer en la cuadrícula (start y
	// goal son la posición real de los pies, no el centro de una celda):
	// axisAlign los parte en escuadra antes de suavizar.
	pts = axisAlign(append(pts, goal), avoid)
	pts = smooth(pts, avoid)

	var path []vec
	for _, p := range pts[1:] {
		path = append(path, vec{p.x - feetOffset.x, p.y - feetOffset.y})
	}
	path[len(path)-1] = to // exacto, sin el redondeo de restar y sumar
	return path
}

// arrivalSpawn es donde aparecen los pacientes nuevos: fuera de la
// pantalla, a la derecha, con los pies en la franja de abajo de la acera.
func arrivalSpawn() vec {
	laneY := (sidewalkRect.Max.Y-feetBox.Max.Y-navCell/2)/navCell*navCell + navCell/2 // centro de la fila más baja que cabe
	return vec{ScreenWidth + navCell, float64(laneY) - feetOffset.y}
}
