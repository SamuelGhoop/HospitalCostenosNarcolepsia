package sfx

import (
	"math"
	"strings"
	"time"
)

// Música de fondo (pedido de Samuel el 2026-10-19: "add a background music
// of the same style"). Es un tema original — no copia ninguna canción de
// Pokémon — con el aire tranquilo de un Centro Pokémon: Do mayor, 120
// pulsos por minuto, 16 compases (32 s) que se repiten en bucle. Suena con
// los mismos tres canales de la Game Boy que los efectos:
//   - melodía: pulso al 25 %,
//   - arpegio: pulso al 12,5 %, bajito,
//   - bajo: la triangular de 4 bits.
// Como los efectos, es una función pura (sin azar ni Ebitengine), así que
// se prueba sola; gui/ui/sound.go la pone a sonar en bucle.

// musicStepSamples: una corchea a 120 pulsos por minuto (0,25 s). Es un
// número entero de muestras, así el bucle cae siempre exacto.
const musicStepSamples = SampleRate / 4

// musicBars: compases del tema, de 8 corcheas cada uno.
const musicBars = 16

// musicMelody: una línea por compás, 8 corcheas separadas por espacios.
// Un nombre de nota (E5, F#4…) empieza una nota, "." alarga la anterior
// una corchea y "_" es silencio. Parte A (compases 1 a 8) y parte B (9 a
// 16); el último compás sube a Mi5 para volver al principio.
var musicMelody = []string{
	// A
	"E5 . G5 . C6 . G5 .",
	"A5 . . . E5 . C5 .",
	"F5 . A5 . C6 . A5 .",
	"G5 . . . . . _ _",
	"E5 . G5 . C6 . D6 .",
	"E6 . D6 . C6 . A5 .",
	"F5 . A5 . G5 . B5 .",
	"C6 . . . . . _ _",
	// B
	"A5 . C6 . A5 . F5 .",
	"G5 . B5 . D6 . B5 .",
	"G5 . E5 . B4 . E5 .",
	"A5 . . . C6 . . .",
	"A5 . G5 . F5 . A5 .",
	"G5 . F5 . E5 . D5 .",
	"E5 . C5 . G5 . E5 .",
	"D5 . . . G4 . B4 D5",
}

// musicChords: dos acordes por compás (uno por cada mitad), en el mismo
// orden que musicMelody.
var musicChords = []string{
	"C C", "Am Am", "F F", "G G", "C C", "Am Am", "Dm G", "C C",
	"F F", "G G", "Em Em", "Am Am", "F F", "G G", "C C", "G G",
}

// chordTones: las tres notas de cada acorde para el arpegio (todas cerca
// de la octava 4, para que no tape la melodía) y la fundamental del bajo.
var chordTones = map[string]struct {
	arp  [3]string
	bass string
}{
	"C":  {[3]string{"C4", "E4", "G4"}, "C3"},
	"Am": {[3]string{"A3", "C4", "E4"}, "A2"},
	"F":  {[3]string{"F4", "A4", "C5"}, "F2"},
	"G":  {[3]string{"G3", "B3", "D4"}, "G2"},
	"Dm": {[3]string{"D4", "F4", "A4"}, "D3"},
	"Em": {[3]string{"E4", "G4", "B4"}, "E3"},
}

// Envolventes de las notas: ataque casi inmediato y caída, como el
// registro de volumen de la Game Boy.
var (
	leadEnvelope = Envelope{Attack: 0.02, Decay: 0.25, Sustain: 0.6, Release: 0.15}
	bassEnvelope = Envelope{Attack: 0.02, Decay: 0.3, Sustain: 0.7, Release: 0.2}
)

// Volúmenes de cada canal antes de normalizar (solo importa la proporción
// entre ellos: Normalize después lleva el total a LevelMusic).
const (
	leadVolume = 0.4
	arpVolume  = 0.12
	bassVolume = 0.35
)

// noteFreq pasa un nombre de nota ("A4", "F#4", "C6") a su frecuencia en
// Hz, con La4 = 440 Hz y temperamento igual.
func noteFreq(name string) float64 {
	semitones := map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
	semi := semitones[name[0]]
	rest := name[1:]
	if strings.HasPrefix(rest, "#") {
		semi++
		rest = rest[1:]
	}
	octave := int(rest[0] - '0')
	midi := 12*(octave+1) + semi // número MIDI: La4 es el 69
	return 440 * math.Pow(2, float64(midi-69)/12)
}

// addNote sintetiza una nota de steps corcheas y la suma al buffer mix
// desde la corchea start.
func addNote(mix []float64, wave Wave, freq float64, start, steps int, volume float64, env Envelope) {
	dur := time.Duration(steps*musicStepSamples) * time.Second / SampleRate
	note := Tone(wave, freq, freq, dur, volume, env, nil)
	offset := start * musicStepSamples
	for i := 0; i*4+3 < len(note) && offset+i < len(mix); i++ {
		mix[offset+i] += sampleAt(note, i)
	}
}

// Music sintetiza una vuelta completa del tema (musicBars compases), ya
// normalizada a LevelMusic. El final empalma con el principio: todas las
// notas empiezan y terminan en silencio.
func Music() []byte {
	totalSteps := musicBars * 8
	mix := make([]float64, totalSteps*musicStepSamples)

	// Melodía: se recorre corchea por corchea; cada nota dura hasta el
	// siguiente nombre de nota o silencio.
	step := 0
	noteName, noteStart := "", 0
	flush := func(end int) {
		if noteName != "" {
			addNote(mix, Pulse25, noteFreq(noteName), noteStart, end-noteStart, leadVolume, leadEnvelope)
		}
		noteName = ""
	}
	for _, bar := range musicMelody {
		for _, tok := range strings.Fields(bar) {
			switch tok {
			case ".":
			case "_":
				flush(step)
			default:
				flush(step)
				noteName, noteStart = tok, step
			}
			step++
		}
	}
	flush(step)

	// Arpegio y bajo: por cada mitad de compás (4 corcheas), el arpegio
	// toca 1-3-5-3 del acorde y el bajo la fundamental y su octava, una
	// negra cada una.
	for bar, pair := range musicChords {
		for half, name := range strings.Fields(pair) {
			chord := chordTones[name]
			start := bar*8 + half*4
			for i, tone := range []int{0, 1, 2, 1} {
				addNote(mix, Pulse12, noteFreq(chord.arp[tone]), start+i, 1, arpVolume, leadEnvelope)
			}
			root := noteFreq(chord.bass)
			addNote(mix, Triangle, root, start, 2, bassVolume, bassEnvelope)
			addNote(mix, Triangle, root*2, start+2, 2, bassVolume, bassEnvelope)
		}
	}

	data := make([]byte, len(mix)*4)
	for i, v := range mix {
		writeSample(data, i, v)
	}
	return Normalize(data, TargetLoudness*LevelMusic)
}
