package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// CreditsScene son los créditos: el mismo paisaje de la portada con un
// panel de papel encima. Enter o Esc vuelven a la portada.
type CreditsScene struct {
	ticks int
}

// NewCreditsScene arma los créditos.
func NewCreditsScene() *CreditsScene { return &CreditsScene{} }

// Update lee el teclado y avanza los créditos.
func (s *CreditsScene) Update() (Scene, error) { return s.update(readInput()) }

func (s *CreditsScene) update(in input) (Scene, error) {
	s.ticks++
	if in.confirm || in.back {
		return NewTitleScene(), nil
	}
	return s, nil
}

// Draw dibuja el paisaje y el panel con los créditos centrados.
func (s *CreditsScene) Draw(screen *ebiten.Image) {
	drawBeach(screen, s.ticks)

	paper := image.Rect(220, 160, 1060, 560)
	fillRect(screen, paper.Add(image.Pt(8, 10)), colDrop)
	fillRect(screen, paper, colInk)
	fillRect(screen, paper.Inset(6), colPaper)

	center := func(t string, size float64) int { return ScreenWidth/2 - pixelTextWidth(t, size)/2 }
	drawPixelText(screen, creditsTitle, center(creditsTitle, 24), 200, 24, []textLayer{{0, 0, colBlue, false}}, 1)
	for i, line := range creditsLines {
		drawPixelText(screen, line, center(line, 16), 268+i*44, 16, []textLayer{{0, 0, colInk, false}}, 1)
	}
	drawPixelText(screen, creditsHint, center(creditsHint, 8), 528, 8, []textLayer{{0, 0, colMuted, false}}, 1)
}
