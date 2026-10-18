package ui

import "github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"

// gameSoundEvents compara dos fotos del modo juego y dice qué efectos
// tienen que sonar (GAME_DESIGN §10.11, pedido de Samuel el 2026-10-17).
// No hay ninguna regla de juego acá: solo se comparan los valores de las
// dos Snapshot, igual que ya hace hud.update para animar la barra. Es una
// función pura, así se prueba con una tabla de casos sin tocar Ebitengine
// ni un dispositivo de audio.
//
// prev es la foto del tick anterior; cur, la de ahora. El llamador
// (GameScene.update) se encarga de no comparar antes de tener una
// primera foto de verdad.
func gameSoundEvents(prev, cur game.Snapshot) []string {
	var events []string

	if cur.Money > prev.Money {
		events = append(events, sfxCoin)
	}
	if prev.Streak > 0 && cur.Streak == 0 {
		events = append(events, sfxStreakLost)
	}

	seen := map[game.ReputationChange]bool{}
	for _, c := range prev.ReputationChanges {
		seen[c] = true
	}
	for _, c := range cur.ReputationChanges {
		if seen[c] {
			continue
		}
		if c.Delta < 0 {
			events = append(events, sfxStarDown)
		} else if c.Delta > 0 {
			events = append(events, sfxStarUp)
		}
	}

	if !prev.RushHour && cur.RushHour {
		events = append(events, sfxRush)
	}

	prevStage := map[string]game.Stage{}
	for _, p := range prev.Patients {
		prevStage[p.ID] = p.Stage
	}
	for _, p := range cur.Patients {
		before, had := prevStage[p.ID]
		switch {
		case p.Stage == game.Collapsed && (!had || before != game.Collapsed):
			events = append(events, sfxCollapse)
		case p.Stage == game.AwaitingReview && had && before != game.AwaitingReview:
			events = append(events, sfxSnore) // se acaba de dormir en una cama (§10.11, opcional)
		}
	}

	if !prev.DayOver && cur.DayOver {
		events = append(events, sfxDayEnd)
	}
	if prev.GameOver == game.NotOver && cur.GameOver != game.NotOver {
		events = append(events, sfxGameOver)
	}

	return events
}

// soundTracker recuerda la última foto para poder llamar gameSoundEvents
// en el siguiente tick, y reproduce lo que salga (playSound y, para
// "coin", playCoin con la racha de la foto nueva). nil hasta la primera
// foto: la primera nunca deja nada de sonido (no hay "antes" con qué
// comparar), igual que hud.started.
type soundTracker struct {
	prev    *game.Snapshot
	started bool
}

// update avanza un tick con la foto actual: toca los efectos que
// correspondan (salvo la primera vez, que solo guarda la foto) y la deja
// lista para la próxima comparación.
func (t *soundTracker) update(cur game.Snapshot) {
	if !t.started {
		t.started = true
		t.prev = &cur
		return
	}
	for _, name := range gameSoundEvents(*t.prev, cur) {
		if name == sfxCoin {
			playCoin(cur.Streak)
			continue
		}
		playSound(name)
	}
	t.prev = &cur
}
