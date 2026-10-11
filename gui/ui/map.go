package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// Imágenes del mapa (GAME_DESIGN §10.7), dentro de assets.Map.
const (
	// backgroundPath: la maqueta hospital.dc.html renderizada a 1280×720,
	// solo con lo que nunca se mueve, y con las 3 camas libres.
	backgroundPath = "map/hospital_map_1280x720.png"
	// blanketPath: la cobija subida (56×48, con transparencia), que va
	// ENCIMA del paciente acostado.
	blanketPath = "map/bed_blanket.png"
	steamPath   = "map/steam_pot.png"
	// palmCrownPath: la copa de una palmera (104×104), que va ENCIMA de los
	// personajes: quien pasa cerca queda debajo de las hojas.
	palmCrownPath = "map/palm_crown.png"
	// Las 3 capas de los carros (132×72, en el mismo punto): detalles sin
	// teñir, carrocería en gris que se tiñe y el letrero de los taxis.
	carDetailsPath = "map/car_details.png"
	carBodyPath    = "map/car_body.png"
	carSignPath    = "map/car_taxi_sign.png"
)

// Las imágenes ya cargadas para Ebitengine (ver loadMap).
var backgroundImg, blanketImg, steamImg, palmCrownImg, carDetailsImg, carBodyImg, carSignImg *ebiten.Image

// loadMap carga las imágenes embebidas del mapa.
func loadMap() error {
	for _, img := range []struct {
		path string
		dst  **ebiten.Image // puntero a la variable que se llena
	}{
		{backgroundPath, &backgroundImg},
		{blanketPath, &blanketImg},
		{steamPath, &steamImg},
		{palmCrownPath, &palmCrownImg},
		{carDetailsPath, &carDetailsImg},
		{carBodyPath, &carBodyImg},
		{carSignPath, &carSignImg},
	} {
		decoded, err := decodePNG(assets.Map, img.path)
		if err != nil {
			return err
		}
		*img.dst = ebiten.NewImageFromImage(decoded)
	}
	return nil
}

// drawMap dibuja el hospital vacío: el mapa de fondo (pisos, paredes,
// muebles, camas libres, calle) y, encima, el rótulo de cada zona, que la
// imagen no trae.
func drawMap(dst *ebiten.Image) {
	dst.DrawImage(backgroundImg, nil)

	// Rótulo de cada zona, en el sitio que calcula zoneLabelBox (el mismo
	// que revisa el test de rótulos sin solaparse).
	for _, z := range zoneOrder {
		box := zoneLabelBox(z)
		drawText(dst, zones[z].label, float64(box.Min.X+4), float64(box.Min.Y+2), smallFace, colInk)
	}
}

// La cama se dibuja en dos capas para que el paciente se vea arropado:
//
//	el fondo (cama libre) → el paciente → drawBlankets (la cobija subida)

// drawBlankets dibuja la cobija subida ENCIMA del paciente, en las camas
// donde hay alguien acostado (lyingIn[número]). En las demás se ve la cama
// libre del fondo, con la cobija doblada al pie.
func drawBlankets(dst *ebiten.Image, rooms []game.RoomView, lyingIn map[int]bool) {
	for _, r := range rooms {
		if !lyingIn[r.Number] {
			continue
		}
		p := blanketSpot(roomZone(r.Number))
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(p.X), float64(p.Y))
		dst.DrawImage(blanketImg, op)
	}
}
