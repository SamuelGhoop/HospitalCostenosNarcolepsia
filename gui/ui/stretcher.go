package ui

import (
	"container/heap"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Los traslados van en camilla (pedido de Samuel, 2026-10-11): quien lleva
// al paciente, camillero o médico, camina DETRÁS de la camilla empujándola,
// y el paciente va acostado encima. La camilla va horizontal yendo a
// izquierda o derecha y vertical yendo arriba o abajo.
//
// Para las rutas cuenta la HUELLA de la camilla en el piso (como los pies
// de un personaje), no el dibujo del paciente acostado encima (48×45 a ×3),
// que sobresale por la vista 3/4. Al cambiar de dirección gira 90° de golpe,
// en un tick (GAME_DESIGN §10.8, 2026-10-15; antes giraba de a poco en 18).

// dir es hacia dónde se mueve alguien. dirNone (solo para el A* de nav.go)
// es "todavía no se movió": el primer paso de una ruta no paga el costo de
// girar.
type dir int

const (
	dirNone dir = iota - 1
	dirRight
	dirLeft
	dirUp
	dirDown
)

func (d dir) String() string { return [...]string{"derecha", "izquierda", "arriba", "abajo"}[d] }

// horizontal dice si la camilla va acostada a lo largo de x.
func (d dir) horizontal() bool { return d == dirRight || d == dirLeft }

// step es el paso de una celda en esa dirección.
func (d dir) step() cell {
	return [...]cell{{1, 0}, {-1, 0}, {0, -1}, {0, 1}}[d]
}

// dirOf es la dirección del eje en el que más se movió (dx, dy); ok es
// false si no se movió.
func dirOf(dx, dy float64) (d dir, ok bool) {
	switch {
	case dx == 0 && dy == 0:
		return 0, false
	case math.Abs(dx) >= math.Abs(dy) && dx > 0:
		return dirRight, true
	case math.Abs(dx) >= math.Abs(dy):
		return dirLeft, true
	case dy < 0:
		return dirUp, true
	default:
		return dirDown, true
	}
}

// Medidas de la camilla, en px de pantalla.
const (
	stretcherLen  = 54 // a lo largo de la marcha
	stretcherWid  = 24 // a lo ancho: lo que pisa en el piso
	stretcherGap  = 4  // entre los pies de quien empuja y la camilla
	stretcherLegs = 9  // las patas y las ruedas, debajo de la sábana (solo dibujo)

	// stretcherReachUp: subiendo, la camilla va más adelante, con la punta
	// cerca de la cintura de quien empuja. En la vista 3/4, lo que está
	// justo delante de sus pies queda escondido detrás de su cuerpo. Más de
	// 20 no se puede: la camilla ya no cabría al lado de la cama, entre el
	// suero y el borde de la habitación.
	stretcherReachUp = 20
)

// stretcherFootprint es la huella en el piso de la camilla que empujan unos
// pies en feet yendo en d: delante de ellos, centrada en su línea de marcha.
// Yendo a los lados va a la altura de los pies (su borde de abajo con el de
// la huella de los pies).
func stretcherFootprint(feet image.Point, d dir) image.Rectangle {
	fb := feetBox.Add(feet)
	switch d {
	case dirRight:
		return image.Rect(fb.Max.X+stretcherGap, fb.Max.Y-stretcherWid, fb.Max.X+stretcherGap+stretcherLen, fb.Max.Y)
	case dirLeft:
		return image.Rect(fb.Min.X-stretcherGap-stretcherLen, fb.Max.Y-stretcherWid, fb.Min.X-stretcherGap, fb.Max.Y)
	case dirUp:
		end := fb.Min.Y - stretcherGap - stretcherReachUp
		return image.Rect(feet.X-stretcherWid/2, end-stretcherLen, feet.X+stretcherWid/2, end)
	default:
		return image.Rect(feet.X-stretcherWid/2, fb.Max.Y+stretcherGap, feet.X+stretcherWid/2, fb.Max.Y+stretcherGap+stretcherLen)
	}
}

// ─── Dibujo ────────────────────────────────────────────────────────────

// Colores de la camilla dibujada con código.
var (
	colStretcherFrame = color.RGBA{0x8a, 0x90, 0x99, 0xff}
	colStretcherSheet = color.RGBA{0xff, 0xff, 0xff, 0xff}
)

// stretcherArt dibuja la camilla con código, mientras no haya sprite:
// marco gris con borde negro, sábana blanca y, debajo, dos patas con su
// rueda. La imagen mide la huella más las patas.
func stretcherArt(vertical bool) *image.RGBA {
	w, h := stretcherLen, stretcherWid
	if vertical {
		w, h = stretcherWid, stretcherLen
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h+stretcherLegs))
	fill := func(r image.Rectangle, c color.RGBA) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	top := image.Rect(0, 0, w, h)
	fill(top, colInk)                       // borde
	fill(top.Inset(1), colStretcherFrame)   // marco
	fill(top.Inset(4), colStretcherSheet)   // sábana
	for _, x := range []int{4, w - 4 - 6} { // dos patas con su rueda
		fill(image.Rect(x+2, h, x+4, h+stretcherLegs-3), colStretcherFrame)
		fill(image.Rect(x, h+stretcherLegs-3, x+6, h+stretcherLegs), colInk)
	}
	return img
}

