package sfx

import (
	"math"
	"math/rand"
	"time"
)

// Los efectos del juego (GAME_DESIGN §10.11), todos de 50 a 600 ms. Desde
// el 2026-10-19 suenan como una Game Boy (Pokémon): pulsos delgados al 25
// y al 12,5 % en vez de la cuadrada al 50 %, que era la más dura, y cada
// uno pasa por Normalize con el nivel de su grupo (loudness.go), así todos
// se oyen igual de fuertes. rnd es el ruido de Hire y Collapse; los demás
// no lo usan, pero lo reciben igual para que la tabla de efectos de
// gui/ui/sound.go tenga una sola firma.

// atLevel normaliza un efecto ya armado al nivel de su grupo (LevelUI,
// LevelEvent o LevelAmbient).
func atLevel(level float64, data []byte) []byte {
	return Normalize(data, TargetLoudness*level)
}

// semitone es la razón de un semitono (temperamento igual, 12 por octava):
// multiplicar una frecuencia por semitone^n la sube n semitonos.
var semitone = math.Pow(2, 1.0/12)

// Click: blip corto y agudo (botón o menú).
func Click(rnd *rand.Rand) []byte {
	return atLevel(LevelUI, Tone(Pulse12, 1046.5, 1046.5, 50*time.Millisecond, 1, DefaultEnvelope, rnd)) // Do6
}

// Open: el portapapeles se abre — barrido hacia arriba (bolsa, pausa,
// reporte, fin de día, Game Over).
func Open(rnd *rand.Rand) []byte {
	return atLevel(LevelUI, Tone(Pulse25, 392, 784, 120*time.Millisecond, 1, DefaultEnvelope, rnd)) // Sol4 a Sol5
}

// Close: el portapapeles se cierra — el mismo barrido, al revés.
func Close(rnd *rand.Rand) []byte {
	return atLevel(LevelUI, Tone(Pulse25, 784, 392, 120*time.Millisecond, 1, DefaultEnvelope, rnd))
}

// Hire: caja registradora al contratar — dos blips agudos y un ruido
// corto (el "cajón" que se abre).
func Hire(rnd *rand.Rand) []byte {
	blip1 := Tone(Pulse25, 1318.5, 1318.5, 60*time.Millisecond, 1, DefaultEnvelope, rnd) // Mi6
	blip2 := Tone(Pulse25, 1760, 1760, 60*time.Millisecond, 1, DefaultEnvelope, rnd)     // La6
	clatter := LowPass(Tone(Noise, 0, 0, 120*time.Millisecond, 0.6, DefaultEnvelope, rnd), 0.3)
	return atLevel(LevelEvent, Concat(blip1, blip2, clatter))
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
	const note1, note2 = 987.77, 1318.5 // Si5 y Mi6: el "cling" de las monedas de Pokémon
	high := note2 * math.Pow(semitone, float64(streak))
	first := Tone(Pulse25, note1, note1, 70*time.Millisecond, 1, DefaultEnvelope, rnd)
	second := Tone(Pulse25, high, high, 90*time.Millisecond, 1, DefaultEnvelope, rnd)
	return atLevel(LevelEvent, Concat(first, second))
}

// StreakLost: dos notas descendentes cortas, cuando se rompe la racha.
func StreakLost(rnd *rand.Rand) []byte {
	first := Tone(Square, 493.88, 493.88, 70*time.Millisecond, 1, DefaultEnvelope, rnd)  // Si4
	second := Tone(Square, 349.23, 349.23, 90*time.Millisecond, 1, DefaultEnvelope, rnd) // Fa4
	return atLevel(LevelEvent, Concat(first, second))
}

// StarDown: barrido descendente grave cuando baja la reputación.
func StarDown(rnd *rand.Rand) []byte {
	return atLevel(LevelEvent, Tone(Triangle, 500, 180, 350*time.Millisecond, 1, DefaultEnvelope, rnd))
}

// StarUp: barrido ascendente cuando sube.
func StarUp(rnd *rand.Rand) []byte {
	return atLevel(LevelEvent, Tone(Triangle, 180, 500, 350*time.Millisecond, 1, DefaultEnvelope, rnd))
}

