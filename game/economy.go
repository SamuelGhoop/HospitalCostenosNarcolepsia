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
		lines = append(lines, PayrollLine{Name: s.Name(), IsDoctor: s.isDoctor(), Salary: s.salary})
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
		Report:     g.reportLocked(),
	}

	if g.money < 0 {
		g.loseLocked(Bankrupt)
	}
}