// stretcherImgs: la camilla horizontal y la vertical. Hoy salen de
// stretcherArt; el día que haya sprite basta con cargarlo aquí (la lógica
// solo usa stretcherFootprint y drawStretcher).
var stretcherImgs [2]*ebiten.Image

func loadStretcher() error {
	for i, vertical := range []bool{false, true} {
		stretcherImgs[i] = ebiten.NewImageFromImage(stretcherArt(vertical))
	}
	return nil
}

// drawStretcher dibuja la camilla con su huella en fp: la sábana sobre la
// huella y las patas abajo.
func drawStretcher(dst *ebiten.Image, fp image.Rectangle, d dir) {
	img := stretcherImgs[0]
	if !d.horizontal() {
		img = stretcherImgs[1]
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(fp.Min.X), float64(fp.Min.Y-stretcherLegs))
	dst.DrawImage(img, op)
}

// riderPos es dónde va el cuadro (esquina) del paciente acostado sobre la
// camilla fp: lo que se ve de su cuerpo, centrado a lo largo de la camilla
// y apoyado en la mitad de la sábana, así se ven la otra mitad y las patas
// (lo de arriba sobresale: es el dibujo en 3/4).
func riderPos(fp image.Rectangle) vec {
	cx := fp.Min.X + fp.Dx()/2
	bottom := fp.Max.Y - stretcherLegs - stretcherWid/2
	return vec{float64(cx - (floorBody.Min.X+floorBody.Max.X)/2), float64(bottom - floorBody.Max.Y)}
}

// riderPlacement es la GeoM del paciente sobre la camilla fp yendo en d,
// con la cabeza hacia quien empuja: horizontal, tal cual (volteado si van
// a la izquierda); vertical, rotado como en la cama (90° horario si bajan,
// antihorario si suben).
func riderPlacement(fp image.Rectangle, d dir) ebiten.GeoM {
	var g ebiten.GeoM
	switch d {
	case dirRight, dirLeft:
		if d == dirLeft {
			g.Scale(-1, 1)
			g.Translate(cellWidth, 0)
		}
		g.Scale(spriteScale, spriteScale)
		at := riderPos(fp)
		g.Translate(at.x, at.y)
	default:
		if d == dirDown {
			g.Rotate(math.Pi / 2) // la cabeza (a la izquierda del cuadro) queda arriba, hacia quien empuja
			g.Translate(cellHeight, 0)
		} else {
			g.Rotate(-math.Pi / 2) // la cabeza queda abajo, hacia quien empuja
			// Al rotar al revés, el cuerpo (filas 9 a 24 del cuadro) queda a la
			// derecha: se corre para que quede arriba a la izquierda, como en
			// la cama (bedBody).
			g.Translate(-float64(floorBody.Min.Y/spriteScale), cellWidth)
		}
		g.Scale(spriteScale, spriteScale)
		// El cuerpo rotado (bedBody) centrado a lo ancho de la camilla y un
		// poco arriba, para que se vea su punta de abajo y las patas.
		cx, cy := fp.Min.X+fp.Dx()/2, fp.Min.Y-stretcherLegs+fp.Dy()/2-6
		g.Translate(float64(cx-bedBody.Dx()/2), float64(cy-bedBody.Dy()/2))
	}
	return g
}

// ─── Ruta empujando la camilla ─────────────────────────────────────────

// carryFree dice si quien empuja cabe en la celda k con la camilla delante
// yendo en d: ni sus pies ni la camilla tocan paredes, muebles ni avoid.
func carryFree(k cell, d dir, avoid []image.Rectangle) bool {
	return cellFree(k, avoid) && !rectBlocked(stretcherFootprint(k.center(), d), avoid)
}

// carrySegmentClear revisa un tramo recto de los pies de quien empuja,
// píxel a píxel, con la camilla delante yendo en d.
func carrySegmentClear(a, b vec, d dir, avoid []image.Rectangle) bool {
	steps := int(math.Ceil(math.Hypot(b.x-a.x, b.y-a.y)))
	for i := 0; i <= steps; i++ {
		t := 1.0
		if steps > 0 {
			t = float64(i) / float64(steps)
		}
		p := image.Pt(int(math.Round(a.x+(b.x-a.x)*t)), int(math.Round(a.y+(b.y-a.y)*t)))
		if rectBlocked(feetBox.Add(p), avoid) || rectBlocked(stretcherFootprint(p, d), avoid) {
			return false
		}
	}
	return true
}

