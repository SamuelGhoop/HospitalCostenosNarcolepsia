package ui

import (
	"image/color"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

func TestAutoAdvance_FiresEveryThreeSecondsOnlyWhenOn(t *testing.T) {
	var a autoAdvance

	for i := 0; i < 1000; i++ {
		if a.tick() {
			t.Fatal("apagado no debe avanzar nunca")
		}
	}

	a.toggle()
	for i := 1; i < autoAdvanceTicks; i++ {
		if a.tick() {
			t.Fatalf("avanzó en el tick %d; debía esperar %d", i, autoAdvanceTicks)
		}
	}
	if !a.tick() {
		t.Errorf("no avanzó en el tick %d (3 s a 60 ticks por segundo)", autoAdvanceTicks)
	}
}

func TestAutoAdvance_ResetRestartsTheCount(t *testing.T) {
	var a autoAdvance
	a.toggle()
	for i := 0; i < autoAdvanceTicks-1; i++ {
		a.tick()
	}

	a.reset() // el usuario avanzó a mano: se cuentan otros 3 s completos

	if a.tick() {
		t.Error("después de reset no debe avanzar en el siguiente tick")
	}
}

func TestMoveToward_ReachesTheTargetWithoutOvershooting(t *testing.T) {
	p := vec{0, 0}
	target := vec{30, 40} // distancia 50

	p = moveToward(p, target, 10)
	if p != (vec{6, 8}) {
		t.Errorf("tras un paso de 10 px: %v; se esperaba {6 8}", p)
	}
	for i := 0; i < 10; i++ {
		p = moveToward(p, target, 10)
	}
	if p != target {
		t.Errorf("debía quedar exactamente en %v y quedó en %v", target, p)
	}
}

func TestLevelColor_FollowsTheMockup(t *testing.T) {
	tests := []struct {
		level hospital.NarcolepsyLevel
		want  color.RGBA
		bolts int
	}{
		{hospital.Mild, colYellow, 1},
		{hospital.Moderate, colOrange, 2},
		{hospital.Severe, colRed, 3},
	}
	for _, tt := range tests {
		c, n := levelStyle(tt.level)
		if c != tt.want || n != tt.bolts {
			t.Errorf("levelStyle(%v) = (%v, %d); se esperaba (%v, %d)", tt.level, c, n, tt.want, tt.bolts)
		}
	}
}

func TestBubbleFor_DependsOnWhereThePatientSleeps(t *testing.T) {
	if text, bg, ok := bubbleFor(hospital.AsleepInHallway); !ok || text != "Zzz!" || bg != colBubbleHallway {
		t.Errorf("pasillo = (%q, %v, %v); se esperaba la burbuja roja \"Zzz!\"", text, bg, ok)
	}
	if text, bg, ok := bubbleFor(hospital.AsleepInBed); !ok || text != "Zzz" || bg != colBubbleBed {
		t.Errorf("cama = (%q, %v, %v); se esperaba la burbuja azul \"Zzz\"", text, bg, ok)
	}
	if _, _, ok := bubbleFor(hospital.Awake); ok {
		t.Error("un paciente despierto no lleva burbuja")
	}
}
