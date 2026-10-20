package ui

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// juiceFX es lo que GameScene recuerda del pase de "juice" (§12.10) entre
// cuadros: nada de esto es una regla del juego, solo movimiento y sonido
// de la interfaz. step() avanza todo un tick; cada bloque (clics, despacho,
// desplome, cobro...) agrega su propio disparador en su propio commit.
type juiceFX struct {
	shake   screenShake
	hitStop hitStop

	punches map[string]*figurePunch // squash&stretch + destello por personaje
	lines   []*dashLine             // líneas de despacho exitoso
}

func newJuiceFX(seed int64) juiceFX {
	return juiceFX{shake: newScreenShake(seed), punches: map[string]*figurePunch{}}
}

// step avanza un tick todas las piezas del pase de juice y descarta las que
// ya terminaron.
func (fx *juiceFX) step() {
	fx.shake.step()
	fx.hitStop.step()
	for id, p := range fx.punches {
		if p.step() {
			delete(fx.punches, id)
		}
	}
	kept := fx.lines[:0]
	for _, l := range fx.lines {
		if l.step() {
			kept = append(kept, l)
		}
	}
	fx.lines = kept
}

// punch arranca (o reinicia) el squash&stretch + destello de id, con el
// color de destello c (blanco: clic que sirvió; rojo: inválido).
func (fx *juiceFX) punch(id string, c color.RGBA) {
	fx.punches[id] = newFigurePunch(c)
}

// dispatchSuccess deja la línea punteada de un despacho exitoso, de from
// a to (los centros de las cajas del personal y el paciente en el
// momento del clic).
func (fx *juiceFX) dispatchSuccess(from, to vec) {
	fx.lines = append(fx.lines, &dashLine{from: from, to: to})
}

// invalidClick es "clic inválido" (BLOQUE 2, §12.10): shake de pantalla
// (shakeInvalidClick) y, aparte, quien llama ya pintó el destello rojo con
// fx.punch.
func (fx *juiceFX) invalidClick() {
	fx.shake.kick(shakeInvalidClick)
}

// ─── screenShake ────────────────────────────────────────────────────────

// screenShake es el temblor de pantalla: arranca en una magnitud (px) y
// decae en línea recta a 0 en shakeDecayTicks. Cada tick que sigue
// temblando sortea un offset nuevo (para que se vea como temblor, no como
// un salto fijo), con un azar propio de la interfaz (rng), igual que el
// tráfico decorado o el personal deambulando.
type screenShake struct {
	mag0      float64
	ticksLeft int
	rng       *rand.Rand
}

func newScreenShake(seed int64) screenShake {
	return screenShake{rng: rand.New(rand.NewSource(seed))}
}

// current es la magnitud de ahora (decae en línea recta con los ticks que
// quedan).
func (s *screenShake) current() float64 {
	if s.ticksLeft <= 0 {
		return 0
	}
	return s.mag0 * float64(s.ticksLeft) / float64(shakeDecayTicks)
}

// kick agrega un temblor de esta magnitud: si ya estaba temblando más
// fuerte, no lo debilita, pero sí le renueva el tiempo que le queda
// (varios golpes seguidos se sienten como uno solo que no para de temblar).
func (s *screenShake) kick(mag float64) {
	if mag > s.current() {
		s.mag0 = mag
	}
	s.ticksLeft = shakeDecayTicks
}

func (s *screenShake) step() {
	if s.ticksLeft > 0 {
		s.ticksLeft--
	}
}

// offset es el desplazamiento entero al azar de este tick (0,0 sin temblor).
func (s *screenShake) offset() (dx, dy int) {
	m := s.current()
	if m <= 0 {
		return 0, 0
	}
	return int(math.Round((s.rng.Float64()*2 - 1) * m)), int(math.Round((s.rng.Float64()*2 - 1) * m))
}

// ─── hitStop ─────────────────────────────────────────────────────────────

// hitStop congela las animaciones de la interfaz (nunca Game.Tick, que
// corre en su propia goroutine): mientras ticksLeft > 0, GameScene.update
// no llama a s.animate() (BLOQUE 4, §12.10).
type hitStop struct{ ticksLeft int }

// trigger pide ticks de hit-stop: si ya había más por correr, no lo acorta.
func (h *hitStop) trigger(ticks int) {
	if ticks > h.ticksLeft {
		h.ticksLeft = ticks
	}
}

func (h *hitStop) active() bool { return h.ticksLeft > 0 }

func (h *hitStop) step() {
	if h.ticksLeft > 0 {
		h.ticksLeft--
	}
}

// ─── figurePunch: squash & stretch + destello ──────────────────────────

// figurePunch es el "golpe" visual de un personaje recién clicado: se
// aplasta y rebota en squashTicks (~120 ms) y, los primeros
// squashFlashTicks cuadros, su silueta destella (blanco o rojo, según
// flashColor) con blend aditivo (drawSquashed, layers_juice.go).
type figurePunch struct {
	ticksLeft  int
	flashLeft  int
	flashColor color.RGBA
}

func newFigurePunch(c color.RGBA) *figurePunch {
	return &figurePunch{ticksLeft: squashTicks, flashLeft: squashFlashTicks, flashColor: c}
}

