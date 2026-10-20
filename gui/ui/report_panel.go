package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Reporte del día (§9, drawDayOver en game_scene.go): vista aprobada en
// Figma ("Reporte A – día con ganancia", frame 17:224; "Reporte B – día
// con pérdida", 17:225, vacío salvo el mapa, así que usa el mismo diseño
// de A con los negativos en rojo). Todo esto es dibujo puro: la plata, la
// nómina y las cuatro consultas (5.1 a 5.4) ya las calculó game/report.go
// (DayReport), acá solo se acomodan en el tablero.
//
// Un par de cosas del Figma no se copiaron tal cual (quedan en el reporte
// de la tarea, no en el código):
//   - Las chinchetas de colores ("report_pin_*.png") no tienen una
//     posición propia en el frame (el nodo de cada una quedó en 0,0): no
//     hay dónde ponerlas sin inventar el número, así que no se dibujan.
//   - El cuadro de PUNTAJE y el botón SIGUIENTE DÍA, en el Figma, quedaron
//     en gris plano con texto negro puro (#000000, no el #141414 de la
//     paleta): es el color por defecto de una figura nueva en Figma, no
//     una elección de diseño (el mismo gris/negro aparece en el bloque de
//     abajo duplicado del Game Over C2, que GAME_DESIGN ya marca como
//     error de Figma). Acá se dibujan con el estilo de ficha normal
//     (papel + texto de tinta) y el botón en azul con texto blanco, como
//     ya tenía el código viejo.

// ─── Tablero de corcho ──────────────────────────────────────────────────

var reportCorkboardImg *ebiten.Image

// reportCorkboardAt y su tamaño: Figma "tablero de corcho
// (report_corkboard.png 1226x693, en 32,16)".
var reportCorkboardAt = image.Pt(32, 16)

// loadReportPanel carga el PNG del tablero. Se llama perezosa (la primera
// vez que se dibuja un panel de fin de día), no desde loadAssets
// (app.go): ese archivo lo está tocando otra sesión ahora mismo, así que
// este panel se las arregla solo en vez de sumarse a esa lista.
func loadReportPanel() error {
	if reportCorkboardImg != nil {
		return nil
	}
	img, err := decodePNG(assets.Report, "report/report_corkboard.png")
	if err != nil {
		return err
	}
	reportCorkboardImg = ebiten.NewImageFromImage(img)
	return nil
}

// ─── Columna de dinero ───────────────────────────────────────────────────

// formatMoneySigned es la plata con su signo siempre puesto: "+$ 1.900" o
// "-$ 600" (a diferencia de formatMoney, hud.go, que no le pone "+").
func formatMoneySigned(n int) string {
	if n < 0 {
		return "-" + formatMoney(-n)
	}
	return "+" + formatMoney(n)
}

// formatSignedInt es un entero con su signo siempre puesto: "+123", "-34",
// "+0" (PLATA ÷ 10 y ESTRELLAS × 200 del Game Over, gameover_panel.go).
func formatSignedInt(n int) string {
	if n < 0 {
		return fmt.Sprint(n) // ya trae el "-"
	}
	return "+" + fmt.Sprint(n)
}

// moneyColor: verde si n ≥ 0, rojo si es negativo (Reporte B, §9: "los
// negativos usan el rojo que ya usan las vistas").
func moneyColor(n int) color.Color {
	if n < 0 {
		return colRed
	}
	return colGreenDark
}

// ─── Bloques con "+ N MÁS" (5.1 a 5.4) ───────────────────────────────────

// renderLine es un renglón de una de las cuatro fichas de consulta: texto a
// la izquierda y, si right no es "", un dato a la derecha (el total de
// episodios de un médico, el ID del paciente en su habitación...).
type renderLine struct {
	text       string
	color      color.Color
	right      string
	rightColor color.Color
}