// carryPlan es un traslado en camilla ya planeado: desde dónde arranca
// quien empuja (pickup, cerca del paciente) y los tramos rectos que
// recorre, cada uno con su dirección.
type carryPlan struct {
	pickup vec   // esquina del cuadro de quien empuja al arrancar
	path   []vec // final de cada tramo (esquinas)
	dirs   []dir // dirección de cada tramo
}

// anchor es donde termina quien empuja: con la camilla pegada a la cama.
func (p carryPlan) anchor() vec { return p.path[len(p.path)-1] }

// carryTurnCost: lo que cuesta girar la camilla, en celdas. Así la ruta
// sale con pocas vueltas (en L) y no en escalera. Como ahora gira de golpe
// en un tick (sin barrer la pared mientras gira; GAME_DESIGN §10.8,
// 2026-10-15), ya no hace falta un costo de más por girar sin espacio: la
// huella solo tiene que caber en las dos direcciones, antes y después, y
// eso ya lo revisa carryFree en cada una.
const carryTurnCost = 6

// carryPickupCost: lo que cuesta cada celda de distancia entre la camilla
// recién aparecida y el paciente: así aparece junto a él si se puede.
const carryPickupCost = 4

// carrySideCost: lo que cuesta cada celda de paso de costado (correrse con
// la camilla sin girarla). En el pasillo 1, que mide 66 px de alto, la
// camilla vertical no cabe: quien sube desde la recepción gira la camilla
// en la puerta y se corre de costado hasta el pasillo. Cuesta más que
// avanzar, así que solo se usa donde hace falta.
const carrySideCost = 3

// opposite es la dirección contraria.
func (d dir) opposite() dir { return [...]dir{dirLeft, dirRight, dirDown, dirUp}[d] }

// bedApproach dice si con los pies en la celda k, yendo en d, la camilla
// queda pegada al costado derecho de la cama de la habitación, a la altura
// de la cama: de ahí el paciente pasa de lado. Puede llegar horizontal
// (empujada hacia la izquierda) o vertical (subiendo desde la puerta, con
// quien empuja abajo): a ×3, a la derecha de la cama no queda sitio para
// llegar por la derecha.
func bedApproach(k cell, d dir, room zone) bool {
	fp := stretcherFootprint(k.center(), d)
	bed := bedRect(room)
	cy := fp.Min.Y + fp.Dy()/2
	return fp.Min.X >= bed.Max.X && fp.Min.X <= bed.Max.X+navCell*2 && cy > bed.Min.Y+navCell && cy < bed.Max.Y-navCell
}

// planCarry planea el traslado en camilla desde donde está quien atiende
// (from, esquina de su cuadro) hasta la cama de room. Es un Dijkstra sobre
// (celda, hacia dónde apunta la camilla), con pasos rectos: hacia delante,
// girando (la camilla tiene que caber en las dos direcciones en la celda
// del giro, con un costo por girar) o de costado (sin girarla, con más
// costo); nunca hacia atrás. En cada celda la camilla cabe delante.
//
// patientAt es la esquina del cuadro del paciente acostado en el piso: la
// camilla aparece lo más cerca posible de él (cada celda de distancia
// entre la camilla y su cuerpo cuesta carryPickupCost), y de ahí sale.
func planCarry(patientAt vec, room zone, avoid []image.Rectangle) (carryPlan, bool) {
	chain, ok := carrySearch(patientAt, avoid, func(cell) bool { return true },
		func(s carryState) bool { return bedApproach(s.k, s.d, room) })
	if !ok {
		return carryPlan{}, false
	}
	corner := func(k cell) vec { return toVec(k.center()).add(vec{-feetOffset.x, -feetOffset.y}) }
	plan := carryPlan{pickup: corner(chain[0].k)}
	if len(chain) == 1 { // ya arranca junto a la cama
		plan.path, plan.dirs = []vec{corner(chain[0].k)}, []dir{chain[0].d}
		return plan, true
	}
	// Un tramo por cada dirección seguida de la camilla y del paso: se
	// guarda el final de cada uno.
	move := func(i int) cell { return cell{chain[i].k.c - chain[i-1].k.c, chain[i].k.r - chain[i-1].k.r} }
	for i := 1; i < len(chain); i++ {
		if i+1 < len(chain) && chain[i+1].d == chain[i].d && move(i+1) == move(i) {
			continue
		}
		plan.path = append(plan.path, corner(chain[i].k))
		plan.dirs = append(plan.dirs, chain[i].d)
	}
	return plan, true
}

