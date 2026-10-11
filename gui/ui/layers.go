package ui

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"io/fs"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/gui/assets"
)

// Los personajes se dibujan por capas (GAME_DESIGN §10.3): el paciente con
// cuerpo → camisa → pelo → sombrero, y el médico con su hoja (la bata) y
// el pelo encima. Todas las capas usan el mismo cuadro y la misma GeoM, así
// rotan en la cama y se voltean juntas. La camisa y el pelo vienen en gris y
// se tiñen con ColorScale: cada canal se multiplica por el color, así el
// contorno negro sigue negro.

// layer es una pasada del dibujo de un personaje: un cuadro de una hoja y su
// tinte.
type layer struct {
	sheet string          // ruta de la hoja dentro de assets.Sprites
	rect  image.Rectangle // el cuadro dentro de la hoja
	tint  color.Color     // nil: tal cual (cuerpo y sombrero)
}

// Las hojas de cada pieza. Un tono de piel por índice de SkinTone (4 tonos,
// GAME_DESIGN §7); el pelo calvo y "sin sombrero" no tienen hoja.
var (
	bodySheets   = skinSheets(assets.Sprites, "patient_body")
	doctorSheets = skinSheets(assets.Sprites, "staff_doctor")
	shirtSheets  = map[game.ShirtPattern]string{
		game.PlainShirt:   "sprites/patient_shirt_lisa.png",
		game.FloralShirt:  "sprites/patient_shirt_florida.png",
		game.StripedShirt: "sprites/patient_shirt_rayas.png",
	}
	hairSheets = map[game.Hair]string{
		game.ShortHair:  "sprites/patient_hair_corto.png",
		game.AfroHair:   "sprites/patient_hair_afro.png",
		game.BraidsHair: "sprites/patient_hair_trenzas.png",
	}
	hatSheets = map[game.Hat]string{
		game.VueltiaoHat: "sprites/patient_hat_vueltiao.png",
		game.CapHat:      "sprites/patient_hat_gorra.png",
	}
)

// Las paletas: game guarda solo el índice y la interfaz decide el color
// (GAME_DESIGN §7). Las camisas son los 6 colores del catálogo de sprites de
// la maqueta; el pelo, negro, castaño oscuro, castaño claro y canoso.
var (
	shirtPalette = [6]color.RGBA{
		{0x2f, 0x6f, 0xd6, 0xff}, // azul
		{0xe5, 0x48, 0x4d, 0xff}, // rojo
		{0x2f, 0xb8, 0x4f, 0xff}, // verde
		{0xff, 0xd2, 0x3f, 0xff}, // amarillo
		{0x22, 0xb8, 0xb0, 0xff}, // turquesa
		{0xf5, 0xa6, 0x23, 0xff}, // naranja
	}
	hairPalette = [4]color.RGBA{
		{0x2e, 0x2a, 0x28, 0xff}, // negro (no del todo, para que se vea el contorno)
		{0x5b, 0x3a, 0x24, 0xff}, // castaño oscuro
		{0x9c, 0x6b, 0x3f, 0xff}, // castaño claro
		{0xd9, 0xd6, 0xd0, 0xff}, // canoso
	}
)

// skinSheets arma la hoja de cada tono de piel: sprites/<prefix>_skin<n>.png.
// El tono que todavía no tiene hoja usa la de la piel 1 (hoy falta la 2);
// así, el día que llegue el PNG, entra solo.
func skinSheets(fsys fs.FS, prefix string) [4]string {
	var out [4]string
	for tone := range out {
		p := fmt.Sprintf("sprites/%s_skin%d.png", prefix, tone+1)
		if _, err := fs.Stat(fsys, p); err != nil {
			p = fmt.Sprintf("sprites/%s_skin1.png", prefix)
		}
		out[tone] = p
	}
	return out
}

// patientLayers son las capas del cuadro frame de la animación a de un
// paciente con esa apariencia: cuerpo, camisa, pelo (si no es calvo) y
// sombrero (si tiene), todas con el mismo cuadro.
func patientLayers(look game.Appearance, a animation, frame int) []layer {
	r := frameRect(a, frame)
	ls := []layer{
		{bodySheets[look.SkinTone], r, nil},
		{shirtSheets[look.ShirtPattern], r, shirtPalette[look.ShirtColor]},
	}
	if look.Hair != game.Bald {
		ls = append(ls, layer{hairSheets[look.Hair], r, hairPalette[look.HairColor]})
	}
	if look.Hat != game.NoHat {
		ls = append(ls, layer{hatSheets[look.Hat], r, nil})
	}
	return ls
}

// doctorLayers son las capas de un médico: su hoja (con la bata, sin camisa
// ni sombrero) y el pelo del paciente encima. En quieto y caminando la
// cabeza está donde la del paciente, así que el pelo usa el mismo cuadro;
// al atender la cabeza no se mueve, así que usa el cuadro 1 de quieto.
func doctorLayers(look game.Appearance, a animation, frame int) []layer {
	ls := []layer{{doctorSheets[look.SkinTone], staffRect(a, frame), nil}}
	if look.Hair != game.Bald {
		ha, hf := a, frame
		if a == animAttend {
			ha, hf = animIdle, 0
		}
		ls = append(ls, layer{hairSheets[look.Hair], frameRect(ha, hf), hairPalette[look.HairColor]})
	}
	return ls
}

// sheetImgs son las hojas ya cargadas, por ruta (las carga loadSprites).
var sheetImgs = map[string]*ebiten.Image{}

// drawLayers dibuja las capas en orden, todas con la misma GeoM.
func drawLayers(dst *ebiten.Image, ls []layer, g ebiten.GeoM) {
	for _, l := range ls {
		op := &ebiten.DrawImageOptions{GeoM: g}
		if l.tint != nil {
			op.ColorScale.ScaleWithColor(l.tint)
		}
		// SubImage recorta el cuadro de la hoja sin copiar píxeles.
		dst.DrawImage(sheetImgs[l.sheet].SubImage(l.rect).(*ebiten.Image), op)
	}
}

// demoLook es la apariencia de un personaje de la demo, que no la trae de
// game: sale de game.RandomAppearance con una semilla fija sacada del ID,
// así el mismo personaje siempre se ve igual.
func demoLook(id string) game.Appearance { return saltedLook(id, demoSeedSalt) }

// demoSeedSalt se mezcla con el ID en la semilla. Con esta, los 2 médicos
// de la demo salen con piel distinta y no hay dos pacientes con el mismo
// sombrero y el mismo color de camisa (las dos reglas las fija un test).
// De las que cumplen, es la primera en la que además los médicos tienen
// pelo distinto y los pacientes llevan 3 pieles y los 3 sombreros.
const demoSeedSalt = 83

func saltedLook(id string, salt uint64) game.Appearance {
	h := fnv.New64a()
	h.Write([]byte(id))
	return game.RandomAppearance(rand.New(rand.NewSource(int64(h.Sum64() ^ salt))))
}
