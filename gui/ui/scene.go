// Package ui dibuja el Hospital de los Costeños con Ebitengine.
//
// Aquí no hay reglas de juego: las escenas solo leen las vistas que entrega
// el paquete game (DemoSnapshot, Report…) y llaman sus métodos.
package ui

import "github.com/hajimehoshi/ebiten/v2"

// Scene es una pantalla: la demostración, el Shift Report, más adelante el
// juego… La App siempre tiene una escena actual y le delega Update y Draw.
//
// Es otra interfaz implícita, como hospital.Attender: DemoScene y
// ReportScene la cumplen solo por tener estos dos métodos.
type Scene interface {
	// Update avanza un tick (60 por segundo) y devuelve la escena que
	// sigue: ella misma si no hay cambio de pantalla.
	Update() (Scene, error)
	// Draw dibuja la escena en la pantalla lógica de 1280×720.
	Draw(screen *ebiten.Image)
}
