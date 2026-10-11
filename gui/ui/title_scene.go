package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// TitleScene es la portada (GAME_DESIGN §9.3): el paisaje animado, el logo
// y el menú en el portapapeles.
type TitleScene struct {
	menu   menu
	ticks  int // para las animaciones del paisaje y del logo
	notice int // ticks que le quedan al aviso "Próximamente"; 0 = no hay aviso
}

// comingSoonTicks: el aviso "Próximamente" dura 1,5 s.
const comingSoonTicks = 90

// clipboardAt: el portapapeles se dibuja a 1× con la esquina aquí
// (TITLE_ASSETS.md).
var clipboardAt = image.Pt(797, 241)

// titleHitboxes son los renglones del portapapeles para el mouse (los
// carga loadTitle; si faltan, el menú solo se usa con el teclado).
var titleHitboxes hitboxes

// NewTitleScene arma la portada. CONTINUAR va en gris: todavía no hay
// partidas guardadas (llegan en F6).
func NewTitleScene() *TitleScene {
	return &TitleScene{menu: newTitleMenu(false)}
}

// Update lee el teclado y el mouse y avanza la portada.
func (s *TitleScene) Update() (Scene, error) { return s.update(readInput()) }

// update hace el trabajo de Update con la entrada ya leída (los tests la
// arman a mano).
func (s *TitleScene) update(in input) (Scene, error) {
	s.ticks++
	if s.notice > 0 {
		s.notice--
	}
	chosen, done := s.menu.update(in, titleHitboxes)
	if !done {
		return s, nil
	}
	switch chosen {
	case optNew, optSettings: // NUEVA PARTIDA llega en F3 y AJUSTES en F6
		s.notice = comingSoonTicks
	case optDemo:
		return NewDemoScene(game.NewDemo()), nil
	case optCredits:
		return NewCreditsScene(), nil
	case optExit:
		return s, ebiten.Termination // RunGame termina sin error: cierre limpio
	}
	return s, nil
}

// Colores del logo (maqueta Title Screen).
var (
	colLogoYellow = color.RGBA{0xff, 0xd2, 0x3f, 0xff}
	colLogoRed    = color.RGBA{0xc4, 0x30, 0x2b, 0xff}
	colDrop       = color.RGBA{0, 0, 0, 0x38} // sombra al 22 %
)

// Draw dibuja la portada por capas: paisaje → logo → portapapeles → textos.
func (s *TitleScene) Draw(screen *ebiten.Image) {
	drawBeach(screen, s.ticks)
	drawLogo(screen, s.ticks)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(clipboardAt.X), float64(clipboardAt.Y))
	screen.DrawImage(titleImgs[s.menu.image()], op)

	// Abajo a la izquierda, centrado en el 61 % de la pantalla: "PRESIONA
	// ENTER" parpadeando, o el aviso "Próximamente" mientras dure.
	msg, show := pressEnterText, pressEnterVisible(s.ticks)
	if s.notice > 0 {
		msg, show = comingSoonText, true
	}
	if show {
		x := ScreenWidth*61/100/2 - pixelTextWidth(msg, 16)/2
		drawPixelText(screen, msg, x, 664, 16, append(outlined(4, colInk), textLayer{0, 0, colWhite}), 1)
	}
	drawPixelText(screen, versionText, ScreenWidth-32-pixelTextWidth(versionText, 8), 687, 8, []textLayer{{0, 0, colInk}}, 1)
}

