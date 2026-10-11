package ui

import (
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// El giro de la camilla (pedido de Samuel, 2026-10-11): cuando la ruta
// cambia de dirección, la camilla no aparece de golpe del otro lado de
// quien empuja; gira de a poco alrededor de sus pies, que se quedan
// quietos, en unos 0,3 s, con el paciente encima.
//
// Lo que se apoya en el piso (la huella de la camilla) no toca paredes ni
// muebles antes ni después del giro: lo revisa planCarry. Mientras gira, el
// DIBUJO se puede montar sobre una pared, como las cabezas en el pasillo 1;
// por eso planCarry prefiere girar donde hay espacio (habitaciones,
// recepción): un giro que barre una pared cuesta más.

// turnTicks: lo que dura un giro (18 ticks = 0,3 s).
const turnTicks = 18

// sub resta dos vectores.
func (v vec) sub(w vec) vec { return vec{v.x - w.x, v.y - w.y} }

// scale multiplica un vector por k.
func (v vec) scale(k float64) vec { return vec{v.x * k, v.y * k} }

// rotate gira un vector theta radianes; en la pantalla (y hacia abajo), un
// ángulo positivo gira en el sentido de las agujas del reloj, como GeoM.Rotate.
func (v vec) rotate(theta float64) vec {
	sin, cos := math.Sincos(theta)
	return vec{v.x*cos - v.y*sin, v.x*sin + v.y*cos}
}

// unit es el vector de una dirección en la pantalla.
func (d dir) unit() vec { return [...]vec{{1, 0}, {-1, 0}, {0, -1}, {0, 1}}[d] }

// turnAngle es lo que gira la camilla de from a to: +90° o -90°.
func turnAngle(from, to dir) float64 {
	a, b := from.unit(), to.unit()
	return math.Atan2(a.x*b.y-a.y*b.x, a.x*b.x+a.y*b.y)
}

// rectCenter es el centro de un rectángulo.
func rectCenter(r image.Rectangle) vec {
	return vec{float64(r.Min.X+r.Max.X) / 2, float64(r.Min.Y+r.Max.Y) / 2}
}

// swing es un giro de la camilla alrededor de los pies de quien empuja
// (pivot), de la dirección from a la dirección to. Su avance t va de 0
// (como iba) a 1 (ya apunta hacia to).
type swing struct {
	pivot    image.Point
	from, to dir
	ts       []float64 // el avance en cada tick: el paciente recorre lo mismo en cada uno
	tick     int       // ticks que lleva girando
}

// newSwing arma un giro y reparte su avance en turnTicks ticks, de modo que
// el paciente encima recorra lo mismo en cada uno (no da saltos).
func newSwing(pivot image.Point, from, to dir) *swing {
	w := &swing{pivot: pivot, from: from, to: to}
	const samples = 96
	lengths := make([]float64, samples+1) // lo recorrido por el paciente hasta cada muestra
	prev := w.riderAt(0)
	for i := 1; i <= samples; i++ {
		p := w.riderAt(float64(i) / samples)
		lengths[i] = lengths[i-1] + math.Hypot(p.x-prev.x, p.y-prev.y)
		prev = p
	}
	for k := 1; k <= turnTicks; k++ {
		want := lengths[samples] * float64(k) / turnTicks
		i := 1
		for i < samples && lengths[i] < want {
			i++
		}
		f := 1.0
		if seg := lengths[i] - lengths[i-1]; seg > 0 {
			f = (want - lengths[i-1]) / seg
		}
		w.ts = append(w.ts, (float64(i-1)+f)/samples)
	}
	w.ts[turnTicks-1] = 1
	return w
}

// progress es el avance del giro en el tick actual.
func (w *swing) progress() float64 {
	if w.tick == 0 {
		return 0
	}
	return w.ts[min(w.tick, len(w.ts))-1]
}

// done dice si la camilla ya terminó de girar.
func (w *swing) done() bool { return w.tick >= len(w.ts) }

// angle es lo que lleva girado la camilla a t del giro.
func (w *swing) angle(t float64) float64 { return turnAngle(w.from, w.to) * t }

// center es el centro de la huella de la camilla a t del giro: gira con la
// camilla alrededor de los pies y, de paso, se corrige poco a poco lo que
// cambia por la vista 3/4 (subiendo, la camilla va más adelante). A t=1
// queda justo donde la pone stretcherFootprint.
func (w *swing) center(t float64) vec {
	p := toVec(w.pivot)
	o0 := rectCenter(stretcherFootprint(w.pivot, w.from)).sub(p)
	o1 := rectCenter(stretcherFootprint(w.pivot, w.to)).sub(p)
	phi := turnAngle(w.from, w.to)
	return p.add(o0.rotate(phi * t)).add(o1.sub(o0.rotate(phi)).scale(t))
}

// riderAt es dónde va el cuadro del paciente a t del giro: con el centro de
// la camilla, de riderPos de antes a riderPos de después.
func (w *swing) riderAt(t float64) vec {
	fp0, fp1 := stretcherFootprint(w.pivot, w.from), stretcherFootprint(w.pivot, w.to)
	rel0 := riderPos(fp0).sub(rectCenter(fp0))
	rel1 := riderPos(fp1).sub(rectCenter(fp1))
	return w.center(t).add(rel0.add(rel1.sub(rel0).scale(t)))
}

// riderStep es lo que avanza el paciente en cada tick del giro.
func (w *swing) riderStep() float64 {
	a, b := w.riderAt(0), w.riderAt(w.ts[0])
	return math.Hypot(b.x-a.x, b.y-a.y)
}

// points son los píxeles de la huella girada a t del giro, cada 3 px (con
// los bordes): la huella de antes, girada alrededor de su centro.
func (w *swing) points(t float64) []image.Point {
	size := stretcherFootprint(w.pivot, w.from).Size()
	c, theta := w.center(t), w.angle(t)
	axis := func(n int) []float64 { // de un borde al otro, cada 3 px como mucho
		k := int(math.Ceil(float64(n-1)/3)) + 1
		out := make([]float64, k)
		for i := range out {
			out[i] = -float64(n)/2 + 0.5 + float64(n-1)*float64(i)/float64(k-1)
		}
		return out
	}
	var out []image.Point
	for _, u := range axis(size.X) {
		for _, v := range axis(size.Y) {
			q := c.add(vec{u, v}.rotate(theta))
			out = append(out, image.Pt(int(math.Floor(q.x)), int(math.Floor(q.y))))
		}
	}
	return out
}

// bounds es el rectángulo que encierra la huella girada a t del giro.
func (w *swing) bounds(t float64) image.Rectangle {
	var r image.Rectangle
	for i, p := range w.points(t) {
		px := image.Rect(p.X, p.Y, p.X+1, p.Y+1)
		if i == 0 {
			r = px
		} else {
			r = r.Union(px)
		}
	}
	return r
}

// swingSteps: en cuántos ángulos se revisa un giro (cada 10°).
const swingSteps = 9

// swingClear dice si la camilla puede girar de from a to alrededor de los
// pies en pivot sin que su huella girada toque paredes, muebles ni avoid
// en ningún momento: un giro "con espacio".
//
// Lo de paredes y muebles no cambia nunca, así que se calcula una sola vez
// por giro (swingWalls); avoid (los pies de los demás) se revisa aparte, y
// punto por punto solo si toca lo que barre el giro.
func swingClear(pivot image.Point, from, to dir, avoid []image.Rectangle) bool {
	walls := swingWalls(pivot, from, to)
	if !walls.clear {
		return false
	}
	for _, a := range avoid {
		if !a.Overlaps(walls.box) {
			continue
		}
		w := swing{pivot: pivot, from: from, to: to}
		for i := 0; i <= swingSteps; i++ {
			for _, p := range w.points(float64(i) / swingSteps) {
				if p.In(a) {
					return false
				}
			}
		}
	}
	return true
}

// swingWallsResult es lo que se guarda de un giro: si barre una pared o un
// mueble y el rectángulo que encierra todo lo que barre.
type swingWallsResult struct {
	clear bool
	box   image.Rectangle
}

type swingKey struct {
	pivot    image.Point
	from, to dir
}

var (
	swingMu   sync.Mutex
	swingMemo = map[swingKey]swingWallsResult{}
)

// swingWalls revisa (una vez por giro) si la huella girada toca una pared o
// un mueble en algún momento del giro.
func swingWalls(pivot image.Point, from, to dir) swingWallsResult {
	key := swingKey{pivot, from, to}
	swingMu.Lock()
	r, ok := swingMemo[key]
	swingMu.Unlock()
	if ok {
		return r
	}
	navMaps()
	w := swing{pivot: pivot, from: from, to: to}
	r.clear = true
	for i := 0; i <= swingSteps; i++ {
		for _, p := range w.points(float64(i) / swingSteps) {
			r.box = r.box.Union(image.Rect(p.X, p.Y, p.X+1, p.Y+1))
			if r.clear && count(blockedSum, image.Rect(p.X, p.Y, p.X+1, p.Y+1)) > 0 {
				r.clear = false
			}
		}
	}
	swingMu.Lock()
	swingMemo[key] = r
	swingMu.Unlock()
	return r
}

// drawSwing dibuja la camilla girando: la imagen de antes del giro, girada
// alrededor del centro de la sábana. rider dibuja al paciente encima con la
// GeoM que le toca.
func drawSwing(dst *ebiten.Image, w *swing, rider func(ebiten.GeoM)) {
	t := w.progress()
	legs := vec{0, stretcherLegs} // la sábana se dibuja encima de las patas
	sheet0 := rectCenter(stretcherFootprint(w.pivot, w.from)).sub(legs)
	sheet := w.center(t).sub(legs)

	img := stretcherImgs[0]
	if !w.from.horizontal() {
		img = stretcherImgs[1]
	}
	size := stretcherFootprint(w.pivot, w.from).Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(size.X)/2, -float64(size.Y)/2)
	op.GeoM.Rotate(w.angle(t))
	op.GeoM.Translate(sheet.x, sheet.y)
	dst.DrawImage(img, op)

	rider(w.riderGeoM(t, sheet0, sheet))
}

