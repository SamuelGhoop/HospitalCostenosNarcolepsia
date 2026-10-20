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
	case juiceRushBanner:
		s.fx.rushBanner = newSweepBanner("¡HORA PICO!", colOrange, rushBannerSize,
			rushBannerEnterTicks, rushBannerHoldTicks, rushBannerExitTicks)
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
