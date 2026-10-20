package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// La barra de arriba del modo juego (F3b), según la maqueta
// gui/design/claude-design/Hospital de los Costeños/HUD.dc.html (decisiones
// de Samuel, 2026-10-11, GAME_DESIGN §9.1):
//
//   - a la izquierda, las 5 estrellas con la reputación ("3,7") y la plata;
//   - en el centro, "DÍA N";
//   - a la derecha, el reloj, la hora y la barra del turno.
//
// Los iconos son los mapas de píxeles de la maqueta, copiados tal cual (un
// test los compara con ella) y dibujados ×3. Los textos van en Press Start
// 2P de 16 px. La reputación que se ve se mueve 0,1 a la vez hacia la real
// con un destello, y la plata cuenta hacia la real; cada cambio de plata deja
// un "+$450" o un "-$X" que sube y se desvanece.

// hudPalette es la HUD_PAL de la maqueta: una letra por color.
var hudPalette = map[byte]color.RGBA{
	'K': {0x14, 0x14, 0x14, 0xff}, 'Y': {0xff, 0xd2, 0x3f, 0xff}, 'O': {0xf5, 0xa6, 0x23, 0xff},
	'H': {0xff, 0xf2, 0xa6, 0xff}, 'E': {0xd9, 0xd3, 0xc5, 0xff}, 'e': {0xc9, 0xc2, 0xb2, 0xff},
	'R': {0xe5, 0x48, 0x4d, 0xff}, 'r': {0xb8, 0x32, 0x3a, 0xff}, 'W': {0xff, 0xff, 0xff, 0xff},
	'w': {0xcf, 0xd6, 0xe3, 0xff}, 'G': {0x2f, 0xb8, 0x4f, 0xff}, 'g': {0x1f, 0x8a, 0x3a, 0xff},
	'L': {0xbf, 0xf1, 0xea, 0xff},
}

// Los mapas de píxeles de la maqueta ('.' es transparente).
var (
	hudStar = []string{
		".....K.....",
		"....KHK....",
		"...KHYYK...",
		".KKKYYYKKK.",
		"KYYYYYYYYOK",
		".KYYYYYYOK.",
		"..KYYYYOK..",
		"..KYYYYOK..",
		".KYYOKYYOK.",
		".KOOK.KOOK.",
		"..KK...KK..",
	}
	hudMoneyIcon = []string{
		".KKKKKKKKKKKKKK.",
		"KGGGGGGGGGGGGGGK",
		"KGLLLLLLLLLLLLGK",
		"KGLGGGGggGGGGLGK",
		"KGLGGGgGGgGGGLGK",
		"KGLGGGgGGgGGGLGK",
		"KGLGGGGggGGGGLGK",
		"KGLLLLLLLLLLLLGK",
		"KggggggggggggggK",
		".KKKKKKKKKKKKKK.",
	}
	hudClockIcon = []string{
		"...KKKKK...",
		"..KRRRRRK..",
		".KRRWKWRRK.",
		"KRWWWKWWWRK",
		"KRWWWKWWWRK",
		"KRWWWKKKWRK",
		"KRWWWWWWwrK",
		"KRWWWWWwwrK",
		".KRRWWwrrK.",
		"..KRRrrrK..",
		"...KKKKK...",
	}
	// hudStarEmpty es la estrella vacía: el amarillo y el brillo en E, el
	// naranja en e (en la maqueta, ESTRELLA.map con esos reemplazos).
	hudStarEmpty = func() []string {
		r := strings.NewReplacer("Y", "E", "H", "E", "O", "e")
		out := make([]string, len(hudStar))
		for i, row := range hudStar {
			out[i] = r.Replace(row)
		}
		return out
	}()
)

// hudIconScale: los iconos se dibujan ×3 (1 px de la maqueta = 3 px).
const hudIconScale = 3

// Medidas de la placa de un aviso de reputación (§12.8): fondo crema con
// borde de 2 px, como los avisos del juego (drawNotices en game_scene.go,
// panel() en theme.go), debajo de las estrellas.
const (
	repIconScale   = 2 // la estrella del aviso, más chica que la de arriba (×2, no ×3)
	repPlatePadX   = 6 // margen de la placa a los lados del texto
	repPlatePadY   = 4 // margen de la placa arriba y abajo del contenido
	repIconGap     = 4 // separación entre el número, el ícono y el motivo
	repPlateGapY   = 6 // de la sombra de la barra a la primera placa (para no tocarla)
	repPlateRowGap = 4 // entre una placa y la siguiente, apiladas hacia abajo
)

