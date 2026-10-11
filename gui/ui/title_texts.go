package ui

// Los textos de la portada y de los créditos, en un solo sitio: los dibuja
// Press Start 2P y un test revisa que la fuente tenga todos sus caracteres.
const (
	logoTop        = "HOSPITAL DE LOS"
	logoMain       = "COSTEÑOS"
	logoBand       = "con Narcolepsia"
	pressEnterText = "PRESIONA ENTER"
	versionText    = "v1.0 · Universidad EIA · POO en Go"
	comingSoonText = "Próximamente"

	creditsTitle = "CRÉDITOS"
	creditsHint  = "ENTER o ESC: volver"
)

// creditsLines son los renglones de los créditos, tal como los pidió Samuel.
var creditsLines = []string{
	"Hospital de los Costeños con Narcolepsia",
	"Samuel Giraldo",
	"Código con ayuda de Claude",
	"Fuente Press Start 2P (OFL)",
	"Motor: Ebitengine",
}

// pixelTexts son todos los textos que se escriben con Press Start 2P.
func pixelTexts() []string {
	return append([]string{logoTop, logoMain, logoBand, pressEnterText, versionText, comingSoonText,
		creditsTitle, creditsHint, "zZ"}, creditsLines...)
}
