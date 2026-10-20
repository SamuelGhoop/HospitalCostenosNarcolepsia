package sfx

import "math"

// Volumen parejo (pedido de Samuel el 2026-10-19: "los sonidos están muy
// duros y disparejos"). Antes cada efecto tenía su propio volumen a ojo y,
// como una cuadrada suena mucho más fuerte que una triangular al mismo
// volumen, entre el más suave y el más fuerte había 14 dB. Ahora cada
// efecto se normaliza: se mide qué tan fuerte se oye (Loudness) y se
// escala para que quede en TargetLoudness por el nivel de su grupo.

// TargetLoudness es el nivel de un evento del juego (−18 dBFS de RMS en la
// ventana más fuerte): bajo a propósito, para que dos o tres efectos a la
// vez sigan sin saturar.
const TargetLoudness = 0.125

// Niveles por grupo, como razón de amplitud respecto a TargetLoudness:
// los eventos (cobro, estrella, desplome…) a 0 dB; la interfaz (clic,
// abrir, cerrar), que suena todo el tiempo, 3 dB más bajo; el ronquido,
// que es ambiente, 9 dB más bajo; la música de fondo (music.go), que no
// para nunca, 12 dB más baja, para que cualquier efecto se oiga encima.
const (
	LevelEvent   = 1.0
	LevelUI      = 0.708 // 10^(−3/20)
	LevelAmbient = 0.355 // 10^(−9/20)
	LevelMusic   = 0.251 // 10^(−12/20)
)

// PeakCeiling: ninguna muestra normalizada pasa de aquí (la mitad de la
// escala). Así queda espacio para que se sumen varios efectos sin recorte.
const PeakCeiling = 0.5

// loudnessWindow: 30 ms, más o menos lo que el oído "integra" para decidir
// qué tan fuerte es un sonido corto.
const loudnessWindow = SampleRate * 30 / 1000

// Loudness es el RMS de la ventana de 30 ms más fuerte del efecto (con
// pasos de 10 ms). Se usa la ventana más fuerte y no el promedio de todo
// el efecto porque, si no, un efecto con silencios o una cola larga
// parecería más suave de lo que se oye.
func Loudness(data []byte) float64 {
	n := len(data) / 4
	win := loudnessWindow
	if n < win {
		win = n
	}
	if win == 0 {
		return 0
	}
	best := 0.0
	for start := 0; start+win <= n; start += win / 3 {
		sum := 0.0
		for i := start; i < start+win; i++ {
			v := sampleAt(data, i)
			sum += v * v
		}
		best = math.Max(best, math.Sqrt(sum/float64(win)))
		if win/3 == 0 {
			break
		}
	}
	return best
}

// Normalize escala el efecto para que su Loudness sea target, sin que
// ninguna muestra pase de PeakCeiling (si el pico no lo permite, queda un
// poco más suave). Un efecto mudo se devuelve igual.
func Normalize(data []byte, target float64) []byte {
	out := make([]byte, len(data))
	loud := Loudness(data)
	if loud == 0 {
		copy(out, data)
		return out
	}
	n := len(data) / 4
	peak := 0.0
	for i := 0; i < n; i++ {
		peak = math.Max(peak, math.Abs(sampleAt(data, i)))
	}
	gain := target / loud
	if peak*gain > PeakCeiling {
		gain = PeakCeiling / peak
	}
	for i := 0; i < n; i++ {
		writeSample(out, i, sampleAt(data, i)*gain)
	}
	return out
}
