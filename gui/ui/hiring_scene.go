package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// La bolsa de empleo (GAME_DESIGN §6, F4): una capa encima del juego, como
// la pausa, que muestra los candidatos del día (Snapshot().Candidates) uno
// por hoja y contrata con Game.Hire. No decide ninguna regla: solo lee la
// foto y llama al método.
//
// Medidas y textos: gui/assets/hiring/HIRING_ASSETS.md (vista aprobada el
// 2026-10-15, Figma "Bolsa de empleo", frames A, B y C).

// hiringPhase es en qué momento de abrir/cerrar está la bolsa.
type hiringPhase int

const (
	hiringOpening hiringPhase = iota // 0,22 s, apareciendo desde el portapapeles del HUD
	hiringOpen                       // quieta, en el centro: acepta clics y teclas
	hiringClosing                    // 0,18 s, volviendo al portapapeles
)

// Tiempos de la animación, redondeados a ticks (60 por segundo).
const (
	hiringOpenTicks  = 13 // 0,22 s
	hiringCloseTicks = 11 // 0,18 s
	hiringPageTicks  = 6  // 0,10 s del cambio de hoja (3 subiendo, 3 bajando)
)

// hiringMinScale, hiringClosedCenter y hiringOpenCenter son los extremos de
// la animación (HIRING_ASSETS.md): cerrada, del tamaño y en el sitio del
// portapapeles chiquito del HUD; abierta, a su tamaño normal y centrada.
const hiringMinScale = 0.167

var (
	hiringClosedCenter = vec{1036, 674}
	hiringOpenCenter   = vec{640, 360}
)

// HiringScene es la bolsa de empleo abierta encima de la partida (pausada
// mientras dura).
type HiringScene struct {
	under *GameScene
	snap  game.Snapshot // la plata y los candidatos, refrescados cada tick (Hire los cambia al toque)

	phase hiringPhase
	ticks int // ticks en la fase actual (abriendo/cerrando)

	index       int // hoja actual
	pendingPage int // a dónde va la hoja cuando termine la transición
	pageTicks   int // ticks que faltan de la transición de hoja; 0 = ninguna en curso

	leftFlash, rightFlash int // ticks que falta mostrar la flecha presionada
}

// newHiringScene abre la bolsa: pausa la partida y arranca la animación de
// abrir desde el portapapeles del HUD.
func newHiringScene(under *GameScene) *HiringScene {
	under.onPause()
	s := &HiringScene{under: under, phase: hiringOpening}
	s.snap = under.g.Snapshot()
	return s
}

// newHiringSceneOpen es la del backdoor de pruebas (-scene hiring): la
// misma bolsa, pero ya abierta de una vez (sin la animación), para que la
// captura muestre el tablero quieto.
func newHiringSceneOpen(under *GameScene) *HiringScene {
	s := newHiringScene(under)
	s.phase = hiringOpen
	return s
}

// Update lee el teclado y el mouse y avanza la bolsa.
func (s *HiringScene) Update() (Scene, error) { return s.update(readInput()) }

func (s *HiringScene) update(in input) (Scene, error) {
	s.snap = s.under.g.Snapshot() // Hire cambia la plata y la lista al toque, sin esperar al motor
	// La barra de arriba es la de la escena del juego, que está en pausa
	// debajo y no corre su Update: se le pasa la misma foto para que la
	// plata cuente hacia abajo y salga el descuento en rojo al contratar.
	s.under.snap = s.snap
	s.under.hud.update(s.snap)
	switch s.phase {
	case hiringOpening:
		s.ticks++
		if s.ticks >= hiringOpenTicks {
			s.phase, s.ticks = hiringOpen, 0
		}
		return s, nil // mientras anima no se aceptan clics (HIRING_ASSETS.md)
	case hiringClosing:
		s.ticks++
		if s.ticks >= hiringCloseTicks {
			s.under.onResume()
			return s.under, nil
		}
		return s, nil
	}

	if s.pageTicks > 0 {
		s.pageTicks--
		if s.pageTicks == hiringPageTicks/2 {
			s.index = s.pendingPage
		}
	}
	if s.leftFlash > 0 {
		s.leftFlash--
	}
	if s.rightFlash > 0 {
		s.rightFlash--
	}
	s.clampIndex()

	switch {
	case in.back:
		s.startClose()
	case in.click && !s.clickInsideBoard(in.mouse):
		s.startClose()
	case in.left || (in.click && in.mouse.In(hiringLeftArrow)):
		s.leftFlash = hiringPageTicks
		s.page(-1)
	case in.right || (in.click && in.mouse.In(hiringRightArrow)):
		s.rightFlash = hiringPageTicks
		s.page(1)
	case in.click && in.mouse.In(hiringHireButton):
		s.hire()
	}
	return s, nil
}