// reportBlock es lo que ocupa UN registro de la lista (un médico con sus
// episodios, un paciente severo con su nota...): una o más renderLine que
// se muestran enteras o no se muestran, nunca a la mitad.
type reportBlock []renderLine

// fitBlocks corta blocks a lo que quepa en budget renglones: agrega
// bloques completos mientras quepan Y, si después de éste todavía queda
// alguno más, deja un renglón libre para avisarlo ("+ N MÁS" lo arma quien
// llama, con moreCount). Nunca corta un bloque a la mitad.
func fitBlocks(blocks []reportBlock, budget int) (shown []reportBlock, moreCount int) {
	used := 0
	for i, b := range blocks {
		need := len(b)
		if i+1 < len(blocks) {
			need++ // un renglón de más, por si hiciera falta el aviso después de éste
		}
		if used+need > budget {
			break
		}
		used += len(b)
		shown = append(shown, b)
	}
	return shown, len(blocks) - len(shown)
}

// maxQueryCardLines: renglones de 8 px que caben en una ficha de consulta
// (5.1 a 5.4), de la primera fila (66 px bajo la esquina de la ficha, en
// las cuatro) hasta 10 px antes de su borde de abajo, en pasos de 14 px
// (grounded: Figma "ficha N contorno", ficha N divisor, filas de ejemplo).
const (
	queryRowStartOffset = 66
	queryRowStep        = 14
	queryBottomMargin   = 10
)

func maxQueryCardLines(box image.Rectangle) int {
	avail := box.Max.Y - queryBottomMargin - (box.Min.Y + queryRowStartOffset)
	return avail / queryRowStep
}

// ─── 5.1 Pasillo ──────────────────────────────────────────────────────────

func hallwayBlocks(hallway []game.PatientView) []reportBlock {
	var out []reportBlock
	for _, p := range hallway {
		out = append(out, reportBlock{{text: strings.ToUpper(fmt.Sprintf("%s %s (%s)", p.ID, p.Name, p.Location)), color: colInk}})
	}
	return out
}

// ─── 5.2 Médicos ──────────────────────────────────────────────────────────

// maxEpisodesPerDoctor: episodios que se listan de cada médico antes de
// "+ N MÁS" (Figma: 2 líneas y después el aviso).
const maxEpisodesPerDoctor = 2

// episodeDestination es a dónde fue el paciente del episodio: el número de
// la habitación, o "PASILLO" si se quedó sin cama.
func episodeDestination(e game.EpisodeLine) string {
	if e.Room == 0 {
		return "PASILLO"
	}
	return fmt.Sprint(e.Room)
}

func doctorBlock(d game.DoctorEpisodes) reportBlock {
	block := reportBlock{{text: strings.ToUpper(d.DoctorName), color: colBlue, right: fmt.Sprint(len(d.Episodes)), rightColor: colInk}}
	eps := d.Episodes
	overflow := 0
	if len(eps) > maxEpisodesPerDoctor {
		overflow = len(eps) - maxEpisodesPerDoctor
		eps = eps[:maxEpisodesPerDoctor]
	}
	for _, e := range eps {
		line := fmt.Sprintf("  EP-%s %s %s → %s", e.ID, e.Time, e.PatientID, episodeDestination(e))
		block = append(block, renderLine{text: line, color: colInk})
	}
	if overflow > 0 {
		block = append(block, renderLine{text: fmt.Sprintf("  + %d MÁS", overflow), color: colMuted})
	}
	return block
}

func doctorBlocks(episodes []game.DoctorEpisodes) []reportBlock {
	var out []reportBlock
	for _, d := range episodes {
		out = append(out, doctorBlock(d))
	}
	return out
}

// ─── 5.3 Habitaciones ─────────────────────────────────────────────────────