// repPlateRect es el rectángulo de la placa del aviso de reputación
// i-ésimo (0 es el más viejo que sigue en pantalla): alineada a la
// izquierda bajo las estrellas (starsMinX), apilada debajo de las
// anteriores sin encimarlas, con su tope en top0 (0: arranca justo en
// top0; i > 0: debajo de la anterior). El ancho crece con el texto, así el
// motivo nunca se corta.
//
// top0 lo calcula quien dibuja (hud.draw): como mínimo, debajo del borde y
// la sombra de la barra (nunca los toca); si ya hay avisos del juego
// dibujados ahí (drawNotices, game_scene.go: errores, altas), debajo de
// ellos, para no encimarse con "P-xxx se fue enojado", que deja los dos
// avisos a la vez.
func repPlateRect(starsMinX, top0, i int, amount, reason string) image.Rectangle {
	iconW, iconH := repIconScale*len(hudStar[0]), repIconScale*len(hudStar)
	plateH := iconH + 2*repPlatePadY
	contentW := pixelTextWidth(amount, hudTextSize) + repIconGap + iconW + repIconGap + pixelTextWidth(reason, hudTextSize)
	plateW := contentW + 2*repPlatePadX
	top := top0 + i*(plateH+repPlateRowGap)
	return image.Rect(starsMinX, top, starsMinX+plateW, top+plateH)
}

