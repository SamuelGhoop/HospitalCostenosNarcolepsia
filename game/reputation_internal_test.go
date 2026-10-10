package game

import "testing"

// §5.9: la reputación va de 0 a 5 ★ (0 a 500 centésimas).
func TestReputation_StaysBetween0And5Stars(t *testing.T) {
	g := newInternalGame(t)

	g.reputation = 495
	g.changeReputationLocked(+10)
	if g.reputation != 500 {
		t.Errorf("495 + 10 = %d; el tope es 500", g.reputation)
	}

	g.reputation = 10
	g.changeReputationLocked(-25)
	if g.reputation != 0 {
		t.Errorf("10 − 25 = %d; no baja de 0", g.reputation)
	}
}