// startClose arranca la animación de cerrar (si no estaba ya cerrando).
func (s *HiringScene) startClose() {
	if s.phase != hiringClosing {
		s.phase, s.ticks = hiringClosing, 0
	}
}

// clickInsideBoard dice si p cayó en el tablero o en una de las flechas (lo
// de afuera cierra la bolsa).
func (s *HiringScene) clickInsideBoard(p image.Point) bool {
	return p.In(hiringBoardRect) || p.In(hiringLeftArrow) || p.In(hiringRightArrow)
}

// page pide cambiar de hoja dir (−1 o 1); si ya hay una transición en curso
// o no hay hoja en esa dirección, no hace nada (HIRING_ASSETS.md: "las
// otras pulsaciones de flecha se ignoran").
func (s *HiringScene) page(dir int) {
	if s.pageTicks > 0 {
		return
	}
	n := s.index + dir
	if n < 0 || n >= len(s.snap.Candidates) {
		return
	}
	s.pendingPage, s.pageTicks = n, hiringPageTicks
}

// clampIndex mantiene la hoja dentro de la lista si se achicó (contrataron
// al candidato de una hoja de más adelante).
func (s *HiringScene) clampIndex() {
	if n := len(s.snap.Candidates); s.index >= n {
		s.index = max(0, n-1)
	}
}

// hire contrata al candidato de la hoja actual, si la plata alcanza. Los
// errores del juego (que no deberían pasar: ya se revisó Affordable) salen
// como aviso de la escena de abajo, igual que los del modo juego.
func (s *HiringScene) hire() {
	cands := s.snap.Candidates
	if s.index < 0 || s.index >= len(cands) || !cands[s.index].Affordable {
		return
	}
	if err := s.under.g.Hire(s.index); err != nil {
		s.under.say(err.Error())
		return
	}
	s.snap = s.under.g.Snapshot()
	s.clampIndex()
}

// pageOffset es cuánto sube o baja el contenido de la hoja (texto, foto,
// puntos y botón; no el tablero de madera ni las flechas) durante el
// cambio de hoja: 0 a −6 px en la primera mitad, 6 a 0 en la segunda.
func (s *HiringScene) pageOffset() int {
	if s.pageTicks <= 0 {
		return 0
	}
	half := hiringPageTicks / 2
	elapsed := hiringPageTicks - s.pageTicks
	if elapsed < half {
		return -6 * elapsed / half
	}
	return 6 * (half - (elapsed - half)) / half
}

// openness es cuánto se ve la bolsa: 0 cerrada (del tamaño del
// portapapeles del HUD), 1 abierta (a su tamaño, centrada). Abrir es
// ease-out (1 − (1 − t)³); cerrar, ease-in al revés (1 − t³).
func (s *HiringScene) openness() float64 {
	switch s.phase {
	case hiringOpening:
		t := float64(s.ticks) / float64(hiringOpenTicks)
		return 1 - math.Pow(1-t, 3)
	case hiringClosing:
		t := float64(s.ticks) / float64(hiringCloseTicks)
		return 1 - math.Pow(t, 3)
	default:
		return 1
	}
}

