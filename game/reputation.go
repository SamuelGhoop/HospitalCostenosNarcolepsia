package game

import (
	"fmt"
	"time"
)

// La reputación se guarda como ENTERO en centésimas de estrella (300 = 3 ★),
// no como float64: así sumar 0,1 y restar 0,25 muchas veces no acumula
// errores de redondeo (GAME_DESIGN §5.9).

// ReputationChange es un cambio de reputación con su motivo, para que la
// interfaz muestre por qué bajan (o suben) las estrellas (§12.8, decisión de
// Samuel del 2026-10-16: en la prueba del 16-10 "le bajaban las estrellas
// solas" sin saber por qué). Delta va en centésimas, igual que Reputation en
// el Snapshot; Reason va en español, con el ID del paciente cuando aplica.
type ReputationChange struct {
	Delta  int
	Reason string
}

// changeReputationLocked suma delta (en centésimas) a la reputación, sin
// pasar de 5 ★ ni bajar de 0. Si reason no es "", el cambio queda en
// g.repChanges para el Snapshot; los llamados sin motivo (como los de los
// tests internos) no agregan nada a la cola.
func (g *Game) changeReputationLocked(delta int, reason string) {
	g.reputation = min(max(g.reputation+delta, 0), maxReputation)
	if reason == "" {
		return
	}
	g.repChanges = append(g.repChanges, ReputationChange{Delta: delta, Reason: reason})
	if extra := len(g.repChanges) - maxReputationChanges; extra > 0 {
		g.repChanges = g.repChanges[extra:]
	}
}

// waitPenaltiesDue dice cuántas penalizaciones por espera le tocan a un
// paciente que lleva waited esperando: la primera a los 30 s y otra cada
// 10 s más (firstWaitPenaltyAt, game/config.go).
func waitPenaltiesDue(waited time.Duration) int {
	if waited < firstWaitPenaltyAt {
		return 0
	}
	return 1 + int((waited-firstWaitPenaltyAt)/waitPenaltyEvery)
}

// applyWaitPenaltiesLocked cobra las penalizaciones que le tocan a pt y
// todavía no se le han cobrado en este episodio: la primera −0,5 y las
// siguientes −0,25. threshold es el instante de espera que dispara ESTA
// penalización (firstWaitPenaltyAt, firstWaitPenaltyAt + waitPenaltyEvery…),
// para el motivo: "P-003 lleva 30 s esperando revisión".
func (g *Game) applyWaitPenaltiesLocked(pt *patient) {
	for due := waitPenaltiesDue(pt.waited); pt.penalties < due; pt.penalties++ {
		g.streak = 0 // cae una penalización de espera: se rompe la racha (§5.8)
		threshold := firstWaitPenaltyAt + time.Duration(pt.penalties)*waitPenaltyEvery
		reason := fmt.Sprintf("%s lleva %d s esperando revisión", pt.p.ID(), int(threshold/time.Second))
		if pt.penalties == 0 {
			g.changeReputationLocked(-firstWaitPenalty, reason)
		} else {
			g.changeReputationLocked(-nextWaitPenalty, reason)
		}
	}
}
