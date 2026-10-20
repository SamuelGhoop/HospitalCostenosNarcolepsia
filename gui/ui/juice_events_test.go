package ui

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

func TestDetectJuiceEvents(t *testing.T) {
	cases := []struct {
		name string
		prev game.Snapshot
		cur  game.Snapshot
		want []juiceEvent
	}{
		{
			name: "sin ningún cambio no pasa nada",
			prev: game.Snapshot{},
			cur:  game.Snapshot{},
			want: nil,
		},
		{
			name: "se revisa a un paciente: cobro",
			prev: game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.AwaitingReview}}},
			cur:  game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.InBed, LastFee: 350}}},
			want: []juiceEvent{{kind: juicePayment, patientID: "P-001", amount: 350}},
		},
		{
			name: "llegar a la cama (AwaitingReview) no es un cobro",
			prev: game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.InHallway}}},
			cur:  game.Snapshot{Patients: []game.GamePatientView{{PatientView: game.PatientView{ID: "P-001"}, Stage: game.AwaitingReview}}},
			want: nil,
		},
		{
			name: "la racha sube: el monto es el multiplicador (como el HUD), no la racha en bruto",
			prev: game.Snapshot{Streak: 1, StreakMultiplier: 100},
			cur:  game.Snapshot{Streak: 2, StreakMultiplier: 125},
			want: []juiceEvent{{kind: juiceStreakUp, amount: 125}},
		},
		{
			name: "la racha se rompe",
			prev: game.Snapshot{Streak: 3},
			cur:  game.Snapshot{Streak: 0},
			want: []juiceEvent{{kind: juiceStreakLost}},
		},
		{
			name: "alguien recibe el alta",
			prev: game.Snapshot{},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: +10, Reason: "P-004 se fue de alta"}}},
			want: []juiceEvent{{kind: juiceDischarge, patientID: "P-004"}},
		},
		{
			name: "alguien se va enojado",
			prev: game.Snapshot{},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-005 se fue enojado"}}},
			want: []juiceEvent{{kind: juiceAngry, patientID: "P-005"}},
		},
		{
			name: "recogido rápido: sube la estrella (con los dos puntos en el motivo)",
			prev: game.Snapshot{},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: +10, Reason: "P-002: lo recogieron rápido"}}},
			want: []juiceEvent{{kind: juiceStarUp, patientID: "P-002"}},
		},
		{
			name: "esperando revisión: baja la estrella",
			prev: game.Snapshot{},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-003 lleva 30 s esperando revisión"}}},
			want: []juiceEvent{{kind: juiceStarDown, patientID: "P-003"}},
		},
		{
			name: "un cambio de reputación que ya estaba en la foto anterior no vuelve a salir",
			prev: game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-003 lleva 30 s esperando revisión"}}},
			cur:  game.Snapshot{ReputationChanges: []game.ReputationChange{{Delta: -50, Reason: "P-003 lleva 30 s esperando revisión"}}},
			want: nil,
		},
		{
			name: "empieza un día nuevo: banner de día",
			prev: game.Snapshot{Day: 1},
			cur:  game.Snapshot{Day: 2},
			want: []juiceEvent{{kind: juiceDayBanner, amount: 2}},
		},
		{
			name: "el mismo día no deja banner",
			prev: game.Snapshot{Day: 2},
			cur:  game.Snapshot{Day: 2},
			want: nil,
		},
		{
			name: "entra una hora pico: banner de hora pico",
			prev: game.Snapshot{RushHour: false},
			cur:  game.Snapshot{RushHour: true},
			want: []juiceEvent{{kind: juiceRushBanner}},
		},
		{
			name: "la hora pico ya estaba activa no vuelve a salir",
			prev: game.Snapshot{RushHour: true},
			cur:  game.Snapshot{RushHour: true},
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectJuiceEvents(c.prev, c.cur)
			if !equalJuiceEvents(got, c.want) {
				t.Errorf("detectJuiceEvents = %+v; se esperaba %+v", got, c.want)
			}
		})
	}
}

func equalJuiceEvents(a, b []juiceEvent) bool {
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
