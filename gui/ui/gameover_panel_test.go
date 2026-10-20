package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// TestDaysSurvived_BankruptcyCountsTheDayThatJustClosed: la quiebra se
// dispara en la nómina de las 20:00 (ese día SÍ terminó completo); la
// reputación en 0 y el colapso pueden pasar a mitad del día (ese no se
// cuenta como sobrevivido).
func TestDaysSurvived_BankruptcyCountsTheDayThatJustClosed(t *testing.T) {
	cases := []struct {
		day  int
		over game.GameOver
		want int
	}{
		{4, game.LostLicense, 3},
		{2, game.HospitalCollapsed, 1},
		{5, game.Bankrupt, 5},
		{1, game.LostLicense, 0}, // perdió el primer día: 0 sobrevividos
	}
	for _, c := range cases {
		if got := daysSurvived(c.day, c.over); got != c.want {
			t.Errorf("daysSurvived(día %d, %v) = %d; se esperaban %d", c.day, c.over, got, c.want)
		}
	}
}

// TestGameOverReason_MatchesEachMotive: el texto de cada motivo (§5.10),
// con los valores reales de la partida.
func TestGameOverReason_MatchesEachMotive(t *testing.T) {
	if got := gameOverReason(game.LostLicense, 0); got != "LA REPUTACIÓN LLEGÓ A 0 ESTRELLAS" {
		t.Errorf("LostLicense: %q", got)
	}
	if got := gameOverReason(game.Bankrupt, -340); got != "LA NÓMINA DEJÓ LA PLATA EN -$ 340" {
		t.Errorf("Bankrupt: %q; se esperaba la plata real formateada", got)
	}
	if got := gameOverReason(game.HospitalCollapsed, 0); got != "5 DORMIDOS SIN ATENDER A LA VEZ" {
		t.Errorf("HospitalCollapsed: %q; se esperaba el límite real (game.CollapseLimit)", got)
	}
	if got := gameOverReason(game.NotOver, 0); got != "" {
		t.Errorf("NotOver: %q; se esperaba vacío (no debería dibujarse nunca)", got)
	}
}

// TestGameOverIconName_OneFileForEachMotive: cada motivo tiene su propio
// ícono (gui/assets/gameover/GAMEOVER_ASSETS.md) y NotOver no tiene
// ninguno (el panel no se dibuja si GameOver == NotOver).
func TestGameOverIconName_OneFileForEachMotive(t *testing.T) {
	want := map[game.GameOver]string{
		game.LostLicense:       "gameover_icon_licencia.png",
		game.Bankrupt:          "gameover_icon_quiebra.png",
		game.HospitalCollapsed: "gameover_icon_colapsado.png",
		game.NotOver:           "",
	}
	for over, name := range want {
		if got := gameOverIconName(over); got != name {
			t.Errorf("gameOverIconName(%v) = %q; se esperaba %q", over, got, name)
		}
	}
}

// ─── Dibujo: solo que no truene (ver report_panel_test.go) ───────────────

// TestDrawGameOverPanel_AllThreeMotivesDoNotPanic: los tres motivos, con
// un puntaje final grande y una plata bien negativa (quiebra), para que
// el ícono, el sello y el desglose del puntaje se ejerciten de verdad.
func TestDrawGameOverPanel_AllThreeMotivesDoNotPanic(t *testing.T) {
	needFonts(t)
	if err := loadHiring(); err != nil { // el portapapeles (hiring_board.png) se reusa acá
		t.Fatalf("loadHiring: %v", err)
	}
	dst := ebiten.NewImage(ScreenWidth, ScreenHeight)

	cases := []game.Snapshot{
		{GameOver: game.LostLicense, Day: 4, Score: 100, Money: 2450, MoneyScore: 245, StarScore: 620, FinalScore: 965},
		{GameOver: game.Bankrupt, Day: 5, Score: 8400, Money: -1234567, MoneyScore: -123456, StarScore: 500, FinalScore: 8766},
		{GameOver: game.HospitalCollapsed, Day: 2, Score: 2350, Money: 1870, MoneyScore: 187, StarScore: 350, FinalScore: 2887},
	}
	for _, snap := range cases {
		drawGameOverPanel(dst, snap)
	}
}
