package ui

import (
	"image"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// TestGameScene_NewDayStartsWithAnEmptyMap: al empezar el día siguiente el
// juego ya sacó a los pacientes y dejó solo al personal inicial
// (game.closeDayLocked). La escena tampoco debe arrastrar nada del día
// anterior: ni pacientes caminando hacia la acera, ni camillas a medias, ni
// a nadie parado en un puesto viejo.
func TestGameScene_NewDayStartsWithAnEmptyMap(t *testing.T) {
	s := newTestGameScene(t)
	id := firstCollapsed(t, s)
	if err := s.g.Dispatch(id, "C-01"); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	playUntil(t, s, 60*30, "arrancara la camilla", func() bool { return len(s.others) > 0 })

	// La foto del día siguiente, como la deja StartNextDay: sin pacientes y
	// con el personal inicial libre.
	day2 := s.snap
	day2.Day++
	day2.Patients = nil
	day2.Staff = nil
	for _, st := range s.snap.Staff[:2] {
		st.Activity, st.PatientID = game.Free, ""
		day2.Staff = append(day2.Staff, st)
	}
	s.snap = day2
	s.startNewDayWorld()
	s.sync()

	if len(s.others) != 0 || s.carry != nil {
		t.Errorf("día 2: quedaron camillas del día anterior (%d en others, carry=%v)", len(s.others), s.carry != nil)
	}
	for who := range s.pos {
		if who != "D-01" && who != "C-01" {
			t.Errorf("día 2: %s sigue en la escena; se esperaba solo el personal inicial", who)
		}
	}
	for _, who := range []string{"D-01", "C-01"} {
		if z := zoneOfFeet(image.Pt(int(s.pos[who].x), int(s.pos[who].y))); z != zoneStaffRoom && z != zoneHallway2 {
			t.Errorf("día 2: %s empieza en %v; se esperaba en la sala del personal (o su desborde)", who, z)
		}
	}
	if s.selected != "" {
		t.Errorf("día 2: quedó elegido %q del día anterior", s.selected)
	}
}
