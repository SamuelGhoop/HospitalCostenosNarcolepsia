package sfx

import (
	"math"
	"math/rand"
	"time"
)

// Los efectos del juego (GAME_DESIGN §10.11), todos de 50 a 600 ms y a
// volumen moderado. rnd es el ruido de Collapse; los demás no lo usan,
// pero lo reciben igual para que la tabla de EffectFuncs tenga una sola
// firma.

// semitone es la razón de un semitono (temperamento igual, 12 por octava):
// multiplicar una frecuencia por semitone^n la sube n semitonos.
var semitone = math.Pow(2, 1.0/12)

// Click: blip corto y agudo (botón o menú).
func Click(rnd *rand.Rand) []byte {
	return Tone(Square, 1300, 1100, 50*time.Millisecond, 0.35, DefaultEnvelope, rnd)
}

// Open: el portapapeles se abre — barrido hacia arriba (bolsa, pausa,
// reporte, fin de día, Game Over).
func Open(rnd *rand.Rand) []byte {
	return Tone(Triangle, 320, 820, 150*time.Millisecond, 0.4, DefaultEnvelope, rnd)
}

// Close: el portapapeles se cierra — el mismo barrido, al revés.
func Close(rnd *rand.Rand) []byte {
	return Tone(Triangle, 820, 320, 150*time.Millisecond, 0.4, DefaultEnvelope, rnd)
}

// Hire: caja registradora al contratar — dos blips agudos y un ruido
// corto (el "cajón" que se abre).
func Hire(rnd *rand.Rand) []byte {
	blip1 := Tone(Square, 1500, 1500, 60*time.Millisecond, 0.4, DefaultEnvelope, rnd)
	blip2 := Tone(Square, 1900, 1900, 60*time.Millisecond, 0.4, DefaultEnvelope, rnd)
	clatter := Tone(Noise, 0, 0, 120*time.Millisecond, 0.25, DefaultEnvelope, rnd)
	return Concat(blip1, blip2, clatter)
}

// coinStreakCap: la racha solo sube la nota hasta 4 puntos (GAME_DESIGN
// §12.9: el multiplicador de pago también se topa ahí).
const coinStreakCap = 4

// Coin: cobro de una revisión o alta — dos notas ascendentes. Con racha,
// la segunda nota sube un semitono por cada punto (hasta 4), para que la
// racha "se sienta".
func Coin(streak int, rnd *rand.Rand) []byte {
	if streak < 0 {
		streak = 0
	}
	if streak > coinStreakCap {
		streak = coinStreakCap
	}
	const note1, note2 = 880.0, 1320.0
	first := Tone(Square, note1, note1, 70*time.Millisecond, 0.35, DefaultEnvelope, rnd)
	second := Tone(Square, note2*math.Pow(semitone, float64(streak)), note2*math.Pow(semitone, float64(streak)), 90*time.Millisecond, 0.35, DefaultEnvelope, rnd)
	return Concat(first, second)
}

// StreakLost: dos notas descendentes cortas, cuando se rompe la racha.
func StreakLost(rnd *rand.Rand) []byte {
	first := Tone(Square, 500, 500, 70*time.Millisecond, 0.3, DefaultEnvelope, rnd)
	second := Tone(Square, 350, 350, 90*time.Millisecond, 0.3, DefaultEnvelope, rnd)
	return Concat(first, second)
}

// StarDown: barrido descendente grave cuando baja la reputación.
func StarDown(rnd *rand.Rand) []byte {
	return Tone(Triangle, 500, 180, 350*time.Millisecond, 0.4, DefaultEnvelope, rnd)
}

// StarUp: barrido ascendente cuando sube.
func StarUp(rnd *rand.Rand) []byte {
	return Tone(Triangle, 180, 500, 350*time.Millisecond, 0.4, DefaultEnvelope, rnd)
}

// Collapse: golpe sordo cuando un paciente se desploma — ruido blanco
// pasado por un pasabajos (LowPass), mezclado con un tono grave corto
// para que se sienta el "golpe".
func Collapse(rnd *rand.Rand) []byte {
	noise := Tone(Noise, 0, 0, 220*time.Millisecond, 0.6, Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.4, Release: 0.3}, rnd)
	filtered := LowPass(noise, 0.12)
	thump := Tone(Triangle, 90, 60, 220*time.Millisecond, 0.5, Envelope{Attack: 0.02, Decay: 0.3, Sustain: 0.2, Release: 0.3}, rnd)
	return Mix(filtered, thump)
}

// Rush: alerta de dos tonos al empezar una hora pico.
func Rush(rnd *rand.Rand) []byte {
	first := Tone(Square, 700, 700, 100*time.Millisecond, 0.35, DefaultEnvelope, rnd)
	second := Tone(Square, 900, 900, 100*time.Millisecond, 0.35, DefaultEnvelope, rnd)
	return Concat(first, second)
}

// DayEnd: fanfarria corta de 4 notas al cerrar el día.
func DayEnd(rnd *rand.Rand) []byte {
	notes := []float64{523.25, 659.25, 784.0, 1046.5} // Do-Mi-Sol-Do, una octava arriba
	parts := make([][]byte, len(notes))
	for i, f := range notes {
		parts[i] = Tone(Square, f, f, 100*time.Millisecond, 0.35, DefaultEnvelope, rnd)
	}
	return Concat(parts...)
}

// GameOver: 3 notas descendentes tristes.
func GameOver(rnd *rand.Rand) []byte {
	notes := []float64{440.0, 349.23, 261.63} // La-Fa-Do, descendiendo
	parts := make([][]byte, len(notes))
	for i, f := range notes {
		parts[i] = Tone(Triangle, f, f, 150*time.Millisecond, 0.4, DefaultEnvelope, rnd)
	}
	return Concat(parts...)
}

// Snore: muy bajo, cuando un paciente cae dormido en cama.
func Snore(rnd *rand.Rand) []byte {
	return Tone(Triangle, 150, 100, 300*time.Millisecond, 0.12, DefaultEnvelope, rnd)
}