// drawLogo dibuja "HOSPITAL DE LOS", "COSTEÑOS", la banda azul con la
// almohada y "con Narcolepsia", y las Zzz que suben a su derecha.
func drawLogo(dst *ebiten.Image, ticks int) {
	center := func(s string, size float64) int { return ScreenWidth/2 - pixelTextWidth(s, size)/2 }

	// HOSPITAL DE LOS: blanco, contorno negro, un borde negro abajo y sombra.
	top := []textLayer{{5, 13, colDrop}}
	for _, dx := range []int{-4, 0, 4} {
		top = append(top, textLayer{dx, 8, colInk})
	}
	top = append(top, outlined(4, colInk)...)
	drawPixelText(dst, logoTop, center(logoTop, 40), 14, 40, append(top, textLayer{0, 0, colWhite}), 1)

	drawPixelText(dst, logoMain, center(logoMain, 72), 104, 72, logoMainLayers(), 1)

	// La banda: azul con borde negro, un poco más oscura abajo y con sombra.
	const bandW, bandH, bandY = 349, 63, 212
	band := image.Rect(ScreenWidth/2-bandW/2, bandY, ScreenWidth/2-bandW/2+bandW, bandY+bandH)
	fillRect(dst, band.Add(image.Pt(6, 9)), colDrop)
	fillRect(dst, band, colInk)
	inner := band.Inset(6)
	fillRect(dst, inner, colBlue)
	fillRect(dst, image.Rect(inner.Min.X, inner.Max.Y-6, inner.Max.X, inner.Max.Y), colBandShade)
	drawPillow(dst, image.Pt(inner.Min.X+23, inner.Min.Y+12))
	drawPixelText(dst, logoBand, inner.Min.X+23+36+15, inner.Min.Y+16, 16, []textLayer{{0, 0, colWhite}}, 1)

	// Las Zzz: z, Z, Z, cada una sube y se desvanece en un ciclo de 2,4 s de
	// 6 pasos, desfasadas 0,8 s.
	letters := []struct {
		s          string
		size       float64
		dx, bottom int
	}{{"z", 16, 8, -26}, {"Z", 16, 52, -18}, {"Z", 24, 116, 0}}
	for k, l := range letters {
		step := (ticks + k*48) % 144 / 24
		dy := int(l.size * (0.4 - float64(step)/6))
		x, y := band.Max.X+l.dx, bandY-l.bottom-int(l.size)+dy
		drawPixelText(dst, l.s, x, y, l.size, append(outlined(3, colInk), textLayer{0, 0, colWhite}), stepFade(step))
	}
}

// logoOutline: el contorno negro de "COSTEÑOS", en px.
const logoOutline = 6

// logoMainLayers son las pasadas de "COSTEÑOS": sombra, relieve rojo con
// borde negro, contorno negro y el relleno amarillo. Como en la maqueta, la
// virgulilla de la Ñ lleva las mismas pasadas que las letras.
func logoMainLayers() []textLayer {
	layers := []textLayer{{8, 28, colDrop}}
	for _, dx := range []int{-logoOutline, 0, logoOutline} {
		layers = append(layers, textLayer{dx, 20, colInk}, textLayer{dx, 14, colInk})
	}
	layers = append(layers, textLayer{0, 14, colLogoRed}, textLayer{0, 10, colLogoRed})
	layers = append(layers, outlined(logoOutline, colInk)...)
	return append(layers, textLayer{0, 0, colLogoYellow})
}

// colBandShade: el azul oscuro de abajo de la banda.
var colBandShade = color.RGBA{0x21, 0x52, 0xa8, 0xff}

// drawPillow dibuja la almohada de la banda (el svg de 18×12 de la maqueta,
// ×2) con su esquina en at.
func drawPillow(dst *ebiten.Image, at image.Point) {
	gray := color.RGBA{0xc4, 0xca, 0xd6, 0xff}
	rects := []struct {
		r image.Rectangle
		c color.Color
	}{
		{image.Rect(2, 1, 16, 11), colInk},
		{image.Rect(0, 0, 3, 3), colInk}, {image.Rect(15, 0, 18, 3), colInk},
		{image.Rect(0, 9, 3, 12), colInk}, {image.Rect(15, 9, 18, 12), colInk},
		{image.Rect(2, 2, 16, 10), colWhite},
		{image.Rect(1, 1, 3, 3), colWhite}, {image.Rect(15, 1, 17, 3), colWhite},
		{image.Rect(1, 9, 3, 11), colWhite}, {image.Rect(15, 9, 17, 11), colWhite},
		{image.Rect(3, 8, 15, 10), gray}, {image.Rect(14, 3, 16, 9), gray}, {image.Rect(15, 9, 17, 11), gray},
		{image.Rect(6, 5, 9, 6), gray}, {image.Rect(9, 6, 11, 7), gray},
	}
	for _, p := range rects {
		r := image.Rect(p.r.Min.X*2, p.r.Min.Y*2, p.r.Max.X*2, p.r.Max.Y*2).Add(at)
		fillRect(dst, r, p.c)
	}
}
