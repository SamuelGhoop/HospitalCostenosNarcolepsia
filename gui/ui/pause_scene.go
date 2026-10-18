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
//
// Desde el 2026-10-17 (pedido de Samuel: "hazle así animados los menús
// como el de contratar que me gustó mucho") abre y cierra con la misma
// animación de portapapeles que la bolsa de empleo (anim.go, extraída de
// hiring_scene.go): mientras anima no acepta teclado ni mouse, y la acción
// pendiente (reanudar o salir al menú) se hace al terminar de cerrar, no
// al elegirla.
type PauseScene struct {
	under  pausable // la escena pausada
	menu   menu
	notice int // ticks que le quedan al aviso "Próximamente"; 0 = no hay aviso

	anim        *animPanel
	closeAction func() (Scene, error) // qué hacer cuando termine de cerrar
}

// Medidas de PAUSE_ASSETS.md: el portapapeles a 1× con la esquina en
// (442, 213), de 407×311 (TestPauseAssets_SixClipboardsOf407x311), y,
// detrás, un negro con alfa 140 (≈55 %).
var (
	pauseAt  = image.Pt(442, 213)
	pauseDim = color.RGBA{0, 0, 0, 140}

	pauseOpenCenter = vec{float64(pauseAt.X) + 407.0/2, float64(pauseAt.Y) + 311.0/2}
)

// Tiempos de la animación de la pausa, los mismos que la bolsa de empleo
// (hiring_scene.go: hiringOpenTicks/hiringCloseTicks).
const (
	pauseOpenTicks  = 13 // 0,22 s
	pauseCloseTicks = 11 // 0,18 s
	pauseMinScale   = 0.167
)

// pauseAnimFrom es desde dónde aparece el portapapeles de la pausa: del
// botón PAUSA del HUD si la escena pausada es el modo juego (tiene ese
// botón); del centro de la pantalla si no (la demo, que no tiene HUD).
func pauseAnimFrom(under pausable) vec {
	if _, ok := under.(*GameScene); ok {
		c := pauseButton.Min.Add(pauseButton.Max).Div(2)
		return vec{float64(c.X), float64(c.Y)}
	}
	return vec{640, 360}
}

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

// newPauseScene pausa under y arranca abriendo el portapapeles (en
// REANUDAR) desde pauseAnimFrom.
func newPauseScene(under pausable) *PauseScene {
	under.onPause()
	return &PauseScene{
		under: under, menu: newPauseMenu(),
		anim: newAnimPanel(pauseOpenTicks, pauseCloseTicks, pauseAnimFrom(under), pauseOpenCenter, pauseMinScale),
	}
}

// Update lee el teclado y el mouse y avanza el menú de pausa.
func (s *PauseScene) Update() (Scene, error) { return s.update(readInput()) }

// update hace el trabajo de Update con la entrada ya leída (los tests la
// arman a mano).
func (s *PauseScene) update(in input) (Scene, error) {
	if !s.anim.acceptsInput() { // abriendo o cerrando: no se le hace caso al teclado ni al mouse
		if s.anim.step() { // terminó de cerrar: ahí es cuando se hace la acción pendiente
			return s.closeAction()
		}
		return s, nil
	}
	if s.notice > 0 {
		s.notice--
	}
	// Esc es REANUDAR sin esperar al cursor, salvo que ya se esté
	// confirmando otra opción: entonces el menú ya no escucha.
	if in.back && s.menu.confirming == 0 {
		s.startClose(func() (Scene, error) { s.under.onResume(); return s.under, nil })
		return s, nil
	}
	chosen, done := s.menu.update(in, pauseHitboxes)
	if !done {
		return s, nil
	}
	switch chosen {
	case optResume:
		s.startClose(func() (Scene, error) { s.under.onResume(); return s.under, nil })
	case optSettings: // AJUSTES llega en F6
		s.notice = comingSoonTicks
	case optToTitle:
		s.startClose(func() (Scene, error) { s.under.onQuit(); return NewTitleScene(), nil })
	}
	return s, nil
}

// startClose deja action pendiente y arranca el cierre animado: action se
// llama en el tick en que termina (s.anim.step() devuelve true).
func (s *PauseScene) startClose(action func() (Scene, error)) {
	s.closeAction = action
	s.anim.startClose()
}

// Draw dibuja la escena pausada tal como quedó, el negro encima (según lo
// abierto que esté el portapapeles) y el portapapeles escalado y centrado
// con la animación (drawAnimatedPanel, anim.go).
func (s *PauseScene) Draw(screen *ebiten.Image) {
	s.under.Draw(screen)
	drawAnimatedPanel(screen, s.anim, pauseDim.A, func(dst *ebiten.Image) {
		img := pauseImgs[s.menu.image()]
		drawImageAt(dst, img, pauseAt, nil)
		if s.notice > 0 {
			size := img.Bounds().Size()
			x := pauseAt.X + size.X/2 - pixelTextWidth(comingSoonText, 16)/2
			drawPixelText(dst, comingSoonText, x, pauseAt.Y+size.Y+24, 16, append(outlined(4, colInk), textLayer{0, 0, colWhite}), 1)
		}
	})
}
