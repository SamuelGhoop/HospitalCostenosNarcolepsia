package ui

import (
	"math"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// ticksAt pasa segundos a ticks (60 por segundo), redondeando hacia arriba
// para caer justo después del instante pedido.
func ticksAt(seconds float64) int { return int(math.Ceil(seconds * 60)) }

// TITLE_ASSETS.md: las nubes van de x = −42 a 320, 1 px por paso, y vuelven
// a empezar. La A da un paso cada 0,249 s y arranca en 40; la B, cada
// 0,359 s y arranca en 12.
func TestTitleLayers_Clouds(t *testing.T) {
	if x := cloudX(0, cloudA); x != 40 {
		t.Errorf("nube A al empezar: x = %d; se esperaba 40", x)
	}
	if x := cloudX(ticksAt(0.249), cloudA); x != 41 {
		t.Errorf("nube A después de un paso: x = %d; se esperaba 41", x)
	}
	if x := cloudX(ticksAt(0.249*280), cloudA); x != -42 {
		t.Errorf("nube A después de 280 pasos: x = %d; debía volver a −42", x)
	}
	if x := cloudX(0, cloudB); x != 12 {
		t.Errorf("nube B al empezar: x = %d; se esperaba 12", x)
	}
	if x := cloudX(ticksAt(0.359*3), cloudB); x != 15 {
		t.Errorf("nube B después de 3 pasos: x = %d; se esperaba 15", x)
	}
}

// Olas: x = −8 + d, con d = 0, 1, 2, 3, 2, 1… un paso cada 0,6 s. Espuma:
// x = −8 − d, un paso cada 0,45 s.
func TestTitleLayers_WavesAndFoam(t *testing.T) {
	for step, d := range []int{0, 1, 2, 3, 2, 1, 0, 1} {
		if x := wavesX(step * 36); x != -8+d {
			t.Errorf("olas, paso %d: x = %d; se esperaba %d", step, x, -8+d)
		}
		if x := foamX(step * 27); x != -8-d {
			t.Errorf("espuma, paso %d: x = %d; se esperaba %d", step, x, -8-d)
		}
	}
}

// Copas de las palmeras: cuadros 0,1,2,3,4,3,2,1… uno cada 0,75 s; la copa 2
// va medio ciclo desfasada.
func TestTitleLayers_PalmCrowns(t *testing.T) {
	want := []int{0, 1, 2, 3, 4, 3, 2, 1, 0}
	for step, f := range want {
		if got := crownFrame(step*45, false); got != f {
			t.Errorf("copa 1, paso %d: cuadro %d; se esperaba %d", step, got, f)
		}
	}
	if got := crownFrame(0, true); got != 4 {
		t.Errorf("copa 2 al empezar: cuadro %d; se esperaba 4 (medio ciclo después)", got)
	}
}

// El paciente respira (r = 0, −1, −2, −1, un valor por segundo) y cabecea en
// un ciclo de 6 s.
func TestTitleLayers_PatientBreathesAndNods(t *testing.T) {
	for step, r := range []int{0, -1, -2, -1, 0} {
		if got := breathY(step * 60); got != r {
			t.Errorf("respiración, segundo %d: r = %d; se esperaba %d", step, got, r)
		}
	}
	cases := []struct {
		seconds float64
		dx, dy  int
	}{{0, 0, 0}, {4.1, 0, 0}, {4.5, 1, 1}, {5.0, 1, 2}, {5.5, 0, 1}, {5.8, 0, 0}, {6.1, 0, 0}}
	for _, c := range cases {
		if dx, dy := headNod(ticksAt(c.seconds)); dx != c.dx || dy != c.dy {
			t.Errorf("cabeceo a los %.1f s: (%d, %d); se esperaba (%d, %d)", c.seconds, dx, dy, c.dx, c.dy)
		}
	}
}

// Zzz del paciente: ciclo de 3 s en 6 pasos de 0,5 s; z baja de +2 de a un
// píxel; la opacidad sube a 1 al 30 % y se apaga al final. Desfases de 0, 1
// y 2 s.
func TestTitleLayers_PatientZzz(t *testing.T) {
	for step := 0; step < 6; step++ {
		z, alpha := patientZzz(step*30, 0)
		if z != 2-step {
			t.Errorf("zzz1, paso %d: z = %d; se esperaba %d", step, z, 2-step)
		}
		p := float64(step) / 6
		want := p / 0.3
		if p >= 0.3 {
			want = (1 - p) / 0.7
		}
		if math.Abs(float64(alpha)-want) > 0.01 {
			t.Errorf("zzz1, paso %d: opacidad %.2f; se esperaba %.2f", step, alpha, want)
		}
	}
	if z, _ := patientZzz(60, 1); z != 2 { // la zzz2 empieza 1 s después
		t.Errorf("zzz2 al segundo 1: z = %d; se esperaba 2 (recién empieza)", z)
	}
}

// "PRESIONA ENTER" parpadea cada 1,2 s: se ve 0,6 s y se apaga 0,6 s.
func TestTitleLayers_PressEnterBlinks(t *testing.T) {
	for _, c := range []struct {
		tick int
		on   bool
	}{{0, true}, {35, true}, {36, false}, {71, false}, {72, true}} {
		if got := pressEnterVisible(c.tick); got != c.on {
			t.Errorf("tick %d: visible = %v; se esperaba %v", c.tick, got, c.on)
		}
	}
}

// Las capas de la portada están embebidas con las medidas de TITLE_ASSETS.md.
func TestTitleAssets_AreEmbeddedWithTheirSizes(t *testing.T) {
	sizes := map[string][2]int{
		"title/fondo.png": {320, 180}, "title/suelo.png": {320, 180},
		"title/nube_a.png": {42, 15}, "title/nube_b.png": {42, 15},
		"title/olas.png": {336, 28}, "title/espuma.png": {336, 5},
		"title/copa1_frames.png": {370, 42}, "title/copa2_frames.png": {370, 42},
		"title/patient_legs.png": {32, 48}, "title/patient_torso.png": {32, 48}, "title/patient_head.png": {32, 48},
		"title/zzz1.png": {32, 48}, "title/zzz2.png": {32, 48}, "title/zzz3.png": {32, 48},
	}
	// El portapapeles: con partida guardada, una imagen por opción; sin
	// partida, las que dejan CONTINUAR en gris (que no se puede elegir).
	for _, o := range titleOptions {
		for frame := 0; frame < 2; frame++ {
			sizes[clipboardPath(o, true, frame)] = [2]int{425, 447}
			if o != optContinue {
				sizes[clipboardPath(o, false, frame)] = [2]int{425, 447}
			}
		}
	}
	for path, size := range sizes {
		img, err := decodePNG(assets.Title, path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if b := img.Bounds(); b.Dx() != size[0] || b.Dy() != size[1] {
			t.Errorf("%s mide %d×%d; se esperaba %d×%d", path, b.Dx(), b.Dy(), size[0], size[1])
		}
	}
}
