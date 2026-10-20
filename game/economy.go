package game

import "time"

// reviewFee es lo que paga un episodio en la revisión del médico (GAME_DESIGN
// §5.8), según cuánto esperó el paciente desde el desplome: reviewFeeMax
// hasta reviewFeeGrace; después baja reviewFeeDrop por cada segundo
// completo, sin bajar de reviewFeeMin.
//
// La división entera de time.Duration cuenta solo segundos completos: a los
// 15,9 s todavía paga lo máximo.
func reviewFee(waited time.Duration) int {
	if waited <= reviewFeeGrace {
		return reviewFeeMax
	}
	late := int((waited - reviewFeeGrace) / time.Second) // segundos completos de más
	return max(reviewFeeMax-reviewFeeDrop*late, reviewFeeMin)
}

// streakMultiplier es el multiplicador del pago de la revisión según la
// racha ANTERIOR (la que traía el jugador antes de esta revisión), en
// centésimas: 100 = ×1, 125 = ×1,25… hasta streakMaxSteps escalones (200 =
// ×2 con los valores de game/config.go). Pedido de Samuel del 2026-10-17
// (§12.8, §5.8): la jugabilidad estaba "un poco aburrida".
func streakMultiplier(streak int) int {
	steps := min(streak, streakMaxSteps)
	return 100 + streakStep*steps
}

// applyStreakMultiplier aplica streakMultiplier a fee, redondeado al peso
// más cercano (no truncado hacia abajo: +50 antes de dividir entre 100 es el
// truco de redondeo con aritmética entera).
func applyStreakMultiplier(fee, streak int) int {
	return (fee*streakMultiplier(streak) + 50) / 100
}

// payPayrollLocked paga la nómina de todo el personal (§5.8) y arma el
// Shift Report diario (DayReport). Lo llama tickLocked una sola vez, en el
// Tick que llega a las 20:00, justo después de sumar los +500 puntos por
// día completado (§5.10): "al cerrar el día, se suman los +500 y luego se
// paga la nómina". Si la plata queda negativa, la partida pierde por
// quiebra.
func (g *Game) payPayrollLocked() {
	var lines []PayrollLine
	total := 0
	for _, s := range g.staff {
		lines = append(lines, PayrollLine{Name: s.Name(), IsDoctor: s.isDoctor(), Salary: s.salary, RoleLabel: s.roleLabel()})
		total += s.salary
	}
	startMoney := g.dayStartMoney
	g.money -= total

	g.lastDayReport = DayReport{
		Day:             g.clock.day,
		ReviewIncome:    g.dayReviewIncome,
		DischargeIncome: g.dayDischargeIncome,
		Payroll:         lines,
		PayrollTotal:    total,
		// Copia: dayHires sigue siendo de Game hasta que StartNextDay lo
		// reinicia; sin copiar, el DayReport compartiría el arreglo.
		Hires:      append([]HireLine(nil), g.dayHires...),
		StartMoney: startMoney,
		EndMoney:   g.money,
		BestStreak: g.dayBestStreak,
		Report:     g.reportLocked(),
	}

	if g.money < 0 {
		g.loseLocked(Bankrupt)
	}
}
