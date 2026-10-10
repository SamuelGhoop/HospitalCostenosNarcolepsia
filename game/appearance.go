package game

import (
	"fmt"
	"math/rand"
)

// Appearance es cómo se ve un paciente (GAME_DESIGN §7). El juego solo
// guarda números y constantes: la interfaz decide qué sprite o qué color
// corresponde a cada uno.
type Appearance struct {
	SkinTone     int // tono de piel, de 0 a skinTones-1 (el sprite del cuerpo)
	ShirtColor   int // índice en la paleta de camisas de la interfaz
	ShirtPattern ShirtPattern
	Hat          Hat
	Hair         Hair
	HairColor    int // índice en la paleta de pelo de la interfaz (con Bald no se usa)
}

// RandomAppearance sortea una apariencia. Cada campo se sortea por separado,
// con la misma probabilidad y SIN restricciones entre campos: cualquier pelo
// puede salir con cualquier sombrero.
func RandomAppearance(rng *rand.Rand) Appearance {
	return Appearance{
		SkinTone:     rng.Intn(skinTones),
		ShirtColor:   rng.Intn(shirtColors),
		ShirtPattern: ShirtPattern(rng.Intn(shirtPatterns)),
		Hat:          Hat(rng.Intn(hats)),
		Hair:         Hair(rng.Intn(hairStyles)),
		HairColor:    rng.Intn(hairColors),
	}
}

// ShirtPattern es el estampado de la camisa.
type ShirtPattern int

const (
	PlainShirt   ShirtPattern = iota // lisa
	FloralShirt                      // floreada
	StripedShirt                     // a rayas

	shirtPatterns = 3 // cuántos estampados hay (para sortear)
)

// String implementa fmt.Stringer para ShirtPattern.
func (p ShirtPattern) String() string {
	switch p {
	case PlainShirt:
		return "lisa"
	case FloralShirt:
		return "floreada"
	case StripedShirt:
		return "rayas"
	default:
		return fmt.Sprintf("ShirtPattern(%d)", int(p))
	}
}

// Hat es el sombrero (solo pacientes).
type Hat int

const (
	NoHat       Hat = iota // sin sombrero
	VueltiaoHat            // sombrero vueltiao
	CapHat                 // gorra

	hats = 3
)

// String implementa fmt.Stringer para Hat.
func (h Hat) String() string {
	switch h {
	case NoHat:
		return "ninguno"
	case VueltiaoHat:
		return "vueltiao"
	case CapHat:
		return "gorra"
	default:
		return fmt.Sprintf("Hat(%d)", int(h))
	}
}

// Hair es el peinado.
type Hair int

const (
	ShortHair  Hair = iota // corto
	AfroHair               // afro
	BraidsHair             // trenzas
	Bald                   // calvo

	hairStyles = 4
)

// String implementa fmt.Stringer para Hair.
func (h Hair) String() string {
	switch h {
	case ShortHair:
		return "corto"
	case AfroHair:
		return "afro"
	case BraidsHair:
		return "trenzas"
	case Bald:
		return "calvo"
	default:
		return fmt.Sprintf("Hair(%d)", int(h))
	}
}
