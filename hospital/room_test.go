package hospital_test

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

func TestRoomState_String(t *testing.T) {
	tests := []struct {
		state hospital.RoomState
		want  string
	}{
		{hospital.Available, "disponible"},
		{hospital.Occupied, "ocupada"},
		{hospital.RoomState(5), "RoomState(5)"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("RoomState(%d).String() = %q; se esperaba %q", int(tt.state), got, tt.want)
		}
	}
}
