package ui

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// El ícono "falta el médico" sobre el paciente que ya está en la cama pero
// sin revisar (GAME_DESIGN §10.3 y §12.8, pedido de Samuel el 2026-10-17 para
// que se note por qué están bajando las estrellas): un globo con un
// estetoscopio, con la colita apuntando al paciente. Ficha en
// gui/assets/sprites/ICONS_ASSETS.md.
const (
	// needDoctorIconPath: hoja horizontal de 38×18 con 2 cuadros de 19×18.
	needDoctorIconPath    = "sprites/icon_need_doctor.png"
	needDoctorFrameWidth  = 19
	needDoctorFrameHeight = 18
	needDoctorFrames      = 2
	// needDoctorFrameTicks: 0,4 s por cuadro a 60 ticks por segundo.
	needDoctorFrameTicks = 24
)

// needDoctorIconImg: la hoja ya cargada para Ebitengine (ver loadSprites).
var needDoctorIconImg *ebiten.Image

// loadNeedDoctorIcon carga la hoja embebida del ícono.
func loadNeedDoctorIcon() error {
	img, err := decodePNG(assets.Sprites, needDoctorIconPath)
	if err != nil {
		return err
	}
	needDoctorIconImg = ebiten.NewImageFromImage(img)
	return nil
}

// needDoctorBlinkAfter: desde cuándo parpadea el ícono.
//
// Es una decisión de interfaz (Samuel, §12.8), no una regla de juego: avisa
// 10 s antes de que firstWaitPenaltyAt (game/config.go) cobre el primer
// −0,5 ★ a los 30 s de espera, para que el jugador lo vea venir.
const needDoctorBlinkAfter = 20 * time.Second

// needDoctorIconAt da la esquina de arriba a la izquierda del ícono en
// pantalla (1280×720) para las habitaciones con cama (101 a 103): a ×3, en
// (+169, +64) desde la esquina de arriba a la izquierda de cada habitación
// (ICONS_ASSETS.md). Las demás habitaciones (ninguna hoy) no tienen ícono.
func needDoctorIconAt(room int) (image.Point, bool) {
	switch room {
	case 101:
		return image.Pt(184, 104), true
	case 102:
		return image.Pt(457, 104), true
	case 103:
		return image.Pt(730, 104), true
	default:
		return image.Point{}, false
	}
}

// needDoctorFrame es el cuadro que toca después de ticks ticks de
// animación: 0, 1, 0, 1… cada needDoctorFrameTicks (0,4 s), en bucle.
func needDoctorFrame(ticks int) int {
	return ticks / needDoctorFrameTicks % needDoctorFrames
}

// needDoctorVisible dice si el ícono se ve en este tick: siempre antes de
// needDoctorBlinkAfter de espera, y después parpadeando medio segundo
// prendido y medio apagado (30 ticks), como la burbuja roja del paciente
// sin recoger (bubbleBlinkAfter, game_input.go).
func needDoctorVisible(waited time.Duration, ticks int) bool {
	if waited < needDoctorBlinkAfter {
		return true
	}
	return ticks/30%2 == 0
}

// needDoctorPatients filtra los pacientes que deben mostrar el ícono: en
// cama (tienen habitación) y esperando revisión (AwaitingReview).
func needDoctorPatients(patients []game.GamePatientView) []game.GamePatientView {
	var out []game.GamePatientView
	for _, p := range patients {
		if p.Stage == game.AwaitingReview && p.Room != 0 {
			out = append(out, p)
		}
	}
	return out
}

// drawNeedDoctorIcons dibuja el ícono sobre cada paciente que lo necesita,
// con su cuadro de animación y el parpadeo desde needDoctorBlinkAfter.
func drawNeedDoctorIcons(dst *ebiten.Image, patients []game.GamePatientView, ticks int) {
	frame := needDoctorFrame(ticks)
	rect := image.Rect(frame*needDoctorFrameWidth, 0, (frame+1)*needDoctorFrameWidth, needDoctorFrameHeight)
	for _, p := range needDoctorPatients(patients) {
		at, ok := needDoctorIconAt(p.Room)
		if !ok || !needDoctorVisible(p.Waited, ticks) {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(spriteScale, spriteScale)
		op.GeoM.Translate(float64(at.X), float64(at.Y))
		dst.DrawImage(needDoctorIconImg.SubImage(rect).(*ebiten.Image), op)
	}
}
