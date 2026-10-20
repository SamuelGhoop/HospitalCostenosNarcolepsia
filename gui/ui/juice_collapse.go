package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// BLOQUE 3 del pase de juice (§12.10, "dormirse"). Al arrancar la animación
// de desplomarse (animCollapse, patient_anim.go, justo cuando el paciente
// YA llegó a su sitio y empieza a caerse) salen tres cosas, todas puramente
// visuales: una nube de polvo en los pies, un "!" rojo grande sobre la
// cabeza que rebota al aparecer y dura collapseFlourishTicks, y la burbuja
// Zzz (game_input.go) más grande y con vaivén mientras tanto. Un shake
// leve acompaña el golpe contra el piso.

// Partículas de polvo (dustParticleCount, de 8 a 12: se sortea cada vez) y
// cuánto dura el "!"/la burbuja agrandada (collapseFlourishTicks, 1 s).
const (
	collapseFlourishTicks = 60 // 1 s a 60 ticks/s
	exclaimPopTicks       = 12 // 0,2 s: el rebote al aparecer
	exclaimFadeTicks      = 15 // 0,25 s: se desvanece al final
	exclaimBaseSize       = 32 // Press Start 2P, bien grande
	zzzEmphasisScale      = 1.6
	zzzBobAmplitude       = 4.0
	zzzBobPeriodTicks     = 40.0

	dustParticleMin  = 8
	dustParticleMax  = 12
	dustLifeMinTicks = 22
	dustLifeMaxTicks = 32
	dustSpreadSpeed  = 1.4 // px/tick al abrirse a los lados
	dustUpSpeed      = 1.6 // px/tick hacia arriba al empezar
	dustGravity      = 0.09
	dustSize         = 3 // px de cada partícula
)

// collapseFlourish es el aviso de un desplome: cuenta ticks hacia arriba
// desde 0 hasta collapseFlourishTicks, en la posición pos (sobre la cabeza,
// fija: el paciente ya no se mueve mientras se desploma).
type collapseFlourish struct {
	pos   vec
	ticks int
}

// step avanza un tick y dice si ya terminó (se puede olvidar).
func (c *collapseFlourish) step() (done bool) {
	c.ticks++
	return c.ticks >= collapseFlourishTicks
}

// exclaimScale es el tamaño del "!" ahora: un rebote (easeOutBack) los
// primeros exclaimPopTicks y 1 ("tamaño normal") después.
func (c *collapseFlourish) exclaimScale() float64 {
	if c.ticks >= exclaimPopTicks {
		return 1
	}
	return easeOutBack(float64(c.ticks) / float64(exclaimPopTicks))
}

// exclaimAlpha se desvanece en los últimos exclaimFadeTicks.
func (c *collapseFlourish) exclaimAlpha() float32 {
	remain := collapseFlourishTicks - c.ticks
	if remain < exclaimFadeTicks {
		return float32(remain) / float32(exclaimFadeTicks)
	}
	return 1
}

// zzzBob es el vaivén vertical (px) de la burbuja Zzz mientras dura el aviso.
func (c *collapseFlourish) zzzBob() float64 {
	return zzzBobAmplitude * math.Sin(2*math.Pi*float64(c.ticks)/zzzBobPeriodTicks)
}

// easeOutBack: 0 en t=0, se pasa de 1 (rebote) y se asienta en 1 en t=1.
// Fórmula estándar de easing con "overshoot" (easings.net, de Penner).
func easeOutBack(t float64) float64 {
	const c1 = 1.70158
	const c3 = c1 + 1
	t--
	return 1 + c3*t*t*t + c1*t*t
}

// dustParticle es una mota de polvo gris que se abre a los lados, sube un
// poco y cae (gravedad), desvaneciéndose con su vida.
type dustParticle struct {
	pos, vel    vec
	life, total int
}

func (p *dustParticle) step() (alive bool) {
	p.pos.x += p.vel.x
	p.pos.y += p.vel.y
	p.vel.y += dustGravity
	p.life--
	return p.life > 0
}

func (p *dustParticle) alpha() float32 { return float32(p.life) / float32(p.total) }

// drawDust dibuja una mota como un cuadrito gris sin suavizar (pixel art
// nítido: posición entera).
func drawDust(dst *ebiten.Image, p *dustParticle) {
	c := color.RGBA{colLightGray.R, colLightGray.G, colLightGray.B, uint8(255 * p.alpha())}
	x, y := int(math.Round(p.pos.x)), int(math.Round(p.pos.y))
	fillRect(dst, image.Rect(x, y, x+dustSize, y+dustSize), c)
}

// drawExclaim dibuja el "!" rojo grande de c, centrado en c.pos, con su
// rebote y su desvanecido.
func drawExclaim(dst *ebiten.Image, c *collapseFlourish) {
	size := exclaimBaseSize * c.exclaimScale()
	if size <= 0 {
		return
	}
	w := pixelTextWidth("!", size)
	a := c.exclaimAlpha()
	col := color.RGBA{colRed.R, colRed.G, colRed.B, uint8(255 * a)}
	drawPixelText(dst, "!", int(c.pos.x)-w/2, int(c.pos.y), size, []textLayer{{0, 0, col}}, a)
}

// collapse arranca el aviso de desplome de id, con la cabeza en head (la
// esquina superior del cuerpo acostado) y los pies en feet: dispara el
// shake leve, el "!" y la nube de polvo.
func (fx *juiceFX) collapse(id string, head, feet vec) {
	fx.shake.kick(shakeCollapse)
	fx.collapses[id] = &collapseFlourish{pos: vec{head.x, head.y - 10}}
	n := dustParticleMin + fx.rng.Intn(dustParticleMax-dustParticleMin+1)
	for i := 0; i < n; i++ {
		angle := fx.rng.Float64()*2 - 1 // -1..1
		life := dustLifeMinTicks + fx.rng.Intn(dustLifeMaxTicks-dustLifeMinTicks+1)
		fx.dust = append(fx.dust, &dustParticle{
			pos:   feet,
			vel:   vec{angle * dustSpreadSpeed, -fx.rng.Float64() * dustUpSpeed},
			life:  life,
			total: life,
		})
	}
}