// iconImage pinta las primeras cols columnas de un mapa de píxeles,
// escalado entero por scale (nearest, sin suavizado: así se dibujan las
// estrellas grandes de arriba, ×hudIconScale, y la chiquita de los avisos
// de reputación, con otra escala).
func iconImage(m []string, cols, scale int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, len(m[0])*scale, len(m)*scale))
	for y, row := range m {
		for x := 0; x < cols && x < len(row); x++ {
			c, ok := hudPalette[row[x]]
			if !ok {
				continue // '.'
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.SetRGBA(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	return img
}

// starColumns es cuántas columnas de la estrella llena se dibujan encima de
// la vacía en la estrella i (de 0 a 4) con la reputación en décimas: todas
// (11) si está entera, ninguna si no llega y, en la parcial,
// 1 + round(9 × fracción): el borde izquierdo y la parte del relleno.
func starColumns(tenths, i int) int {
	f := tenths - 10*i // décimas que le tocan a esta estrella
	switch {
	case f >= 10:
		return len(hudStar[0])
	case f <= 0:
		return 0
	default:
		return 1 + int(math.Round(9*float64(f)/10))
	}
}

// Colores de la barra (los de la maqueta).
var (
	colHUDPaper   = color.RGBA{0xfb, 0xf8, 0xef, 0xff} // fondo
	colHUDInset   = color.RGBA{0xe3, 0xdd, 0xcc, 0xff} // línea de adentro, abajo; fondo de la barra del turno
	colHUDBlue    = color.RGBA{0x2f, 0x6f, 0xd6, 0xff} // el DÍA y el relleno de la barra del turno
	colHUDBlueDim = color.RGBA{0x21, 0x52, 0xa8, 0xff}
	colHUDSky     = color.RGBA{0x9d, 0xdc, 0xf7, 0xff}
	colHUDGreen   = color.RGBA{0x2f, 0xb8, 0x4f, 0xff} // la plata, el "+$450" y el destello al subir
	colHUDRed     = color.RGBA{0xe5, 0x48, 0x4d, 0xff} // el "-$X" y el destello al bajar
	colHUDShadow  = color.RGBA{0, 0, 0, 56}            // la sombra debajo de la barra (22 %)
)

// Medidas de la barra en una pantalla de 1280 (las de la maqueta, a 1280/464).
const (
	hudHeight    = 39 // el fondo; debajo, el borde de 3 px
	hudBorder    = 3
	hudPad       = 17 // margen a los lados
	hudGap       = 11 // entre un icono y su texto
	hudGroupGap  = 33 // entre las estrellas y la plata
	hudTextSize  = 16 // Press Start 2P
	hudBarWidth  = 132
	hudBarHeight = 17
)

// hudLayout son los rectángulos de cada parte de la barra.
type hudLayout struct {
	bar                                    image.Rectangle // el fondo con el borde
	stars, rep                             image.Rectangle
	moneyIcon, money                       image.Rectangle
	day                                    image.Rectangle // el recuadro azul del DÍA
	clockIcon, clock                       image.Rectangle
	progress                               image.Rectangle // la barra del turno con su borde
	repText, moneyText, dayText, clockText string
}

// layoutHUD acomoda las partes de la barra para esos textos: centradas en
// la altura del fondo, de izquierda a derecha y el DÍA en el centro.
func layoutHUD(repText, moneyText, dayText, clockText string) hudLayout {
	mid := hudHeight / 2
	middle := func(x, w, h int) image.Rectangle { // un rectángulo de w×h centrado en la altura
		y := mid - h/2
		return image.Rect(x, y, x+w, y+h)
	}
	l := hudLayout{bar: image.Rect(0, 0, ScreenWidth, hudHeight+hudBorder), repText: repText, moneyText: moneyText, dayText: dayText, clockText: clockText}
	l.stars = middle(hudPad, (5*13-2)*hudIconScale, len(hudStar)*hudIconScale)
	l.rep = middle(l.stars.Max.X+hudGap, pixelTextWidth(repText, hudTextSize), hudTextSize)
	l.moneyIcon = middle(l.rep.Max.X+hudGroupGap, len(hudMoneyIcon[0])*hudIconScale, len(hudMoneyIcon)*hudIconScale)
	l.money = middle(l.moneyIcon.Max.X+hudGap, pixelTextWidth(moneyText, hudTextSize), hudTextSize)

	dayW := pixelTextWidth(dayText, hudTextSize) + 2*hudGap + 2*hudBorder
	dayH := hudTextSize + 7 + 5 + 2*hudBorder // relleno de arriba 7 y de abajo 5, como la maqueta
	l.day = middle(ScreenWidth/2-dayW/2, dayW, dayH)

	l.progress = middle(ScreenWidth-hudPad-hudBarWidth, hudBarWidth, hudBarHeight)
	clockW := pixelTextWidth(clockText, hudTextSize)
	l.clock = middle(l.progress.Min.X-hudGap-6-clockW, clockW, hudTextSize)
	l.clockIcon = middle(l.clock.Min.X-hudGap-len(hudClockIcon[0])*hudIconScale, len(hudClockIcon[0])*hudIconScale, len(hudClockIcon)*hudIconScale)
	return l
}

// hudStarsRect es el rectángulo de las 5 estrellas (el mismo que calcula
// layoutHUD en l.stars, pero sin tener que formatear ningún texto: no
// depende de nada que cambie entre fotos). Lo usa el pase de juice (§12.10,
// BLOQUE 4) para saber de dónde se desprende la estrella cuando alguien se
// va enojado.
func hudStarsRect() image.Rectangle {
	mid := hudHeight / 2
	w, h := (5*13-2)*hudIconScale, len(hudStar)*hudIconScale
	y := mid - h/2
	return image.Rect(hudPad, y, hudPad+w, y+h)
}

// rushHourText es el letrero de la hora pico (§12.9).
const rushHourText = "HORA PICO"

// rushHourSignRect es dónde va "HORA PICO": a la izquierda del ícono del
// reloj (nunca toca el DÍA ni la barra del turno, que están más a la
// izquierda y más a la derecha; TestHUD_RushHourSignDoesNotOverlapAnything).
func rushHourSignRect(l hudLayout) image.Rectangle {
	w := pixelTextWidth(rushHourText, smallFace.Size)
	x := l.clockIcon.Min.X - hudGap - w
	y := l.clockIcon.Min.Y + (l.clockIcon.Dy()-int(smallFace.Size))/2
	return image.Rect(x, y, x+w, y+int(smallFace.Size))
}

// streakBadgeRect es dónde va "RACHA ×N": debajo de la plata, empezando en
// el mismo x que su texto (l.money.Min.X, que depende del ícono y la
// reputación, nunca de cuántos dígitos tenga la plata: así nunca se corre
// hacia el DÍA por una cifra grande). TestHUD_StreakAndRushHourSignsDoNotOverlapAnything.
func streakBadgeRect(l hudLayout, text string) image.Rectangle {
	w := pixelTextWidth(text, smallFace.Size)
	x := l.money.Min.X
	y := l.money.Max.Y + 1
	return image.Rect(x, y, x+w, y+int(smallFace.Size))
}

// progressFill es lo que se llena de la barra del turno (por dentro del
// borde) con el turno en progress (de 0 a 1).
func progressFill(bar image.Rectangle, progress float64) int {
	inner := bar.Dx() - 2*hudBorder
	return int(math.Round(math.Max(0, math.Min(1, progress)) * float64(inner)))
}

// formatMoney escribe la plata como en la maqueta: "$ 2.450", con puntos de
// miles; si es negativa, "-$ 150".
func formatMoney(n int) string {
	if n < 0 {
		return "-" + formatMoney(-n)
	}
	return "$ " + thousands(n)
}

// thousands escribe n con puntos de miles: 1234567 → "1.234.567".
func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}

// formatDelta es el cambio de plata que sube: "+$450" o "-$1.200".
func formatDelta(d int) string {
	if d < 0 {
		return "-$" + thousands(-d)
	}
	return "+$" + thousands(d)
}

// formatReputation escribe la reputación en décimas como en la maqueta: "3,7".
func formatReputation(tenths int) string { return fmt.Sprintf("%d,%d", tenths/10, tenths%10) }

// formatReputationChange es el aviso con motivo de un cambio de reputación
// (§12.8, decisión de Samuel del 2026-10-16, para que no "le bajen las
// estrellas solas"): "-0,5 ★ P-003 lleva 30 s esperando revisión" o "+0,1 ★
// P-002: lo recogieron rápido". El signo es "-" (no "−": Press Start 2P no
// tiene el signo menos tipográfico, se comprobó con sfnt.GlyphIndex), igual
// que formatDelta con la plata.
//
// El "★" de en medio **no se dibuja**: a ese tamaño, en Press Start 2P sale
// como un cuadro sin forma de estrella (la fuente no trae ese glifo). Queda
// solo como separador de datos entre el número y el motivo: draw() lo parte
// con strings.Cut y dibuja ahí, en su lugar, el ícono de estrella chiquito
// de hudImgs.starSmall, teñido de rojo o verde.
func formatReputationChange(c game.ReputationChange) string {
	return fmt.Sprintf("%s ★ %s", formatReputationStars(c.Delta), c.Reason)
}

// formatReputationStars es el delta en centésimas como estrellas con coma
// decimal, sin ceros de más: -50 → "-0,5", -25 → "-0,25", +10 → "+0,1". Los
// deltas de reputación son todos múltiplos de 5 centésimas (config.go), así
// que nunca hace falta más de dos decimales.
func formatReputationStars(hundredths int) string {
	sign := "+"
	if hundredths < 0 {
		sign, hundredths = "-", -hundredths
	}
	frac := strings.TrimRight(fmt.Sprintf("%02d", hundredths%100), "0")
	if frac == "" {
		frac = "0"
	}
	return fmt.Sprintf("%s%d,%s", sign, hundredths/100, frac)
}

// hudDayLabel es el texto del recuadro del centro.
func hudDayLabel(day int) string { return fmt.Sprintf("DÍA %d", day) }

// formatStreakMultiplier escribe el multiplicador de la racha (centésimas,
// 100 = ×1) con coma decimal y sin ceros de más: 150 → "×1,5", 200 → "×2".
func formatStreakMultiplier(hundredths int) string {
	whole := hundredths / 100
	frac := strings.TrimRight(fmt.Sprintf("%02d", hundredths%100), "0")
	if frac == "" {
		return fmt.Sprintf("×%d", whole)
	}
	return fmt.Sprintf("×%d,%s", whole, frac)
}

// ─── Lo que se mueve ───────────────────────────────────────────────────

// Tiempos de la barra, en ticks (60 por segundo).
const (
	repStepTicks    = 15 // un paso de 0,1 de reputación (y su destello): 0,25 s
	moneyCountTicks = 36 // la plata cuenta hasta la real en 0,6 s
	deltaTicks      = 60 // el "+$450" sube y se desvanece en 1 s
	deltaRise       = 24 // lo que sube, en px
)

// Tiempos de la racha de atención (§12.9): streakPopTicks es el "pop" (un
// salto de 3 px) cuando sube el multiplicador, streakLostTicks el
// parpadeo corto (3 destellos) cuando se pierde.
const (
	streakPopTicks  = 10
	streakLostTicks = 24
	streakPopRise   = 3 // px que salta hacia arriba
)

// moneyDelta es un "+$450" o un "-$X" que sube y se desvanece.
type moneyDelta struct {
	text  string
	color color.RGBA
	ticks int // ticks desde que apareció
}

// offset es lo que lleva subido (negativo: hacia arriba) y su opacidad.
func (d moneyDelta) offset() (rise float64, alpha float32) { return fadeOffset(d.ticks) }

// Tiempos del aviso de reputación, en ticks. El motivo es una frase larga
// ("P-003 lleva 30 s esperando revisión") que en el segundo del "+$450" no
// se alcanza a leer (prueba de Samuel del 17-10): se queda opaco 3 s y se
// desvanece en el último.
const (
	repHoldTicks = 180                         // 3 s opaco…
	repFadeTicks = 60                          // …y 1 s desvaneciéndose
	repShowTicks = repHoldTicks + repFadeTicks // lo que dura en total
)

// repChangeDelta es un aviso de reputación con su motivo (§12.8). No sube
// como el de la plata: se queda en su sitio y se desvanece al final.
type repChangeDelta struct {
	text  string
	color color.RGBA
	ticks int
}

// alpha es la opacidad del aviso: 1 durante repHoldTicks y después baja
// en línea recta hasta 0 en repFadeTicks.
func (d repChangeDelta) alpha() float32 {
	f := float64(d.ticks-repHoldTicks) / repFadeTicks
	return float32(1 - math.Max(0, math.Min(1, f)))
}

// fadeOffset es lo que lleva subido (negativo: hacia arriba) y la opacidad
// de algo que aparece y en deltaTicks sube deltaRise px mientras se
// desvanece (el "+$450" de la plata).
func fadeOffset(ticks int) (rise float64, alpha float32) {
	f := math.Min(1, float64(ticks)/deltaTicks)
	return -deltaRise * f, float32(1 - f)
}

// hud es lo que la barra recuerda entre cuadros: lo que muestra, que se
// acerca de a poco a lo que dice la foto del juego.
type hud struct {
	started bool
	// La reputación que se ve, en décimas, y el paso en curso.
	rep       int
	stepTicks int // ticks que le quedan al paso actual (y a su destello)
	stepUp    bool
	// La plata: la que se ve va de moneyFrom a moneyTo en moneyCountTicks.
	moneyFrom, moneyTo int
	moneyTick          int
	deltas             []moneyDelta
	// Los avisos de reputación con motivo (§12.8): repSeen recuerda cuáles de
	// Snapshot.ReputationChanges ya se mostraron (la cola se recorre entera
	// en cada foto, como Notices con noticeSeen en game_scene.go), y
	// repDeltas son los que todavía se ven, subiendo y desvaneciéndose.
	repSeen   map[game.ReputationChange]bool
	repDeltas []repChangeDelta

	// La racha de atención y la hora pico (§12.9, pedido de Samuel el
	// 2026-10-17). streakShown es el multiplicador que se ve (en
	// centésimas; 100 = sin racha); popTicks cuenta el salto cuando sube
	// y lostTicks el parpadeo corto cuando se pierde (baja a 100).
	// hudTicks es el reloj propio de la barra, para el parpadeo suave de
	// HORA PICO.
	streakShown int
	streakPop   int
	streakLost  int
	hudTicks    int
}

// repTenths es la reputación de la foto (en centésimas) en décimas,
// redondeada: 2,75 se ve 2,8.
func repTenths(hundredths int) int { return (hundredths + 5) / 10 }

// update avanza la barra un tick con la foto actual del juego.
func (h *hud) update(snap game.Snapshot) {
	if !h.started { // la primera foto se muestra tal cual
		h.started = true
		h.rep = repTenths(snap.Reputation)
		h.moneyFrom, h.moneyTo, h.moneyTick = snap.Money, snap.Money, moneyCountTicks
		h.repSeen = map[game.ReputationChange]bool{}
		for _, c := range snap.ReputationChanges { // de antes de abrir la pantalla: no dejan aviso
			h.repSeen[c] = true
		}
		h.streakShown = snap.StreakMultiplier
		return
	}

	h.hudTicks++
	if h.streakPop > 0 {
		h.streakPop--
	}
	if h.streakLost > 0 {
		h.streakLost--
	}
	if snap.StreakMultiplier > h.streakShown {
		h.streakPop = streakPopTicks
	} else if snap.StreakMultiplier < h.streakShown {
		h.streakLost = streakLostTicks
	}
	h.streakShown = snap.StreakMultiplier

	// Reputación: un paso de 0,1 cada repStepTicks, con su destello.
	if h.stepTicks > 0 {
		h.stepTicks--
	}
	if target := repTenths(snap.Reputation); h.stepTicks == 0 && target != h.rep {
		h.stepUp = target > h.rep
		if h.stepUp {
			h.rep++
		} else {
			h.rep--
		}
		h.stepTicks = repStepTicks
	}

	// Plata: cada cambio arranca la cuenta desde lo que se ve y deja su cambio.
	if snap.Money != h.moneyTo {
		d := snap.Money - h.moneyTo
		h.moneyFrom, h.moneyTo, h.moneyTick = h.shownMoney(), snap.Money, 0
		c := colHUDGreen
		if d < 0 {
			c = colHUDRed
		}
		h.deltas = append(h.deltas, moneyDelta{text: formatDelta(d), color: c})
	}
	if h.moneyTick < moneyCountTicks {
		h.moneyTick++
	}
	kept := h.deltas[:0]
	for _, d := range h.deltas {
		if d.ticks++; d.ticks < deltaTicks {
			kept = append(kept, d)
		}
	}
	h.deltas = kept

	// Reputación: cada cambio nuevo de la cola (§12.8) deja un aviso con su
	// motivo, rojo si baja y verde si sube.
	for _, c := range snap.ReputationChanges {
		if h.repSeen[c] {
			continue
		}
		h.repSeen[c] = true
		clr := colHUDGreen
		if c.Delta < 0 {
			clr = colHUDRed
		}
		h.repDeltas = append(h.repDeltas, repChangeDelta{text: formatReputationChange(c), color: clr})
	}
	keptRep := h.repDeltas[:0]
	for _, d := range h.repDeltas {
		if d.ticks++; d.ticks < repShowTicks {
			keptRep = append(keptRep, d)
		}
	}
	h.repDeltas = keptRep
}

// shownMoney es la plata que se ve ahora.
func (h *hud) shownMoney() int {
	f := float64(h.moneyTick) / moneyCountTicks
	return h.moneyFrom + int(math.Round(f*float64(h.moneyTo-h.moneyFrom)))
}

// flashing dice si la reputación está destellando (en un paso).
func (h *hud) flashing() bool { return h.stepTicks > 0 }

// flashColor es el color del destello: verde si sube, rojo si baja.
func (h *hud) flashColor() color.RGBA {
	if h.stepUp {
		return colHUDGreen
	}
	return colHUDRed
}

// ─── Dibujo ────────────────────────────────────────────────────────────

// Las imágenes de los iconos, hechas la primera vez que se dibuja la barra.
var hudImgs struct {
	star, starEmpty, money, clock, starSmall *ebiten.Image
}

func hudImages() {
	if hudImgs.star != nil {
		return
	}
	hudImgs.star = ebiten.NewImageFromImage(iconImage(hudStar, len(hudStar[0]), hudIconScale))
	hudImgs.starEmpty = ebiten.NewImageFromImage(iconImage(hudStarEmpty, len(hudStar[0]), hudIconScale))
	hudImgs.money = ebiten.NewImageFromImage(iconImage(hudMoneyIcon, len(hudMoneyIcon[0]), hudIconScale))
	hudImgs.clock = ebiten.NewImageFromImage(iconImage(hudClockIcon, len(hudClockIcon[0]), hudIconScale))
	// La estrella chiquita de los avisos de reputación (§12.8): la misma
	// forma de arriba, más chica (×2 en vez de ×3), sin teñir todavía (se
	// tiñe de rojo o verde al dibujarla, igual que el destello de las
	// estrellas grandes).
	hudImgs.starSmall = ebiten.NewImageFromImage(iconImage(hudStar, len(hudStar[0]), repIconScale))
}

// fadeAlpha multiplica la opacidad de c por alpha (0 a 1). Devuelve NRGBA
// (no premultiplicado) porque fillRect espera un color "derecho", igual que
// la tilde que se desvanece en drawPixelText (pixelfont.go).
func fadeAlpha(c color.RGBA, alpha float32) color.NRGBA {
	return color.NRGBA{c.R, c.G, c.B, uint8(float32(c.A) * alpha)}
}

// panelFaded es panel() (theme.go) pero con la placa y el borde
// desvaneciéndose: lo usan los avisos de reputación, que se van solos.
func panelFaded(dst *ebiten.Image, r image.Rectangle, bg color.RGBA, alpha float32) {
	fillRect(dst, r, fadeAlpha(bg, alpha))
	outline(dst, r, fadeAlpha(colInk, alpha))
}

// drawImageAt dibuja img con su esquina en at, teñida con tint si no es nil.
func drawImageAt(dst, img *ebiten.Image, at image.Point, tint *color.RGBA) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(at.X), float64(at.Y))
	if tint != nil {
		op.ColorScale.ScaleWithColor(*tint)
	}
	dst.DrawImage(img, op)
}