// Collapse: golpe sordo cuando un paciente se desploma — ruido blanco
// pasado por un pasabajos (LowPass), mezclado con un tono grave corto
// para que se sienta el "golpe".
func Collapse(rnd *rand.Rand) []byte {
	noise := Tone(Noise, 0, 0, 220*time.Millisecond, 0.6, Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.4, Release: 0.3}, rnd)
	filtered := LowPass(noise, 0.12)
	thump := Tone(Triangle, 90, 60, 220*time.Millisecond, 0.5, Envelope{Attack: 0.02, Decay: 0.3, Sustain: 0.2, Release: 0.3}, rnd)
	return atLevel(LevelEvent, Mix(filtered, thump))
}

// Rush: alerta de dos tonos al empezar una hora pico.
func Rush(rnd *rand.Rand) []byte {
	first := Tone(Pulse12, 698.46, 698.46, 100*time.Millisecond, 1, DefaultEnvelope, rnd) // Fa5
	second := Tone(Pulse12, 880, 880, 100*time.Millisecond, 1, DefaultEnvelope, rnd)      // La5
	return atLevel(LevelEvent, Concat(first, second))
}

// DayEnd: fanfarria corta de 4 notas al cerrar el día.
func DayEnd(rnd *rand.Rand) []byte {
	notes := []float64{523.25, 659.25, 784.0, 1046.5} // Do-Mi-Sol-Do, una octava arriba
	parts := make([][]byte, len(notes))
	for i, f := range notes {
		parts[i] = Tone(Pulse25, f, f, 100*time.Millisecond, 1, DefaultEnvelope, rnd)
	}
	return atLevel(LevelEvent, Concat(parts...))
}

// GameOver: 3 notas descendentes tristes.
func GameOver(rnd *rand.Rand) []byte {
	notes := []float64{440.0, 349.23, 261.63} // La-Fa-Do, descendiendo
	parts := make([][]byte, len(notes))
	for i, f := range notes {
		parts[i] = Tone(Triangle, f, f, 150*time.Millisecond, 1, DefaultEnvelope, rnd)
	}
	return atLevel(LevelEvent, Concat(parts...))
}

// Snore: muy bajo, cuando un paciente cae dormido en cama.
func Snore(rnd *rand.Rand) []byte {
	return atLevel(LevelAmbient, Tone(Triangle, 150, 100, 300*time.Millisecond, 1, DefaultEnvelope, rnd))
}

// Pase de juice (GAME_DESIGN §12.10, pedido de Samuel el 2026-10-19): 4
// efectos nuevos, con el mismo estilo Game Boy de arriba.

// Dispatch: blip corto y grave cuando un despacho (Dispatch) sale bien —
// distinto de Click (agudo, de menú): este es "algo se puso en marcha".
func Dispatch(rnd *rand.Rand) []byte {
	return atLevel(LevelUI, Tone(Pulse25, 659.25, 880, 70*time.Millisecond, 1, DefaultEnvelope, rnd)) // Mi5 a La5
}

// InvalidClick: un blip corto que cae en vez de subir — "eso no se puede".
func InvalidClick(rnd *rand.Rand) []byte {
	return atLevel(LevelUI, Tone(Pulse25, 392, 261.63, 90*time.Millisecond, 1, DefaultEnvelope, rnd)) // Sol4 a Do4
}

// Discharge: fanfarria corta (3 notas) cuando un paciente recibe el alta.
func Discharge(rnd *rand.Rand) []byte {
	notes := []float64{659.25, 784.0, 1046.5} // Mi-Sol-Do, subiendo
	parts := make([][]byte, len(notes))
	for i, f := range notes {
		parts[i] = Tone(Pulse25, f, f, 90*time.Millisecond, 1, DefaultEnvelope, rnd)
	}
	return atLevel(LevelEvent, Concat(parts...))
}

// Angry: zumbido grave cuando alguien se va enojado.
func Angry(rnd *rand.Rand) []byte {
	buzz := Tone(Square, 110, 90, 260*time.Millisecond, 0.8, Envelope{Attack: 0.02, Decay: 0.2, Sustain: 0.5, Release: 0.3}, rnd)
	return atLevel(LevelEvent, LowPass(buzz, 0.25))
}
