package ui

import (
	"image"
	"math"
	"testing"
)

// BLOQUE 1: la hitbox crece +50 % alrededor de su centro (nunca se corre).
func TestGrowRect_GrowsAroundItsOwnCenter(t *testing.T) {
	r := image.Rect(10, 20, 30, 60) // 20×40, centro (20, 40)
	got := growRect(r, 1.5)
	wantCx, wantCy := 20.0, 40.0
	gotCx, gotCy := boxCenter(got)
	if gotCx != wantCx || gotCy != wantCy {
		t.Errorf("centro = (%v, %v); se esperaba (%v, %v)", gotCx, gotCy, wantCx, wantCy)
	}
	if got.Dx() != 30 || got.Dy() != 60 { // 20×1.5, 40×1.5
		t.Errorf("tamaño = %dx%d; se esperaba 30x60", got.Dx(), got.Dy())
	}
}

func TestGrowRect_FactorOneIsTheSameBox(t *testing.T) {
	r := image.Rect(5, 5, 25, 15)
	if got := growRect(r, 1); got != r {
		t.Errorf("growRect(r, 1) = %v; se esperaba %v sin cambios", got, r)
	}
}

// ─── BLOQUE 2: shake, hit-stop, squash&stretch y la línea de despacho ──

// El shake decae a 0 en shakeDecayTicks y nunca sube de golpe: cada tick que
// pasa, la magnitud es igual o menor que la de antes.
func TestScreenShake_DecaysToZero(t *testing.T) {
	s := newScreenShake(1)
	s.kick(shakeInvalidClick)
	prev := s.current()
	if prev != shakeInvalidClick {
		t.Fatalf("current() justo después de kick = %v; se esperaba %v", prev, shakeInvalidClick)
	}
	for i := 0; i < shakeDecayTicks; i++ {
		s.step()
		cur := s.current()
		if cur > prev {
			t.Fatalf("tick %d: la magnitud subió de %v a %v", i, prev, cur)
		}
		prev = cur
	}
	if got := s.current(); got != 0 {
		t.Errorf("después de %d ticks, current() = %v; se esperaba 0", shakeDecayTicks, got)
	}
}

// Un segundo kick más débil no debilita un temblor que ya estaba más
// fuerte, pero sí le renueva el tiempo que le queda.
func TestScreenShake_WeakerKickDoesNotWeakenAStrongerShake(t *testing.T) {
	s := newScreenShake(1)
	s.kick(shakeAngry) // el más fuerte de los tres
	for i := 0; i < shakeDecayTicks/2; i++ {
		s.step()
	}
	before := s.current()
	s.kick(shakeCollapse) // más débil: no debería bajarlo
	if got := s.current(); got < before {
		t.Errorf("current() tras un kick más débil = %v; se esperaba al menos %v", got, before)
	}
}

// hitStop cuenta hacia atrás y un trigger más chico no lo acorta.
func TestHitStop_CountsDownAndDoesNotShrink(t *testing.T) {
	var h hitStop
	h.trigger(hitStopTicks)
	if !h.active() {
		t.Fatal("recién disparado, active() debería ser true")
	}
	h.trigger(1) // más chico: no debe acortar lo que ya estaba corriendo
	for i := 0; i < hitStopTicks; i++ {
		if !h.active() {
			t.Fatalf("tick %d: active() ya era false; se esperaban %d ticks", i, hitStopTicks)
		}
		h.step()
	}
	if h.active() {
		t.Error("después de hitStopTicks pasos, active() sigue true")
	}
}

// figurePunch: la escala vale 1,1 en reposo y en los dos extremos de su
// pulso (apenas arranca y justo antes de terminar), y se aplasta (sx>1,
// sy<1) a mitad de camino.
func TestFigurePunch_ScaleStartsAndEndsAtRest(t *testing.T) {
	p := newFigurePunch(colWhite)
	sx, sy := p.scale()
	if sx != 1 || sy != 1 {
		t.Errorf("escala en el primer tick = (%v, %v); se esperaba (1, 1): el pulso vale 0 al empezar", sx, sy)
	}
	for i := 0; i < squashTicks/2; i++ {
		p.step()
	}
	sx, sy = p.scale()
	if sx <= 1 || sy >= 1 {
		t.Errorf("escala a mitad del pulso = (%v, %v); se esperaba aplastado (sx>1, sy<1)", sx, sy)
	}
	for p.ticksLeft > 0 {
		p.step()
	}
	sx, sy = p.scale()
	if sx != 1 || sy != 1 {
		t.Errorf("escala terminado el squash = (%v, %v); se esperaba (1, 1)", sx, sy)
	}
}

