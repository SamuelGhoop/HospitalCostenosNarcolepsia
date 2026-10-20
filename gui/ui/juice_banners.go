package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// BLOQUE 6 del pase de juice (§12.10): "DÍA N" al empezar cada día y
// "¡HORA PICO!" al entrar a una. Los dos son el mismo mecanismo
// (sweepBanner): entran deslizando desde afuera a la izquierda, se quedan
// quietos en el centro un rato y siguen deslizando hasta salir por la
// derecha. Nunca se llevan la entrada ni pausan nada: GameScene los dibuja,
// nada más.

// sweepBanner es un texto grande que barre la pantalla de lado a lado en
// tres fases: entra (enterTicks), se sostiene en el centro (holdTicks) y
// sale (exitTicks).
type sweepBanner struct {
	text                             string
	col                              color.RGBA
	size                             float64
	enterTicks, holdTicks, exitTicks int
	ticks                            int
}

func newSweepBanner(text string, col color.RGBA, size float64, enter, hold, exit int) *sweepBanner {
	return &sweepBanner{text: text, col: col, size: size, enterTicks: enter, holdTicks: hold, exitTicks: exit}
}

// step avanza un tick y dice si todavía se ve.
func (b *sweepBanner) step() (alive bool) {
	b.ticks++
	return b.ticks < b.enterTicks+b.holdTicks+b.exitTicks
}

// centerX es la posición horizontal del CENTRO del texto ahora: de afuera a
// la izquierda (-halfWidth-margin) al centro de la pantalla mientras entra,
// quieto en el centro mientras se sostiene, y del centro a afuera a la
// derecha mientras sale. halfWidth es la mitad del ancho del texto (para
// que arranque y termine completamente fuera de la pantalla, no solo su
// centro).
func (b *sweepBanner) centerX(halfWidth float64) float64 {
	const margin = 40 // bien afuera, para que no se note el recorte
	off := halfWidth + margin
	mid := float64(ScreenWidth) / 2
	switch {
	case b.ticks < b.enterTicks:
		t := float64(b.ticks) / float64(b.enterTicks)
		return -off + (mid+off)*t
	case b.ticks < b.enterTicks+b.holdTicks:
		return mid
	default:
		t := float64(b.ticks-b.enterTicks-b.holdTicks) / float64(b.exitTicks)
		return mid + (mid+off)*t
	}
}

// drawSweepBanner dibuja b en banner.Y, centrado, con un borde negro
// (outlined, pixelfont.go) para que se lea bien sobre cualquier fondo.
func drawSweepBanner(dst *ebiten.Image, b *sweepBanner, y float64) {
	w := float64(pixelTextWidth(b.text, b.size))
	x := b.centerX(w/2) - w/2
	layers := append(outlined(2, colInk), textLayer{0, 0, b.col})
	drawPixelText(dst, b.text, int(x), int(y), b.size, layers, 1)
}
