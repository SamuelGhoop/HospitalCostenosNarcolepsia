package ui

// applyJuiceEvent traduce un juiceEvent (detectJuiceEvents, juice_events.go)
// a lo que corresponde en s.fx: dónde en pantalla (s.pos, s.figureBox) y
// qué sonido.
func (s *GameScene) applyJuiceEvent(e juiceEvent) {
	switch e.kind {
	case juicePayment:
		if at, ok := s.overHead(e.patientID); ok {
			s.fx.payment(at, e.amount)
		}
	case juiceStreakUp:
		s.fx.streakUp(e.amount)
	case juiceStreakLost:
		s.fx.streakLostBanner()
	case juiceDischarge:
		if at, ok := s.overHead(e.patientID); ok {
			s.fx.dischargeBurst(at)
		}
		playSound(sfxDischarge)
	case juiceAngry:
		if at, ok := s.overHead(e.patientID); ok {
			s.fx.angryBurst(at, hudStarsCenter())
		}
		playSound(sfxAngry)
	case juiceStarUp:
		s.fx.starShimmer()
	case juiceStarDown:
		s.fx.starBreak()
	case juiceDayBanner:
		s.fx.dayBanner = newSweepBanner(hudDayLabel(e.amount), colHUDBlue, dayBannerSize,
			dayBannerEnterTicks, dayBannerHoldTicks, dayBannerExitTicks)
		// Bloque 7 (§12.10): "banner de día". Reusa "open" (el barrido hacia
		// arriba de abrir un portapapeles, §10.11): no hay un sonido propio
		// de "empieza el día" todavía y la idea de "algo se abre" le cae
		// bien a un día nuevo, sin inventar un efecto más.
		playSound(sfxOpen)
	case juiceRushBanner:
		s.fx.rushBanner = newSweepBanner("¡HORA PICO!", colOrange, rushBannerSize,
			rushBannerEnterTicks, rushBannerHoldTicks, rushBannerExitTicks)
		// El sonido de la hora pico (sfxRush) ya suena solo: gameSoundEvents
		// (sound_events.go) lo dispara con la misma condición exacta
		// (!prev.RushHour && cur.RushHour, §10.11, de antes de este pase de
		// juice). No hace falta repetirlo acá.
	}
}

// overHead es un punto sobre la cabeza de id (paciente o personal), para
// anclar ahí un floatText o una ráfaga de partículas: un poco arriba de su
// caja actual (de pie, acostado o en la cama: figureBox ya sabe cuál toca).
func (s *GameScene) overHead(id string) (vec, bool) {
	f, ok := s.figure(id)
	if !ok {
		return vec{}, false
	}
	box := s.figureBox(f)
	return vec{float64(box.Min.X+box.Max.X) / 2, float64(box.Min.Y - 4)}, true
}

// hudStarsCenter es el centro del grupo de estrellas de la barra (de dónde
// se desprende la que cae al enojarse alguien, BLOQUE 4): no de una
// estrella en particular, para no complicar cuál es "la última llena".
func hudStarsCenter() vec {
	r := hudStarsRect()
	return vec{float64(r.Min.X+r.Max.X) / 2, float64(r.Min.Y+r.Max.Y) / 2}
}