// El destello dura justo squashFlashTicks cuadros.
func TestFigurePunch_FlashesForExactlyItsTicks(t *testing.T) {
	p := newFigurePunch(colWhite)
	for i := 0; i < squashFlashTicks; i++ {
		if !p.flashing() {
			t.Fatalf("tick %d: flashing() ya era false; se esperaban %d cuadros", i, squashFlashTicks)
		}
		p.step()
	}
	if p.flashing() {
		t.Error("después de squashFlashTicks cuadros, flashing() sigue true")
	}
}

// step() dice "terminado" justo cuando ya no queda ni squash ni destello.
func TestFigurePunch_StepReportsDoneOnlyAtTheVeryEnd(t *testing.T) {
	p := newFigurePunch(colWhite)
	done := false
	ticks := 0
	for !done && ticks < 100 {
		done = p.step()
		ticks++
	}
	if ticks != squashTicks {
		t.Errorf("terminó a los %d ticks; se esperaban %d (squashTicks, que dura más que el destello)", ticks, squashTicks)
	}
}

// ─── BLOQUE 3: dormirse (desplome) ──────────────────────────────────────

// easeOutBack vale 0 al empezar y 1 al terminar, pasándose de 1 en algún
// punto intermedio (el "rebote").
func TestEaseOutBack_StartsAtZeroEndsAtOneAndOvershoots(t *testing.T) {
	const tolerance = 1e-9
	if got := easeOutBack(0); math.Abs(got) > tolerance {
		t.Errorf("easeOutBack(0) = %v; se esperaba ~0", got)
	}
	if got := easeOutBack(1); math.Abs(got-1) > tolerance {
		t.Errorf("easeOutBack(1) = %v; se esperaba ~1", got)
	}
	over := false
	for i := 0; i <= 20; i++ {
		t0 := float64(i) / 20
		if easeOutBack(t0) > 1.001 {
			over = true
		}
	}
	if !over {
		t.Error("easeOutBack nunca pasó de 1; se esperaba que rebotara (overshoot)")
	}
}

// El aviso de desplome dura collapseFlourishTicks y, a mitad de camino, el
// "!" ya está en su tamaño normal (el rebote solo dura exclaimPopTicks).
func TestCollapseFlourish_LastsItsTicksAndSettlesAfterThePop(t *testing.T) {
	c := &collapseFlourish{}
	for i := 0; i < exclaimPopTicks; i++ {
		c.step()
	}
	if got := c.exclaimScale(); got != 1 {
		t.Errorf("exclaimScale tras exclaimPopTicks = %v; se esperaba 1 (el rebote ya terminó)", got)
	}
	done := false
	ticks := exclaimPopTicks
	for !done {
		done = c.step()
		ticks++
	}
	if ticks != collapseFlourishTicks {
		t.Errorf("terminó a los %d ticks; se esperaban %d", ticks, collapseFlourishTicks)
	}
}

// El polvo cae (gravedad) y se desvanece con su vida, sin volverse a subir
// de opacidad.
func TestDustParticle_FallsAndFadesOut(t *testing.T) {
	p := &particle{vel: vec{0, -1}, life: 10, total: 10, gravity: dustGravity}
	prevVY := p.vel.y
	prevAlpha := p.alpha()
	for p.step() {
		if p.vel.y <= prevVY {
			t.Fatalf("la velocidad vertical no subió con la gravedad: %v → %v", prevVY, p.vel.y)
		}
		if p.alpha() > prevAlpha {
			t.Fatalf("la opacidad subió: %v → %v", prevAlpha, p.alpha())
		}
		prevVY, prevAlpha = p.vel.y, p.alpha()
	}
	if p.life != 0 {
		t.Errorf("life terminó en %d; se esperaba 0", p.life)
	}
}

// ─── BLOQUE 4: cobro, racha, alta y enojo ───────────────────────────────

