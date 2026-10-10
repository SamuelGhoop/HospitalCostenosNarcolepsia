package game

import "time"

// La reputación se guarda como ENTERO en centésimas de estrella (300 = 3 ★),
// no como float64: así sumar 0,1 y restar 0,25 muchas veces no acumula
// errores de redondeo (GAME_DESIGN §5.9).

// changeReputationLocked suma delta (en centésimas) a la reputación, sin
// pasar de 5 ★ ni bajar de 0.
func (g *Game) changeReputationLocked(delta int) {
	g.reputation = min(max(g.reputation+delta, 0), maxReputation)
}

// waitPenaltiesDue dice cuántas penalizaciones por espera le tocan a un
// paciente que lleva waited esperando: la primera a los 20 s y otra cada
// 10 s más.
func waitPenaltiesDue(waited time.Duration) int {
	if waited < firstWaitPenaltyAt {
		return 0
	}
	return 1 + int((waited-firstWaitPenaltyAt)/waitPenaltyEvery)
}

// applyWaitPenaltiesLocked cobra las penalizaciones que le tocan a pt y
// todavía no se le han cobrado en este episodio: la primera −0,5 y las
// siguientes −0,25.
func (g *Game) applyWaitPenaltiesLocked(pt *patient) {
	for due := waitPenaltiesDue(pt.waited); pt.penalties < due; pt.penalties++ {
		if pt.penalties == 0 {
			g.changeReputationLocked(-firstWaitPenalty)
		} else {
			g.changeReputationLocked(-nextWaitPenalty)
		}
	}
}