// pickupRadius: a cuántas celdas del paciente puede aparecer la camilla
// (6 celdas = 48 px): junto a él, así no se desliza lejos para subirse.
const pickupRadius = 6

// carryState es un estado del Dijkstra de la camilla: dónde están los pies
// de quien empuja y hacia dónde apunta la camilla.
type carryState struct {
	k cell
	d dir
}

// carrySearch es el Dijkstra de planCarry: desde los sitios junto al
// paciente acostado en patientAt donde puede aparecer la camilla, hasta el
// primer estado que cumple goal, pasando solo por celdas que cumplen
// inside. Devuelve los estados del camino, del primero al último.
func carrySearch(patientAt vec, avoid []image.Rectangle, inside func(cell) bool, goal func(carryState) bool) ([]carryState, bool) {
	idx := func(s carryState) int { return (s.k.r*navCols+s.k.c)*4 + int(s.d) }
	dist := map[int]float64{}
	prev := map[int]carryState{}
	open := &statePQ{}

	body := floorBody.Add(image.Pt(int(patientAt.x), int(patientAt.y)))
	bc := body.Min.Add(body.Max).Div(2)
	start := feetCellOf(toVec(bc))
	for dc := -pickupRadius; dc <= pickupRadius; dc++ {
		for dr := -pickupRadius; dr <= pickupRadius; dr++ {
			k := cell{start.c + dc, start.r + dr}
			for d := dirRight; d <= dirDown; d++ {
				if !inside(k) || !carryFree(k, d, avoid) {
					continue
				}
				fp := stretcherFootprint(k.center(), d)
				fc := fp.Min.Add(fp.Max).Div(2)
				cost := carryPickupCost * math.Hypot(float64(fc.X-bc.X), float64(fc.Y-bc.Y)) / navCell
				s := carryState{k, d}
				dist[idx(s)] = cost
				heap.Push(open, stateItem{s.k, s.d, cost})
			}
		}
	}
	done := map[int]bool{}
	for open.Len() > 0 {
		it := heap.Pop(open).(stateItem)
		s := carryState{it.k, it.d}
		if done[idx(s)] {
			continue
		}
		done[idx(s)] = true
		if goal(s) {
			var chain []carryState // de atrás para adelante
			for {
				chain = append([]carryState{s}, chain...)
				p, ok := prev[idx(s)]
				if !ok {
					return chain, true
				}
				s = p
			}
		}
		relax := func(n carryState, cost float64) {
			if old, seen := dist[idx(n)]; inside(n.k) && !done[idx(n)] && (!seen || cost < old) {
				dist[idx(n)] = cost
				prev[idx(n)] = s
				heap.Push(open, stateItem{n.k, n.d, cost})
			}
		}
		for m := dirRight; m <= dirDown; m++ {
			if m == s.d.opposite() {
				continue // hacia atrás no: la camilla va siempre delante
			}
			next := cell{s.k.c + m.step().c, s.k.r + m.step().r}
			// Avanzar (o girar y avanzar): la camilla apunta hacia donde va.
			if carryFree(s.k, m, avoid) && carryFree(next, m, avoid) {
				cost := dist[idx(s)] + cellCost(next)
				if m != s.d {
					cost += carryTurnCost
				}
				relax(carryState{next, m}, cost)
			}
			// De costado: se corre sin girar la camilla.
			if m != s.d && carryFree(next, s.d, avoid) {
				relax(carryState{next, s.d}, dist[idx(s)]+cellCost(next)*carrySideCost)
			}
		}
	}
	return nil, false
}

// carryCanLeave dice si a un paciente acostado en patientAt, en la zona z,
// lo puede recoger la camilla junto a él y sacarlo derecho al pasillo 1
// por la puerta de su zona, sin pasar por otra (así un puesto de piso no
// queda, por ejemplo, debajo de una mesa de la cafetería).
func carryCanLeave(patientAt vec, z zone) bool {
	inside := func(k cell) bool {
		zz, ok := zoneAt(toVec(k.center()))
		return !ok || zz == z || zz == zoneHallway1 // los huecos de las puertas no son de ninguna zona
	}
	goal := func(s carryState) bool {
		zz, ok := zoneAt(toVec(s.k.center()))
		return ok && zz == zoneHallway1
	}
	_, ok := carrySearch(patientAt, nil, inside, goal)
	return ok
}

// stateItem y statePQ: la cola de prioridad de planCarry.
type stateItem struct {
	k    cell
	d    dir
	cost float64
}
type statePQ []stateItem

func (q statePQ) Len() int           { return len(q) }
func (q statePQ) Less(i, j int) bool { return q[i].cost < q[j].cost }
func (q statePQ) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *statePQ) Push(x any)        { *q = append(*q, x.(stateItem)) }
func (q *statePQ) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}
