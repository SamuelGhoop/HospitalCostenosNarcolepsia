package ui

import (
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

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

// Los PNG del mapa van embebidos con sus medidas.
func TestMapAssets_AreEmbedded(t *testing.T) {
	sizes := map[string][2]int{
		backgroundPath: {1280, 720},
		blanketPath:    {56, 48},
		steamPath:      {3 * steamFrameWidth, 36},
		palmCrownPath:  {104, 104},
		carDetailsPath: {carWidth, 72},
		carBodyPath:    {carWidth, 72},
		carSignPath:    {carWidth, 72},
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

// La copa de las palmeras (palm_crown.png) se dibuja encima de los
// personajes justo donde está en el fondo: en (771, 469) y (1047, 469).
func TestPalmCrowns_SitWhereTheBackgroundHasThem(t *testing.T) {
	want := []image.Point{{771, 469}, {1047, 469}}
	if len(palmCrowns) != len(want) {
		t.Fatalf("hay %d copas; se esperaban %d", len(palmCrowns), len(want))
	}
	bg, err := decodePNG(assets.Map, backgroundPath)
	if err != nil {
		t.Fatal(err)
	}
	crown, err := decodePNG(assets.Map, palmCrownPath)
	if err != nil {
		t.Fatal(err)
	}
	for i, at := range palmCrowns {
		if at != want[i] {
			t.Errorf("copa %d en %v; se esperaba en %v", i, at, want[i])
		}
		diff := 0
		for y := 0; y < 104; y++ {
			for x := 0; x < 104; x++ {
				if _, _, _, a := crown.At(x, y).RGBA(); a == 0xffff && crown.At(x, y) != bg.At(at.X+x, at.Y+y) {
					diff++
				}
			}
		}
		if diff > 0 {
			t.Errorf("copa %d: %d píxeles no coinciden con el fondo", i, diff)
		}
	}
}
