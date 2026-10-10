package game

import (
	"errors"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// §5.10: con la reputación en 0 pierde la licencia. La partida queda
// congelada: Tick no hace nada y las acciones devuelven ErrGameOver.
func TestDefeat_LicenseLostAtZeroReputationFreezesTheGame(t *testing.T) {
	g := newInternalGame(t)
	g.reputation = 50
	id, err := g.CollapseForTest("Yeimy Padilla", hospital.Severe, Cafeteria)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ { // 20 s: la primera penalización (−50) la deja en 0
		g.Tick(tickInterval)
	}

	snap := g.Snapshot()
	if snap.Reputation != 0 || snap.GameOver != LostLicense || snap.GameOver.String() != "¡PERDIÓ LA LICENCIA!" {
		t.Fatalf("reputación %d, GameOver = %v; se esperaba 0 y ¡PERDIÓ LA LICENCIA!", snap.Reputation, snap.GameOver)
	}

	g.Tick(tickInterval)
	if g.Snapshot().Progress != snap.Progress { // Progress y no Clock: un tick no alcanza a cambiar el minuto
		t.Error("con la partida perdida, el reloj siguió avanzando")
	}
	if err := g.Dispatch(id, "C-01"); !errors.Is(err, ErrGameOver) {
		t.Errorf("Dispatch con la partida perdida: err = %v; se esperaba ErrGameOver", err)
	}
}
