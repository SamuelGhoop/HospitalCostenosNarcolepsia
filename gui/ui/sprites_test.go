package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// La hoja del paciente viene embebida en el binario y mide 4 columnas ×
// 10 filas de cuadros de 16×24.
func TestPatientSheet_IsEmbeddedWithTenRowsOfFourFrames(t *testing.T) {
	img, err := decodePNG(assets.Sprites, patientSheetPath)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", patientSheetPath, err)
	}

	if got, want := img.Bounds().Size(), image.Pt(4*cellWidth, 10*cellHeight); got != want {
		t.Errorf("la hoja mide %v; se esperaba %v", got, want)
	}
}

func TestFrameRect_CutsTheRightCellOfTheSheet(t *testing.T) {
	tests := []struct {
		anim  animation
		frame int
		want  image.Rectangle
	}{
		{animIdle, 0, image.Rect(0, 0, 16, 24)},
		{animIdle, 1, image.Rect(16, 0, 32, 24)},
		{animWalkLeft, 3, image.Rect(48, 72, 64, 96)},
		{animSleepBed, 1, image.Rect(16, 168, 32, 192)},
		{animWave, 0, image.Rect(0, 216, 16, 240)},
	}
	for _, tt := range tests {
		if got := frameRect(tt.anim, tt.frame); got != tt.want {
			t.Errorf("frameRect(fila %d, cuadro %d) = %v; se esperaba %v", tt.anim, tt.frame, got, tt.want)
		}
	}
}

func TestFrameAt_LoopsOrStopsOnTheLastFrame(t *testing.T) {
	// Quieto: 2 cuadros de 500 ms (30 ticks) que se repiten.
	if got := frameAt(animIdle, 29); got != 0 {
		t.Errorf("quieto a los 29 ticks: cuadro %d; se esperaba 0", got)
	}
	if got := frameAt(animIdle, 30); got != 1 {
		t.Errorf("quieto a los 30 ticks: cuadro %d; se esperaba 1", got)
	}
	if got := frameAt(animIdle, 60); got != 0 {
		t.Errorf("quieto a los 60 ticks: cuadro %d; se esperaba 0 (vuelve a empezar)", got)
	}

	// Desplomarse: 4 cuadros de 120 ms (7 ticks) que NO se repiten.
	if got := frameAt(animCollapse, 1000); got != 3 {
		t.Errorf("desplomarse al final: cuadro %d; se esperaba quedarse en el 3", got)
	}
	if finished(animCollapse, 27) || !finished(animCollapse, 28) {
		t.Error("desplomarse debe terminar exactamente a los 4×7 = 28 ticks")
	}
	if finished(animIdle, 100000) {
		t.Error("una animación que se repite nunca termina")
	}
}

func TestWalkAnimation_PicksTheDirectionAndFlipsForTheRight(t *testing.T) {
	tests := []struct {
		dx, dy   float64
		want     animation
		wantFlip bool
	}{
		{-3, 1, animWalkLeft, false},
		{3, -1, animWalkLeft, true}, // derecha = izquierda volteada
		{1, -3, animWalkUp, false},
		{0, 3, animWalkDown, false},
	}
	for _, tt := range tests {
		got, flip := walkAnimation(tt.dx, tt.dy)
		if got != tt.want || flip != tt.wantFlip {
			t.Errorf("walkAnimation(%v, %v) = (%d, %v); se esperaba (%d, %v)", tt.dx, tt.dy, got, flip, tt.want, tt.wantFlip)
		}
	}
}

// Pedido por Samuel: la rotación de 90° se aplica SOLO en la cama.
func TestSpritePlacement_RotatesOnlyInsideTheBed(t *testing.T) {
	bed := bedRect(zoneRoom101)
	at := vec{200, 150}

	tests := []struct {
		name   string
		anim   animation
		frame  int
		hasBed bool
		rotate bool
	}{
		{"dormido en cama", animSleepBed, 0, true, true},
		{"dormido en cama, cuadro 2", animSleepBed, 1, true, true},
		{"dormido en el piso (aunque tenga cama asignada)", animSleepFloor, 0, true, false},
		{"desplomarse", animCollapse, 3, false, false},
		{"quieto", animIdle, 0, false, false},
		{"dormido en cama pero sin cama conocida", animSleepBed, 0, false, false},
	}
	for _, tt := range tests {
		_, rotated := spritePlacement(tt.anim, tt.frame, bed, tt.hasBed, at, false)
		if rotated != tt.rotate {
			t.Errorf("%s: rotado = %v; se esperaba %v", tt.name, rotated, tt.rotate)
		}
	}
}

