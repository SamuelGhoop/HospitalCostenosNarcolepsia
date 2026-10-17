package ui

import (
	"image"
	"math"
	"math/rand"
	"sort"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Personal libre: dónde descansa y cómo deambula (GAME_DESIGN §10.8 punto
// 7, pedido de Samuel el 2026-10-17: "cuando contrato gente solo hay 3
// espacios donde aparece el personal, entonces aparecen 4 en el mismo
// lugar" y "el personal no esté tan quieto").

// ─── Puesto fijo en la sala (con su desborde al pasillo 2) ─────────────

// staffHomeSpot es el i-ésimo puesto para el personal libre: primero los de
// pie de la sala del personal (spotsOf) y, si no alcanzan, el desborde del
// pasillo 2 junto a su puerta (staffOverflowSpots). Mismo respaldo que
// slot: si tampoco alcanzan, se repite el último (no debería pasar, porque
// hay 6 puestos para los 6 que deja contratar game.maxStaff).
func staffHomeSpot(i int) image.Point {
	s := spotsOf(zoneStaffRoom).standing
	if i < len(s) {
		return s[i]
	}
	o := staffOverflowSpots()
	switch j := i - len(s); {
	case j < len(o):
		return o[j]
	case len(o) > 0:
		return o[len(o)-1]
	case len(s) > 0:
		return s[len(s)-1]
	default:
		return zones[zoneStaffRoom].rect.Min
	}
}

// staffHomeTarget da el puesto de id en la sala del personal: book.take le
// da siempre el mismo índice a la misma persona (y el primero libre a
// quien llega), así nadie se encima del puesto del que ya estaba (el bug
// de Samuel: antes se usaba la posición de la foto, que no es estable ni
// respeta cuántos puestos hay de verdad).
func staffHomeTarget(book spotBook, id string) image.Point {
	return staffHomeSpot(book.take(zoneStaffRoom, id))
}

// staffOverflowCache guarda staffOverflowSpots: se calcula una sola vez,
// como spotCache.
var staffOverflowCache []image.Point

// staffOverflowSpots son los puestos de pie en el pasillo 2, junto a la
// puerta de la sala del personal (doorways[8]), para cuando la sala ya no
// alcanza: el cupo general de la zona (spotsOf, el que reparte entre
// quienes deambulan por ahí) más otros que buscan igual (sin tocar
// muebles, con los rótulos adentro de la zona) pero SIN dejarle el sitio
// al paciente que podría desplomarse en el pasillo 2 (floorOf se lo deja
// al cupo general; acá no, porque el pasillo es tan chico que si también
// se reservara ese espacio no alcanzaría para el personal). En la
// práctica el personal casi no llega a usar los que no son del cupo
// general (se necesitan más de 4 libres a la vez) y, si alguna vez un
// paciente se desploma justo en uno de esos, en el peor caso un rótulo se
// monta sobre el otro, igual que el resto de los puestos relajados de la
// sección 10.8 punto 3 (pendiente del visto bueno de Samuel).
func staffOverflowSpots() []image.Point {
	if staffOverflowCache != nil {
		return staffOverflowCache
	}
	z := zoneHallway2
	r := zones[z].rect
	door := doorways[8].gap // pasillo 2 ↔ sala del personal

	out := append([]image.Point(nil), spotsOf(z).standing...) // el cupo general, primero
	var taken []shape
	for _, p := range out {
		taken = append(taken, standingShape(p))
	}
	for y := r.Min.Y; y+figHeight <= r.Max.Y; y += spotStep {
		for x := r.Min.X; x+figWidth <= r.Max.X; x += spotStep {
			at := image.Pt(x, y)
			sh := standingShape(at)
			if spotFree(z, at, standingBody.Add(at), sh, false, taken) {
				out = append(out, at)
				taken = append(taken, sh)
			}
		}
	}
	doorCenter := image.Pt(door.Min.X+door.Dx()/2, door.Min.Y+door.Dy()/2)
	center := func(p image.Point) image.Point {
		b := standingBody.Add(p)
		return b.Min.Add(b.Max).Div(2)
	}
	sort.Slice(out, func(i, j int) bool {
		ci, cj := center(out[i]), center(out[j])
		di := math.Hypot(float64(ci.X-doorCenter.X), float64(ci.Y-doorCenter.Y))
		dj := math.Hypot(float64(cj.X-doorCenter.X), float64(cj.Y-doorCenter.Y))
		return di < dj
	})
	staffOverflowCache = out
	return out
}

// ─── Deambular mientras está libre ──────────────────────────────────────

// restZonesFor son las zonas entre las que deambula el personal libre
// (Activity Free): siempre la sala y, además, por donde dijo Samuel el
// 2026-10-17 que anda cada uno cuando no tiene nada que hacer. Los médicos,
// entre la sala, la cafetería ("por un tinto") y el pasillo 2; los
// camilleros, entre la sala y la recepción, de pie junto a la entrada.
func restZonesFor(doctor bool) []zone {
	if doctor {
		return []zone{zoneStaffRoom, zoneCafeteria, zoneHallway2}
	}
	return []zone{zoneStaffRoom, zoneLobby}
}

// restWanderTicks: cuánto se queda el personal libre en un lugar antes de
// cambiar, de 8 a 15 s (480 a 900 ticks de Update, a 60 por segundo). El
// azar es propio de la interfaz (rng), nunca el de la partida (g.rng):
// así no cambia la secuencia de los tests con semilla fija del juego.
func restWanderTicks(rng *rand.Rand) int {
	const minTicks, maxTicks = 8 * 60, 15 * 60
	return minTicks + rng.Intn(maxTicks-minTicks+1)
}

// staffRestZoneSpot da el puesto de pie de la zona z para el personal
// libre: el de la sala (con su desborde, staffHomeTarget) o el cupo
// general de la zona (compartido con quien deambula o llega ahí, igual que
// los pacientes).
func staffRestZoneSpot(book spotBook, z zone, id string) image.Point {
	if z == zoneStaffRoom {
		return staffHomeTarget(book, id)
	}
	return slot(z, book.take(z, id))
}

// staffWanderRNG es el azar PROPIO DE LA INTERFAZ con el que el personal
// libre decide a qué zona ir y cuánto se queda: nunca el de la partida
// (game.Game.rng), para no cambiar la secuencia de los tests con semilla
// fija del juego. Sale de seedSource, igual que el tráfico decorado
// (traffic.go): con el backdoor de pruebas (-agent -seed N) es
// determinista. Se vuelve a armar en cada partida nueva (se nota porque
// cambia el puntero de la escena).
var (
	staffWanderRNG   *rand.Rand
	staffWanderScene *GameScene
)

// wanderRNG da el azar de deambular de esta escena.
func (s *GameScene) wanderRNG() *rand.Rand {
	if staffWanderScene != s {
		staffWanderRNG, staffWanderScene = rand.New(rand.NewSource(seedSource())), s
	}
	return staffWanderRNG
}

// staffRestSpot es a dónde camina el personal libre (Activity Free, "la
// actividad de reposo" de la foto): cada 8 a 15 s cambia de zona entre las
// suyas (restZonesFor), para que no se vea tan quieto (pedido de Samuel,
// 2026-10-17: "las animaciones ya están mejor así... también no sé si
// estoy jugándolo mal" — y, aparte, que el personal no esté tan quieto).
// Mientras está ocupado (Carrying, GoingToReview…) este método no se
// llama, así que el reloj de deambular queda congelado.
func (s *GameScene) staffRestSpot(a *actor, st game.StaffMemberView) image.Point {
	if a.restTicks > 0 {
		a.restTicks--
	} else {
		zs := restZonesFor(st.IsDoctor)
		next := a.restZone
		for next == a.restZone && len(zs) > 1 {
			next = zs[s.wanderRNG().Intn(len(zs))]
		}
		s.standing.release(st.ID, next)
		a.restZone = next
		a.restTicks = restWanderTicks(s.wanderRNG())
	}
	return staffRestZoneSpot(s.standing, a.restZone, st.ID)
}
