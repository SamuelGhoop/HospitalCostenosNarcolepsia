package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Lo que se mueve en el mapa sin que tenga que ver con el hospital
// (GAME_DESIGN §10.7): el taxi de la calle y el humo de la olla. Sus PNG ya
// vienen escalados, así que se dibujan a 1×.
const (
	taxiWidth    = 132 // ancho de taxi.png
	taxiY        = 651 // esquina superior izquierda, en el carril de abajo
	taxiSpeed    = 3   // píxeles por tick (cruza la pantalla en unos 8 s)
	taxiGapTicks = 360 // 6 s sin taxi antes de que vuelva a entrar

	steamFrameWidth = 44 // steam_pot.png: 3 cuadros de 44×36 en fila
	steamFrames     = 3
	steamFrameTicks = 18 // ~300 ms por cuadro a 60 ticks por segundo
)

// steamAt es la esquina superior izquierda del humo, sobre la olla.
var steamAt = image.Pt(40, 246)

// taxiCrossTicks: cuántos ticks tarda el taxi en cruzar, desde que asoma
// por la izquierda hasta que sale del todo por la derecha (división
// redondeando hacia arriba).
const taxiCrossTicks = (ScreenWidth + taxiWidth + taxiSpeed - 1) / taxiSpeed

// scenery cuenta los ticks del decorado. Todo lo demás se calcula desde esa
// cuenta, así no hay estado que se desincronice: el taxi y el humo dependen
// solo de cuántos ticks pasaron.
type scenery struct {
	ticks int
}

// update avanza el decorado un tick. Lo llama Update de la escena.
func (s *scenery) update() { s.ticks++ }

// taxi dice dónde está el taxi y si se ve. Cada vuelta dura lo que tarda en
// cruzar más la espera; durante la espera no se dibuja.
func (s scenery) taxi() (x, y float64, visible bool) {
	t := s.ticks % (taxiCrossTicks + taxiGapTicks)
	if t >= taxiCrossTicks {
		return 0, taxiY, false
	}
	return float64(-taxiWidth + t*taxiSpeed), taxiY, true
}

// steamFrame es el cuadro del humo que toca: 0, 1, 2, 0, 1…
func (s scenery) steamFrame() int {
	return s.ticks / steamFrameTicks % steamFrames
}

// draw dibuja el humo y el taxi encima del mapa.
func (s scenery) draw(dst *ebiten.Image) {
	f := s.steamFrame()
	frame := image.Rect(f*steamFrameWidth, 0, (f+1)*steamFrameWidth, steamImg.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(steamAt.X), float64(steamAt.Y))
	dst.DrawImage(steamImg.SubImage(frame).(*ebiten.Image), op)

	if x, y, visible := s.taxi(); visible {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, y)
		dst.DrawImage(taxiImg, op)
	}
}
