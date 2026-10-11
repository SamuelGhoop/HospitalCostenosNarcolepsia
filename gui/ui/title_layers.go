package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// El paisaje de la portada (gui/assets/title/TITLE_ASSETS.md): capas de
// 320×180 que se dibujan ×4 para llenar 1280×720. Cada animación es una
// función de los ticks que pasaron (60 por segundo): no guarda estado, así
// que no se puede desincronizar.

// titleScale: la escena nativa es de 320×180; se dibuja ×4.
const titleScale = 4

// msOf pasa ticks a milisegundos.
func msOf(ticks int) int { return ticks * 1000 / 60 }

// pingPong recorre 0, 1, …, n−1, n−2, …, 1 y vuelve a empezar.
func pingPong(step, n int) int {
	period := 2 * (n - 1)
	i := step % period
	if i < n {
		return i
	}
	return period - i
}

// cloud es una de las dos nubes.
type cloud int

const (
	cloudA cloud = iota // y = 57, un paso cada 0,249 s, arranca en x = 40
	cloudB              // y = 29, un paso cada 0,359 s, arranca en x = 12
)

// cloudX: la nube va de x = −42 a 319, 1 px por paso, y vuelve a empezar.
func cloudX(ticks int, c cloud) int {
	stepMs, start := 249, 40
	if c == cloudB {
		stepMs, start = 359, 12
	}
	const span = 320 + 42
	steps := msOf(ticks) / stepMs
	return -42 + (start+42+steps)%span
}

// wavesX: las olas van y vienen (d = 0, 1, 2, 3, 2, 1…), un paso cada 0,6 s.
func wavesX(ticks int) int { return -8 + pingPong(ticks/36, 4) }

// foamX: la espuma, igual que las olas pero al revés y un paso cada 0,45 s.
func foamX(ticks int) int { return -8 - pingPong(ticks/27, 4) }

// crownFrame: el cuadro de la copa de una palmera (0 a 4, ida y vuelta),
// uno cada 0,75 s; la segunda va medio ciclo desfasada.
func crownFrame(ticks int, second bool) int {
	step := ticks / 45
	if second {
		step += 4
	}
	return pingPong(step, 5)
}

// breathY: el torso del paciente sube y baja (0, −1, −2, −1), un valor por
// segundo.
func breathY(ticks int) int {
	return []int{0, -1, -2, -1}[ticks/60%4]
}

// headNod: el cabeceo del paciente en un ciclo de 6 s.
func headNod(ticks int) (dx, dy int) {
	switch ms := msOf(ticks) % 6000; {
	case ms < 4200:
		return 0, 0
	case ms < 4680:
		return 1, 1
	case ms < 5160:
		return 1, 2
	case ms < 5640:
		return 0, 1
	default:
		return 0, 0
	}
}

// patientZzz: la Zzz número i (0, 1, 2) del paciente, en un ciclo de 3 s de
// 6 pasos de 0,5 s, desfasada i segundos. Sube de a un píxel desde +2 y su
// opacidad llega a 1 al 30 % del ciclo y se apaga al final.
func patientZzz(ticks, i int) (dy int, alpha float32) {
	phase := ((ticks-i*60)%180 + 180) % 180
	step := phase / 30
	return 2 - step, stepFade(step)
}

// stepFade es la opacidad del paso step de 6: 0 → 1 al 30 % → 0.
func stepFade(step int) float32 {
	p := float32(step) / 6
	if p < 0.3 {
		return p / 0.3
	}
	return (1 - p) / 0.7
}

// pressEnterVisible: "PRESIONA ENTER" parpadea cada 1,2 s.
func pressEnterVisible(ticks int) bool { return ticks%72 < 36 }

// ─── Imágenes y dibujo ─────────────────────────────────────────────────

var titleImgs = map[string]*ebiten.Image{}

// loadTitle carga las capas y el portapapeles de la portada.
func loadTitle() error {
	names := []string{"fondo", "nube_a", "nube_b", "olas", "espuma", "suelo", "copa1_frames", "copa2_frames",
		"patient_legs", "patient_torso", "patient_head", "zzz1", "zzz2", "zzz3"}
	var paths []string
	for _, n := range names {
		paths = append(paths, "title/"+n+".png")
	}
	for _, o := range menuOptions {
		for frame := 0; frame < 2; frame++ {
			paths = append(paths, clipboardPath(o, true, frame))
			if o != optContinue {
				paths = append(paths, clipboardPath(o, false, frame))
			}
		}
	}
	for _, p := range paths {
		img, err := decodePNG(assets.Title, p)
		if err != nil {
			return err
		}
		titleImgs[p] = ebiten.NewImageFromImage(img)
	}
	return nil
}

// drawLayer dibuja una capa (o un recorte) de 320×180 ×4 con su esquina en
// (x, y) de la escena nativa.
func drawLayer(dst, img *ebiten.Image, x, y int, alpha float32) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(titleScale, titleScale)
	op.GeoM.Translate(float64(x*titleScale), float64(y*titleScale))
	op.ColorScale.ScaleAlpha(alpha)
	dst.DrawImage(img, op)
}

// crown devuelve el cuadro f (de 74×42) de una hoja de copas.
func crown(sheet *ebiten.Image, f int) *ebiten.Image {
	return sheet.SubImage(image.Rect(f*74, 0, f*74+74, 42)).(*ebiten.Image)
}

// drawBeach dibuja el paisaje animado completo (lo usan la portada y los
// créditos), en el orden de TITLE_ASSETS.md.
func drawBeach(dst *ebiten.Image, ticks int) {
	img := func(n string) *ebiten.Image { return titleImgs["title/"+n+".png"] }
	drawLayer(dst, img("fondo"), 0, 0, 1)
	drawLayer(dst, img("nube_a"), cloudX(ticks, cloudA), 57, 1)
	drawLayer(dst, img("nube_b"), cloudX(ticks, cloudB), 29, 1)
	drawLayer(dst, img("olas"), wavesX(ticks), 100, 1)
	drawLayer(dst, img("espuma"), foamX(ticks), 123, 1)
	drawLayer(dst, img("suelo"), 0, 0, 1)
	drawLayer(dst, crown(img("copa1_frames"), crownFrame(ticks, false)), -13, 79, 1)
	drawLayer(dst, crown(img("copa2_frames"), crownFrame(ticks, true)), 145, 85, 1)

	r := breathY(ticks)
	cx, cy := headNod(ticks)
	drawLayer(dst, img("patient_legs"), 15, 112, 1)
	drawLayer(dst, img("patient_torso"), 15, 112+r, 1)
	drawLayer(dst, img("patient_head"), 15+cx, 112+r+cy, 1)
	for i, n := range []string{"zzz1", "zzz2", "zzz3"} {
		z, alpha := patientZzz(ticks, i)
		drawLayer(dst, img(n), 15, 112+z, alpha)
	}
}