// riderBodyCenter es el centro del cuerpo acostado en la hoja (filas 9 a 24
// de un cuadro de 16 de ancho): con él se corrige al paciente al girar.
var riderBodyCenter = vec{8, 16.5}

// riderGeoM es la GeoM del paciente a t del giro: la de antes, girada con
// la sábana, más una corrección que crece con t, para que al terminar su
// cuerpo quede donde lo pone riderPlacement con la camilla ya girada.
func (w *swing) riderGeoM(t float64, sheet0, sheet vec) ebiten.GeoM {
	turned := func(t float64, sheet vec) ebiten.GeoM {
		g := riderPlacement(stretcherFootprint(w.pivot, w.from), w.from)
		g.Translate(-sheet0.x, -sheet0.y)
		g.Rotate(w.angle(t))
		g.Translate(sheet.x, sheet.y)
		return g
	}
	end := turned(1, w.center(1).sub(vec{0, stretcherLegs}))
	want := riderPlacement(stretcherFootprint(w.pivot, w.to), w.to)
	ex, ey := end.Apply(riderBodyCenter.x, riderBodyCenter.y)
	wx, wy := want.Apply(riderBodyCenter.x, riderBodyCenter.y)
	g := turned(t, sheet)
	g.Translate((wx-ex)*t, (wy-ey)*t)
	return g
}
