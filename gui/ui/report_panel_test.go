package ui

import (
	"image"
	"strconv"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// Estos tests son funciones puras (GAME_DESIGN §9, "no adivines mirando
// capturas"): ninguno dibuja y lee píxeles de vuelta porque, en este
// módulo, leer un *ebiten.Image (At/ReadPixels) antes de que arranque la
// ventana de verdad entra en pánico ("ReadPixels cannot be called before
// the game starts"), aunque DIBUJAR sí se puede (TestDrawDayOverPanel_*
// más abajo son justamente eso: que no truene). La forma correcta a mano
// es el backdoor de pruebas (-agent -shot/-dump, GAME_DESIGN §10.9), que
// sí corre con una ventana real.

func line(s string) renderLine { return renderLine{text: s} }

func blockOf(lines ...string) reportBlock {
	b := make(reportBlock, len(lines))
	for i, l := range lines {
		b[i] = line(l)
	}
	return b
}

// TestFitBlocks_FitsOrReservesTheMásLine: tabla de casos del recorte
// genérico que usan las cuatro fichas de consulta (5.1 a 5.4).
func TestFitBlocks_FitsOrReservesTheMásLine(t *testing.T) {
	cases := []struct {
		name      string
		blocks    []reportBlock
		budget    int
		wantShown int
		wantMore  int
	}{
		{"vacío", nil, 9, 0, 0},
		{"todo entra, sin sobrar renglón", []reportBlock{blockOf("a"), blockOf("b")}, 2, 2, 0},
		{"todo entra, con renglones de sobra", []reportBlock{blockOf("a"), blockOf("b")}, 9, 2, 0},
		// budget=1 es un caso extremo que no pasa con las fichas reales
		// (su presupuesto siempre es 9, TestMaxQueryCardLines_...): ni
		// siquiera "a" sola deja lugar para avisar que falta "b", así que
		// no se muestra ninguna (mejor nada que un aviso a medias).
		{"budget no alcanza ni para un bloque con su aviso: no muestra nada", []reportBlock{blockOf("a"), blockOf("b")}, 1, 0, 2},
		{"bloques de más de un renglón: corta el que no entra entero", []reportBlock{blockOf("a", "a2"), blockOf("b", "b2")}, 3, 1, 1},
		{"cabe justo, pero no deja lugar para avisar que falta uno: lo deja afuera", []reportBlock{blockOf("a"), blockOf("b"), blockOf("c")}, 2, 1, 2},
		{"el último bloque no necesita reservar renglón: puede usarlos todos", []reportBlock{blockOf("a"), blockOf("b")}, 2, 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			shown, more := fitBlocks(c.blocks, c.budget)
			if len(shown) != c.wantShown || more != c.wantMore {
				t.Errorf("fitBlocks(%d bloques, budget=%d) = (%d mostrados, +%d MÁS); se esperaba (%d, +%d)",
					len(c.blocks), c.budget, len(shown), more, c.wantShown, c.wantMore)
			}
		})
	}
}

// TestMaxQueryCardLines_SameForTheFourCards: las cuatro fichas de consulta
// miden lo mismo (280×206, Figma) así que caben los mismos renglones.
func TestMaxQueryCardLines_SameForTheFourCards(t *testing.T) {
	for _, box := range []image.Rectangle{hallwayCardBox, doctorsCardBox, roomsCardBox, severeCardBox} {
		if got := maxQueryCardLines(box); got != 9 {
			t.Errorf("maxQueryCardLines(%v) = %d; se esperaban 9", box, got)
		}
	}
}

// TestDoctorBlock_CapsEpisodesAndKeepsTheRealCount: el encabezado siempre
// dice el total de episodios (aunque se recorten), y después hay como
// mucho maxEpisodesPerDoctor líneas de detalle más, si sobran, un "+ N
// MÁS".
func TestDoctorBlock_CapsEpisodesAndKeepsTheRealCount(t *testing.T) {
	d := game.DoctorEpisodes{
		DoctorName: "Dr. Hernando Polo",
		Episodes: []game.EpisodeLine{
			{ID: "021", Time: "19:12", PatientID: "P-017", Room: 101},
			{ID: "022", Time: "19:20", PatientID: "P-018", Room: 102},
			{ID: "024", Time: "19:48", PatientID: "P-019", Room: 0},
		},
	}
	b := doctorBlock(d)
	if len(b) != 1+maxEpisodesPerDoctor+1 {
		t.Fatalf("el bloque tiene %d renglones; se esperaban %d (encabezado + %d episodios + MÁS)", len(b), 1+maxEpisodesPerDoctor+1, maxEpisodesPerDoctor)
	}
	if b[0].right != "3" {
		t.Errorf("el encabezado dice %q episodios; se esperaban los 3 reales, no solo los que se muestran", b[0].right)
	}
	if got := b[len(b)-1].text; got != "  + 1 MÁS" {
		t.Errorf("el último renglón es %q; se esperaba el aviso del episodio que falta", got)
	}
	if dest := episodeDestination(d.Episodes[2]); dest != "PASILLO" {
		t.Errorf("episodeDestination sin habitación = %q; se esperaba PASILLO", dest)
	}
	if dest := episodeDestination(d.Episodes[0]); dest != "101" {
		t.Errorf("episodeDestination con habitación 101 = %q; se esperaba 101", dest)
	}
}

