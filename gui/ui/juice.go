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
