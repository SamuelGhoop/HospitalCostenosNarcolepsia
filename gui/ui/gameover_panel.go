package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// El Game Over (§5.10, §9, drawGameOver en game_scene.go): vista aprobada
// en Figma ("Game Over C1 – perdió la licencia" 17:226, "C2 – quiebra"
// 17:227, "C3 – hospital colapsado" 18:348), un motivo por cada
// game.GameOver. El portapapeles es el mismo PNG de la bolsa de empleo
// (hiring_board.png, ya cargado por loadHiring: Figma reusa esa capa con
// el nombre "hiring_board.png" en los tres frames, así que acá no hace
// falta un PNG propio).
//
// Dos cosas del Figma no se copiaron tal cual (quedan en el reporte de la
// tarea, no en el código): el oscurecido de fondo de dos de los tres
// frames dice "alfa 170/255" en el NOMBRE de la capa pero su color real es
// el mismo de alfa 140 que ya usa el resto de los paneles (pauseDim); acá
// se dejó pauseDim para los tres, en vez de inventar un cuarto valor a
// partir de un nombre que no coincide con su propio color. Y el frame C2
// (quiebra) tiene, de la ficha PUNTAJE para abajo, un bloque entero en
// gris plano (#d9d9d9) con texto negro puro (#000000, no el #141414 de la
// paleta): es el mismo color "recién creado, sin estilo" que ya aparece
// duplicado en ese frame (GAME_DESIGN, "17:227 tiene un bloque duplicado
// al final: ignóralo"), así que acá los tres motivos usan los mismos
// colores de C1/C3 (que sí están bien estilados y coinciden entre sí).

// ─── Íconos del motivo ───────────────────────────────────────────────────

var gameOverIcons = map[string]*ebiten.Image{}

// gameOverIconName es el PNG del ícono según el motivo (32×32; Figma lo
// dibuja ×3, como el resto de los sprites del juego).
func gameOverIconName(over game.GameOver) string {
	switch over {
	case game.LostLicense:
		return "gameover_icon_licencia.png"
	case game.Bankrupt:
		return "gameover_icon_quiebra.png"
	case game.HospitalCollapsed:
		return "gameover_icon_colapsado.png"
	default:
		return ""
	}
}

// loadGameOverIcons carga los tres PNG, perezoso (ver report_panel.go:
// app.go lo está tocando otra sesión ahora mismo, así que este panel no se
// suma a loadAssets y se las arregla solo la primera vez que hace falta).
func loadGameOverIcons() error {
	if len(gameOverIcons) > 0 {
		return nil
	}
	for _, over := range []game.GameOver{game.LostLicense, game.Bankrupt, game.HospitalCollapsed} {
		name := gameOverIconName(over)
		img, err := decodePNG(assets.GameOver, "gameover/"+name)
		if err != nil {
			return err
		}
		gameOverIcons[name] = ebiten.NewImageFromImage(img)
	}
	return nil
}

// gameOverIconAt y su tamaño en pantalla: Figma "ícono ×3", 96×96 en
// (592, 196) en los tres frames.
var gameOverIconAt = image.Pt(592, 196)

const gameOverIconScale = 3 // el PNG va a 32×32; en pantalla, ×3 = 96×96

// ─── El motivo (§5.10) ────────────────────────────────────────────────────

// gameOverReason es el renglón bajo el sello ("LA REPUTACIÓN LLEGÓ A 0
// ESTRELLAS", "LA NÓMINA DEJÓ LA PLATA EN -$ 340", "5 DORMIDOS SIN ATENDER
// A LA VEZ"): el mismo texto de las vistas de Figma, con los valores
// reales de esta partida en vez de los de ejemplo.
func gameOverReason(over game.GameOver, money int) string {
	switch over {
	case game.LostLicense:
		return "LA REPUTACIÓN LLEGÓ A 0 ESTRELLAS"
	case game.Bankrupt:
		return fmt.Sprintf("LA NÓMINA DEJÓ LA PLATA EN %s", formatMoney(money))
	case game.HospitalCollapsed:
		return fmt.Sprintf("%d DORMIDOS SIN ATENDER A LA VEZ", game.CollapseLimit)
	default:
		return ""
	}
}

// daysSurvived (§9, "DÍAS SOBREVIVIDOS"): la quiebra se dispara en la
// nómina de las 20:00, así que ese día SÍ terminó completo (el jugador lo
// sobrevivió entero, solo que la plata quedó en negativo). La reputación
// en 0 y el colapso, en cambio, pueden pasar a mitad del día: ese día que
// estaba en curso no se cuenta como sobrevivido.
func daysSurvived(day int, over game.GameOver) int {
	if over == game.Bankrupt {
		return day
	}
	return day - 1
}

// ─── Dibujo ────────────────────────────────────────────────────────────

// gameOverColumnLeft, gameOverColumnRight: la columna de texto del
// portapapeles (334 a 946 en los tres frames de Figma).
const (
	gameOverColumnLeft  = 334
	gameOverColumnRight = 946
)

// drawGameOverRow dibuja una fila "ETIQUETA ... valor", con la etiqueta a
// la izquierda (colMuted) y el valor alineado a la derecha en
// gameOverColumnRight.
func drawGameOverRow(dst *ebiten.Image, y int, label string, value string, valueColor color.Color) {
	drawPixelText(dst, label, gameOverColumnLeft, y, 16, []textLayer{{0, 0, colMuted}}, 1)
	drawPixelText(dst, value, gameOverColumnRight-pixelTextWidth(value, 16), y, 16, []textLayer{{0, 0, valueColor}}, 1)
}

