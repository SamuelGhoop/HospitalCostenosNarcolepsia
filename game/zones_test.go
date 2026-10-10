package game_test

import (
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// El String() de cada zona es la ubicación que recibe RegisterEpisode y que
// queda en el historial del modelo.
func TestZone_StringIsTheLocationForTheModel(t *testing.T) {
	want := map[game.Zone]string{
		game.Lobby:     "recepción",
		game.Cafeteria: "cafetería",
		game.Hallway1:  "pasillo 1",
		game.Hallway2:  "pasillo 2",
		game.Radiology: "radiología",
	}
	for zone, name := range want {
		if zone.String() != name {
			t.Errorf("Zone(%d).String() = %q; se esperaba %q", int(zone), zone.String(), name)
		}
	}
}
