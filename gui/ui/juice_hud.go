package ui

import "math"

// BLOQUE 5 del pase de juice (§12.10, "HUD vivo"). La plata y las estrellas
// ya se animaban desde el §12.8/12.9 (hud.go: cuentan, destellan, dejan su
// "+$450"); esto les suma el "pop" de la plata y las chispas/fragmentos de
// las estrellas que pidió el bloque 5. starShimmer y starBreak reusan el
// mismo tipo particle que el polvo del desplome y el confeti del alta
// (juice_particles.go), saliendo del centro del grupo de estrellas
// (hudStarsCenter, juice_apply.go).

// starShimmer es la reputación subiendo: unas chispas blancas breves sobre
// las estrellas.
func (fx *juiceFX) starShimmer() {
	origin := hudStarsCenter()
	for i := 0; i < starShimmerCount; i++ {
		angle := fx.rng.Float64() * 2 * math.Pi
		speed := starShimmerSpeed * (0.5 + 0.5*fx.rng.Float64())
		fx.hudSparks = append(fx.hudSparks, &particle{
			pos: origin, vel: vec{math.Cos(angle) * speed, math.Sin(angle) * speed},
			life: starShimmerLifeTicks, total: starShimmerLifeTicks, col: colWhite, size: 2, gravity: 0,
		})
	}
}

// starBreak es la reputación bajando (sin ser un enojo, que ya tiene su
// propia estrella cayendo: angryBurst, juice_payoff.go): unos fragmentos
// grises que caen de las estrellas, como si una se rompiera un poco.
func (fx *juiceFX) starBreak() {
	origin := hudStarsCenter()
	for i := 0; i < starBreakCount; i++ {
		angle := fx.rng.Float64()*2 - 1
		fx.hudSparks = append(fx.hudSparks, &particle{
			pos: origin, vel: vec{angle * starBreakSpeed, -fx.rng.Float64() * starBreakSpeed},
			life: starBreakLifeTicks, total: starBreakLifeTicks, col: colMuted, size: 2, gravity: starBreakGravity,
		})
	}
}
