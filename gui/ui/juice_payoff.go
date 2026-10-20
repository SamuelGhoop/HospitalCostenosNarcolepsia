package ui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// BLOQUE 4 del pase de juice (§12.10): "cobro/racha/alta/enojo". Nada de
// esto es una regla del juego: los montos, la racha y los motivos ya los
// calculó game/ (dispatch.go, reputation.go, patients.go); detectJuiceEvents
// (juice_events.go) los detecta comparando dos Snapshot y esto solo los
// anima.

// ─── floatText: un texto que sube (o cae) y se desvanece ───────────────

// floatText es un texto que arranca en pos y se mueve hacia pos.y-riseTotal
// (si riseTotal es negativo, CAE en vez de subir) mientras se desvanece, en
// total ticks. Lo usan el "+$350" del cobro y "RACHA PERDIDA".
type floatText struct {
	text         string
	pos          vec
	col          color.RGBA
	size         float64
	ticks, total int
	riseTotal    float64
}

func (f *floatText) step() (alive bool) {
	f.ticks++
	return f.ticks < f.total
}

// y es la posición vertical ahora (en línea recta de pos.y a pos.y-riseTotal).
func (f *floatText) y() float64 {
	t := float64(f.ticks) / float64(f.total)
	return f.pos.y - f.riseTotal*t
}

func (f *floatText) alpha() float32 {
	return float32(1 - float64(f.ticks)/float64(f.total))
}

func drawFloatText(dst *ebiten.Image, f *floatText) {
	a := f.alpha()
	w := pixelTextWidth(f.text, f.size)
	c := color.RGBA{f.col.R, f.col.G, f.col.B, uint8(255 * a)}
	drawPixelText(dst, f.text, int(f.pos.x)-w/2, int(f.y()), f.size, []textLayer{{0, 0, c}}, a)
}

// ─── streakBanner: "¡RACHA ×N!" con rebote elástico ────────────────────

// streakBanner es el banner grande que salta cuando sube la racha: aparece
// con un rebote elástico (easeOutBack, como el "!" del desplome) y se
// desvanece al final.
type streakBanner struct {
	text  string
	col   color.RGBA
	pos   vec
	ticks int
}

func (b *streakBanner) step() (alive bool) {
	b.ticks++
	return b.ticks < streakBannerTicks
}

func (b *streakBanner) scale() float64 {
	if b.ticks >= streakBannerPopTicks {
		return 1
	}
	return easeOutBack(float64(b.ticks) / float64(streakBannerPopTicks))
}

func (b *streakBanner) alpha() float32 {
	remain := streakBannerTicks - b.ticks
	if remain < streakBannerFadeTicks {
		return float32(remain) / float32(streakBannerFadeTicks)
	}
	return 1
}

func drawStreakBanner(dst *ebiten.Image, b *streakBanner) {
	size := streakBannerBaseSize * b.scale()
	if size <= 0 {
		return
	}
	a := b.alpha()
	w := pixelTextWidth(b.text, size)
	c := color.RGBA{b.col.R, b.col.G, b.col.B, uint8(255 * a)}
	drawPixelText(dst, b.text, int(b.pos.x)-w/2, int(b.pos.y), size, []textLayer{{0, 0, c}}, a)
}

// streakBannerColor es más cálido mientras más alta la racha (en el
// multiplicador, centésimas: 100 = ×1, 200 = ×2, como config.go streakStep
// de a 25).
func streakBannerColor(multiplierHundredths int) color.RGBA {
	switch {
	case multiplierHundredths >= 175:
		return colRed
	case multiplierHundredths >= 150:
		return colOrange
	default:
		return colYellow
	}
}

// ─── fallingStar: la estrella que se desprende del HUD (enojo) ────────

// fallingStar es la estrella de reputación que, al enojarse alguien, se
// desprende del contador del HUD y cae girando y desvaneciéndose.
type fallingStar struct {
	pos, vel vec
	angle    float64
	ticks    int
}

func (s *fallingStar) step() (alive bool) {
	s.pos.x += s.vel.x
	s.pos.y += s.vel.y
	s.vel.y += fallingStarGravity
	s.angle += fallingStarSpin
	s.ticks++
	return s.ticks < fallingStarTicks
}

func (s *fallingStar) alpha() float32 {
	return float32(1 - float64(s.ticks)/float64(fallingStarTicks))
}