// floatText sube (riseTotal positivo) y se desvanece hasta el final.
func TestFloatText_RisesAndFadesToZero(t *testing.T) {
	f := &floatText{pos: vec{0, 100}, total: 10, riseTotal: 20}
	prevY, prevAlpha := f.y(), f.alpha()
	if prevY != 100 || prevAlpha != 1 {
		t.Fatalf("en el primer tick: y=%v alpha=%v; se esperaba y=100 alpha=1", prevY, prevAlpha)
	}
	for f.step() {
		if f.y() >= prevY {
			t.Fatalf("no subió: y pasó de %v a %v", prevY, f.y())
		}
		if f.alpha() > prevAlpha {
			t.Fatalf("la opacidad subió: %v → %v", prevAlpha, f.alpha())
		}
		prevY, prevAlpha = f.y(), f.alpha()
	}
	if f.ticks != f.total {
		t.Errorf("terminó a los %d ticks; se esperaban %d", f.ticks, f.total)
	}
}

// floatText con riseTotal negativo CAE en vez de subir ("RACHA PERDIDA").
func TestFloatText_NegativeRiseFallsInstead(t *testing.T) {
	f := &floatText{pos: vec{0, 100}, total: 10, riseTotal: -20}
	f.step()
	if f.y() <= 100 {
		t.Errorf("y() = %v; con riseTotal negativo se esperaba que bajara de 100", f.y())
	}
}

// El banner de racha dura streakBannerTicks y, pasado el rebote
// (streakBannerPopTicks), ya está en su tamaño normal.
func TestStreakBanner_LastsItsTicksAndSettlesAfterThePop(t *testing.T) {
	b := &streakBanner{}
	for i := 0; i < streakBannerPopTicks; i++ {
		b.step()
	}
	if got := b.scale(); got != 1 {
		t.Errorf("scale() tras el rebote = %v; se esperaba 1", got)
	}
	remaining, alive := 0, true
	for alive {
		alive = b.step()
		remaining++
	}
	if total := streakBannerPopTicks + remaining; total != streakBannerTicks {
		t.Errorf("duró %d ticks en total; se esperaban %d", total, streakBannerTicks)
	}
}

// Más racha, color más cálido (amarillo → naranja → rojo).
func TestStreakBannerColor_WarmerAtHigherStreak(t *testing.T) {
	if c := streakBannerColor(100); c != colYellow {
		t.Errorf("×1 = %v; se esperaba amarillo", c)
	}
	if c := streakBannerColor(150); c != colOrange {
		t.Errorf("×1,5 = %v; se esperaba naranja", c)
	}
	if c := streakBannerColor(200); c != colRed {
		t.Errorf("×2 = %v; se esperaba rojo", c)
	}
}

// La estrella que cae gira, cae (gravedad) y se desvanece hasta el final.
func TestFallingStar_SpinsFallsAndFades(t *testing.T) {
	s := &fallingStar{vel: vec{0, 0}}
	prevAngle, prevY, prevAlpha := s.angle, s.pos.y, s.alpha()
	for s.step() {
		if s.angle <= prevAngle {
			t.Fatalf("el ángulo no avanzó: %v → %v", prevAngle, s.angle)
		}
		if s.pos.y < prevY {
			t.Fatalf("y bajó en vez de caer: %v → %v", prevY, s.pos.y)
		}
		if s.alpha() > prevAlpha {
			t.Fatalf("la opacidad subió: %v → %v", prevAlpha, s.alpha())
		}
		prevAngle, prevY, prevAlpha = s.angle, s.pos.y, s.alpha()
	}
	if s.ticks != fallingStarTicks {
		t.Errorf("terminó a los %d ticks; se esperaban %d", s.ticks, fallingStarTicks)
	}
}

// La línea de despacho se desvanece en dispatchLineTicks y nunca sube de
// opacidad.
func TestDashLine_FadesToZero(t *testing.T) {
	d := &dashLine{from: vec{0, 0}, to: vec{100, 0}}
	prev := d.alpha()
	if prev != 1 {
		t.Fatalf("alpha() recién creada = %v; se esperaba 1", prev)
	}
	for d.step() {
		cur := d.alpha()
		if cur > prev {
			t.Fatalf("la opacidad subió de %v a %v", prev, cur)
		}
		prev = cur
	}
	if d.ticks != dispatchLineTicks {
		t.Errorf("se desvaneció a los %d ticks; se esperaban %d", d.ticks, dispatchLineTicks)
	}
}