// roomStateWord: "OCUPADA"/"LIBRE" (Figma; distinto de RoomState.String(),
// que da "ocupada"/"disponible" para el letrero OCUPADA/DISPONIBLE del
// mapa, overlays.go). Es un rótulo de esta ficha, no una regla del modelo.
func roomStateWord(s hospital.RoomState) string {
	if s == hospital.Occupied {
		return "OCUPADA"
	}
	return "LIBRE"
}

// roomStateColor: rojo ocupada, verde libre (Figma: fill_98adi8/fill_wxxslu).
func roomStateColor(s hospital.RoomState) color.Color {
	if s == hospital.Occupied {
		return colRed
	}
	return colGreenDark
}

func roomBlock(r game.RoomView) reportBlock {
	occupant := ""
	if len(r.Occupants) > 0 {
		occupant = strings.Join(r.Occupants, ", ")
	}
	return reportBlock{{
		text: fmt.Sprintf("%d  %s", r.Number, roomStateWord(r.State)), color: roomStateColor(r.State),
		right: occupant, rightColor: colInk,
	}}
}

func roomBlocks(rooms []game.RoomView) []reportBlock {
	var out []reportBlock
	for _, r := range rooms {
		out = append(out, roomBlock(r))
	}
	return out
}

// ─── 5.4 Severos ──────────────────────────────────────────────────────────

func severeBlock(sv game.SevereLine) reportBlock {
	block := reportBlock{{text: strings.ToUpper(fmt.Sprintf("%s %s", sv.PatientID, sv.PatientName)), color: colInk, right: fmt.Sprint(sv.Episodes), rightColor: colInk}}
	if sv.Note != "" {
		block = append(block, renderLine{text: "  " + strings.ToUpper(sv.Note), color: colMuted})
	}
	return block
}

func severeBlocks(severe []game.SevereLine) []reportBlock {
	var out []reportBlock
	for _, sv := range severe {
		out = append(out, severeBlock(sv))
	}
	return out
}

// ─── Dibujo de las cuatro fichas de consulta (5.1 a 5.4) ─────────────────

// queryCardBoxes: las cuatro fichas de consulta, en el orden del Shift
// Report (Figma "Reporte A", fichas 1 a 4).
var (
	hallwayCardBox = image.Rect(72, 386, 352, 592)
	doctorsCardBox = image.Rect(360, 390, 640, 596)
	roomsCardBox   = image.Rect(648, 384, 928, 590)
	severeCardBox  = image.Rect(936, 388, 1216, 594)
)

// drawQueryCardChrome dibuja el papel, el título y el subtítulo comunes a
// las cuatro fichas de consulta (offsets iguales en las cuatro: título en
// +12,+18, subtítulo en +12,+40, divisor en +54, Figma).
func drawQueryCardChrome(dst *ebiten.Image, box image.Rectangle, title, subtitle string) {
	panel(dst, box, colPaper)
	drawPixelText(dst, title, box.Min.X+12, box.Min.Y+18, 16, []textLayer{{0, 0, colBlue}}, 1)
	if subtitle != "" {
		drawPixelText(dst, subtitle, box.Min.X+12, box.Min.Y+40, 8, []textLayer{{0, 0, colMuted}}, 1)
	}
	fillRect(dst, image.Rect(box.Min.X+12, box.Min.Y+54, box.Max.X-12, box.Min.Y+56), colPaperShade)
}

// drawBlockList dibuja blocks (ya recortados con fitBlocks) y, si
// moreCount > 0, el renglón "+ N MÁS"; si no hay ninguno, emptyText.
func drawBlockList(dst *ebiten.Image, box image.Rectangle, blocks []reportBlock, moreCount int, emptyText string) {
	x := box.Min.X + 12
	y := box.Min.Y + queryRowStartOffset
	rightX := box.Max.X - 12
	if len(blocks) == 0 {
		drawPixelText(dst, emptyText, x, y, 8, []textLayer{{0, 0, colMuted}}, 1)
		return
	}
	for _, b := range blocks {
		for _, l := range b {
			drawPixelText(dst, l.text, x, y, 8, []textLayer{{0, 0, l.color}}, 1)
			if l.right != "" {
				w := pixelTextWidth(l.right, 8)
				drawPixelText(dst, l.right, rightX-w, y, 8, []textLayer{{0, 0, l.rightColor}}, 1)
			}
			y += queryRowStep
		}
	}
	if moreCount > 0 {
		drawPixelText(dst, fmt.Sprintf("+ %d MÁS", moreCount), x, y, 8, []textLayer{{0, 0, colMuted}}, 1)
	}
}

