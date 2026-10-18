package sfx

import (
	"math/rand"
	"testing"
	"time"
)

// sampleInt16 lee la muestra i como lo que de verdad quedó escrito (un
// int16), para comparar sin el redondeo de sampleAt.
func sampleInt16(data []byte, i int) int16 {
	j := i * 4
	return int16(uint16(data[j]) | uint16(data[j+1])<<8)
}

// Tone da el largo correcto: muestras × 4 bytes (16 bits, 2 canales).
func TestTone_LengthMatchesDuration(t *testing.T) {
	cases := []struct {
		name string
		dur  time.Duration
	}{
		{"50ms", 50 * time.Millisecond},
		{"200ms", 200 * time.Millisecond},
		{"600ms", 600 * time.Millisecond},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := Tone(Square, 440, 440, c.dur, 0.5, DefaultEnvelope, nil)
			wantSamples := int(float64(SampleRate) * c.dur.Seconds())
			if got, want := len(data), wantSamples*4; got != want {
				t.Errorf("len = %d; se esperaba %d (%d muestras × 4 bytes)", got, want, wantSamples)
			}
		})
	}
}

// Ninguna onda (cuadrada, triangular o ruido) se recorta: con volume ≤ 1
// ninguna muestra llega a desbordar int16.
func TestTone_NoClipping(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	for _, w := range []Wave{Square, Triangle, Noise} {
		data := Tone(w, 300, 900, 300*time.Millisecond, 1.0, DefaultEnvelope, rnd)
		for i := 0; i*4+3 < len(data); i++ {
			s := sampleInt16(data, i)
			if s == -32768 { // el único valor que int16 no puede "reflejar": sería recorte
				t.Fatalf("onda %v, muestra %d: %d está en el borde del recorte", w, i, s)
			}
		}
	}
}

// Todo efecto empieza y termina en silencio (amplitud 0 en la primera y la
// última muestra): así no suena un "clic" al arrancar o al cortar.
func TestTone_StartsAndEndsSilent(t *testing.T) {
	data := Tone(Square, 440, 880, 200*time.Millisecond, 0.8, DefaultEnvelope, nil)
	n := len(data) / 4
	if s := sampleInt16(data, 0); s != 0 {
		t.Errorf("primera muestra = %d; se esperaba 0", s)
	}
	if s := sampleInt16(data, n-1); s != 0 {
		t.Errorf("última muestra = %d; se esperaba 0", s)
	}
}

// El volumen efectivamente limita la amplitud: con volume 0,3 ninguna
// muestra pasa de ese nivel (dentro del redondeo de 16 bits).
func TestTone_VolumeCapsAmplitude(t *testing.T) {
	data := Tone(Square, 440, 440, 100*time.Millisecond, 0.3, DefaultEnvelope, nil)
	volume := 0.3
	limit := int16(volume * 32767)
	for i := 0; i*4+3 < len(data); i++ {
		if s := sampleInt16(data, i); s > limit+1 || s < -limit-1 {
			t.Fatalf("muestra %d = %d; se esperaba como mucho ±%d (volumen 0,3)", i, s, limit)
		}
	}
}

// Un barrido de freqFrom a freqTo iguales es un tono fijo: Square con la
// misma frecuencia cruza por cero siempre al mismo ritmo (medio período).
func TestTone_FixedFrequencySquareWave_HasExpectedPeriod(t *testing.T) {
	const freq = 440.0
	data := Tone(Square, freq, freq, 100*time.Millisecond, 1, Envelope{Sustain: 1}, nil) // sin ataque/caída: amplitud 1 todo el tiempo salvo las puntas
	period := SampleRate / freq
	// A medio período el signo debería cambiar.
	first := sampleInt16(data, 10)
	half := sampleInt16(data, 10+int(period/2))
	if (first > 0) == (half > 0) {
		t.Errorf("a medio período (%d muestras) la onda cuadrada no cambió de signo", int(period/2))
	}
}

// LowPass suaviza: el resultado de filtrar ruido blanco varía menos de
// muestra a muestra que el original (es "más grave").
func TestLowPass_SmoothsNoise(t *testing.T) {
	rnd := rand.New(rand.NewSource(2))
	noise := Tone(Noise, 0, 0, 100*time.Millisecond, 1, Envelope{Sustain: 1}, rnd)
	filtered := LowPass(noise, 0.1)
	diffSum := func(data []byte) float64 {
		var sum float64
		for i := 1; i*4+3 < len(data); i++ {
			sum += absf(sampleAt(data, i) - sampleAt(data, i-1))
		}
		return sum
	}
	if diffSum(filtered) >= diffSum(noise) {
		t.Error("el ruido filtrado varía tanto o más entre muestras que el original; se esperaba más suave")
	}
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Concat pega los efectos uno después del otro: el largo es la suma, y
// cada tramo sigue siendo el mismo que por separado.
func TestConcat_AppendsInOrder(t *testing.T) {
	a := Tone(Square, 440, 440, 50*time.Millisecond, 0.5, DefaultEnvelope, nil)
	b := Tone(Square, 880, 880, 70*time.Millisecond, 0.5, DefaultEnvelope, nil)
	got := Concat(a, b)
	if len(got) != len(a)+len(b) {
		t.Fatalf("len = %d; se esperaba %d", len(got), len(a)+len(b))
	}
	if string(got[:len(a)]) != string(a) || string(got[len(a):]) != string(b) {
		t.Error("Concat no dejó los dos tramos tal cual")
	}
}

// Mix suma dos efectos que suenan a la vez y rellena con silencio el más
// corto hasta el largo del más largo.
func TestMix_LengthIsTheLongerOne(t *testing.T) {
	short := Tone(Square, 440, 440, 50*time.Millisecond, 0.3, DefaultEnvelope, nil)
	long := Tone(Triangle, 90, 60, 200*time.Millisecond, 0.3, DefaultEnvelope, nil)
	got := Mix(short, long)
	if len(got) != len(long) {
		t.Errorf("len = %d; se esperaba %d (el más largo)", len(got), len(long))
	}
}