// TestDoctorBlock_NoOverflowLineWhenEverythingFits: con 2 episodios o
// menos no hay "+ N MÁS".
func TestDoctorBlock_NoOverflowLineWhenEverythingFits(t *testing.T) {
	d := game.DoctorEpisodes{DoctorName: "Dra. Rosiris Arrieta", Episodes: []game.EpisodeLine{
		{ID: "018", Time: "17:30", PatientID: "P-015", Room: 103},
		{ID: "022", Time: "19:20", PatientID: "P-018", Room: 102},
	}}
	b := doctorBlock(d)
	if len(b) != 3 {
		t.Fatalf("el bloque tiene %d renglones; se esperaban 3 (encabezado + 2 episodios, sin MÁS)", len(b))
	}
}

// TestSevereBlock_NoteLineOnlyWhenThereIsOne: el paciente severo que sigue
// en el mapa (Note == "") no lleva segundo renglón.
func TestSevereBlock_NoteLineOnlyWhenThereIsOne(t *testing.T) {
	withNote := severeBlock(game.SevereLine{PatientID: "P-003", PatientName: "Wilfrido Berrío", Episodes: 3, Note: "(alta)"})
	if len(withNote) != 2 {
		t.Fatalf("con nota, el bloque tiene %d renglones; se esperaban 2", len(withNote))
	}
	withoutNote := severeBlock(game.SevereLine{PatientID: "P-019", PatientName: "Yuleidis Pertuz", Episodes: 1})
	if len(withoutNote) != 1 {
		t.Fatalf("sin nota, el bloque tiene %d renglones; se esperaba 1", len(withoutNote))
	}
}

// TestRoomStateWordAndColor_MatchesTheFigmaWording: "OCUPADA"/"LIBRE" (no
// "ocupada"/"disponible", que es RoomState.String(): ese texto es del
// letrero del mapa, overlays.go, no de esta ficha).
func TestRoomStateWordAndColor_MatchesTheFigmaWording(t *testing.T) {
	if w := roomStateWord(hospital.Occupied); w != "OCUPADA" {
		t.Errorf("roomStateWord(Occupied) = %q; se esperaba OCUPADA", w)
	}
	if w := roomStateWord(hospital.Available); w != "LIBRE" {
		t.Errorf("roomStateWord(Available) = %q; se esperaba LIBRE", w)
	}
	if roomStateColor(hospital.Occupied) != colRed {
		t.Error("roomStateColor(Occupied) debía ser colRed")
	}
	if roomStateColor(hospital.Available) != colGreenDark {
		t.Error("roomStateColor(Available) debía ser colGreenDark")
	}
}

// TestFormatMoneySigned y TestFormatSignedInt: con puntos de miles y el
// signo siempre puesto (a diferencia de formatMoney/formatDelta, hud.go).
func TestFormatMoneySigned(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{{1900, "+$ 1.900"}, {-600, "-$ 600"}, {0, "+$ 0"}, {-1450, "-$ 1.450"}}
	for _, c := range cases {
		if got := formatMoneySigned(c.n); got != c.want {
			t.Errorf("formatMoneySigned(%d) = %q; se esperaba %q", c.n, got, c.want)
		}
	}
}

func TestFormatSignedInt(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{{123, "+123"}, {-34, "-34"}, {0, "+0"}}
	for _, c := range cases {
		if got := formatSignedInt(c.n); got != c.want {
			t.Errorf("formatSignedInt(%d) = %q; se esperaba %q", c.n, got, c.want)
		}
	}
}

