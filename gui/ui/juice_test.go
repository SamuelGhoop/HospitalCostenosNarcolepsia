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
