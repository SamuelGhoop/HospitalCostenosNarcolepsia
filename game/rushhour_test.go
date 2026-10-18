package game_test

import (
	"strings"
	"testing"
	"time"
)

// §5.3 y §12.8: Snapshot.RushHour sigue las franjas de rushBands (config.go)
// y, al ENTRAR a una hora pico, sale el aviso "¡Hora pico! Llegan más
// pacientes". Se usa newQuietGame (sin llegadas por la calle) porque esto
// solo depende del reloj, no de quién aparece.
//
// La conversión es la del reloj del juego (clock.go): 1 s real = 2,4 min de
// juego, así que de 08:00 a una hora de juego dada faltan
// (minuto − 480) / 2,4 s reales.
func TestRushHour_TracksTheBandsAndNoticesWhenOneStarts(t *testing.T) {
	g := newQuietGame(t)

	if snap := g.Snapshot(); snap.RushHour {
		t.Fatalf("a las 08:00 (tranquilo) RushHour = true; se esperaba false")
	}

	tickFor(g, 50*time.Second) // (600−480)/2,4 = 50 s: son las 10:00, entra el primer pico
	snap := g.Snapshot()
	if !snap.RushHour {
		t.Fatalf("a las 10:00 RushHour = %v; se esperaba true (entra el pico)", snap.RushHour)
	}
	if n := len(snap.Notices); n == 0 || !strings.Contains(snap.Notices[n-1], "Hora pico") {
		t.Errorf("avisos = %q; se esperaba uno de hora pico al entrar al primer pico", snap.Notices)
	}

	tickFor(g, 50*time.Second) // 100 s: (720−480)/2,4 = 100 s: son las 12:00, sale del pico
	if snap := g.Snapshot(); snap.RushHour {
		t.Fatalf("a las 12:00 RushHour = true; se esperaba false (vuelve a normal)")
	}

	tickFor(g, 75*time.Second) // 175 s: (900−480)/2,4 = 175 s: son las 15:00, entra el segundo pico
	snap = g.Snapshot()
	if !snap.RushHour {
		t.Fatalf("a las 15:00 RushHour = %v; se esperaba true (segundo pico)", snap.RushHour)
	}
	peaks := 0
	for _, notice := range snap.Notices {
		if strings.Contains(notice, "Hora pico") {
			peaks++
		}
	}
	if peaks != 2 {
		t.Errorf("avisos de hora pico = %d; se esperaban 2 (uno por cada pico que empezó)", peaks)
	}
}
