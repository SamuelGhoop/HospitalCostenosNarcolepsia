package sfx

import (
	"math/rand"
	"testing"
)

// Los 13 efectos (GAME_DESIGN §10.11) duran entre 50 y 600 ms, no superan
// el volumen (ninguna muestra pasa de 32767) y empiezan y terminan en
// silencio, como cualquier Tone. Coin además prueba cada racha de 0 a 4.
func TestEffects_DurationAndSilence(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	effects := map[string][]byte{
		"click":      Click(rnd),
		"open":       Open(rnd),
		"close":      Close(rnd),
		"hire":       Hire(rnd),
		"coin(0)":    Coin(0, rnd),
		"coin(4)":    Coin(4, rnd),
		"streakLost": StreakLost(rnd),
		"starDown":   StarDown(rnd),
		"starUp":     StarUp(rnd),
		"collapse":   Collapse(rnd),
		"rush":       Rush(rnd),
		"dayEnd":     DayEnd(rnd),
		"gameOver":   GameOver(rnd),
		"snore":      Snore(rnd),
	}
	for name, data := range effects {
		if len(data) == 0 {
			t.Errorf("%s: no generó nada", name)
			continue
		}
		ms := float64(len(data)/4) / float64(SampleRate) * 1000
		if ms < 50 || ms > 600 {
			t.Errorf("%s dura %.0f ms; se esperaba entre 50 y 600", name, ms)
		}
		// Tolerancia chica (no exactamente 0): Collapse mezcla un pasabajos,
		// que deja un resto casi inaudible por el "arrastre" del filtro,
		// aunque la envolvente ya haya llegado a 0. Los demás (sin filtro)
		// sí quedan en 0 exacto.
		const silence = 50 // de 32767: muy por debajo de lo audible
		n := len(data) / 4
		if s := int16(uint16(data[0]) | uint16(data[1])<<8); s > silence || s < -silence {
			t.Errorf("%s: primera muestra %d; se esperaba cerca de 0", name, s)
		}
		last := (n - 1) * 4
		if s := int16(uint16(data[last]) | uint16(data[last+1])<<8); s > silence || s < -silence {
			t.Errorf("%s: última muestra %d; se esperaba cerca de 0", name, s)
		}
	}
}

// Con racha, la segunda nota de Coin sube de tono: más racha, frecuencia
// más alta (se nota en que cruza por cero más seguido).
func TestCoin_HigherStreakMeansHigherPitch(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	zeroCrossings := func(data []byte) int {
		n := len(data) / 4
		count := 0
		prev := int16(0)
		for i := 0; i < n; i++ {
			s := int16(uint16(data[i*4]) | uint16(data[i*4+1])<<8)
			if i > 0 && (s > 0) != (prev > 0) && s != 0 {
				count++
			}
			if s != 0 {
				prev = s
			}
		}
		return count
	}
	prevCrossings := 0
	for streak := 0; streak <= coinStreakCap; streak++ {
		data := Coin(streak, rnd)
		// Solo la segunda nota (la mitad de atrás) cambia con la racha.
		half := data[len(data)/2:]
		c := zeroCrossings(half)
		if streak > 0 && c <= prevCrossings {
			t.Errorf("racha %d: %d cruces por cero en la 2ª nota; se esperaban más que en la racha %d (%d)", streak, c, streak-1, prevCrossings)
		}
		prevCrossings = c
	}
}

// La racha se topa en 4: una racha mayor no sigue subiendo el tono.
func TestCoin_StreakCapsAtFour(t *testing.T) {
	a := Coin(4, rand.New(rand.NewSource(1)))
	b := Coin(9, rand.New(rand.NewSource(1)))
	if string(a) != string(b) {
		t.Error("una racha de 9 sonó distinto que una de 4; debería topar ahí")
	}
}