// En la cama, la cabeza (a la izquierda del cuadro horizontal) queda hacia
// la cabecera (arriba) y el cuerpo queda dentro de la cama.
func TestSpritePlacement_InBedTheHeadPointsToTheHeadboard(t *testing.T) {
	bed := bedRect(zoneRoom101)

	g, _ := spritePlacement(animSleepBed, 0, bed, true, vec{}, false)

	headX, headY := g.Apply(2, 12)  // la cabeza, a la izquierda del cuadro
	feetX, feetY := g.Apply(14, 12) // los pies, a la derecha
	if headY >= feetY {
		t.Errorf("la cabeza (y=%.0f) debe quedar arriba de los pies (y=%.0f)", headY, feetY)
	}
	for _, p := range []image.Point{{int(headX), int(headY)}, {int(feetX), int(feetY)}} {
		if !p.In(bed) {
			t.Errorf("el punto %v quedó fuera de la cama %v", p, bed)
		}
	}
}

// Pedido por Samuel: al despertarse, el cuadro 1 va dentro de la cama
// (rotado) y desde el cuadro 2 (sentado) va de pie al lado, sin rotar.
func TestSpritePlacement_WakeUpStartsInBedThenStandsBesideIt(t *testing.T) {
	bed := bedRect(zoneRoom101)
	beside := toVec(wakeSpot(zoneRoom101))

	g, rotated := spritePlacement(animWakeUp, 0, bed, true, beside, false)
	x, y := g.Apply(8, 12)
	if !rotated || !image.Pt(int(x), int(y)).In(bed) {
		t.Errorf("cuadro 1 de despertarse: rotado=%v en (%.0f, %.0f); se esperaba rotado y dentro de la cama %v", rotated, x, y, bed)
	}

	for frame := 1; frame < 4; frame++ {
		g, rotated := spritePlacement(animWakeUp, frame, bed, true, beside, false)
		x, y := g.Apply(0, 0)
		if rotated || x != beside.x || y != beside.y {
			t.Errorf("cuadro %d de despertarse: rotado=%v en (%.0f, %.0f); se esperaba sin rotar en %v", frame+1, rotated, x, y, beside)
		}
	}

	// "Al lado" de verdad: el cuadro de pie no se monta sobre la cama, y
	// queda a la IZQUIERDA (la derecha es para la etiqueta del que esté en cama).
	p := wakeSpot(zoneRoom101)
	if image.Rect(p.X, p.Y, p.X+cellWidth, p.Y+cellHeight).Overlaps(bed) {
		t.Errorf("el puesto al lado de la cama %v se monta sobre la cama %v", p, bed)
	}
	if p.X+cellWidth > bed.Min.X {
		t.Errorf("el que se despierta debe quedar a la izquierda de la cama: x=%d, cama desde x=%d", p.X, bed.Min.X)
	}
}

// Caminar a la derecha: la animación de la izquierda volteada, sin moverla
// de su cuadro.
func TestSpritePlacement_FlipKeepsTheSpriteInItsCell(t *testing.T) {
	at := vec{100, 50}

	g, _ := spritePlacement(animWalkLeft, 0, image.Rectangle{}, false, at, true)

	var want ebiten.GeoM
	want.Scale(-1, 1)
	want.Translate(cellWidth, 0)
	want.Translate(at.x, at.y)
	for _, px := range []float64{0, 15} {
		gx, gy := g.Apply(px, 0)
		wx, wy := want.Apply(px, 0)
		if gx != wx || gy != wy {
			t.Errorf("x=%v: (%v, %v); se esperaba (%v, %v)", px, gx, gy, wx, wy)
		}
	}
	if x, _ := g.Apply(0, 0); x != at.x+cellWidth {
		t.Errorf("el borde izquierdo del sprite debe quedar a la derecha del cuadro: x=%v", x)
	}
}
