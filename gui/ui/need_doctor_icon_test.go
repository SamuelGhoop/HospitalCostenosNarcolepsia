package ui

import (
	"image"
	"testing"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// El ícono "falta el médico" (GAME_DESIGN §10.3 y §12.8): hoja de 38×18 con
// 2 cuadros de 19×18, 0,4 s cada uno (24 ticks a 60/s).
func TestNeedDoctorIcon_IsEmbeddedWithTheRightSize(t *testing.T) {
	img, err := decodePNG(assets.Sprites, needDoctorIconPath)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 2*needDoctorFrameWidth || b.Dy() != needDoctorFrameHeight {
		t.Errorf("tamaño %dx%d; se esperaba %dx%d", b.Dx(), b.Dy(), 2*needDoctorFrameWidth, needDoctorFrameHeight)
	}
}

// needDoctorFrame alterna 0 → 1 → 0 cada 24 ticks (0,4 s a 60/s), en bucle.
func TestNeedDoctorIcon_FrameLoopsEveryFourTenthsOfASecond(t *testing.T) {
	want := map[int]int{0: 0, 23: 0, 24: 1, 47: 1, 48: 0}
	for ticks, w := range want {
		if got := needDoctorFrame(ticks); got != w {
			t.Errorf("ticks %d: cuadro %d; se esperaba %d", ticks, got, w)
		}
	}
}

// needDoctorIconAt da la esquina de arriba a la izquierda del ícono en
// pantalla (1280×720) para las habitaciones 101 a 103 (GAME_DESIGN §12.8,
// ICONS_ASSETS.md), y nada para una habitación sin ese ícono.
func TestNeedDoctorIcon_PositionPerRoom(t *testing.T) {
	cases := []struct {
		room int
		at   image.Point
		ok   bool
	}{
		{101, image.Pt(184, 104), true},
		{102, image.Pt(457, 104), true},
		{103, image.Pt(730, 104), true},
		{104, image.Point{}, false},
		{0, image.Point{}, false},
	}
	for _, c := range cases {
		at, ok := needDoctorIconAt(c.room)
		if ok != c.ok || (ok && at != c.at) {
			t.Errorf("needDoctorIconAt(%d) = (%v, %v); se esperaba (%v, %v)", c.room, at, ok, c.at, c.ok)
		}
	}
}

// El ícono se ve quieto (siempre visible) mientras el paciente lleva menos
// de needDoctorBlinkAfter esperando, y parpadea desde ahí: el umbral es de
// interfaz (10 s antes del −0,5 ★ que impone firstWaitPenaltyAt = 30 s en
// game/config.go, §5.9), para que el jugador lo vea venir.
func TestNeedDoctorIcon_BlinksFromItsThreshold(t *testing.T) {
	if needDoctorVisible(needDoctorBlinkAfter-time.Second, 29) != true {
		t.Error("antes del umbral debe verse siempre, sin importar el tick")
	}
	if needDoctorVisible(needDoctorBlinkAfter-time.Second, 31) != true {
		t.Error("antes del umbral debe verse siempre, sin importar el tick")
	}
	cases := []struct {
		ticks int
		want  bool
	}{
		{0, true}, {29, true}, {30, false}, {59, false}, {60, true},
	}
	for _, c := range cases {
		if got := needDoctorVisible(needDoctorBlinkAfter, c.ticks); got != c.want {
			t.Errorf("tick %d: visible = %v; se esperaba %v", c.ticks, got, c.want)
		}
	}
}

// drawNeedDoctorIcons dibuja uno solo por paciente AwaitingReview con cama
// asignada, y nada para los demás estados o sin habitación.
func TestNeedDoctorIcon_OnlyForPatientsAwaitingReviewInARoom(t *testing.T) {
	patients := []game.GamePatientView{
		{PatientView: game.PatientView{ID: "P-001", Room: 101}, Stage: game.AwaitingReview},
		{PatientView: game.PatientView{ID: "P-002", Room: 102}, Stage: game.InBed},
		{PatientView: game.PatientView{ID: "P-003", Room: 0}, Stage: game.AwaitingReview},
		{PatientView: game.PatientView{ID: "P-004", Room: 103}, Stage: game.Collapsed},
	}
	got := needDoctorPatients(patients)
	if len(got) != 1 || got[0].ID != "P-001" {
		t.Errorf("needDoctorPatients = %v; se esperaba solo P-001", got)
	}
}
