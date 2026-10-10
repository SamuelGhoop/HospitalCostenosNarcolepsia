package game_test

import (
	"math/rand"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// §7: cada campo de la apariencia se sortea por separado, con la misma
// probabilidad y sin restricciones entre campos.
func TestRandomAppearance_EachFieldIsUniformAndIndependent(t *testing.T) {
	const n = 24000
	rng := rand.New(rand.NewSource(1))

	// Cuántas veces sale cada valor de cada campo.
	skin, shirtColor, hairColor := map[int]int{}, map[int]int{}, map[int]int{}
	pattern, hat, hair := map[game.ShirtPattern]int{}, map[game.Hat]int{}, map[game.Hair]int{}
	hairWithHat := map[[2]int]int{} // pares (pelo, sombrero)

	for i := 0; i < n; i++ {
		a := game.RandomAppearance(rng)
		skin[a.SkinTone]++
		shirtColor[a.ShirtColor]++
		hairColor[a.HairColor]++
		pattern[a.ShirtPattern]++
		hat[a.Hat]++
		hair[a.Hair]++
		hairWithHat[[2]int{int(a.Hair), int(a.Hat)}]++
	}

	// uniform revisa que haya exactamente k valores distintos y que cada uno
	// salga n/k veces, con un margen del 10 %.
	uniform := func(field string, counts []int, k int) {
		t.Helper()
		if len(counts) != k {
			t.Errorf("%s: salieron %d valores distintos; se esperaban %d", field, len(counts), k)
		}
		want := n / k
		for _, c := range counts {
			if c < want*9/10 || c > want*11/10 {
				t.Errorf("%s: un valor salió %d veces; se esperaban unas %d", field, c, want)
			}
		}
	}
	uniform("piel", values(skin), 4)
	uniform("color de camisa", values(shirtColor), 6)
	uniform("color de pelo", values(hairColor), 4)
	uniform("estampado", values(pattern), 3)
	uniform("sombrero", values(hat), 3)
	uniform("pelo", values(hair), 4)

	// Sin restricciones: las 12 combinaciones de pelo y sombrero salen, y
	// cada una más o menos n/12 veces (por ejemplo, afro con y sin sombrero).
	uniform("pelo con sombrero", values(hairWithHat), 12)
	if hairWithHat[[2]int{int(game.AfroHair), int(game.NoHat)}] == 0 {
		t.Error("nunca salió afro sin sombrero")
	}
}

// values devuelve los conteos de un mapa, sin importar la clave.
func values[K comparable](m map[K]int) []int {
	out := make([]int, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}
