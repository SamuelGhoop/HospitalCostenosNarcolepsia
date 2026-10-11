package ui

import (
	"image"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// Lo que se mueve en el mapa sin que tenga que ver con el hospital
// (GAME_DESIGN §10.7): el tráfico de la calle (traffic.go) y el humo de la
// olla. Sus PNG ya vienen escalados, así que se dibujan a 1×.
const (
	steamFrameWidth = 44 // steam_pot.png: 3 cuadros de 44×36 en fila
	steamFrames     = 3
	steamFrameTicks = 18 // ~300 ms por cuadro a 60 ticks por segundo
)

// steamAt es la esquina superior izquierda del humo, sobre la olla.
var steamAt = image.Pt(40, 246)

// scenery es el decorado que se mueve: el humo depende solo de cuántos
// ticks pasaron, y el tráfico lleva su propio estado.
type scenery struct {
	ticks   int
	traffic *traffic
}

// newScenery arma el decorado con el azar cosmético de la interfaz.
func newScenery(rng *rand.Rand) scenery { return scenery{traffic: newTraffic(rng)} }

// update avanza el decorado un tick. Lo llama Update de la escena.
func (s *scenery) update() {
	s.ticks++
	if s.traffic != nil {
		s.traffic.update()
	}
}

// steamFrame es el cuadro del humo que toca: 0, 1, 2, 0, 1…
func (s scenery) steamFrame() int {
	return s.ticks / steamFrameTicks % steamFrames
}

// draw dibuja el humo y los carros encima del mapa.
func (s scenery) draw(dst *ebiten.Image) {
	f := s.steamFrame()
	frame := image.Rect(f*steamFrameWidth, 0, (f+1)*steamFrameWidth, steamImg.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(steamAt.X), float64(steamAt.Y))
	dst.DrawImage(steamImg.SubImage(frame).(*ebiten.Image), op)

	if s.traffic != nil {
		s.traffic.draw(dst)
	}
}

// palmCrowns son las esquinas de las copas de las 2 palmeras: en la maqueta,
// put(copa, cx − 19, 171.5) con cx = 300 y 400 (la copa trae 1 px de borde a
// 1,5 px de mundo cada uno).
var palmCrowns = []image.Point{fromMockup(279.5, 170, 0, 0).Min, fromMockup(379.5, 170, 0, 0).Min}

// drawPalmCrowns dibuja las copas de las palmeras ENCIMA de los personajes
// (el fondo también las trae, en el mismo sitio): quien pasa cerca de una
// palmera queda debajo de las hojas.
func drawPalmCrowns(dst *ebiten.Image) {
	for _, at := range palmCrowns {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		dst.DrawImage(palmCrownImg, op)
	}
}