// drawHallwayCard: 5.1, dormidos en el pasillo.
func drawHallwayCard(dst *ebiten.Image, hallway []game.PatientView) {
	drawQueryCardChrome(dst, hallwayCardBox, "5.1 PASILLO", "DORMIDOS SIN CAMA")
	blocks, more := fitBlocks(hallwayBlocks(hallway), maxQueryCardLines(hallwayCardBox))
	drawBlockList(dst, hallwayCardBox, blocks, more, "(NADIE: TODOS TIENEN CAMA)")
}

// drawDoctorsCard: 5.2, episodios atendidos por cada médico.
func drawDoctorsCard(dst *ebiten.Image, episodes []game.DoctorEpisodes) {
	drawQueryCardChrome(dst, doctorsCardBox, "5.2 MÉDICOS", "EPISODIOS ATENDIDOS")
	blocks, more := fitBlocks(doctorBlocks(episodes), maxQueryCardLines(doctorsCardBox))
	drawBlockList(dst, doctorsCardBox, blocks, more, "(NINGUNO)")
}

// drawRoomsCard: 5.3, estado de las habitaciones y el último error de
// AssignRoom. Las habitaciones son siempre pocas (§2, 3 habitaciones): no
// hace falta "+ N MÁS" para ellas, así que se dibujan aparte del error.
func drawRoomsCard(dst *ebiten.Image, report game.Report) {
	box := roomsCardBox
	drawQueryCardChrome(dst, box, "5.3 HABITACIONES", "ESTADO Y ÚLTIMO ERROR")
	drawBlockList(dst, box, roomBlocks(report.Rooms), 0, "(SIN HABITACIONES)")
	if report.LastAssignError == "" {
		return
	}
	x := box.Min.X + 12
	label := "ÚLTIMO ERROR DE ASSIGNROOM:"
	drawPixelText(dst, label, x, box.Min.Y+122, 8, []textLayer{{0, 0, colMuted}}, 1)
	avail := box.Max.X - 12 - x
	for i, line := range wrapTwoLines(strings.ToUpper(report.LastAssignError), avail, 8) {
		drawPixelText(dst, line, x, box.Min.Y+150+i*14, 8, []textLayer{{0, 0, colRed}}, 1)
	}
}

// drawSevereCard: 5.4, pacientes Severe con sus episodios de hoy.
func drawSevereCard(dst *ebiten.Image, severe []game.SevereLine) {
	drawQueryCardChrome(dst, severeCardBox, "5.4 SEVEROS", "EPISODIOS EN ESTA PARTIDA")
	blocks, more := fitBlocks(severeBlocks(severe), maxQueryCardLines(severeCardBox))
	drawBlockList(dst, severeCardBox, blocks, more, "(NINGUNO)")
}

// ─── Las cuatro fichas de la economía del día y el puntaje ───────────────

var (
	titleCardBox   = image.Rect(400, 50, 880, 104)
	balanceCardBox = image.Rect(72, 122, 512, 366)
	payrollCardBox = image.Rect(528, 126, 888, 366)
	hiresCardBox   = image.Rect(904, 120, 1212, 366)
	scoreCardBox   = image.Rect(72, 610, 512, 660)
)

// balanceMoneyRight: el borde derecho donde terminan los valores de la
// ficha BALANCE DEL DÍA y el de PUNTAJE (496: el ancho de las dos fichas,
// Figma).
const balanceMoneyRight = 496