// TestExpenseAmount_PlainWhenZero: "$ 0" en gris si no hubo gasto, el
// monto en rojo con signo si sí.
func TestExpenseAmount_PlainWhenZero(t *testing.T) {
	if text, clr := expenseAmount(0); text != "$ 0" || clr != colMuted {
		t.Errorf("expenseAmount(0) = (%q, %v); se esperaba (\"$ 0\", colMuted)", text, clr)
	}
	if text, clr := expenseAmount(600); text != "-$ 600" || clr != colRed {
		t.Errorf("expenseAmount(600) = (%q, %v); se esperaba (\"-$ 600\", colRed)", text, clr)
	}
}

// TestMoneyColor: verde si no es negativo, rojo si lo es (Reporte B, §9:
// "los negativos usan el rojo que ya usan las vistas").
func TestMoneyColor(t *testing.T) {
	if moneyColor(0) != colGreenDark || moneyColor(500) != colGreenDark {
		t.Error("moneyColor(≥0) debía ser colGreenDark")
	}
	if moneyColor(-1) != colRed {
		t.Error("moneyColor(<0) debía ser colRed")
	}
}

// TestHiresTotal suma el costo de las contrataciones del día.
func TestHiresTotal(t *testing.T) {
	got := hiresTotal([]game.HireLine{{Cost: 600}, {Cost: 250}})
	if got != 850 {
		t.Errorf("hiresTotal = %d; se esperaban 850", got)
	}
	if got := hiresTotal(nil); got != 0 {
		t.Errorf("hiresTotal(nil) = %d; se esperaban 0", got)
	}
}

// ─── Dibujo: solo que no truene (ver el comentario del principio) ───────

// TestDrawDayOverPanel_EmptyDayDoesNotPanic: el primer día, sin
// contrataciones, sin pasillo, sin severos, nómina del personal inicial.
func TestDrawDayOverPanel_EmptyDayDoesNotPanic(t *testing.T) {
	needFonts(t)
	dst := ebiten.NewImage(ScreenWidth, ScreenHeight)
	report := game.DayReport{
		Day: 1, StartMoney: 2000, EndMoney: 1150,
		Payroll:      []game.PayrollLine{{Name: "Hernando Polo", IsDoctor: true, Salary: 550, RoleLabel: "Médico"}, {Name: "Éder Cassiani", Salary: 300, RoleLabel: "Camillero"}},
		PayrollTotal: 850,
		Report:       game.Report{Rooms: []game.RoomView{{Number: 101, Capacity: 1}, {Number: 102, Capacity: 1}, {Number: 103, Capacity: 1}}},
	}
	drawDayOverPanel(dst, report, 850)
}

// TestDrawDayOverPanel_OverflowingListsDoNotPanic: listas largas (más
// médicos y episodios de los que caben, pasillo y severos largos, varias
// contrataciones) para que fitBlocks y los "+ N MÁS" se ejerciten de
// verdad, no solo con casos vacíos.
func TestDrawDayOverPanel_OverflowingListsDoNotPanic(t *testing.T) {
	needFonts(t)
	dst := ebiten.NewImage(ScreenWidth, ScreenHeight)

	var hallway []game.PatientView
	for i := 0; i < 15; i++ {
		hallway = append(hallway, game.PatientView{ID: "P-0" + strconv.Itoa(i), Name: "Paciente", Location: "pasillo 2"})
	}
	var episodes []game.DoctorEpisodes
	for d := 0; d < 4; d++ {
		var eps []game.EpisodeLine
		for e := 0; e < 6; e++ {
			eps = append(eps, game.EpisodeLine{ID: strconv.Itoa(e), Time: "19:12", PatientID: "P-017", Room: 101})
		}
		episodes = append(episodes, game.DoctorEpisodes{DoctorName: "Dr. Alguien", Episodes: eps})
	}
	var severe []game.SevereLine
	for i := 0; i < 15; i++ {
		severe = append(severe, game.SevereLine{PatientID: "P-0" + strconv.Itoa(i), PatientName: "Alguien", Episodes: i, Note: "(alta)"})
	}
	var hires []game.HireLine
	for i := 0; i < 3; i++ {
		hires = append(hires, game.HireLine{Name: "Candidato", IsDoctor: i%2 == 0, Cost: 500, RoleLabel: "Médico"})
	}

	report := game.DayReport{
		Day: 5, StartMoney: 2000, EndMoney: -200, // Reporte B: día con pérdida
		Payroll:      []game.PayrollLine{{Name: "A", IsDoctor: true, Salary: 550, RoleLabel: "Médico"}},
		PayrollTotal: 550,
		Hires:        hires,
		Report:       game.Report{Hallway: hallway, Episodes: episodes, Severe: severe, LastAssignError: hospital.ErrNoRoomAvailable.Error()},
	}
	drawDayOverPanel(dst, report, 123456)
}
