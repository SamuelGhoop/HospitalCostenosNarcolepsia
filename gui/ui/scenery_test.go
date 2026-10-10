package ui

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// §10.7: el taxi pasa por el carril de abajo (esquina superior izquierda en
// y = 651), de izquierda a derecha. Cuando sale por la derecha desaparece
// un rato y vuelve a entrar por la izquierda.
func TestScenery_TheTaxiCrossesAndComesBack(t *testing.T) {
	var s scenery

	x, y, visible := s.taxi()
	if !visible || x != -taxiWidth || y != 651 {
		t.Fatalf("al empezar el taxi está en (%v, %v), visible=%v; se esperaba entrando por la izquierda en (−%d, 651)", x, y, visible, taxiWidth)
	}

	for i := 0; i < 10; i++ {
		s.update()
	}
	if x, _, _ := s.taxi(); x != -taxiWidth+10*taxiSpeed {
		t.Errorf("después de 10 ticks el taxi está en x=%v; se esperaba %v", x, -taxiWidth+10*taxiSpeed)
	}

	// Avanza hasta que sale por la derecha. Con un límite: si el taxi no
	// sale nunca, el test falla en vez de quedarse en un bucle infinito.
	for out := false; !out; {
		x, _, visible := s.taxi()
		switch {
		case !visible:
			out = true
		case x > ScreenWidth:
			t.Fatalf("el taxi sigue visible en x=%v, ya fuera de la pantalla", x)
		case s.ticks > 2*ScreenWidth:
			t.Fatalf("después de %d ticks el taxi no ha salido de la pantalla", s.ticks)
		default:
			s.update()
		}
	}
	for i := 0; i < taxiGapTicks-1; i++ { // el rato que no se ve
		s.update()
		if _, _, visible := s.taxi(); visible {
			t.Fatalf("el taxi reapareció %d ticks después de salir; debía esperar %d", i+1, taxiGapTicks)
		}
	}
	s.update()
	if x, _, visible := s.taxi(); !visible || x != -taxiWidth {
		t.Errorf("después de la espera el taxi está en x=%v, visible=%v; debía volver a entrar por la izquierda", x, visible)
	}
}

// §10.7: el humo de la olla cambia de cuadro cada ~300 ms (18 ticks), en
// bucle 0 → 1 → 2 → 0.
func TestScenery_TheSteamLoopsThroughItsThreeFrames(t *testing.T) {
	var s scenery
	want := map[int]int{0: 0, 17: 0, 18: 1, 35: 1, 36: 2, 53: 2, 54: 0}
	for tick := 0; tick <= 54; tick++ {
		if w, ok := want[tick]; ok {
			if got := s.steamFrame(); got != w {
				t.Errorf("tick %d: cuadro del humo %d; se esperaba %d", tick, got, w)
			}
		}
		s.update()
	}
}

// Los 4 PNG del mapa van embebidos con sus medidas.
func TestMapAssets_AreEmbedded(t *testing.T) {
	sizes := map[string][2]int{
		backgroundPath: {1280, 720},
		blanketPath:    {56, 48},
		taxiPath:       {132, 72},
		steamPath:      {3 * steamFrameWidth, 36},
	}
	for path, size := range sizes {
		img, err := decodePNG(assets.Map, path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if b := img.Bounds(); b.Dx() != size[0] || b.Dy() != size[1] {
			t.Errorf("%s mide %d×%d; se esperaba %d×%d", path, b.Dx(), b.Dy(), size[0], size[1])
		}
	}
}
