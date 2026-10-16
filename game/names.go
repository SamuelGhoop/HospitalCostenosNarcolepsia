package game

import "math/rand"

// Nombres costeños (GAME_DESIGN §7). Los nombres van en dos listas para poder
// ponerles "Dra." o "Dr." a los médicos.
var (
	femaleFirstNames = []string{"Yeimy", "Yuleidis", "Nayibe", "Ledys", "Rosiris", "Yorledis", "Marelvis"}
	maleFirstNames   = []string{"Wilfrido", "Dairo", "Éder", "Keyner", "Yeferson", "Aníbal", "Hernando", "Dagoberto", "Ronaldo"}
	surnames         = []string{"Berrío", "Padilla", "Barrios", "Arrieta", "Mendoza", "Julio", "Polo", "Pertuz", "Cassiani", "Ospina", "Castro", "Herrera", "Altamar", "Cantillo"}
)

// randomName arma "Nombre Apellido" al azar. female dice si el nombre salió
// de la lista femenina.
//
// El nombre se sortea UNA vez sobre las dos listas juntas, así cada nombre
// tiene la misma probabilidad aunque las listas tengan distinto largo.
func randomName(rng *rand.Rand) (name string, female bool) {
	i := rng.Intn(len(femaleFirstNames) + len(maleFirstNames))
	var first string
	if i < len(femaleFirstNames) {
		first, female = femaleFirstNames[i], true
	} else {
		first = maleFirstNames[i-len(femaleFirstNames)]
	}
	return first + " " + surnames[rng.Intn(len(surnames))], female
}

// doctorNameAndGender es como doctorName, pero además devuelve female: lo
// usa la bolsa de empleo (§6, F4) para decidir "Médica" en la hoja del
// candidato con el mismo sorteo que ya pone "Dra." en el nombre, sin volver
// a tirar el dado.
func doctorNameAndGender(rng *rand.Rand) (name string, female bool) {
	n, female := randomName(rng)
	if female {
		return "Dra. " + n, true
	}
	return "Dr. " + n, false
}

// doctorName es un nombre al azar con su título: "Dra. Yeimy Polo".
func doctorName(rng *rand.Rand) string {
	name, _ := doctorNameAndGender(rng)
	return name
}

// randomAge devuelve una edad al azar entre lo y hi, los dos incluidos.
func randomAge(rng *rand.Rand, lo, hi int) int {
	return lo + rng.Intn(hi-lo+1)
}
