package ui

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// El tráfico de la calle (pedido de Samuel, 2026-10-11), en lugar del taxi
// único: carros de colores que van por el único carril, de izquierda a
// derecha. Cada uno acelera y frena suave, guarda distancia con el de
// adelante y, cuando hay espacio, vuelve a su velocidad. Es decorado: usa
// su propio azar (con semilla fija en los tests) y avanza por ticks.
//
// Queda lista la base del bus del evento de F5: un carro con stopAt frena
// hasta que su frente queda ahí (el paradero), y los de atrás hacen fila.

// Medidas del tráfico, en px de pantalla y ticks.
const (
	carWidth    = 132  // las 3 capas del carro miden 132×72
	carY        = 600  // esquina superior izquierda de los carros, en el carril
	carMinGap   = 40   // distancia mínima con el de adelante
	carMaxAccel = 0.05 // lo más que cambia la velocidad en un tick, acelerando o frenando
	carMinSpeed = 2.5  // velocidad deseada: al azar entre carMinSpeed…
	carMaxSpeed = 4.5  // …y carMaxSpeed px por tick
	carSpawnMin = 120  // un carro nuevo cada carSpawnMin…
	carSpawnMax = 480  // …a carSpawnMax ticks (de 2 a 8 s)
	taxiPercent = 40   // de cada 100 carros, más o menos 40 son taxis
)

// Colores de los carros: el taxi y los demás.
var (
	taxiColor = color.RGBA{255, 220, 70, 255}
	carColors = []color.RGBA{
		{229, 72, 77, 255},   // rojo
		{80, 140, 230, 255},  // azul
		{245, 245, 240, 255}, // blanco
		{60, 190, 95, 255},   // verde
		{150, 156, 165, 255}, // gris
	}
)

// car es un carro de la calle. x es su borde izquierdo y v lo que avanza por
// tick.
type car struct {
	x, v   float64
	want   float64 // la velocidad a la que quiere ir
	color  color.RGBA
	taxi   bool    // con letrero
	stopAt float64 // si es mayor que 0, frena para que su frente quede ahí (el bus en el paradero, F5)
}

// traffic son los carros de la calle, del de adelante (el más a la derecha)
// al de atrás.
type traffic struct {
	rng     *rand.Rand
	cars    []*car
	next    int // ticks que faltan para que intente entrar otro carro
	spawned int // carros que han entrado (para los tests)
}

func newTraffic(rng *rand.Rand) *traffic {
	return &traffic{rng: rng}
}

// brakingDistance es lo que avanza todavía, frenando al máximo, un carro
// que va a v: v−a + v−2a + … mientras sea positivo.
func brakingDistance(v float64) float64 {
	n := math.Floor(v / carMaxAccel)
	return n*v - carMaxAccel*n*(n+1)/2
}

// safeSpeed es la mayor velocidad u para este tick, entre lo que permite
// acelerar o frenar (de v−a a v+a) y su velocidad deseada, con la que el
// carro de borde izquierdo x no se acerca a menos de carMinGap de un
// obstáculo cuyo borde de atrás está en rear y va a vRear: ni ahora ni si,
// de aquí en adelante, los dos frenan al máximo. Si ninguna sirve, frena al
// máximo.
func safeSpeed(x, v, want, rear, vRear float64) float64 {
	lo := math.Max(0, v-carMaxAccel)
	hi := math.Max(lo, math.Min(v+carMaxAccel, want))
	safe := func(u float64) bool {
		front := x + u + carWidth
		return rear-front >= carMinGap && rear+brakingDistance(vRear)-(front+brakingDistance(u)) >= carMinGap
	}
	if safe(hi) {
		return hi
	}
	if !safe(lo) {
		return lo
	}
	for i := 0; i < 40; i++ { // bisección: safe es verdadero para u chicos y falso para grandes
		mid := (lo + hi) / 2
		if safe(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	if lo < carCreep && v <= carMaxAccel {
		return 0 // casi pegado: se detiene en vez de arrastrarse
	}
	return lo
}

// carCreep: por debajo de esto (px por tick) un carro que puede frenar del
// todo se detiene.
const carCreep = 0.01

// update avanza el tráfico un tick: mueve los carros (del de adelante al de
// atrás, así cada uno ya sabe dónde quedó el suyo), saca los que salieron
// por la derecha e intenta que entre uno nuevo.
func (t *traffic) update() {
	for i, c := range t.cars {
		rear, vRear := math.Inf(1), 0.0 // sin nadie adelante
		if i > 0 {
			rear, vRear = t.cars[i-1].x, t.cars[i-1].v
		}
		if c.stopAt > 0 && c.stopAt+carMinGap < rear {
			rear, vRear = c.stopAt+carMinGap, 0 // un obstáculo quieto donde tiene que parar
		}
		c.v = safeSpeed(c.x, c.v, c.want, rear, vRear)
		c.x += c.v
	}
	for len(t.cars) > 0 && t.cars[0].x > ScreenWidth {
		t.cars = t.cars[1:]
	}

	if t.next > 0 {
		t.next--
		return
	}
	c := &car{x: -carWidth, want: carMinSpeed + t.rng.Float64()*(carMaxSpeed-carMinSpeed)}
	c.v = c.want
	if n := len(t.cars); n > 0 {
		last := t.cars[n-1]
		if safeSpeed(c.x-c.v, c.v, c.want, last.x, last.v) < c.want {
			return // la entrada está ocupada: lo intenta en el próximo tick
		}
	}
	if t.rng.Intn(100) < taxiPercent {
		c.color, c.taxi = taxiColor, true
	} else {
		c.color = carColors[t.rng.Intn(len(carColors))]
	}
	t.cars = append(t.cars, c)
	t.spawned++
	t.next = carSpawnMin + t.rng.Intn(carSpawnMax-carSpawnMin+1)
}

// draw dibuja los carros: los detalles (contorno, vidrios, luces y sombra),
// la carrocería teñida de su color y, en los taxis, el letrero encima.
func (t *traffic) draw(dst *ebiten.Image) {
	for _, c := range t.cars {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(c.x), carY)
		dst.DrawImage(carDetailsImg, op)
		body := *op
		body.ColorScale.ScaleWithColor(c.color)
		dst.DrawImage(carBodyImg, &body)
		if c.taxi {
			dst.DrawImage(carSignImg, op)
		}
	}
}
