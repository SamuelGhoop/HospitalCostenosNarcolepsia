package ui

import (
	"image"
	"math"
	"time"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// El modo juego no decide nada del hospital: cada tick lee la foto de game/
// (Snapshot) y de ahí saca a dónde camina cada personaje. Lo que dura una
// caminata lo dice el juego (Left): la escena reparte lo que le falta de
// ruta en los ticks que quedan, así el personal llega justo cuando el juego
// dice que llegó. Los pacientes que deambulan caminan a su paso.

// gameZone es la zona del mapa de una zona del juego.
func gameZone(z game.Zone) zone { return zoneFor(z.String()) }

// drawnState es el estado con el que se dibuja un paciente del juego: el
// desplomado y el del pasillo, dormidos en el piso; el que espera revisión
// y el revisado, en la cama; los demás, despiertos. (El modelo no se entera
// del desplome hasta que lo recogen, pero en pantalla ya está en el piso.)
func drawnState(st game.Stage) hospital.PatientState {
	switch st {
	case game.Collapsed, game.InHallway:
		return hospital.AsleepInHallway
	case game.AwaitingReview, game.InBed:
		return hospital.AsleepInBed
	default:
		return hospital.Awake
	}
}

// spotBook lleva quién ocupa cada puesto de una zona (por índice), para que
// cada uno conserve el suyo mientras siga en la zona.
type spotBook map[zone][]string

// take le da a id el puesto que ya tenía en z o el primero libre.
func (b spotBook) take(z zone, id string) int {
	for i, who := range b[z] {
		if who == id {
			return i
		}
	}
	for i, who := range b[z] {
		if who == "" {
			b[z][i] = id
			return i
		}
	}
	b[z] = append(b[z], id)
	return len(b[z]) - 1
}

// release libera los puestos de id en todas las zonas menos keep.
func (b spotBook) release(id string, keep zone) {
	for z, ids := range b {
		if z == keep {
			continue
		}
		for i, who := range ids {
			if who == id {
				ids[i] = ""
			}
		}
	}
}

// noZone: para liberar los puestos de alguien en todas las zonas.
const noZone zone = -2

// actor es lo que la escena recuerda de cada personaje entre fotos.
type actor struct {
	left      time.Duration // el Left de la última foto (para notar cuando cambia)
	ticksLeft int           // ticks que le quedan a la caminata cronometrada (0 = a su paso)
	activity  game.Activity // la última actividad (personal)
	stage     game.Stage    // la última etapa (pacientes)
	attend    int           // ticks que le quedan atendiendo junto a la cama, después de revisar
	attendAt  vec           // dónde atiende
	carried   string        // a quién ya llevó en el traslado actual (la camilla puede llegar antes que el juego)
	exiting   bool          // ya salió del juego: camina fuera de la pantalla y desaparece

	restZone  zone // dónde descansa el personal libre (§10.8 punto 7): empieza en su sala
	restTicks int  // ticks que le quedan ahí antes de cambiar de lugar (0 al principio: arranca en la sala sin esperar)
}

func (s *GameScene) actorOf(id string) *actor {
	a, ok := s.actors[id]
	if !ok {
		a = &actor{}
		s.actors[id] = a
	}
	return a
}

// timed cronometra la caminata de a: si el juego trae un Left nuevo, le
// quedan esos ticks (a 60 por segundo).
func (a *actor) timed(left time.Duration) {
	if left != a.left {
		a.left = left
		a.ticksLeft = int(math.Ceil(left.Seconds() * 60))
	}
}

// untimed: camina a su paso.
func (a *actor) untimed() { a.left, a.ticksLeft = 0, 0 }

// doorInside es dónde entra el que llega por la puerta principal: justo
// adentro, en la recepción.
func doorInside() vec {
	g := doorways[len(doorways)-1].gap
	return vec{float64(g.Min.X+g.Dx()/2) - feetOffset.x, float64(g.Min.Y-2*navCell) - feetOffset.y}
}

// doorOutside es dónde termina de salir el que se va: justo afuera de la
// puerta principal, en la acera.
func doorOutside() vec {
	g := doorways[len(doorways)-1].gap
	return vec{float64(g.Min.X+g.Dx()/2) - feetOffset.x, float64(g.Max.Y+2*navCell) - feetOffset.y}
}

// sync lee la foto y arma los personajes con su destino: los pacientes por
// su etapa y el personal por lo que está haciendo. También arranca los
// traslados en camilla que el juego empezó.
func (s *GameScene) sync() {
	s.figures = s.figures[:0]
	present := map[string]bool{}
	patients := map[string]game.GamePatientView{}

	for _, p := range s.snap.Patients {
		present[p.ID] = true
		patients[p.ID] = p
		a := s.actorOf(p.ID)
		f := figure{id: p.ID, patient: true, state: drawnState(p.Stage), level: p.Level, room: p.Room, look: p.Look}
		if _, seen := s.pos[p.ID]; !seen {
			s.pos[p.ID] = arrivalSpawn() // llega por la acera, desde fuera de la pantalla
			s.anims[p.ID] = &patientAnim{state: hospital.Awake}
		}
		pa := s.anims[p.ID]
		pa.observe(f.state, p.Room)
		pa.drowsy = p.Stage == game.Drowsy
		z := gameZone(p.Zone)

		switch p.Stage {
		case game.Arriving:
			s.standing.release(p.ID, noZone)
			f.target = doorInside()
			a.timed(p.Left)
		case game.Wandering:
			s.lying.release(p.ID, noZone)
			s.standing.release(p.ID, z)
			f.target = toVec(slot(z, s.standing.take(z, p.ID)))
			a.untimed()
		case game.Drowsy, game.Collapsed, game.InHallway:
			s.standing.release(p.ID, noZone)
			s.lying.release(p.ID, z) // si cambió de zona, suelta el puesto de la otra
			f.target = toVec(floorSpot(z, s.lying.take(z, p.ID)))
			a.untimed()
		case game.AwaitingReview, game.InBed:
			s.standing.release(p.ID, noZone)
			f.target = bedCell(roomZone(p.Room))
			a.untimed()
		case game.Leaving:
			s.standing.release(p.ID, noZone)
			s.lying.release(p.ID, noZone)
			f.target = doorOutside()
			a.timed(p.Left)
		}
		if p.Stage != game.Drowsy && p.Stage != game.Collapsed && p.Stage != game.InHallway {
			s.lying.release(p.ID, noZone)
		}
		a.stage = p.Stage
		s.figures = append(s.figures, f)
	}

	for _, st := range s.snap.Staff {
		present[st.ID] = true
		a := s.actorOf(st.ID)
		f := figure{id: st.ID, doctor: st.IsDoctor, look: st.Look}
		if _, seen := s.pos[st.ID]; !seen {
			// El puesto es fijo por persona (s.standing.take), no por la
			// posición en la foto: antes se usaba slot(zoneStaffRoom, i) con
			// i = la posición en s.snap.Staff, y con más de 3 contratados (la
			// sala solo tiene 3 puestos de pie) todos los que sobraban caían
			// en el mismo lugar (aviso de Samuel, 2026-10-17). staffHomeTarget
			// reparte también el desborde del pasillo 2 (§10.7 punto 7).
			a.restZone = zoneStaffRoom
			a.restTicks = restWanderTicks(s.wanderRNG()) // no deambula de una: se queda un rato en la sala
			s.pos[st.ID] = toVec(staffHomeTarget(s.standing, st.ID))
			s.staff[st.ID] = &staffAnim{}
		}
		if a.activity == game.GoingToReview && st.Activity != game.GoingToReview {
			a.attend, a.attendAt = attendTicks, s.pos[st.ID] // acaba de revisar: atiende junto a la cama
		}
		a.activity = st.Activity
		if st.Activity != game.Carrying {
			a.carried = ""
		}

		p, hasPatient := patients[st.PatientID]
		switch {
		case st.Activity == game.Carrying && hasPatient:
			s.standing.release(st.ID, noZone) // suelta su puesto de reposo mientras trabaja
			if a.carried != p.ID && s.carryOf(st.ID) == nil && s.carryOf(p.ID) == nil {
				s.startCarry(st.ID, p, st.Left)
				a.carried = p.ID // una sola camilla por traslado, aunque llegue antes que el juego
			}
			a.untimed()
		case (st.Activity == game.GoingToPickUp || st.Activity == game.GoingToTransfer) && hasPatient:
			s.standing.release(st.ID, noZone)
			f.target = helperSpot(toVec(floorSpot(gameZone(p.Zone), s.lying.take(gameZone(p.Zone), p.ID))))
			a.timed(st.Left)
		case st.Activity == game.GoingToReview && hasPatient:
			s.standing.release(st.ID, noZone)
			f.target = toVec(wakeSpot(roomZone(p.Room))) // al lado de la cama (un puesto revisado: piso libre)
			a.timed(st.Left)
		case a.attend > 0:
			s.standing.release(st.ID, noZone)
			f.target = a.attendAt
			a.untimed()
		default: // libre: deambula entre su sala y sus demás zonas (§10.8 punto 7)
			f.target = toVec(s.staffRestSpot(a, st))
			a.untimed()
		}
		s.figures = append(s.figures, f)
	}

	// Los que ya salieron del juego (cruzaron la puerta) siguen caminando
	// por la acera hasta salir de la pantalla, y ahí desaparecen.
	for id := range s.pos {
		if present[id] {
			continue
		}
		a := s.actorOf(id)
		if !a.exiting {
			a.exiting = true
			a.untimed()
			s.anims[id].observe(hospital.Awake, 0)
		}
		f := figure{id: id, patient: true, state: hospital.Awake, target: arrivalSpawn()}
		if s.pos[id] == f.target {
			delete(s.pos, id)
			delete(s.anims, id)
			delete(s.actors, id)
			delete(s.paths, id)
			s.standing.release(id, noZone)
			s.lying.release(id, noZone)
			continue
		}
		s.figures = append(s.figures, f)
	}
}

// startCarry arranca el traslado en camilla que el juego empezó: pusher
// lleva a p a su cama en left (lo que dice el juego). La velocidad sale de
// repartir el camino en el tiempo que queda, descontando lo que tarda el
// paciente en pasarse a la camilla y a la cama. El giro ya no resta tiempo:
// desde el 2026-10-15 es de golpe, en el mismo tick en que se llega a la
// esquina (GAME_DESIGN §10.8).
func (s *GameScene) startCarry(pusher string, p game.GamePatientView, left time.Duration) {
	room := roomZone(p.Room)
	from := s.pos[p.ID]
	plan, ok := planCarry(from, room, s.stillFeetExcept(pusher, p.ID))
	if !ok {
		plan, ok = planCarry(from, room, nil)
	}
	if !ok {
		return // no debería pasar (lo revisa un test): el paciente queda donde está y la foto lo pone en la cama
	}
	dist := math.Hypot(plan.pickup.x-s.pos[pusher].x, plan.pickup.y-s.pos[pusher].y)
	prev := plan.pickup
	for _, q := range plan.path {
		dist += math.Hypot(q.x-prev.x, q.y-prev.y)
		prev = q
	}
	ticks := left.Seconds()*60 - 2*float64(figHeight)/hopSpeed
	speed := staffSpeed
	if ticks > 0 {
		speed = math.Max(1, math.Min(6, dist/ticks))
	}
	s.others = append(s.others, &carrying{order: &carryOrder{pusher: pusher, patient: p.ID, room: room, plan: plan}, speed: speed})
}

// stillFeetExcept son las huellas de los pies de los que están quietos en su
// sitio (menos except): la camilla no los pisa.
func (s *GameScene) stillFeetExcept(except ...string) []image.Rectangle {
	var out []image.Rectangle
	for _, f := range s.figures {
		skip := s.onBed(f) || s.inCarry(f.id)
		for _, e := range except {
			skip = skip || f.id == e
		}
		if skip || s.pos[f.id] != f.target {
			continue
		}
		out = append(out, feetRectAt(f.target))
	}
	return out
}

// routeLeft es lo que le falta de ruta a id hasta su destino.
func (s *GameScene) routeLeft(id string, from vec) float64 {
	d, prev := 0.0, from
	for _, p := range s.paths[id] {
		d += math.Hypot(p.x-prev.x, p.y-prev.y)
		prev = p
	}
	return d
}

// animate hace un tick de movimiento: los traslados en camilla y, los
// demás, hacia su destino (a su paso o al ritmo que marca el juego).
func (s *GameScene) animate() {
	before := map[string]vec{}
	for id, p := range s.pos {
		before[id] = p
	}
	kept := s.others[:0]
	for _, c := range s.others {
		s.stepCarry(c)
		if c.stage != carryDone {
			kept = append(kept, c)
		}
	}
	s.others = kept

	for _, f := range s.figures {
		old := before[f.id]
		a := s.actorOf(f.id)
		pa, isPatient := s.anims[f.id]
		if !s.inCarry(f.id) && !(isPatient && pa.holdsStill()) {
			next := s.nextWaypoint(f.id, old, f.target)
			speed := speedOf(f)
			if a.ticksLeft > 0 {
				speed = math.Max(0.5, s.routeLeft(f.id, old)/float64(a.ticksLeft))
				a.ticksLeft--
			}
			s.pos[f.id] = moveToward(old, next, speed)
		}
		now := s.pos[f.id]
		if isPatient {
			pa.step(now.x-old.x, now.y-old.y, s.onBed(f))
		}
		if sa, ok := s.staff[f.id]; ok {
			c := s.carryOf(f.id)
			// Solo el camillero tiene fila de empujar; un médico que empuja
			// (no había camillero libre) sigue con la animación de caminar.
			pushing := !f.doctor && c != nil && c.visible() && c.order.pusher == f.id
			sa.step(now.x-old.x, now.y-old.y, a.attend > 0 && now == old, pushing)
		}
		if a.attend > 0 && now == f.target {
			a.attend--
		}
	}
}