// hiringTransform es la escala y el centro de la bolsa con esa openness
// (HIRING_ASSETS.md): el punto p del tablero completo (dibujado en su sitio
// final) queda en centro + (p − (640, 360)) × escala.
func hiringTransform(o float64) (scale float64, center vec) {
	scale = hiringMinScale + (1-hiringMinScale)*o
	center = vec{
		hiringClosedCenter.x + (hiringOpenCenter.x-hiringClosedCenter.x)*o,
		hiringClosedCenter.y + (hiringOpenCenter.y-hiringClosedCenter.y)*o,
	}
	return
}

// ─── Medidas e imágenes ────────────────────────────────────────────────

var (
	hiringBoardAt   = image.Pt(280, 81)
	hiringBoardRect = image.Rect(280, 81, 280+730, 81+556)

	hiringLeftArrow  = image.Rect(208, 320, 208+56, 320+80)
	hiringRightArrow = image.Rect(1016, 320, 1016+56, 320+80)

	hiringHireButton = image.Rect(334, 500, 946, 564)

	hudClipboardAt = image.Pt(976, 634)
	hireButton     = image.Rect(976, 640, 976+120, 640+68) // el cuerpo del portapapeles chiquito del HUD
)

// ─── Nombre del candidato ──────────────────────────────────────────────

// candidateNameMaxWidth es lo que mide la columna de datos de la hoja, de
// x = 526 a x = 946 (HIRING_ASSETS.md): el nombre tiene que caber ahí.
const candidateNameMaxWidth = 946 - 526

// candidateNameLineGap es cuánto baja el segundo renglón del nombre cuando
// hay que partirlo en dos (BUG F4: a 24 px "Dra. Yorledis Cassiani" se salía
// del papel). No pisa el rol, que va en y = 250.
const candidateNameLineGap = 18

// fitCandidateName decide a qué tamaño y en cuántos renglones cabe name en
// maxWidth: primero prueba 24 px (el tamaño de la vista aprobada); si no
// cabe, 16 px; si tampoco cabe entero, lo parte en dos renglones por
// palabras a 16 px. Con los nombres de GAME_DESIGN §7 nunca hace falta
// partir (el más largo entra a 16 px), pero queda listo si algún día se
// agrega un nombre más largo a las listas.
func fitCandidateName(name string, maxWidth int) (lines []string, size float64) {
	if pixelTextWidth(name, 24) <= maxWidth {
		return []string{name}, 24
	}
	if pixelTextWidth(name, 16) <= maxWidth {
		return []string{name}, 16
	}
	return wrapNameInTwoLines(name, maxWidth, 16), 16
}

// wrapNameInTwoLines llena el primer renglón con todas las palabras que
// quepan en maxWidth y deja el resto en el segundo. Si una sola palabra ya
// no cabe (no pasa con los nombres de §7), se deja como está: no hay por
// dónde partirla.
func wrapNameInTwoLines(name string, maxWidth int, size float64) []string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return []string{name}
	}
	line1 := words[0]
	i := 1
	for i < len(words) {
		grown := line1 + " " + words[i]
		if pixelTextWidth(grown, size) > maxWidth {
			break
		}
		line1 = grown
		i++
	}
	if i >= len(words) {
		return []string{line1}
	}
	return []string{line1, strings.Join(words[i:], " ")}
}

// drawCandidateName dibuja el nombre del candidato con fitCandidateName, con
// la esquina de arriba a la izquierda en (x, y).
func drawCandidateName(dst *ebiten.Image, name string, x, y, maxWidth int) {
	lines, size := fitCandidateName(name, maxWidth)
	for i, line := range lines {
		drawPixelText(dst, line, x, y+i*candidateNameLineGap, size, []textLayer{{0, 0, colInk}}, 1)
	}
}

// hiringImgs son las imágenes de la bolsa y del portapapeles chiquito del
// HUD, por nombre de archivo. Las carga loadHiring.
var hiringImgs = map[string]*ebiten.Image{}