// drawFallingStar la dibuja con el ícono chiquito de estrella del HUD
// (hudImgs.starSmall, hud.go), rotada y teñida de rojo mientras cae.
func drawFallingStar(dst *ebiten.Image, s *fallingStar) {
	hudImages()
	img := hudImgs.starSmall
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	var g ebiten.GeoM
	g.Translate(-float64(w)/2, -float64(h)/2)
	g.Rotate(s.angle)
	g.Translate(s.pos.x, s.pos.y)
	a := s.alpha()
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, GeoM: g}
	op.ColorScale.ScaleWithColor(color.RGBA{colRed.R, colRed.G, colRed.B, uint8(255 * a)})
	dst.DrawImage(img, op)
}

// ─── Disparadores en juiceFX ────────────────────────────────────────────

// payment es el momento del cobro (§5.5, §5.8): hit-stop, flash blanco de
// pantalla y el "+$350" verde que sube sobre el paciente.
func (fx *juiceFX) payment(overHead vec, amount int) {
	fx.hitStop.trigger(hitStopTicks)
	fx.screenFlash.trigger(color.RGBA{0xff, 0xff, 0xff, paymentFlashAlpha}, paymentFlashTicks)
	fx.payments = append(fx.payments, &floatText{
		text: formatDelta(amount), pos: overHead, col: colGreenDark, size: hudTextSize,
		total: paymentPopupTicks, riseTotal: paymentPopupRise,
	})
}

// streakUp es la racha subiendo: el banner grande y sus chispas.
func (fx *juiceFX) streakUp(multiplierHundredths int) {
	txt := "¡RACHA " + formatStreakMultiplier(multiplierHundredths) + "!"
	fx.streak = &streakBanner{text: txt, col: streakBannerColor(multiplierHundredths), pos: streakBannerPos}
	for i := 0; i < streakSparkCount; i++ {
		angle := fx.rng.Float64() * 2 * math.Pi
		speed := streakSparkSpeed * (0.6 + 0.4*fx.rng.Float64())
		life := streakSparkLifeTicks
		fx.sparks = append(fx.sparks, &particle{
			pos: streakBannerPos, vel: vec{math.Cos(angle) * speed, math.Sin(angle) * speed},
			life: life, total: life, col: colYellow, size: 3, gravity: 0.05,
		})
	}
}

// streakLostBanner es la racha rompiéndose: "RACHA PERDIDA" cae y se
// desvanece.
func (fx *juiceFX) streakLostBanner() {
	fx.streakLostText = &floatText{
		text: "RACHA PERDIDA", pos: streakBannerPos, col: colRed, size: hudTextSize,
		total: streakLostBannerTicks, riseTotal: -streakLostFall,
	}
}

// dischargeBurst es el alta: corazones y confeti multicolor saltando sobre
// el paciente que sale (confetti: colores variados; corazones: cuadritos
// rosados — simplificación: nada de sprites nuevos, se distinguen del
// confeti solo por el color).
func (fx *juiceFX) dischargeBurst(overHead vec) {
	palette := []color.RGBA{colRed, colGreen, colBlue, colYellow, colOrange, colTeal}
	heartColor := color.RGBA{0xff, 0x6b, 0x9d, 0xff} // rosado
	for i := 0; i < dischargeParticleCount; i++ {
		col := heartColor
		if i%2 == 1 {
			col = palette[fx.rng.Intn(len(palette))]
		}
		angle := fx.rng.Float64()*2 - 1
		life := dischargeLifeMinTicks + fx.rng.Intn(dischargeLifeMaxTicks-dischargeLifeMinTicks+1)
		fx.discharge = append(fx.discharge, &particle{
			pos: overHead, vel: vec{angle * dischargeSpread, -fx.rng.Float64() * dischargeUpSpeed},
			life: life, total: life, col: col, size: dischargeSize, gravity: dischargeGravity,
		})
	}
}

// angryBurst es alguien yéndose enojado: humo rojo sobre la cabeza, una
// estrella que cae del HUD (starOrigin) y un shake fuerte.
func (fx *juiceFX) angryBurst(overHead, starOrigin vec) {
	fx.shake.kick(shakeAngry)
	for i := 0; i < angrySmokeCount; i++ {
		fx.angrySmoke = append(fx.angrySmoke, &particle{
			pos: overHead, vel: vec{(fx.rng.Float64()*2 - 1) * angrySmokeDrift, -angrySmokeRise},
			life: angrySmokeLifeTicks, total: angrySmokeLifeTicks, col: colRedDark, size: angrySmokeSize, gravity: 0,
		})
	}
	fx.fallingStars = append(fx.fallingStars, &fallingStar{
		pos: starOrigin, vel: vec{(fx.rng.Float64()*2 - 1) * 0.5, 0.6},
	})
}