// drawMoneyRow dibuja "ETIQUETA" a la izquierda (x, y) y un valor
// alineado a la derecha en right (mismo y).
func drawMoneyRow(dst *ebiten.Image, x, y int, label string, labelColor color.Color, value string, valueColor color.Color, right, size int) {
	drawPixelText(dst, label, x, y, 16, []textLayer{{0, 0, labelColor}}, 1)
	w := pixelTextWidth(value, float64(size))
	drawPixelText(dst, value, right-w, y, float64(size), []textLayer{{0, 0, valueColor}}, 1)
}

// drawTitleCard dibuja "REPORTE DEL DÍA N" centrada en su ficha.
func drawTitleCard(dst *ebiten.Image, day int) {
	panel(dst, titleCardBox, colPaper)
	title := fmt.Sprintf("REPORTE DEL DÍA %d", day)
	x := titleCardBox.Min.X + (titleCardBox.Dx()-pixelTextWidth(title, 24))/2
	drawPixelText(dst, title, x, 70, 24, []textLayer{{0, 0, colInk}}, 1)
}

// expenseAmount es cómo se muestra un gasto del balance (CONTRATACIONES,
// NÓMINA): "$ 0" en gris si no hubo nada, o el monto en rojo con signo.
func expenseAmount(total int) (text string, clr color.Color) {
	if total == 0 {
		return formatMoney(0), colMuted
	}
	return formatMoneySigned(-total), colRed
}

// drawBalanceCard dibuja BALANCE DEL DÍA: el neto del día junto al título,
// el desglose (al empezar, revisiones, altas, contrataciones, nómina) y
// AL FINAL.
func drawBalanceCard(dst *ebiten.Image, r game.DayReport, hiresTotal int) {
	box := balanceCardBox
	panel(dst, box, colPaper)
	drawPixelText(dst, "BALANCE DEL DÍA", box.Min.X+16, box.Min.Y+20, 16, []textLayer{{0, 0, colBlue}}, 1)
	delta := r.EndMoney - r.StartMoney
	deltaText := formatMoneySigned(delta)
	drawPixelText(dst, deltaText, box.Min.X+balanceMoneyRight-pixelTextWidth(deltaText, 16), box.Min.Y+20, 16, []textLayer{{0, 0, moneyColor(delta)}}, 1)
	fillRect(dst, image.Rect(box.Min.X+16, box.Min.Y+46, box.Min.X+balanceMoneyRight, box.Min.Y+48), colPaperShade)

	right := box.Min.X + balanceMoneyRight
	hiresText, hiresColor := expenseAmount(hiresTotal)
	payrollText, payrollColor := expenseAmount(r.PayrollTotal)
	rows := []struct {
		label, value string
		valueColor   color.Color
	}{
		{"AL EMPEZAR", formatMoney(r.StartMoney), colInk},
		{"REVISIONES", formatMoneySigned(r.ReviewIncome), colGreenDark},
		{"ALTAS", formatMoneySigned(r.DischargeIncome), colGreenDark},
		{"CONTRATACIONES", hiresText, hiresColor},
		{"NÓMINA", payrollText, payrollColor},
	}
	for i, row := range rows {
		y := box.Min.Y + 62 + i*24
		drawMoneyRow(dst, box.Min.X+16, y, row.label, colMuted, row.value, row.valueColor, right, 16)
	}
	fillRect(dst, image.Rect(box.Min.X+16, box.Min.Y+186, box.Min.X+balanceMoneyRight, box.Min.Y+189), colInk)
	drawPixelText(dst, "AL FINAL", box.Min.X+16, box.Min.Y+206, 16, []textLayer{{0, 0, colInk}}, 1)
	endText := formatMoney(r.EndMoney)
	drawPixelText(dst, endText, right-pixelTextWidth(endText, 24), box.Min.Y+202, 24, []textLayer{{0, 0, moneyColor(r.EndMoney)}}, 1)
}