// step avanza un tick y dice si ya terminó del todo (ni squash ni destello:
// se puede olvidar).
func (p *figurePunch) step() (done bool) {
	if p.ticksLeft > 0 {
		p.ticksLeft--
	}
	if p.flashLeft > 0 {
		p.flashLeft--
	}
	return p.ticksLeft <= 0 && p.flashLeft <= 0
}

// scale es (sx, sy) ahora: 1,1 en reposo. Un solo pulso de aplastar
// (sx>1, sy<1) y estirar de vuelta (sx<1, sy>1), que vale 0 al empezar y al
// terminar (seno), así no hay salto ni al arrancar ni al volver a 1.
func (p *figurePunch) scale() (sx, sy float64) {
	if p.ticksLeft <= 0 {
		return 1, 1
	}
	t := 1 - float64(p.ticksLeft)/float64(squashTicks)
	amt := math.Sin(t*math.Pi) * squashAmount
	return 1 + amt, 1 - amt
}

// flashing dice si todavía toca dibujar el destello aditivo.
func (p *figurePunch) flashing() bool { return p.flashLeft > 0 }

// ─── dashLine: despacho exitoso ────────────────────────────────────────

// dashLine es la línea punteada animada del personal al paciente que se
// desvanece en dispatchLineTicks (0,6 s, BLOQUE 2).
type dashLine struct {
	from, to vec
	ticks    int
}

// step avanza un tick y dice si todavía se ve.
func (d *dashLine) step() (stillVisible bool) {
	d.ticks++
	return d.ticks < dispatchLineTicks
}

// alpha baja en línea recta de 1 a 0.
func (d *dashLine) alpha() float32 {
	return float32(1 - float64(d.ticks)/float64(dispatchLineTicks))
}

// phase son los píxeles que ya "avanzaron" los guiones (dashSpeed por tick).
func (d *dashLine) phase() float64 { return float64(d.ticks) * dashSpeed }

// drawDashLine dibuja los guiones de d entre from y to, avanzando con la
// fase y desvaneciéndose con su alpha.
func drawDashLine(dst *ebiten.Image, d *dashLine) {
	dx, dy := d.to.x-d.from.x, d.to.y-d.from.y
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	ux, uy := dx/length, dy/length
	period := dashLen + dashGap
	phase := math.Mod(d.phase(), period)
	a := d.alpha()
	if a <= 0 {
		return
	}
	c := color.RGBA{colYellow.R, colYellow.G, colYellow.B, uint8(255 * a)}
	for pos := -phase; pos < length; pos += period {
		segStart, segEnd := math.Max(pos, 0), math.Min(pos+dashLen, length)
		if segEnd <= segStart {
			continue
		}
		x0, y0 := d.from.x+ux*segStart, d.from.y+uy*segStart
		x1, y1 := d.from.x+ux*segEnd, d.from.y+uy*segEnd
		vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), dashWidth, c, false)
	}
}

// ─── worldCanvas: el mundo aparte, para poder temblarlo con el shake ───

// worldCanvasImg es donde GameScene.drawWorld pinta el mapa, los
// personajes y sus rótulos; Draw lo pega en la pantalla con el offset
// entero del shake, sin tocar el HUD (game_scene.go).
var worldCanvasImg *ebiten.Image

func worldCanvas() *ebiten.Image {
	if worldCanvasImg == nil {
		worldCanvasImg = ebiten.NewImage(ScreenWidth, ScreenHeight)
	}
	return worldCanvasImg
}

// ─── drawSquashed: aplica el squash&stretch y el destello al dibujar ───

// figurePunchCanvasImg es el lienzo aparte donde se dibuja UN personaje con
// su squash&stretch (reusado entre personajes: nunca hay dos a la vez en
// mitad de dibujarse), igual que panelContentCanvas con los paneles.
var figurePunchCanvasImg *ebiten.Image

func figurePunchCanvas() *ebiten.Image {
	if figurePunchCanvasImg == nil {
		figurePunchCanvasImg = ebiten.NewImage(ScreenWidth, ScreenHeight)
	}
	return figurePunchCanvasImg
}

// drawSquashed dibuja lo que pinta draw (un personaje completo, en sus
// coordenadas de pantalla normales) con el squash&stretch de p, con el
// pivote en pivot (sus pies: así se aplasta contra el suelo, no flota). Si
// p es nil, dibuja normal, sin pasar por el lienzo aparte. El destello se
// pinta aparte, con blend aditivo (ColorScale/CompositeModeLighter, como
// pide GAME_DESIGN §12.10): la misma silueta, teñida de blanco o rojo.
func drawSquashed(dst *ebiten.Image, p *figurePunch, pivot vec, draw func(*ebiten.Image)) {
	if p == nil {
		draw(dst)
		return
	}
	canvas := figurePunchCanvas()
	canvas.Clear()
	draw(canvas)

	sx, sy := p.scale()
	var geo ebiten.GeoM
	geo.Translate(-pivot.x, -pivot.y)
	geo.Scale(sx, sy)
	geo.Translate(pivot.x, pivot.y)

	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, GeoM: geo}
	dst.DrawImage(canvas, op)

	if p.flashing() {
		flashOp := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, GeoM: geo, CompositeMode: ebiten.CompositeModeLighter}
		flashOp.ColorScale.ScaleWithColor(p.flashColor)
		dst.DrawImage(canvas, flashOp)
	}
}