// hiringImageNames son los archivos que carga loadHiring.
var hiringImageNames = []string{
	"hiring_board.png",
	"hiring_arrow_left_0.png", "hiring_arrow_left_1.png",
	"hiring_arrow_right_0.png", "hiring_arrow_right_1.png",
	"hud_hire_clipboard.png",
}

// loadHiring carga las imágenes de la bolsa de empleo.
func loadHiring() error {
	for _, name := range hiringImageNames {
		img, err := decodePNG(assets.Hiring, "hiring/"+name)
		if err != nil {
			return err
		}
		hiringImgs[name] = ebiten.NewImageFromImage(img)
	}
	return nil
}

// hiringCellImg es el lienzo de 16×24 donde se arma el cuadro quieto 0 de
// un candidato (cuerpo + pelo teñido), reusado cuadro a cuadro para no
// crear una imagen nueva cada vez.
var hiringCellImg *ebiten.Image

// candidateCell dibuja el cuadro quieto 0 de un candidato (médico u
// camillero, con su apariencia) en hiringCellImg y lo devuelve.
func candidateCell(c game.CandidateView) *ebiten.Image {
	if hiringCellImg == nil {
		hiringCellImg = ebiten.NewImage(cellWidth, cellHeight)
	}
	hiringCellImg.Clear()
	var ls []layer
	if c.IsDoctor {
		ls = doctorLayers(c.Look, animIdle, 0)
	} else {
		ls = orderlyLayers(c.Look, animIdle, 0)
	}
	drawLayers(hiringCellImg, ls, ebiten.GeoM{})
	return hiringCellImg
}

// ─── Dibujo ────────────────────────────────────────────────────────────

