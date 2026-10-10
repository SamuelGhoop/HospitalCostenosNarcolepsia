package game

import "fmt"

// GameOver dice si la partida terminó y por qué (GAME_DESIGN §5.10).
type GameOver int

const (
	NotOver           GameOver = iota // la partida sigue
	LostLicense                       // la reputación llegó a 0
	HospitalCollapsed                 // 5 o más pacientes desplomados o en el pasillo a la vez
	// La quiebra (no alcanza para la nómina) llega en F4.
)

// String es el letrero que muestra la pantalla de Game Over.
func (o GameOver) String() string {
	switch o {
	case NotOver:
		return ""
	case LostLicense:
		return "¡PERDIÓ LA LICENCIA!"
	case HospitalCollapsed:
		return "¡HOSPITAL COLAPSADO!"
	default:
		return fmt.Sprintf("GameOver(%d)", int(o))
	}
}

// checkDefeatLocked se llama al final de cada Tick, cuando ya se aplicó
// todo lo de ese paso. Para el colapso cuentan solo los desplomados y los
// del pasillo, no los que esperan revisión (§5.9).
func (g *Game) checkDefeatLocked() {
	if g.reputation == 0 {
		g.loseLocked(LostLicense)
		return
	}
	down := 0
	for _, pt := range g.patients {
		if pt.stage == Collapsed || pt.stage == InHallway {
			down++
		}
	}
	if down >= collapseLimit {
		g.loseLocked(HospitalCollapsed)
	}
}

// finalScoreLocked es el puntaje final (§5.10): el acumulado + plata ÷ 10 +
// estrellas × 200. La reputación está en centésimas de estrella, así que
// las estrellas × 200 son reputation × 200 / 100 (aritmética entera).
func (g *Game) finalScoreLocked() int {
	return g.score + g.money/finalMoneyDivisor + g.reputation*finalStarPoints/100
}

// loseLocked termina la partida. Desde aquí Tick no hace nada y las
// acciones del jugador devuelven ErrGameOver.
func (g *Game) loseLocked(why GameOver) {
	g.over = why
	g.noticeLocked(why.String())
}