// drawPayrollCard dibuja NÓMINA: una fila por cada miembro del personal
// (nombre, rol con género y sueldo) y el TOTAL pagado.
func drawPayrollCard(dst *ebiten.Image, box image.Rectangle, payroll []game.PayrollLine, total int) {
	panel(dst, box, colPaper)
	drawPixelText(dst, "NÓMINA", box.Min.X+16, box.Min.Y+20, 16, []textLayer{{0, 0, colBlue}}, 1)
	note := "PAGADA A LAS 20:00"
	drawPixelText(dst, note, box.Min.X+200, box.Min.Y+24, 8, []textLayer{{0, 0, colMuted}}, 1)
	fillRect(dst, image.Rect(box.Min.X+16, box.Min.Y+46, box.Max.X-16, box.Min.Y+48), colPaperShade)

	roleX, salaryRight := box.Min.X+200, box.Max.X-16
	for i, p := range payroll {
		y := box.Min.Y + 60 + i*20
		drawPixelText(dst, strings.ToUpper(p.Name), box.Min.X+16, y, 8, []textLayer{{0, 0, colInk}}, 1)
		roleColor := colGreenDark
		if p.IsDoctor {
			roleColor = colBlue
		}
		drawPixelText(dst, strings.ToUpper(p.RoleLabel), roleX, y, 8, []textLayer{{0, 0, roleColor}}, 1)
		salary := formatMoney(p.Salary)
		drawPixelText(dst, salary, salaryRight-pixelTextWidth(salary, 8), y, 8, []textLayer{{0, 0, colInk}}, 1)
	}
	fillRect(dst, image.Rect(box.Min.X+16, box.Min.Y+190, box.Max.X-16, box.Min.Y+193), colInk)
	totalText, totalColor := formatMoney(total), colRed
	if total == 0 {
		totalColor = colMuted
	}
	drawPixelText(dst, "TOTAL", box.Min.X+16, box.Min.Y+206, 16, []textLayer{{0, 0, colInk}}, 1)
	drawPixelText(dst, totalText, salaryRight-pixelTextWidth(totalText, 16), box.Min.Y+206, 16, []textLayer{{0, 0, totalColor}}, 1)
}

// drawHiresCard dibuja CONTRATADOS: dos renglones por cada contratación de
// hoy (nombre; rol con género y costo) y el TOTAL.
func drawHiresCard(dst *ebiten.Image, box image.Rectangle, hires []game.HireLine, total int) {
	panel(dst, box, colReportYellow)
	drawPixelText(dst, "CONTRATADOS", box.Min.X+16, box.Min.Y+20, 16, []textLayer{{0, 0, colBlue}}, 1)
	drawPixelText(dst, "HOY, PAGADOS POR ADELANTADO", box.Min.X+16, box.Min.Y+42, 8, []textLayer{{0, 0, colMuted}}, 1)
	fillRect(dst, image.Rect(box.Min.X+16, box.Min.Y+56, box.Max.X-16, box.Min.Y+58), colSand)

	right := box.Max.X - 16
	if len(hires) == 0 {
		drawPixelText(dst, "(NINGUNO)", box.Min.X+16, box.Min.Y+70, 8, []textLayer{{0, 0, colMuted}}, 1)
	}
	for i, h := range hires {
		y := box.Min.Y + 70 + i*32
		drawPixelText(dst, strings.ToUpper(h.Name), box.Min.X+16, y, 8, []textLayer{{0, 0, colInk}}, 1)
		roleColor := colGreenDark
		if h.IsDoctor {
			roleColor = colBlue
		}
		drawPixelText(dst, strings.ToUpper(h.RoleLabel), box.Min.X+16, y+12, 8, []textLayer{{0, 0, roleColor}}, 1)
		cost := formatMoneySigned(-h.Cost)
		drawPixelText(dst, cost, right-pixelTextWidth(cost, 8), y+12, 8, []textLayer{{0, 0, colRed}}, 1)
	}
	fillRect(dst, image.Rect(box.Min.X+16, box.Min.Y+196, box.Max.X-16, box.Min.Y+199), colInk)
	totalText, totalColor := formatMoney(0), colMuted
	if total != 0 {
		totalText, totalColor = formatMoneySigned(-total), colRed
	}
	drawPixelText(dst, "TOTAL", box.Min.X+16, box.Min.Y+212, 16, []textLayer{{0, 0, colInk}}, 1)
	drawPixelText(dst, totalText, right-pixelTextWidth(totalText, 16), box.Min.Y+212, 16, []textLayer{{0, 0, totalColor}}, 1)
}