// drawGameOverPanel dibuja el Game Over entero (Figma "Game Over C1/C2/
// C3", §9, §5.10): el portapapeles, el ícono del motivo, el sello con su
// nombre, el motivo, los días sobrevividos, el desglose del puntaje final
// y los dos botones. snap es la foto de la partida perdida (GameOver,
// Day, Score, FinalScore, MoneyScore, StarScore, Money); las acciones de
// los botones las maneja GameScene.updateGameOver (game_scene.go).
func drawGameOverPanel(dst *ebiten.Image, snap game.Snapshot) {
	drawImageAt(dst, hiringImgs["hiring_board.png"], hiringBoardAt, nil)

	if err := loadGameOverIcons(); err == nil {
		icon := gameOverIcons[gameOverIconName(snap.GameOver)]
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
		op.GeoM.Scale(gameOverIconScale, gameOverIconScale)
		op.GeoM.Translate(float64(gameOverIconAt.X), float64(gameOverIconAt.Y))
		dst.DrawImage(icon, op)
	}

	drawPixelText(dst, "FIN DE LA PARTIDA", gameOverColumnLeft, 158, 16, []textLayer{{0, 0, colBlue}}, 1)
	day := fmt.Sprintf("DÍA %d", snap.Day)
	drawPixelText(dst, day, gameOverColumnRight-pixelTextWidth(day, 16), 158, 16, []textLayer{{0, 0, colMuted}}, 1)
	fillRect(dst, image.Rect(gameOverColumnLeft, 184, gameOverColumnRight, 187), colInk)

	drawStamp(dst, snap.GameOver.String())
	// El motivo va a una sola línea en la vista aprobada (una plata de
	// ejemplo modesta, "-$ 340"), pero una quiebra con números grandes
	// podría no caber: se parte en dos renglones si hace falta
	// (wrapTwoLines, igual que los subtítulos de la demo), cada uno
	// centrado por su cuenta.
	reason := gameOverReason(snap.GameOver, snap.Money)
	for i, line := range wrapTwoLines(reason, gameOverColumnRight-gameOverColumnLeft, 16) {
		drawPixelText(dst, line, 640-pixelTextWidth(line, 16)/2, 358+i*18, 16, []textLayer{{0, 0, colMuted}}, 1)
	}

	fillRect(dst, image.Rect(gameOverColumnLeft, 386, gameOverColumnRight, 388), colPaperShade)
	drawGameOverRow(dst, 396, "DÍAS SOBREVIVIDOS", fmt.Sprint(daysSurvived(snap.Day, snap.GameOver)), colInk)
	drawGameOverRow(dst, 418, "PUNTAJE", thousands(snap.Score), colInk)
	drawGameOverRow(dst, 440, "PLATA ÷ 10", formatSignedInt(snap.MoneyScore), moneyColor(snap.MoneyScore))
	drawGameOverRow(dst, 462, "ESTRELLAS × 200", formatSignedInt(snap.StarScore), moneyColor(snap.StarScore))
	fillRect(dst, image.Rect(gameOverColumnLeft, 482, gameOverColumnRight, 485), colInk)

	final := thousands(snap.FinalScore)
	drawPixelText(dst, "PUNTAJE FINAL", gameOverColumnLeft, 498, 16, []textLayer{{0, 0, colInk}}, 1)
	drawPixelText(dst, final, gameOverColumnRight-pixelTextWidth(final, 24), 494, 24, []textLayer{{0, 0, colInk}}, 1)

	panel(dst, newGameButton, colGreen)
	drawCenteredButtonLabel(dst, newGameButton, "NUEVA PARTIDA")
	panel(dst, menuButton, colBlue)
	drawCenteredButtonLabel(dst, menuButton, "MENÚ")
}

// drawCenteredButtonLabel centra s (24 px, blanco) dentro de r, 16 px bajo
// su borde de arriba (Figma: los dos botones del Game Over).
func drawCenteredButtonLabel(dst *ebiten.Image, r image.Rectangle, s string) {
	x := r.Min.X + (r.Dx()-pixelTextWidth(s, 24))/2
	drawPixelText(dst, s, x, r.Min.Y+16, 24, []textLayer{{0, 0, colWhite}}, 1)
}

// drawStamp dibuja el sello rojo con el motivo ("¡PERDIÓ LA LICENCIA!",
// "¡QUIEBRA!", "¡HOSPITAL COLAPSADO!"), centrado en x=640: el papel mide
// el texto + 16 px de cada lado, y el borde 4 px más por lado (medidas
// grounded: Figma "¡PERDIO LA LICENCIA!" da 512×36 de papel con esta
// fórmula, y "¡QUIEBRA!" da 248×36).
func drawStamp(dst *ebiten.Image, title string) {
	const cx = 640
	tw := pixelTextWidth(title, 24)
	papelW := tw + 32
	borderW := papelW + 8
	border := image.Rect(cx-borderW/2, 300, cx+borderW/2, 344)
	papel := image.Rect(cx-papelW/2, 304, cx+papelW/2, 340)
	fillRect(dst, border, colRed)
	fillRect(dst, papel, colPaper)
	drawPixelText(dst, title, cx-tw/2, 311, 24, []textLayer{{0, 0, colRed}}, 1)
}
