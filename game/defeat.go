package game

import "fmt"

// GameOver dice si la partida terminó y por qué (GAME_DESIGN §5.10).
type GameOver int

const (
	NotOver           GameOver = iota // la partida sigue
	LostLicense                       // la reputación llegó a 0
	HospitalCollapsed                 // 5 o más pacientes desplomados o en el pasillo a la vez
	Bankrupt                          // la nómina de las 20:00 dejó la plata en negativo (§5.8, F4)
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
	case Bankrupt:
		return "¡QUIEBRA!"
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

// finalScoreBreakdownLocked son las dos partes que se suman al acumulado
// para el puntaje final (§5.10, §9 "PLATA ÷ 10" y "ESTRELLAS × 200" del
// Game Over): plata ÷ 10 y estrellas × 200. La reputación está en
// centésimas de estrella, así que las estrellas × 200 son
// reputation × 200 / 100 (aritmética entera). Vive en su propia función
// (Snapshot la suma a g.score para FinalScore) para que la interfaz pueda
// mostrar cada parte por separado sin repetir las constantes de
// game/config.go afuera del paquete.
func (g *Game) finalScoreBreakdownLocked() (moneyScore, starScore int) {
	return g.money / finalMoneyDivisor, g.reputation * finalStarPoints / 100
}

// loseLocked termina la partida. Desde aquí Tick no hace nada y las
// acciones del jugador devuelven ErrGameOver.
func (g *Game) loseLocked(why GameOver) {
	g.over = why
	g.noticeLocked(why.String())
}
