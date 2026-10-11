package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// pausable es una escena que se puede pausar con Esc: la demo y, en F3, el
// juego. Otra interfaz implícita: basta con tener estos métodos.
//
// Mientras la pausa está abierta la escena no recibe Update, así que ya se
// queda quieta sola; los avisos sirven para lo que corre por fuera de la
// escena (en F3, la goroutine motor del juego).
type pausable interface {
	Scene
	onPause()  // se abrió la pausa (F3: Game.Pause)
	onResume() // se cerró la pausa y la escena sigue (F3: Game.Resume)
	onQuit()   // el jugador salió al menú y la escena se descarta (F3: cancela el context)
}

// PauseScene es el menú de pausa (GAME_DESIGN §9.4): una capa encima de la
// escena pausada, que se sigue viendo quieta y oscurecida.
type PauseScene struct {
	under  pausable // la escena pausada
	menu   menu
	notice int // ticks que le quedan al aviso "Próximamente"; 0 = no hay aviso
}

// Medidas de PAUSE_ASSETS.md: el portapapeles a 1× con la esquina en
// (442, 213) y, detrás, un negro con alfa 140 (≈55 %).
var (
	pauseAt  = image.Pt(442, 213)
	pauseDim = color.RGBA{0, 0, 0, 140}
)

// pauseImgs son los 6 portapapeles, por ruta; pauseHitboxes, sus renglones
// para el mouse. Los carga loadPause.
var (
	pauseImgs     = map[string]*ebiten.Image{}
	pauseHitboxes hitboxes
)

// loadPause carga los portapapeles de la pausa y sus renglones.
func loadPause() error {
	for _, o := range pauseOptions {
		for frame := 0; frame < 2; frame++ {
			p := pausePath(o, frame)
			img, err := decodePNG(assets.Pause, p)
			if err != nil {
				return err
			}
			pauseImgs[p] = ebiten.NewImageFromImage(img)
		}
	}
	boxes, err := loadPauseHitboxes()
	if err != nil {
		return err
	}
	pauseHitboxes = boxes
	return nil
}

// newPauseScene pausa under y abre el menú en REANUDAR.
func newPauseScene(under pausable) *PauseScene {
	under.onPause()
	return &PauseScene{under: under, menu: newPauseMenu()}
}

// Update lee el teclado y el mouse y avanza el menú de pausa.
func (s *PauseScene) Update() (Scene, error) { return s.update(readInput()) }

// update hace el trabajo de Update con la entrada ya leída (los tests la
// arman a mano).
func (s *PauseScene) update(in input) (Scene, error) {
	if s.notice > 0 {
		s.notice--
	}
	// Esc es REANUDAR sin esperar al cursor, salvo que ya se esté
	// confirmando otra opción: entonces el menú ya no escucha.
	if in.back && s.menu.confirming == 0 {
		s.under.onResume()
		return s.under, nil
	}
	chosen, done := s.menu.update(in, pauseHitboxes)
	if !done {
		return s, nil
	}
	switch chosen {
	case optResume:
		s.under.onResume()
		return s.under, nil
	case optSettings: // AJUSTES llega en F6
		s.notice = comingSoonTicks
	case optToTitle:
		s.under.onQuit()
		return NewTitleScene(), nil
	}
	return s, nil
}

// Draw dibuja la escena pausada tal como quedó, el negro encima, el
// portapapeles y, debajo de él, el aviso "Próximamente" mientras dure.
func (s *PauseScene) Draw(screen *ebiten.Image) {
	s.under.Draw(screen)
	fillRect(screen, image.Rect(0, 0, ScreenWidth, ScreenHeight), pauseDim)

	img := pauseImgs[s.menu.image()]
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(pauseAt.X), float64(pauseAt.Y))
	screen.DrawImage(img, op)

	if s.notice > 0 {
		size := img.Bounds().Size()
		x := pauseAt.X + size.X/2 - pixelTextWidth(comingSoonText, 16)/2
		drawPixelText(screen, comingSoonText, x, pauseAt.Y+size.Y+24, 16, append(outlined(4, colInk), textLayer{0, 0, colWhite}), 1)
	}
}
