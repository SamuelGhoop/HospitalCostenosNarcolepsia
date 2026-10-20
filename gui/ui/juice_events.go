package ui

import (
	"strings"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Pase de juice (§12.10): detectJuiceEvents compara dos Snapshot y dice qué
// pasó, igual que gameSoundEvents (sound_events.go) pero para la parte
// visual del BLOQUE 4 ("cobro/racha/alta/enojo") y, con los eventos de
// estrella, el BLOQUE 5 (HUD). Es pura: no toca Ebitengine ni game.Game, así
// que se prueba con una tabla de casos (juice_events_test.go).

// juiceKind distingue qué pasó.
type juiceKind int

const (
	juicePayment    juiceKind = iota // se cobró una revisión (bloque 4): amount es el pago
	juiceStreakUp                    // la racha subió (bloque 4)
	juiceStreakLost                  // la racha se rompió (bloque 4)
	juiceDischarge                   // alguien recibió el alta (bloque 4)
	juiceAngry                       // alguien se fue enojado (bloque 4)
	juiceStarUp                      // la reputación subió, sin ser alta (bloque 5)
	juiceStarDown                    // la reputación bajó, sin ser un enojo (bloque 5)
)

// juiceEvent trae lo que hace falta para dibujarlo: a quién (patientID, si
// aplica) y cuánto (amount: el pago o la racha nueva, según kind).
type juiceEvent struct {
	kind      juiceKind
	patientID string
	amount    int
}

// detectJuiceEvents: prev es la foto del tick anterior, cur la de ahora.
func detectJuiceEvents(prev, cur game.Snapshot) []juiceEvent {
	var out []juiceEvent

	prevStage := map[string]game.Stage{}
	for _, p := range prev.Patients {
		prevStage[p.ID] = p.Stage
	}
	for _, p := range cur.Patients {
		before, had := prevStage[p.ID]
		if p.Stage == game.InBed && had && before == game.AwaitingReview {
			out = append(out, juiceEvent{kind: juicePayment, patientID: p.ID, amount: p.LastFee})
		}
	}

	switch {
	case cur.Streak > prev.Streak:
		// amount es el multiplicador (centésimas, 100 = ×1), el mismo que ya
		// muestra la barra ("RACHA ×1,5"): así el banner grande dice lo mismo
		// que el HUD, no la racha en bruto.
		out = append(out, juiceEvent{kind: juiceStreakUp, amount: cur.StreakMultiplier})
	case prev.Streak > 0 && cur.Streak == 0:
		out = append(out, juiceEvent{kind: juiceStreakLost})
	}

	seen := map[game.ReputationChange]bool{}
	for _, c := range prev.ReputationChanges {
		seen[c] = true
	}
	for _, c := range cur.ReputationChanges {
		if seen[c] {
			continue
		}
		id := reputationChangeSubject(c.Reason)
		switch {
		case strings.Contains(c.Reason, "se fue de alta"):
			out = append(out, juiceEvent{kind: juiceDischarge, patientID: id})
		case strings.Contains(c.Reason, "se fue enojado"):
			out = append(out, juiceEvent{kind: juiceAngry, patientID: id})
		case c.Delta > 0:
			out = append(out, juiceEvent{kind: juiceStarUp, patientID: id})
		case c.Delta < 0:
			out = append(out, juiceEvent{kind: juiceStarDown, patientID: id})
		}
	}
	return out
}

// reputationChangeSubject es el ID del paciente al principio del motivo
// (game/reputation.go siempre lo pone primero: "P-003 lleva 30 s
// esperando revisión" o "P-002: lo recogieron rápido", con los dos puntos
// pegados al ID en ese caso).
func reputationChangeSubject(reason string) string {
	id, _, _ := strings.Cut(reason, " ")
	return strings.TrimSuffix(id, ":")
}