// hiresTotal suma el costo de las contrataciones de hoy.
func hiresTotal(hires []game.HireLine) int {
	total := 0
	for _, h := range hires {
		total += h.Cost
	}
	return total
}

// drawScoreCard dibuja PUNTAJE (el acumulado, no el final: ese solo existe
// con la partida perdida, gameover_panel.go).
func drawScoreCard(dst *ebiten.Image, score int) {
	box := scoreCardBox
	panel(dst, box, colPaper)
	drawPixelText(dst, "PUNTAJE", box.Min.X+16, box.Min.Y+18, 16, []textLayer{{0, 0, colMuted}}, 1)
	value := thousands(score)
	right := box.Min.X + balanceMoneyRight
	drawPixelText(dst, value, right-pixelTextWidth(value, 24), box.Min.Y+14, 24, []textLayer{{0, 0, colInk}}, 1)
}

// drawNextDayButton dibuja el botón SIGUIENTE DÍA (la acción, con la
// partida en pausa, la maneja GameScene.updateDayOver en game_scene.go).
func drawNextDayButton(dst *ebiten.Image) {
	panel(dst, dayOverButton, colBlue)
	const label = "SIGUIENTE DÍA"
	x := dayOverButton.Min.X + (dayOverButton.Dx()-pixelTextWidth(label, 24))/2
	drawPixelText(dst, label, x, dayOverButton.Min.Y+16, 24, []textLayer{{0, 0, colWhite}}, 1)
}

// drawDayOverPanel dibuja el Reporte del día entero (Figma "Reporte A/B",
// §9): el tablero de corcho y, encima, la ficha del título, el balance, la
// nómina, los contratados, las cuatro consultas de la Sección 5, el
// puntaje y el botón. day es el Shift Report del día que acaba de cerrar
// (Game.DayReport()); score es el puntaje acumulado AHORA (Snapshot.Score:
// puede haber seguido subiendo si el jugador sigue mirando el panel antes
// de apretar SIGUIENTE DÍA, aunque en la práctica el reloj está en pausa).
func drawDayOverPanel(dst *ebiten.Image, day game.DayReport, score int) {
	// No debería fallar (el PNG va embebido): si fallara, se sigue sin el
	// corcho de fondo en vez de dejar la pantalla en blanco o reventar.
	if err := loadReportPanel(); err == nil {
		drawImageAt(dst, reportCorkboardImg, reportCorkboardAt, nil)
	}

	hTotal := hiresTotal(day.Hires)
	drawTitleCard(dst, day.Day)
	drawBalanceCard(dst, day, hTotal)
	drawPayrollCard(dst, payrollCardBox, day.Payroll, day.PayrollTotal)
	drawHiresCard(dst, hiresCardBox, day.Hires, hTotal)
	drawHallwayCard(dst, day.Report.Hallway)
	drawDoctorsCard(dst, day.Report.Episodes)
	drawRoomsCard(dst, day.Report)
	drawSevereCard(dst, day.Report.Severe)
	drawScoreCard(dst, score)
	drawNextDayButton(dst)
}
