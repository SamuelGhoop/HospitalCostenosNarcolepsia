package ui

import (
	"image"
	"math"
)

// Pase de "juice" del modo juego (pedido de Samuel, 2026-10-19;
// GAME_DESIGN §12.10). Todo lo de aquí es puramente visual o sonoro: nunca
// cambia una regla del juego ni toca Game.Tick. Intensidad elegida por
// Samuel: FUERTE/ARCADE (shake de 6-8 px que decae en ~0,3 s, hit-stop de
// ~6 cuadros, bastantes partículas). Las constantes de intensidad viven
// TODAS en este archivo para poder ajustarlas sin perseguirlas por el resto
// de gui/ui.

// ─── BLOQUE 1: clics más fáciles (hitbox e imán) ───────────────────────

// hitboxGrowFactor: la hitbox de cada personaje (figureBox) crece esto
// alrededor de su centro para que sea más fácil darle clic (+50 %).
const hitboxGrowFactor = 1.5

// clickMagnetRadius: si el clic (o el mouse, para el hover) no cae en la
// hitbox de nadie, se elige al personaje clicable más cercano por
// distancia al centro de su caja, si está a esto o menos (en píxeles de
// pantalla, 1280×720).
const clickMagnetRadius = 40.0

// growRect agranda r por factor alrededor de su propio centro (>1 lo hace
// más grande, manteniendo el centro). Redondea al píxel entero: la interfaz
// es pixel art, nada de hitboxes con decimales que no se puedan dibujar.
func growRect(r image.Rectangle, factor float64) image.Rectangle {
	cx, cy := float64(r.Min.X+r.Max.X)/2, float64(r.Min.Y+r.Max.Y)/2
	hw, hh := float64(r.Dx())/2*factor, float64(r.Dy())/2*factor
	return image.Rect(
		int(math.Round(cx-hw)), int(math.Round(cy-hh)),
		int(math.Round(cx+hw)), int(math.Round(cy+hh)),
	)
}

// boxCenter es el centro de r, en punto flotante (para medir distancias).
func boxCenter(r image.Rectangle) (x, y float64) {
	return float64(r.Min.X+r.Max.X) / 2, float64(r.Min.Y+r.Max.Y) / 2
}

// ─── BLOQUE 2: clics contundentes ──────────────────────────────────────

// Squash & stretch del personaje clicado (figurePunch, juice_fx.go):
// squashTicks ~120 ms a 60 ticks/s, squashFlashTicks "2 cuadros" de destello
// (blanco si el clic sirvió, rojo si fue inválido) y squashAmount cuánto se
// aplasta y estira (18 %, FUERTE/ARCADE).
const (
	squashTicks      = 7
	squashFlashTicks = 2
	squashAmount     = 0.18
)

// Línea punteada del despacho exitoso (dashLine, juice_fx.go): se desvanece
// en dispatchLineTicks (0,6 s) y sus guiones avanzan dashSpeed px por tick.
const (
	dispatchLineTicks = 36 // 0,6 s a 60 ticks/s
	dashLen           = 6.0
	dashGap           = 4.0
	dashWidth         = 2.0
	dashSpeed         = 2.0
)

// Shake de pantalla (screenShake, juice_fx.go): decae a 0 en shakeDecayTicks
// (~0,3 s, FUERTE/ARCADE). Las magnitudes de cada evento, de más chico a más
// grande: shakeCollapse (desplome, bloque 3), shakeInvalidClick (clic
// inválido, bloque 2) y shakeAngry (se va enojado, bloque 4, "shake fuerte").
const (
	shakeDecayTicks   = 18 // 0,3 s a 60 ticks/s
	shakeCollapse     = 3.5
	shakeInvalidClick = 7.0
	shakeAngry        = 8.0
)

// Hit-stop (hitStop, juice_fx.go): ~6 cuadros (100 ms) con la interfaz
// congelada; nunca el reloj del juego (Game.Tick sigue en su propia
// goroutine). Se usa en el cobro de una revisión (bloque 4).
const hitStopTicks = 6

// ─── BLOQUE 4: cobro, racha, alta y enojo ──────────────────────────────

// Cobro (juice_payoff.go): hit-stop (hitStopTicks) + flash blanco de
// pantalla breve y de alpha bajo + "+$350" verde que sube sobre el
// paciente y se desvanece.
const (
	paymentFlashTicks = hitStopTicks // el flash dura lo mismo que el hit-stop
	paymentFlashAlpha = 90           // de 255: alpha bajo
	paymentPopupTicks = 60           // 1 s: sube y se desvanece
	paymentPopupRise  = 28           // px que sube
)

// Racha: el banner grande "¡RACHA ×N!" (N es el multiplicador, como ya
// muestra la barra) con rebote elástico y chispas; "RACHA PERDIDA" cae y se
// desvanece.
const (
	streakBannerBaseSize  = 32
	streakBannerPopTicks  = 14 // 0,23 s: el rebote elástico
	streakBannerTicks     = 54 // ~0,9 s en total
	streakBannerFadeTicks = 14

	streakSparkCount     = 8
	streakSparkLifeTicks = 24
	streakSparkSpeed     = 2.2

	streakLostBannerTicks = 50 // cae y se desvanece
	streakLostFall        = 36 // px que cae (riseTotal negativo: ver floatText)
)

// streakBannerPos: centro-arriba del mapa, debajo de la barra.
var streakBannerPos = vec{float64(ScreenWidth) / 2, 140}

// Alta: corazones y confeti multicolor que saltan sobre el paciente.
const (
	dischargeParticleCount = 14
	dischargeLifeMinTicks  = 30
	dischargeLifeMaxTicks  = 45
	dischargeUpSpeed       = 2.4
	dischargeSpread        = 1.6
	dischargeGravity       = 0.08
	dischargeSize          = 4
)

// Enojo: nubecita de humo rojo sobre la cabeza y una estrella que se
// desprende del contador de reputación del HUD y cae girando.
const (
	angrySmokeCount     = 6
	angrySmokeLifeTicks = 45
	angrySmokeDrift     = 0.5
	angrySmokeRise      = 0.35 // gravedad negativa: flota hacia arriba
	angrySmokeSize      = 5

	fallingStarTicks   = 50
	fallingStarSpin    = 0.22 // rad/tick
	fallingStarGravity = 0.12
)
