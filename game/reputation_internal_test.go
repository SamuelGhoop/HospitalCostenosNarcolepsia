package game

import "testing"

// §5.9: la reputación va de 0 a 5 ★ (0 a 500 centésimas).
func TestReputation_StaysBetween0And5Stars(t *testing.T) {
	g := newInternalGame(t)

	g.reputation = 495
	g.changeReputationLocked(+10, "")
	if g.reputation != 500 {
		t.Errorf("495 + 10 = %d; el tope es 500", g.reputation)
	}

	g.reputation = 10
	g.changeReputationLocked(-25, "")
	if g.reputation != 0 {
		t.Errorf("10 − 25 = %d; no baja de 0", g.reputation)
	}
}

// §12.8 (2026-10-16): cada cambio de reputación con motivo (reason != "")
// queda en g.repChanges para que la interfaz muestre por qué bajan o suben
// las estrellas; sin motivo (el helper interno de arriba, por ejemplo) no
// agrega nada.
func TestReputation_ChangeWithReasonIsRecorded(t *testing.T) {
	g := newInternalGame(t)

	g.changeReputationLocked(-50, "P-003 lleva 30 s esperando revisión")
	g.changeReputationLocked(-25, "")
	g.changeReputationLocked(+10, "P-002: lo recogieron rápido")

	want := []ReputationChange{
		{Delta: -50, Reason: "P-003 lleva 30 s esperando revisión"},
		{Delta: +10, Reason: "P-002: lo recogieron rápido"},
	}
	if len(g.repChanges) != len(want) {
		t.Fatalf("repChanges = %+v; se esperaban %+v", g.repChanges, want)
	}
	for i, c := range want {
		if g.repChanges[i] != c {
			t.Errorf("repChanges[%d] = %+v; se esperaba %+v", i, g.repChanges[i], c)
		}
	}
}

// Igual que maxNotices: solo se guardan los maxReputationChanges más
// recientes.
func TestReputation_ChangesQueueDiscardsTheOldest(t *testing.T) {
	g := newInternalGame(t)
	for i := 0; i < maxReputationChanges+2; i++ {
		g.changeReputationLocked(-1, "cambio")
	}
	if len(g.repChanges) != maxReputationChanges {
		t.Fatalf("hay %d cambios guardados; se esperaban %d", len(g.repChanges), maxReputationChanges)
	}
}