// draw dibuja la barra con lo que muestra ahora y la hora, el día y el
// turno de la foto. repMinTop es el Y libre que deja drawNotices
// (game_scene.go) debajo de sus avisos del juego, si hay: los avisos de
// reputación con motivo nunca empiezan más arriba de ahí.
func (h *hud) draw(dst *ebiten.Image, snap game.Snapshot, repMinTop int) {
	hudImages()
	l := layoutHUD(formatReputation(h.rep), formatMoney(h.shownMoney()), hudDayLabel(snap.Day), snap.Clock)

	// El fondo, la línea de adentro, el borde y la sombra.
	fillRect(dst, image.Rect(0, 0, ScreenWidth, hudHeight), colHUDPaper)
	fillRect(dst, image.Rect(0, hudHeight-hudBorder, ScreenWidth, hudHeight), colHUDInset)
	fillRect(dst, image.Rect(0, hudHeight, ScreenWidth, hudHeight+hudBorder), colInk)
	fillRect(dst, image.Rect(0, hudHeight+hudBorder, ScreenWidth, hudHeight+2*hudBorder), colHUDShadow)

	// Las estrellas: la vacía y, encima, la llena hasta donde le toca. Con el
	// destello se tiñen del color del paso, igual que el número.
	var tint *color.RGBA
	repColor := colInk
	if h.flashing() {
		c := h.flashColor()
		tint, repColor = &c, c
	}
	for i := 0; i < 5; i++ {
		at := l.stars.Min.Add(image.Pt(i*13*hudIconScale, 0))
		drawImageAt(dst, hudImgs.starEmpty, at, tint)
		if cols := starColumns(h.rep, i); cols > 0 {
			part := hudImgs.star.SubImage(image.Rect(0, 0, cols*hudIconScale, len(hudStar)*hudIconScale)).(*ebiten.Image)
			drawImageAt(dst, part, at, tint)
		}
	}
	drawPixelText(dst, l.repText, l.rep.Min.X, l.rep.Min.Y, hudTextSize, []textLayer{{0, 0, repColor}}, 1)

	// Los avisos de reputación con motivo (§12.8): una placa crema con
	// borde negro, igual que los avisos del juego, **debajo del todo** de
	// la barra (nunca toca su borde ni su sombra) y debajo de los avisos
	// del juego si los hay (repMinTop; no se encima con ellos), alineada a
	// la izquierda bajo las estrellas. Cada una se apila debajo de la
	// anterior y se desvanece sola. El número y el motivo van separados por
	// el ícono de estrella chiquito (no el glifo "★": a este tamaño, en
	// Press Start 2P sale como un cuadro sin forma de estrella).
	//
	// A diferencia del "+$450" de la plata, la placa **no sube**: solo se
	// desvanece en su sitio. Si subiera los 24 px de fadeOffset volvería a
	// meterse debajo de la barra (el bug que se está arreglando), porque su
	// placa ya arranca pegada al borde de abajo, sin margen arriba.
	barBottom := hudHeight + 2*hudBorder // debajo del borde de 3 px y la sombra de 3 px
	top0 := barBottom + repPlateGapY
	if m := repMinTop + repPlateGapY; m > top0 {
		top0 = m
	}
	for i, d := range h.repDeltas {
		alpha := d.alpha()
		amount, reason, _ := strings.Cut(d.text, " ★ ")
		rect := repPlateRect(l.stars.Min.X, top0, i, amount, reason)
		panelFaded(dst, rect, colHUDPaper, alpha)

		iconW, iconH := repIconScale*len(hudStar[0]), repIconScale*len(hudStar)
		textY := rect.Min.Y + repPlatePadY + (iconH-hudTextSize)/2
		x := rect.Min.X + repPlatePadX
		drawPixelText(dst, amount, x, textY, hudTextSize, []textLayer{{0, 0, d.color}}, alpha)
		x += pixelTextWidth(amount, hudTextSize) + repIconGap
		iconTint := color.RGBA{d.color.R, d.color.G, d.color.B, uint8(255 * alpha)}
		drawImageAt(dst, hudImgs.starSmall, image.Pt(x, rect.Min.Y+repPlatePadY), &iconTint)
		x += iconW + repIconGap
		drawPixelText(dst, reason, x, textY, hudTextSize, []textLayer{{0, 0, d.color}}, alpha)
	}

	// La plata, verde con contorno negro, y los cambios que suben.
	drawImageAt(dst, hudImgs.money, l.moneyIcon.Min, nil)
	drawPixelText(dst, l.moneyText, l.money.Min.X, l.money.Min.Y, hudTextSize, append(outlined(2, colInk), textLayer{0, 0, colHUDGreen}), 1)
	for _, d := range h.deltas {
		rise, alpha := d.offset()
		y := l.money.Min.Y - 6 + int(math.Round(rise))
		drawPixelText(dst, d.text, l.money.Max.X+6, y, hudTextSize, append(outlined(2, colInk), textLayer{0, 0, d.color}), alpha)
	}

	// La racha de atención (§12.9): "RACHA ×1,5" junto a la plata, solo si
	// hay alguna (o si se acaba de perder y todavía está parpadeando),
	// con un salto de unos px cuando sube y un parpadeo corto cuando se
	// pierde.
	if snap.Streak > 0 || h.streakLost > 0 {
		visible := true
		if h.streakLost > 0 {
			visible = (h.streakLost/4)%2 == 0 // 3 destellos en streakLostTicks
		}
		if visible {
			txt := "RACHA " + formatStreakMultiplier(h.streakShown)
			r := streakBadgeRect(l, txt)
			y := r.Min.Y
			if h.streakPop > 0 {
				y -= int(math.Round(streakPopRise * float64(h.streakPop) / streakPopTicks))
			}
			drawPixelText(dst, txt, r.Min.X, y, smallFace.Size, []textLayer{{0, 0, colHUDGreen}}, 1)
		}
	}

	// La hora pico (§12.9): "HORA PICO" junto al reloj, parpadeando suave
	// (seno, nunca tapa ni el DÍA ni la barra del turno: va a la
	// izquierda del ícono del reloj, en el hueco entre el DÍA y ahí).
	if snap.RushHour {
		alpha := float32(0.5 + 0.5*math.Sin(float64(h.hudTicks)/20))
		r := rushHourSignRect(l)
		drawPixelText(dst, rushHourText, r.Min.X, r.Min.Y, smallFace.Size, []textLayer{{0, 0, colOrange}}, alpha)
	}

	// El DÍA: recuadro azul con borde negro, sombra de adentro y letras blancas.
	fillRect(dst, l.day, colInk)
	inner := l.day.Inset(hudBorder)
	fillRect(dst, inner, colHUDBlue)
	fillRect(dst, image.Rect(inner.Min.X, inner.Max.Y-hudBorder, inner.Max.X, inner.Max.Y), colHUDBlueDim)
	drawPixelText(dst, l.dayText, inner.Min.X+hudGap, inner.Min.Y+7, hudTextSize, []textLayer{{0, 0, color.White}}, 1)

	// El reloj, la hora y la barra del turno.
	drawImageAt(dst, hudImgs.clock, l.clockIcon.Min, nil)
	drawPixelText(dst, l.clockText, l.clock.Min.X, l.clock.Min.Y, hudTextSize, []textLayer{{0, 0, colInk}}, 1)
	fillRect(dst, l.progress, colInk)
	bar := l.progress.Inset(hudBorder)
	fillRect(dst, bar, colHUDInset)
	if w := progressFill(l.progress, snap.Progress); w > 0 {
		fill := image.Rect(bar.Min.X, bar.Min.Y, bar.Min.X+w, bar.Max.Y)
		fillRect(dst, fill, colHUDBlue)
		fillRect(dst, image.Rect(fill.Min.X, fill.Min.Y, fill.Max.X, fill.Min.Y+hudBorder), colHUDSky)
		fillRect(dst, image.Rect(fill.Min.X, fill.Max.Y-hudBorder, fill.Max.X, fill.Max.Y), colHUDBlueDim)
		if w < bar.Dx() {
			fillRect(dst, image.Rect(fill.Max.X, bar.Min.Y, min(fill.Max.X+hudBorder, bar.Max.X), bar.Max.Y), colInk) // el borde del relleno
		}
	}
}
