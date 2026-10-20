package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// particle es un cuadrito de color que se mueve con su propia velocidad y
// gravedad y se desvanece con su vida: la pieza que comparten el polvo del
// desplome (bloque 3), las chispas de la racha, el confeti y los corazones
// del alta, y el humo del enojo (bloque 4, §12.10). Cada efecto arma las
// suyas con la velocidad, la gravedad, el color y el tamaño que le toquen;
// particle no sabe ni le importa de cuál se trata.
type particle struct {
	pos, vel    vec
	life, total int
	col         color.RGBA
	size        int
	gravity     float64 // px/tick² hacia abajo; negativo: flota (el humo)
}

// step mueve la partícula un tick (gravedad incluida) y cuenta su vida.
// Dice si todavía sigue viva.
func (p *particle) step() (alive bool) {
	p.pos.x += p.vel.x
	p.pos.y += p.vel.y
	p.vel.y += p.gravity
	p.life--
	return p.life > 0
}

// alpha baja en línea recta de 1 a 0 con la vida que le queda.
func (p *particle) alpha() float32 {
	if p.total == 0 {
		return 0
	}
	return float32(p.life) / float32(p.total)
}

// drawParticle la dibuja como un cuadrito sin suavizar, en su posición
// entera (pixel art nítido): nunca a medio píxel, aunque pos sea flotante.
func drawParticle(dst *ebiten.Image, p *particle) {
	c := color.RGBA{p.col.R, p.col.G, p.col.B, uint8(255 * p.alpha())}
	x, y := int(math.Round(p.pos.x)), int(math.Round(p.pos.y))
	fillRect(dst, image.Rect(x, y, x+p.size, y+p.size), c)
}
