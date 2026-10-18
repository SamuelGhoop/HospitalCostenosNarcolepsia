package ui

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// gameSoundEvents compara dos fotos y dice qué efectos tocan (sound.go,
// sound_events.go): es pura, así que se prueba con una tabla de casos sin
// tocar Ebitengine ni el sonido de verdad.
func TestGameSoundEvents(t *testing.T) {
	cases := []struct {
		name string
		prev game.Snapshot
		cur  game.Snapshot
		want []string
	}{
		{
			name: "sin ningún cambio no suena nada",
			prev: game.Snapshot{Money: 1000},
			cur:  game.Snapshot{Money: 1000},
			want: nil,
		},
		{
			name: "la plata sube: coin",
			prev: game.Snapshot{Money: 1000},
			cur:  game.Snapshot{Money: 1450},
			want: []string{sfxCoin},
		},
		{
			name: "la plata baja (nómina) no toca coin",
			prev: game.Snapshot{Money: 1000},
			cur:  game.Snapshot{Money: 400},
			want: nil,
		},
		{
			name: "la racha se pierde: streakLost",
			prev: game.Snapshot{Streak: 3},
			cur:  game.Snapshot{Streak: 0},
			want: []string{sfxStreakLost},
		},
		{
			name: "la racha sube no deja su propio efecto (va con el coin)",
			prev: game.Snapshot{Streak: 1},
			cur:  game.Snapshot{Streak: 2},
			want: nil,
		},
		{
			name: "un cambio de reputación nuevo, hacia abajo: starDown",
			prev: game.Snapshot{},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-001 lleva 30 s esperando"}}},
			want: []string{sfxStarDown},
		},
		{
			name: "un cambio de reputación nuevo, hacia arriba: starUp",
			prev: game.Snapshot{},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: +10, Reason: "P-001: lo recogieron rápido"}}},
			want: []string{sfxStarUp},
		},
		{
			name: "un cambio de reputación que ya estaba en la foto anterior no vuelve a sonar",
			prev: game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-001 lleva 30 s esperando"}}},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-001 lleva 30 s esperando"}}},
			want: nil,
		},
		{
			name: "la hora pico empieza: rush",
			prev: game.Snapshot{RushHour: false},
			cur:  game.Snapshot{RushHour: true},
			want: []string{sfxRush},
		},
		{
			name: "la hora pico ya estaba activa no vuelve a sonar",
			prev: game.Snapshot{RushHour: true},
			cur:  game.Snapshot{RushHour: true},
			want: nil,
		},
		{
			name: "un paciente se desploma: collapse",
			prev: game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.Drowsy}}},
			cur:  game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.Collapsed}}},
			want: []string{sfxCollapse},
		},
		{
			name: "un paciente que ya estaba desplomado no vuelve a sonar",
			prev: game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.Collapsed}}},
			cur:  game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.Collapsed}}},
			want: nil,
		},
		{
			name: "un paciente cae dormido en una cama: snore",
			prev: game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.InHallway}}},
			cur:  game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.AwaitingReview}}},
			want: []string{sfxSnore},
		},
		{
			name: "termina el día: dayEnd",
			prev: game.Snapshot{DayOver: false},
			cur:  game.Snapshot{DayOver: true},
			want: []string{sfxDayEnd},
		},
		{
			name: "Game Over: gameOver",
			prev: game.Snapshot{GameOver: game.NotOver},
			cur:  game.Snapshot{GameOver: game.LostLicense},
			want: []string{sfxGameOver},
		},
		{
			name: "varios a la vez: todos suenan",
			prev: game.Snapshot{Money: 1000, DayOver: false},
			cur:  game.Snapshot{Money: 1200, DayOver: true},
			want: []string{sfxCoin, sfxDayEnd},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := gameSoundEvents(c.prev, c.cur)
			if !equalStrings(got, c.want) {
				t.Errorf("gameSoundEvents = %v; se esperaba %v", got, c.want)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// soundTracker no deja sonar nada con la primera foto (no hay "antes" con
// qué comparar) y sí desde la segunda.
func TestSoundTracker_FirstSnapshotNeverSounds(t *testing.T) {
	var tr soundTracker
	tr.update(game.Snapshot{Money: 1000}) // la primera: solo se guarda
	if tr.prev == nil || tr.prev.Money != 1000 {
		t.Fatalf("no guardó la primera foto")
	}
	// La segunda sí compara (acá no hay reproductor: solo se prueba que no
	// se cae y que gameSoundEvents es lo que decide).
	tr.update(game.Snapshot{Money: 1450})
	if tr.prev.Money != 1450 {
		t.Errorf("la foto guardada quedó en %d; se esperaba 1450", tr.prev.Money)
	}
}