// Draw dibuja el juego de abajo (quieto, pausado), el oscurecido y el
// tablero, todo escalado según la openness actual.
func (s *HiringScene) Draw(screen *ebiten.Image) {
	s.under.Draw(screen)

	o := s.openness()
	fillRect(screen, image.Rect(0, 0, ScreenWidth, ScreenHeight), color.RGBA{0, 0, 0, uint8(140 * o)})

	content := s.boardCanvas()
	content.Clear()
	s.drawBoard(content)

	scale, center := hiringTransform(o)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(-float64(ScreenWidth)/2, -float64(ScreenHeight)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(center.x, center.y)
	screen.DrawImage(content, op)
}

// boardCanvas es el lienzo de 1280×720 donde se arma el tablero completo
// (imagen, flechas, textos, foto y botón) antes de escalarlo como una sola
// pieza (HIRING_ASSETS.md: "dibujar el tablero completo... y ponerla en
// pantalla con escala s").
var hiringBoardCanvas *ebiten.Image

func (s *HiringScene) boardCanvas() *ebiten.Image {
	if hiringBoardCanvas == nil {
		hiringBoardCanvas = ebiten.NewImage(ScreenWidth, ScreenHeight)
	}
	return hiringBoardCanvas
}

// drawBoard dibuja el tablero en su sitio final (640, 360 es el centro de
// la pantalla de 1280×720): el fondo de madera, las flechas y, encima, el
// contenido de la hoja actual (con su desplazamiento de página).
func (s *HiringScene) drawBoard(dst *ebiten.Image) {
	drawImageAt(dst, hiringImgs["hiring_board.png"], hiringBoardAt, nil)
	s.drawArrow(dst, hiringLeftArrow.Min, "hiring_arrow_left_", s.leftFlash > 0)
	s.drawArrow(dst, hiringRightArrow.Min, "hiring_arrow_right_", s.rightFlash > 0)

	cands := s.snap.Candidates
	if len(cands) == 0 {
		s.drawNoCandidates(dst)
		return
	}
	s.clampIndex()
	s.drawPage(dst, cands[s.index], s.index, len(cands))
}

func (s *HiringScene) drawArrow(dst *ebiten.Image, at image.Point, prefix string, pressed bool) {
	frame := "0"
	if pressed {
		frame = "1"
	}
	drawImageAt(dst, hiringImgs[prefix+frame+".png"], at, nil)
}

// drawNoCandidates: ya contrataron a los 3 de hoy (GAME_DESIGN §6: la
// bolsa se repone al día siguiente). Algo simple, sin foto ni botón.
func (s *HiringScene) drawNoCandidates(dst *ebiten.Image) {
	txt := "YA CONTRATASTE A LOS 3 DE HOY"
	pixelTextCentered(dst, txt, 640, 340, 16, []textLayer{{0, 0, colInk}})
}

// drawPage dibuja la hoja de un candidato: foto, datos, divisores, "i/n" y
// el botón CONTRATAR, todo corrido dy (el desplazamiento de la página).
func (s *HiringScene) drawPage(dst *ebiten.Image, c game.CandidateView, i, n int) {
	dy := s.pageOffset()
	y := func(v int) int { return v + dy }

	drawPixelText(dst, "BOLSA DE EMPLEO", 334, y(158), 16, []textLayer{{0, 0, colBlue}}, 1)
	pixelTextRight(dst, "ESC: CERRAR", 946, y(158), 16, []textLayer{{0, 0, colRed}})
	fillRect(dst, image.Rect(334, y(184), 334+612, y(184)+3), colInk)

	// La foto: marco, fondo y el retrato ×8.
	fillRect(dst, image.Rect(334, y(204), 334+160, y(204)+224), colInk)
	fillRect(dst, image.Rect(339, y(209), 339+150, y(209)+214), colRoomFloor)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Scale(8, 8)
	op.GeoM.Translate(350, float64(y(220)))
	dst.DrawImage(candidateCell(c), op)
	pixelTextCentered(dst, fmt.Sprintf("%d/%d", i+1, n), 414, y(444), 16, []textLayer{{0, 0, colInk}})

	// Los datos, en la columna de la derecha. La vista aprobada va toda en
	// mayúsculas (ROSA MENDOZA, MEDICINA DEL SUEÑO...); es solo
	// presentación, strings.ToUpper al dibujar, nunca sobre los datos de
	// game/ (accentMarks ya sabe dibujar la tilde de las mayúsculas).
	drawCandidateName(dst, strings.ToUpper(c.Name), 526, y(210), candidateNameMaxWidth)
	roleColor := colGreenDark
	if c.IsDoctor {
		roleColor = colBlue
	}
	drawPixelText(dst, strings.ToUpper(c.RoleLabel), 526, y(250), 16, []textLayer{{0, 0, roleColor}}, 1)
	drawPixelText(dst, strings.ToUpper(c.Specialty), 526, y(276), 16, []textLayer{{0, 0, colMuted}}, 1)
	drawPixelText(dst, fmt.Sprintf("EDAD: %d", c.Age), 526, y(302), 16, []textLayer{{0, 0, colMuted}}, 1)
	fillRect(dst, image.Rect(526, y(328), 526+420, y(328)+2), colPaperShade)

	drawPixelText(dst, "RAPIDEZ", 526, y(344), 16, []textLayer{{0, 0, colInk}}, 1)
	drawStatSquares(dst, 706, y(340), c.Speed)
	if c.IsDoctor {
		drawPixelText(dst, "PERICIA", 526, y(380), 16, []textLayer{{0, 0, colInk}}, 1)
		drawStatSquares(dst, 706, y(376), c.Skill)
	} else {
		drawPixelText(dst, "SIN PERICIA: NO REVISA", 526, y(380), 16, []textLayer{{0, 0, colMuted}}, 1)
	}
	fillRect(dst, image.Rect(526, y(406), 526+420, y(406)+2), colPaperShade)

	drawPixelText(dst, "SUELDO POR DÍA", 526, y(418), 16, []textLayer{{0, 0, colMuted}}, 1)
	pixelTextRight(dst, formatMoney(c.Salary), 946, y(418), 16, []textLayer{{0, 0, colInk}})
	drawPixelText(dst, "CONTRATARLO", 526, y(442), 16, []textLayer{{0, 0, colMuted}}, 1)
	pixelTextRight(dst, formatMoney(c.Cost), 946, y(442), 16, []textLayer{{0, 0, colInk}})
	drawPixelText(dst, "TU PLATA", 526, y(466), 16, []textLayer{{0, 0, colMuted}}, 1)
	moneyColor := colGreenDark
	if !c.Affordable {
		moneyColor = colRedDark
	}
	pixelTextRight(dst, formatMoney(s.snap.Money), 946, y(466), 16, []textLayer{{0, 0, moneyColor}})

	s.drawHireButton(dst, c, y)
}

// drawHireButton dibuja el botón CONTRATAR: verde si la plata alcanza,
// gris con "TE FALTAN $ X" si no.
func (s *HiringScene) drawHireButton(dst *ebiten.Image, c game.CandidateView, y func(int) int) {
	fill, shadow, text := colGreen, colGreenDark, colWhite
	if !c.Affordable {
		fill, shadow, text = colLightGray, colMetal, colMuted
	}
	fillRect(dst, image.Rect(334, y(500), 334+612, y(500)+64), colInk)
	fillRect(dst, image.Rect(339, y(505), 339+602, y(505)+54), fill)
	fillRect(dst, image.Rect(339, y(553), 339+602, y(553)+6), shadow)
	pixelTextCentered(dst, "CONTRATAR", 640, y(512), 24, []textLayer{{0, 0, text}})
	if !c.Affordable {
		falta := fmt.Sprintf("TE FALTAN $ %s", thousands(c.Cost-s.snap.Money))
		pixelTextCentered(dst, falta, 640, y(574), 16, []textLayer{{0, 0, colRed}})
	}
}

// drawStatSquares dibuja los 5 cuadros de una estadística (rapidez o
// pericia): contorno negro, lleno amarillo hasta filled, el resto vacío.
func drawStatSquares(dst *ebiten.Image, x0, y, filled int) {
	for i := 0; i < 5; i++ {
		x := x0 + 30*i
		fillRect(dst, image.Rect(x, y, x+24, y+24), colInk)
		c := colPaperShade
		if i < filled {
			c = colYellow
		}
		fillRect(dst, image.Rect(x+3, y+3, x+21, y+21), c)
	}
}

// pixelTextRight y pixelTextCentered alinean el texto de Press Start 2P a
// la derecha o centrado, con la misma firma reducida que usa la bolsa.
func pixelTextRight(dst *ebiten.Image, s string, right, y int, size float64, layers []textLayer) {
	drawPixelText(dst, s, right-pixelTextWidth(s, size), y, size, layers, 1)
}

func pixelTextCentered(dst *ebiten.Image, s string, cx, y int, size float64, layers []textLayer) {
	drawPixelText(dst, s, cx-pixelTextWidth(s, size)/2, y, size, layers, 1)
}

// drawHireClipboard dibuja el portapapeles chiquito de CONTRATAR en el
// HUD: la cabeza (×3) de un candidato de la lista, cambiando cada 2 s, y
// el texto. Sin candidatos, el portapapeles se ve vacío.
const hudCandidateTicks = 120 // 2 s a 60 ticks/s

func (s *GameScene) drawHireClipboard(dst *ebiten.Image) {
	drawImageAt(dst, hiringImgs["hud_hire_clipboard.png"], hudClipboardAt, nil)
	if cands := s.snap.Candidates; len(cands) > 0 {
		c := cands[(s.ticks/hudCandidateTicks)%len(cands)]
		cell := candidateCell(c)
		crop := cell.SubImage(image.Rect(2, 3, 14, 11)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(3, 3)
		op.GeoM.Translate(1018, 659)
		dst.DrawImage(crop, op)
	}
	pixelTextCentered(dst, "CONTRATAR", 1036, 683, 8, []textLayer{{0, 0, colInk}})
}
