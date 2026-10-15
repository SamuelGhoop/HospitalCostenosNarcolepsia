package ui

import (
	"image"
	"image/color"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// La barra de arriba del modo juego sale de la maqueta HUD.dc.html (pedido
// de Samuel, 2026-10-11). Los tests leen la maqueta misma, así los iconos y
// la paleta de Go no se pueden separar de ella sin que se note.
const hudMockupPath = "../design/claude-design/Hospital de los Costeños/HUD.dc.html"

// mockupPixelMap lee de la maqueta el arreglo `static NOMBRE = [ '…', … ];`.
func mockupPixelMap(t *testing.T, src, name string) []string {
	t.Helper()
	block := regexp.MustCompile(`static ` + name + ` = \[([^\]]*)\]`).FindStringSubmatch(src)
	if block == nil {
		t.Fatalf("la maqueta no tiene %s", name)
	}
	var rows []string
	for _, m := range regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(block[1], -1) {
		rows = append(rows, m[1])
	}
	return rows
}

func readHUDMockup(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(hudMockupPath)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Los iconos ESTRELLA, PLATA y RELOJ son los mapas de píxeles de la maqueta,
// tal cual, y VACÍA es la estrella con el amarillo y el brillo en E y el
// naranja en e (como en la maqueta: ESTRELLA.map(replace…)).
func TestHUDIcons_AreTheMockupPixelMaps(t *testing.T) {
	src := readHUDMockup(t)
	for name, got := range map[string][]string{"ESTRELLA": hudStar, "PLATA": hudMoneyIcon, "RELOJ": hudClockIcon} {
		want := mockupPixelMap(t, src, name)
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s no es el de la maqueta:\n%s\nse esperaba:\n%s", name, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
	}
	for i, row := range hudStar {
		want := strings.NewReplacer("Y", "E", "H", "E", "O", "e").Replace(row)
		if hudStarEmpty[i] != want {
			t.Errorf("fila %d de VACÍA: %q; se esperaba %q", i, hudStarEmpty[i], want)
		}
	}
}

// La paleta es la HUD_PAL de la maqueta.
func TestHUDIcons_UseTheMockupPalette(t *testing.T) {
	src := readHUDMockup(t)
	block := regexp.MustCompile(`static HUD_PAL = \{([^}]*)\}`).FindStringSubmatch(src)
	if block == nil {
		t.Fatal("la maqueta no tiene HUD_PAL")
	}
	entries := regexp.MustCompile(`(\w): '#([0-9a-fA-F]{6})'`).FindAllStringSubmatch(block[1], -1)
	if len(entries) != len(hudPalette) {
		t.Errorf("la maqueta tiene %d colores y la paleta de Go %d", len(entries), len(hudPalette))
	}
	for _, e := range entries {
		v, _ := strconv.ParseUint(e[2], 16, 32)
		want := color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
		if got, ok := hudPalette[e[1][0]]; !ok || got != want {
			t.Errorf("color %s: %v; se esperaba %v", e[1], got, want)
		}
	}
}

// Cada píxel del mapa se dibuja como un cuadrado de 3×3 con su color; el
// '.' queda transparente.
func TestHUDIcons_DrawAtTimesThree(t *testing.T) {
	img := iconImage(hudClockIcon, len(hudClockIcon[0]))
	if b := img.Bounds(); b.Dx() != 3*len(hudClockIcon[0]) || b.Dy() != 3*len(hudClockIcon) {
		t.Fatalf("el reloj mide %v; se esperaba ×3", b.Size())
	}
	for y, row := range hudClockIcon {
		for x, ch := range row {
			for _, d := range []image.Point{{0, 0}, {2, 2}} {
				got := img.RGBAAt(3*x+d.X, 3*y+d.Y)
				want := color.RGBA{}
				if ch != '.' {
					want = hudPalette[byte(ch)]
				}
				if got != want {
					t.Fatalf("píxel (%d, %d) del reloj = %v; se esperaba %v (%q)", x, y, got, want, ch)
				}
			}
		}
	}
}

// La estrella parcial: la vacía y, encima, la llena recortada a
// 1 + round(9 × fracción) columnas (3,7 → la cuarta usa 7); las enteras,
// las 11; las que no llegan, ninguna.
func TestHUDStars_PartialStarColumns(t *testing.T) {
	cases := []struct {
		tenths int
		want   [5]int
	}{
		{37, [5]int{11, 11, 11, 7, 0}},
		{31, [5]int{11, 11, 11, 2, 0}},
		{30, [5]int{11, 11, 11, 0, 0}},
		{50, [5]int{11, 11, 11, 11, 11}},
		{0, [5]int{0, 0, 0, 0, 0}},
		{5, [5]int{6, 0, 0, 0, 0}},
	}
	for _, c := range cases {
		for i := 0; i < 5; i++ {
			if got := starColumns(c.tenths, i); got != c.want[i] {
				t.Errorf("con %d décimas, la estrella %d lleva %d columnas llenas; se esperaban %d", c.tenths, i+1, got, c.want[i])
			}
		}
	}
}

// La reputación que se ve se mueve 0,1 a la vez hacia la real, un paso cada
// 0,25 s, con un destello en cada paso: verde si sube y rojo si baja.
func TestHUDReputation_StepsOneTenthAtATimeWithAFlash(t *testing.T) {
	var h hud
	h.update(game.Snapshot{Reputation: 300, Money: 2000}) // la primera foto se muestra tal cual
	if h.rep != 30 || h.flashing() {
		t.Fatalf("al empezar se ven %d décimas (destello %v); se esperaban 30, sin destello", h.rep, h.flashing())
	}
	down := game.Snapshot{Reputation: 250, Money: 2000} // 2,5: baja 5 décimas
	for tick := 1; tick <= 5*repStepTicks; tick++ {
		h.update(down)
		if want := 30 - (tick+repStepTicks-1)/repStepTicks; h.rep != want {
			t.Fatalf("tick %d bajando: se ven %d décimas; se esperaban %d", tick, h.rep, want)
		}
		if !h.flashing() || h.flashColor() != colHUDRed {
			t.Fatalf("tick %d bajando: destello %v %v; se esperaba rojo", tick, h.flashing(), h.flashColor())
		}
	}
	h.update(down)
	if h.rep != 25 || h.flashing() {
		t.Errorf("ya en 2,5: se ven %d décimas, destello %v", h.rep, h.flashing())
	}
	h.update(game.Snapshot{Reputation: 275, Money: 2000}) // 2,75 se ve como 2,8: sube
	if h.rep != 26 || h.flashColor() != colHUDGreen {
		t.Errorf("subiendo: se ven %d décimas con destello %v; se esperaban 26 y verde", h.rep, h.flashColor())
	}
	if repStepTicks != 15 {
		t.Errorf("cada paso dura %d ticks; se esperaban 15 (0,25 s)", repStepTicks)
	}
}

// La plata que se ve cuenta hacia la real en ~0,6 s, y cada cambio deja un
// "+$450" verde o un "-$X" rojo que sube 24 px y se desvanece en 1 s.
func TestHUDMoney_CountsUpAndShowsTheChange(t *testing.T) {
	var h hud
	h.update(game.Snapshot{Reputation: 300, Money: 2000})
	if len(h.deltas) != 0 {
		t.Fatalf("la primera foto dejó %d cambios", len(h.deltas))
	}
	pay := game.Snapshot{Reputation: 300, Money: 2450}
	h.update(pay)
	if len(h.deltas) != 1 || h.deltas[0].text != "+$450" || h.deltas[0].color != colHUDGreen {
		t.Fatalf("al cobrar: cambios %+v; se esperaba \"+$450\" en verde", h.deltas)
	}
	for tick := 2; tick <= moneyCountTicks/2; tick++ {
		h.update(pay)
	}
	if mid := h.shownMoney(); mid <= 2000 || mid >= 2450 {
		t.Errorf("a la mitad se ven $%d; se esperaba entre 2000 y 2450", mid)
	}
	for tick := moneyCountTicks/2 + 1; tick <= moneyCountTicks; tick++ {
		h.update(pay)
	}
	if got := h.shownMoney(); got != 2450 {
		t.Errorf("a los %d ticks se ven $%d; se esperaban 2450", moneyCountTicks, got)
	}
	if moneyCountTicks < 30 || moneyCountTicks > 42 {
		t.Errorf("la plata cuenta en %d ticks; se esperaban unos 0,6 s", moneyCountTicks)
	}

	d := h.deltas[0]
	if rise, alpha := d.offset(); rise >= -1 || alpha <= 0 {
		t.Errorf("a mitad de camino el \"+$450\" sube %v y tiene alfa %v", rise, alpha)
	}
	for tick := 0; tick < deltaTicks; tick++ {
		h.update(pay)
	}
	if len(h.deltas) != 0 {
		t.Errorf("después de 1 s el \"+$450\" sigue ahí")
	}
	last := moneyDelta{ticks: deltaTicks}
	if rise, alpha := last.offset(); rise != -24 || alpha != 0 {
		t.Errorf("al final el cambio sube %v px con alfa %v; se esperaba 24 px y transparente", rise, alpha)
	}

	h.update(game.Snapshot{Reputation: 300, Money: 1250})
	if n := len(h.deltas); n != 1 || h.deltas[0].text != "-$1.200" || h.deltas[0].color != colHUDRed {
		t.Errorf("al pagar: cambios %+v; se esperaba \"-$1.200\" en rojo", h.deltas)
	}
}

func TestHUD_FormatsMoneyWithThousandsDots(t *testing.T) {
	for n, want := range map[int]string{2450: "$ 2.450", 0: "$ 0", 1000000: "$ 1.000.000", -150: "-$ 150", 999: "$ 999"} {
		if got := formatMoney(n); got != want {
			t.Errorf("formatMoney(%d) = %q; se esperaba %q", n, got, want)
		}
	}
	if got := hudDayLabel(3); got != "DÍA 3" {
		t.Errorf("el día 3 se escribe %q", got)
	}
	if got := formatReputation(37); got != "3,7" {
		t.Errorf("3,7 estrellas se escriben %q", got)
	}
}

// Medidas de la maqueta en una pantalla de 1280: la barra mide 42 px con su
// borde; a la izquierda, desde 17 px, las estrellas ×3 y la plata; en el
// centro el DÍA; a la derecha, hasta 1263 px, el reloj, la hora y la barra
// del turno. Nada se encima.
func TestHUD_LayoutFollowsTheMockup(t *testing.T) {
	needFonts(t)
	l := layoutHUD("3,7", "$ 2.450", "DÍA 1", "14:15")
	if l.bar != image.Rect(0, 0, ScreenWidth, 42) {
		t.Errorf("la barra es %v", l.bar)
	}
	if l.stars.Min.X != 17 || l.stars.Dx() != 3*63 || l.stars.Dy() != 33 {
		t.Errorf("las estrellas van en %v; se esperaba desde x=17, de 189×33", l.stars)
	}
	if c := (l.day.Min.X + l.day.Max.X) / 2; c < ScreenWidth/2-1 || c > ScreenWidth/2+1 {
		t.Errorf("el DÍA está centrado en x=%d", c)
	}
	if l.progress.Max.X != ScreenWidth-17 || l.progress.Dx() != 132 {
		t.Errorf("la barra del turno es %v; se esperaba de 132 px hasta x=%d", l.progress, ScreenWidth-17)
	}
	parts := []image.Rectangle{l.stars, l.rep, l.moneyIcon, l.money, l.day, l.clockIcon, l.clock, l.progress}
	for i, a := range parts {
		if !a.In(image.Rect(0, 0, ScreenWidth, 39)) {
			t.Errorf("la parte %d (%v) se sale de la barra", i, a)
		}
		for _, b := range parts[i+1:] {
			if a.Overlaps(b) {
				t.Errorf("las partes %v y %v se enciman", a, b)
			}
		}
	}
	if w := progressFill(l.progress, 0.5); w != (l.progress.Dx()-6)/2 {
		t.Errorf("a la mitad del turno la barra se llena %d px", w)
	}
}
