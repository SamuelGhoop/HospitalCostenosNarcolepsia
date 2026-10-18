// Package sfx sintetiza los efectos de sonido del juego por código: nada
// de archivos de audio ni licencias (pedido de Samuel el 2026-10-17, "añade
// sonidos acordes, que la vida está ahí"; GAME_DESIGN §10.11). Son
// funciones puras — frecuencia, duración, onda y envolvente entran,
// []byte de PCM sale — así se prueban sin Ebitengine ni un dispositivo de
// audio real; gui/ui/sound.go es quien los manda a sonar con
// ebiten/audio.
package sfx

import (
	"math"
	"math/rand"
	"time"
)

// SampleRate: 44100 Hz, el que usa el contexto de audio de la interfaz
// (gui/ui/sound.go).
const SampleRate = 44100

// Wave es la forma de onda de un tono.
type Wave int

const (
	Square   Wave = iota // onda cuadrada: blips, cajas registradoras, alertas
	Triangle             // onda triangular: barridos, tonos más "redondos"
	Noise                // ruido blanco: golpes, chisporroteo
)

// Envelope es una envolvente ADSR corta, en fracciones de la duración
// total (0 a 1): sube en Attack, baja hasta Sustain en Decay, se mantiene
// en Sustain y baja a 0 en Release. Attack+Decay+Release debe ser ≤ 1; lo
// que quede en el medio es el sostenido. En la muestra 0 y en la última
// la amplitud es siempre 0, así ningún efecto "clickea" al empezar o
// terminar.
type Envelope struct {
	Attack, Decay, Release float64 // fracciones de la duración (0 a 1)
	Sustain                float64 // nivel sostenido (0 a 1)
}

// at da la amplitud (0 a 1) de la envolvente en la muestra i de n.
func (e Envelope) at(i, n int) float64 {
	if n <= 1 {
		return 0
	}
	t := float64(i) / float64(n-1)
	switch {
	case t < e.Attack:
		if e.Attack == 0 {
			return 1
		}
		return t / e.Attack
	case t < e.Attack+e.Decay:
		if e.Decay == 0 {
			return e.Sustain
		}
		return 1 - (t-e.Attack)/e.Decay*(1-e.Sustain)
	case t > 1-e.Release:
		if e.Release == 0 {
			return 0
		}
		return e.Sustain * (1 - t) / e.Release
	default:
		return e.Sustain
	}
}

// DefaultEnvelope: ataque 8 %, caída 12 % hasta el 70 % y liberación en el
// último 20 % — corta y pareja para un blip de unos cientos de
// milisegundos.
var DefaultEnvelope = Envelope{Attack: 0.08, Decay: 0.12, Sustain: 0.7, Release: 0.2}

// frac es la parte fraccionaria de x (siempre entre 0 y 1, para una fase
// que crece sin tope).
func frac(x float64) float64 { return x - math.Floor(x) }

// Tone sintetiza dur de un tono cuya frecuencia barre en línea recta de
// freqFrom a freqTo (iguales: tono fijo), con la forma de onda wave,
// la envolvente env y a lo sumo volume de amplitud (0 a 1: más de 1
// recortaría la onda, así que se topa ahí). rnd es el ruido de Noise —
// inyectado para que el mismo efecto salga igual en los tests con semilla
// fija (como el azar de game/, CLAUDE.md regla 7).
//
// El resultado es PCM de 16 bits little-endian, estéreo (los dos canales
// iguales): el formato que pide audio.Context.NewPlayerFromBytes.
func Tone(wave Wave, freqFrom, freqTo float64, dur time.Duration, volume float64, env Envelope, rnd *rand.Rand) []byte {
	if volume > 1 {
		volume = 1
	}
	if volume < 0 {
		volume = 0
	}
	n := int(float64(SampleRate) * dur.Seconds())
	if n < 2 {
		n = 2
	}
	data := make([]byte, n*4) // 2 canales × 2 bytes por muestra
	phase := 0.0
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1)
		freq := freqFrom + (freqTo-freqFrom)*t
		phase += freq / float64(SampleRate)
		var s float64
		switch wave {
		case Square:
			if frac(phase) < 0.5 {
				s = 1
			} else {
				s = -1
			}
		case Triangle:
			s = 4*math.Abs(frac(phase)-0.5) - 1
		case Noise:
			s = rnd.Float64()*2 - 1
		}
		amp := env.at(i, n) * volume
		v := s * amp
		writeSample(data, i, v)
	}
	return data
}

// writeSample escribe v (entre −1 y 1) como una muestra de 16 bits en los
// dos canales de la posición i.
func writeSample(data []byte, i int, v float64) {
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	sample := int16(v * 32767)
	lo, hi := byte(sample), byte(sample>>8)
	j := i * 4
	data[j], data[j+1] = lo, hi
	data[j+2], data[j+3] = lo, hi
}

// LowPass es un filtro pasabajos de un polo (el más simple que hay):
// y[i] = y[i-1] + alpha·(x[i]-y[i-1]). alpha chico deja pasar solo lo
// grave (así el ruido blanco de Collapse se oye como un golpe sordo, no
// como chisporroteo); alpha cerca de 1 casi no filtra nada.
func LowPass(data []byte, alpha float64) []byte {
	out := make([]byte, len(data))
	var y float64
	for i := 0; i*4+3 < len(data); i++ {
		x := sampleAt(data, i)
		y += alpha * (x - y)
		writeSample(out, i, y)
	}
	return out
}

// sampleAt lee la muestra i (de cualquiera de los dos canales, iguales)
// como un float entre −1 y 1.
func sampleAt(data []byte, i int) float64 {
	j := i * 4
	raw := int16(uint16(data[j]) | uint16(data[j+1])<<8)
	return float64(raw) / 32767
}

// Concat pega varios efectos uno después del otro (por ejemplo, las dos
// notas de Coin o las cuatro de DayEnd): cada uno ya empieza y termina en
// silencio, así no hace falta nada más para que no se note la unión.
func Concat(parts ...[]byte) []byte {
	var total int
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// Mix suma dos efectos que suenan a la vez (por ejemplo, el "golpe sordo"
// de Collapse: un tono grave y ruido filtrado juntos). El resultado mide
// lo que mida el más largo; el más corto se completa con silencio.
func Mix(a, b []byte) []byte {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	out := make([]byte, n)
	for i := 0; i*4+3 < n; i++ {
		va := 0.0
		if i*4+3 < len(a) {
			va = sampleAt(a, i)
		}
		vb := 0.0
		if i*4+3 < len(b) {
			vb = sampleAt(b, i)
		}
		writeSample(out, i, va+vb)
	}
	return out
}
