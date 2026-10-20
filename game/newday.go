package game

import "fmt"

// startingStaff es cuánto personal tenía la partida al empezar (§5.2): el
// médico y el camillero iniciales, contratados primero en New. Cada noche
// el personal vuelve a ser solo ese (closeDayLocked).
const startingStaff = 2

// closeDayLocked deja el hospital listo para el día siguiente (pedido de
// Samuel, 2026-10-19: "when the day ends it gets patients out of the
// hospital and staff pool restarts but money and reputation stays"). La
// llama StartNextDay con g.mu tomado. La plata, la reputación, el puntaje
// y el número del día no se tocan.
//
//   - Pacientes: todos salen del mapa, sin penalización. A los que el
//     modelo tiene dormidos (en cama o en el pasillo) se les llama
//     WakePatient, que libera la cama o el pasillo; los demás ya están
//     Awake en el modelo (o ni los había admitido, si venían por la acera),
//     así que no hay nada que decirle. Sus episodios siguen en el historial.
//   - Personal: vuelve a ser solo el inicial, todos libres. El modelo no
//     tiene cómo despedir a nadie, así que los demás siguen en su lista de
//     personal; no importa, porque el despacho por turnos solo elige a
//     quien IsAvailable() dice que sí, y el decorador StaffMember solo lo
//     dice durante un despacho "de guardia" de este juego (§3.2), que ya no
//     los va a incluir. Los contadores de IDs no se reinician: el modelo ya
//     tiene esos IDs y no se pueden repetir.
//   - Avisos y el último error de AssignRoom: se borran, el día arranca limpio.
func (g *Game) closeDayLocked() {
	for _, pt := range g.patients {
		switch pt.stage {
		case AwaitingReview, InHallway, InBed:
			if err := g.h.WakePatient(pt.p); err != nil {
				// No debería pasar: el modelo lo tiene dormido.
				g.noticeLocked(fmt.Sprintf("%s no se pudo despertar: %v", pt.p, err))
			}
		}
		if pt.stage != Arriving && pt.stage != Leaving {
			// Leaving ya tiene su nota ("(alta)" o "(se fue enojado)").
			g.notes[pt.p.ID()] = "(cerró el turno)"
		}
	}
	g.patients = nil

	if len(g.staff) > startingStaff {
		g.staff = g.staff[:startingStaff]
	}
	for _, s := range g.staff {
		s.job = nil
	}

	g.notices = nil
	g.lastAssignErr = ""
}
