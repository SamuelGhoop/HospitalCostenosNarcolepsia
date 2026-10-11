package ui

import (
	"image/color"
	"math"
	"math/rand"
	"testing"
)

// El tráfico de la calle (pedido de Samuel, 2026-10-11): un carril, de
// izquierda a derecha; los carros entran cada 2 a 8 s si hay espacio,
// aceleran y frenan suave y guardan distancia con el de adelante.

// runTraffic avanza el tráfico n ticks y revisa en cada uno las reglas que
// valen siempre: nadie se solapa, la distancia mínima, la aceleración
// máxima y que nadie va hacia atrás.
func runTraffic(t *testing.T, tr *traffic, n int) {
	t.Helper()
	speeds := map[*car]float64{}
	for _, c := range tr.cars {
		speeds[c] = c.v
	}
	for tick := 0; tick < n; tick++ {
		tr.update()
		for i, c := range tr.cars {
			if old, ok := speeds[c]; ok && math.Abs(c.v-old) > carMaxAccel+1e-9 {
				t.Fatalf("tick %d: un carro cambió de velocidad %.3f px/tick en un tick; el máximo es %.2f", tick, c.v-old, carMaxAccel)
			}
			if c.v < 0 {
				t.Fatalf("tick %d: un carro va hacia atrás (%.3f)", tick, c.v)
			}
			speeds[c] = c.v
			if i > 0 {
				ahead := tr.cars[i-1]
				if gap := ahead.x - (c.x + carWidth); gap < carMinGap-1e-9 {
					t.Fatalf("tick %d: dos carros a %.1f px (el mínimo es %d)", tick, gap, carMinGap)
				}
			}
		}
	}
}

func TestTraffic_CarsKeepTheirDistanceAndAccelerateSmoothly(t *testing.T) {
	tr := newTraffic(rand.New(rand.NewSource(1)))
	runTraffic(t, tr, 60*60*5) // 5 minutos
	if tr.spawned < 30 {
		t.Errorf("en 5 minutos entraron %d carros; con uno cada 2 a 8 s se esperaban más de 30", tr.spawned)
	}
}

// Los carros entran por la izquierda, fuera de la pantalla, con su
// velocidad deseada entre 2,5 y 4,5 px por tick; ~40 % son taxis (amarillos,
// con letrero) y los demás de los otros 5 colores.
func TestTraffic_CarsEnterFromTheLeftWithTheirSpeedAndColor(t *testing.T) {
	tr := newTraffic(rand.New(rand.NewSource(2)))
	taxis, total := 0, 0
	seen := map[*car]bool{}
	for tick := 0; tick < 60*60*20; tick++ {
		tr.update()
		for _, c := range tr.cars {
			if seen[c] {
				continue
			}
			seen[c] = true
			total++
			if c.x > -carWidth+carMaxSpeed+1 {
				t.Fatalf("un carro apareció en x=%.1f; se esperaba fuera de la pantalla, por la izquierda", c.x)
			}
			if c.want < carMinSpeed || c.want > carMaxSpeed {
				t.Errorf("velocidad deseada %.2f; se esperaba entre %.1f y %.1f", c.want, carMinSpeed, carMaxSpeed)
			}
			if c.taxi {
				taxis++
				if c.color != taxiColor {
					t.Errorf("un taxi de color %v", c.color)
				}
			} else if !containsColor(carColors, c.color) {
				t.Errorf("un carro de color %v, que no está en la lista", c.color)
			}
		}
	}
	if share := float64(taxis) / float64(total); total < 100 || share < 0.3 || share > 0.5 {
		t.Errorf("%d de %d carros fueron taxis (%.0f %%); se esperaba cerca del 40 %%", taxis, total, share*100)
	}
}

func containsColor(cs []color.RGBA, c color.RGBA) bool {
	for _, x := range cs {
		if x == c {
			return true
		}
	}
	return false
}

// Si la entrada está ocupada (un carro detenido recién entrado), no aparece
// otro encima: espera a que haya espacio.
func TestTraffic_NoCarAppearsWhileTheEntranceIsBusy(t *testing.T) {
	tr := newTraffic(rand.New(rand.NewSource(3)))
	tr.cars = []*car{{x: -carWidth + 10, v: 0, want: 0}} // detenido en la entrada
	tr.next = 0
	for tick := 0; tick < 600; tick++ {
		tr.update()
		if len(tr.cars) != 1 {
			t.Fatalf("tick %d: apareció un carro con la entrada ocupada", tick)
		}
	}
	tr.cars[0].want = carMaxSpeed // se va
	runTraffic(t, tr, 600)
	if tr.spawned == 0 {
		t.Error("cuando la entrada se desocupó no volvió a aparecer ningún carro")
	}
}

// Con un carro detenido adelante, el de atrás frena y se detiene sin chocar,
// cerca de la distancia mínima. Es la base del bus en el paradero (F5): un
// carro con stopAt frena hasta que su frente queda ahí, y los de atrás
// hacen fila.
func TestTraffic_TheCarBehindAStoppedOneBrakesAndStops(t *testing.T) {
	tr := newTraffic(rand.New(rand.NewSource(4)))
	tr.next = math.MaxInt32 // que no entren más
	stopped := &car{x: 700, v: 0, want: 0}
	behind := &car{x: -carWidth, v: carMaxSpeed, want: carMaxSpeed}
	tr.cars = []*car{stopped, behind}
	runTraffic(t, tr, 1200)
	if behind.v != 0 {
		t.Errorf("el de atrás sigue andando a %.2f px/tick", behind.v)
	}
	if gap := stopped.x - (behind.x + carWidth); gap > carMinGap+30 {
		t.Errorf("se detuvo a %.1f px del de adelante; se esperaba cerca de %d", gap, carMinGap)
	}

	tr = newTraffic(rand.New(rand.NewSource(5)))
	tr.next = math.MaxInt32
	bus := &car{x: -carWidth, v: 3, want: 3, stopAt: 500}
	queued := &car{x: -carWidth - 200, v: carMaxSpeed, want: carMaxSpeed}
	tr.cars = []*car{bus, queued}
	runTraffic(t, tr, 1200)
	if front := bus.x + carWidth; bus.v != 0 || front > 500+1e-9 || front < 500-carMinGap {
		t.Errorf("con stopAt=500 el carro quedó con el frente en %.1f, a %.2f px/tick; se esperaba detenido justo antes de 500", front, bus.v)
	}
	if queued.v != 0 {
		t.Errorf("el de atrás del bus sigue andando a %.2f px/tick; se esperaba en fila", queued.v)
	}
}
