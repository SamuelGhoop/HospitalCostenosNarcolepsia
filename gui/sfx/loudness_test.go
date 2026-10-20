package sfx

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// flat es una envolvente sin ataque ni caída (sostenido al 100 %), con una
// liberación mínima para que siga terminando en silencio.
var flat = Envelope{Attack: 0, Decay: 0, Sustain: 1, Release: 0.01}

// dB pasa una razón de amplitudes a decibeles.
func dB(ratio float64) float64 { return 20 * math.Log10(ratio) }

// Una onda cuadrada de amplitud A tiene RMS A: Loudness de una cuadrada
// pareja al 0,5 da 0,5 (dentro del redondeo de 16 bits).
func TestLoudness_SquareEqualsItsAmplitude(t *testing.T) {
	data := Tone(Square, 440, 440, 200*time.Millisecond, 0.5, flat, nil)
	if got := Loudness(data); math.Abs(got-0.5) > 0.01 {
		t.Errorf("Loudness = %.3f; se esperaba 0,5", got)
	}
}

// Normalize lleva cualquier efecto al nivel pedido, suba o baje.
func TestNormalize_ReachesTheTarget(t *testing.T) {
	for _, vol := range []float64{0.05, 0.6} {
		data := Normalize(Tone(Triangle, 300, 600, 200*time.Millisecond, vol, DefaultEnvelope, nil), 0.15)
		if got := Loudness(data); math.Abs(dB(got/0.15)) > 0.5 {
			t.Errorf("volumen %.2f: Loudness = %.3f; se esperaba 0,15 (±0,5 dB)", vol, got)
		}
	}
}

// Normalize nunca deja pasar un pico por encima de PeakCeiling, aunque el
// nivel pedido lo exija (el ruido tiene picos muy por encima de su RMS).
func TestNormalize_NeverExceedsThePeakCeiling(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	data := Normalize(Tone(Noise, 0, 0, 200*time.Millisecond, 0.3, flat, rnd), 0.8)
	for i := 0; i*4+3 < len(data); i++ {
		if v := math.Abs(sampleAt(data, i)); v > PeakCeiling+0.001 {
			t.Fatalf("muestra %d = %.3f; el techo es %.2f", i, v, PeakCeiling)
		}
	}
}

// El pulso al 25 % (el "timbre" Game Boy de Pokémon) pasa un cuarto de
// cada período arriba y el resto abajo.
func TestTone_Pulse25IsHighAQuarterOfTheTime(t *testing.T) {
	data := Tone(Pulse25, 441, 441, 200*time.Millisecond, 0.5, flat, nil)
	n := len(data) / 4
	high := 0
	for i := 0; i < n; i++ {
		if sampleAt(data, i) > 0 {
			high++
		}
	}
	if frac := float64(high) / float64(n); math.Abs(frac-0.25) > 0.02 {
		t.Errorf("arriba el %.0f %% del tiempo; se esperaba 25 %%", frac*100)
	}
}

// Como el canal de onda de la Game Boy, la triangular tiene solo 16
// escalones (4 bits): con volumen y envolvente fijos no hay más de 16
// valores distintos.
func TestTone_TriangleHasSixteenSteps(t *testing.T) {
	data := Tone(Triangle, 220, 220, 100*time.Millisecond, 0.5, Envelope{Sustain: 1}, nil)
	levels := map[int16]bool{}
	n := len(data) / 4
	for i := n / 4; i < n*3/4; i++ { // la parte sostenida
		levels[int16(uint16(data[i*4])|uint16(data[i*4+1])<<8)] = true
	}
	if len(levels) > 16 {
		t.Errorf("%d niveles distintos; se esperaban como mucho 16", len(levels))
	}
}

// El pedido de Samuel (2026-10-19): el volumen parejo en todo el juego.
// Cada efecto queda a TargetLoudness más el nivel de su grupo (interfaz,
// evento o ambiente), con ±1 dB de tolerancia.
func TestEffects_ConsistentLoudness(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	cases := []struct {
		name  string
		data  []byte
		level float64
	}{
		{"click", Click(rnd), LevelUI},
		{"open", Open(rnd), LevelUI},
		{"close", Close(rnd), LevelUI},
		{"hire", Hire(rnd), LevelEvent},
		{"coin(0)", Coin(0, rnd), LevelEvent},
		{"coin(4)", Coin(4, rnd), LevelEvent},
		{"streakLost", StreakLost(rnd), LevelEvent},
		{"starDown", StarDown(rnd), LevelEvent},
		{"starUp", StarUp(rnd), LevelEvent},
		{"collapse", Collapse(rnd), LevelEvent},
		{"rush", Rush(rnd), LevelEvent},
		{"dayEnd", DayEnd(rnd), LevelEvent},
		{"gameOver", GameOver(rnd), LevelEvent},
		{"snore", Snore(rnd), LevelAmbient},
	}
	for _, c := range cases {
		want := TargetLoudness * c.level
		if got := Loudness(c.data); math.Abs(dB(got/want)) > 1 {
			t.Errorf("%s: %.1f dB respecto a lo esperado (Loudness %.3f, se esperaba %.3f)", c.name, dB(got/want), got, want)
		}
	}
}
