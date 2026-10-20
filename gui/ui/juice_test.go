package ui

import (
	"image"
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
