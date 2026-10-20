package sfx

import (
	"math"
	"testing"
)

// noteFreq lee el nombre de una nota (letra, sostenido opcional y octava)
// con La4 = 440 Hz.
func TestNoteFreq(t *testing.T) {
	cases := map[string]float64{"A4": 440, "A5": 880, "C6": 1046.50, "E5": 659.26, "F#4": 369.99, "C3": 130.81}
	for name, want := range cases {
		if got := noteFreq(name); math.Abs(got-want) > 0.05 {
			t.Errorf("noteFreq(%q) = %.2f; se esperaba %.2f", name, got, want)
		}
	}
}

// El tema dura exactamente musicBars compases de 8 corcheas: así el bucle
// cae siempre en el tiempo y no se "corre" al volver a empezar.
func TestMusic_LengthIsAWholeNumberOfBars(t *testing.T) {
	want := musicBars * 8 * musicStepSamples * 4
	if got := len(Music()); got != want {
		t.Errorf("len = %d; se esperaba %d (%d compases)", got, want, musicBars)
	}
}

// El bucle no "clickea" al volver a empezar: empieza y termina en silencio.
func TestMusic_LoopsWithoutAClick(t *testing.T) {
	data := Music()
	n := len(data) / 4
	for _, i := range []int{0, n - 1} {
		if v := math.Abs(sampleAt(data, i)); v > 0.002 {
			t.Errorf("muestra %d = %.4f; se esperaba silencio para que el bucle no clickee", i, v)
		}
	}
}

// La música va por debajo de los efectos (LevelMusic), con ±1 dB.
func TestMusic_SitsAtTheMusicLevel(t *testing.T) {
	want := TargetLoudness * LevelMusic
	if got := Loudness(Music()); math.Abs(dB(got/want)) > 1 {
		t.Errorf("Loudness = %.3f; se esperaba %.3f (±1 dB)", got, want)
	}
	if LevelMusic >= LevelUI {
		t.Error("la música debería ir más baja que los sonidos de la interfaz")
	}
}
