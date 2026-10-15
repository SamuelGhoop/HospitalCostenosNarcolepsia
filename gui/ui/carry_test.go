package ui

import (
	"image"
	"math"
	"testing"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/game"
)

// Todo traslado de la demo va en camilla (pedido de Samuel, 2026-10-11):
// al recogerlo, la camilla aparece junto al paciente; mientras lo llevan,
// el paciente va encima y la camilla delante de quien empuja, horizontal o
// vertical según hacia dónde avanzan, sin pisar a nadie que esté quieto;
// al llegar, el paciente pasa a la cama y la camilla desaparece. Lo hacen
// médicos y camillero.
func TestDemo_TransfersGoOnAStretcher(t *testing.T) {
	needFonts(t)
	s := NewDemoScene(game.NewDemo())
	type seen struct {
		appearedNear, rode, pushedFromBehind, onBed bool
		doctor                                      bool
	}
	carries := map[string]*seen{} // por paciente
	for step := 1; step <= 11; step++ {
		if err := s.advance(); err != nil {
			t.Fatal(err)
		}
		var last *carrying
		for tick := 0; tick < 8000 && s.choreo != nil; tick++ {
			s.animate()
			c := s.carry
			if c == nil {
				if last != nil && last.order != nil {
					o := last.order
					carries[o.patient].onBed = s.pos[o.patient] == bedCell(o.room)
				}
				last = nil
				continue
			}
			o := c.order
			if carries[o.patient] == nil {
				f, _ := s.figure(o.pusher)
				carries[o.patient] = &seen{doctor: f.doctor}
			}
			cs := carries[o.patient]
			if c.visible() && !cs.appearedNear { // el primer tick con camilla
				body := floorBody.Add(image.Pt(int(s.pos[o.patient].x), int(s.pos[o.patient].y)))
				d := math.Hypot(float64(c.fp.Min.X+c.fp.Dx()/2-(body.Min.X+body.Dx()/2)), float64(c.fp.Min.Y+c.fp.Dy()/2-(body.Min.Y+body.Dy()/2)))
				if d > 3*figWidth {
					t.Errorf("paso %d: la camilla apareció a %.0f px de %s; se esperaba junto a él", step, d, o.patient)
				}
				cs.appearedNear = true
			}
			if c.riding() {
				cs.rode = true
				if s.pos[o.patient] != riderPos(c.fp) {
					t.Fatalf("paso %d, tick %d: %s no va sobre la camilla (%v; la camilla pide %v)", step, tick, o.patient, s.pos[o.patient], riderPos(c.fp))
				}
				feet := image.Pt(int(math.Round(s.pos[o.pusher].x+feetOffset.x)), int(math.Round(s.pos[o.pusher].y+feetOffset.y)))
				if c.fp != stretcherFootprint(feet, c.dir) {
					t.Fatalf("paso %d, tick %d: la camilla %v no va delante de %s (se esperaba %v)", step, tick, c.fp, o.pusher, stretcherFootprint(feet, c.dir))
				}
				if (c.fp.Dx() > c.fp.Dy()) != c.dir.horizontal() {
					t.Fatalf("paso %d, tick %d: yendo %v la camilla mide %v", step, tick, c.dir, c.fp.Size())
				}
				cs.pushedFromBehind = true
				for _, f := range s.figures { // la camilla no pisa a nadie que esté quieto
					if s.inCarry(f.id) || (f.patient && s.onBed(f)) || s.pos[f.id] != s.destination(f) || s.pos[f.id].x >= ScreenWidth {
						continue
					}
					feet := feetBox.Add(image.Pt(int(math.Round(s.pos[f.id].x+feetOffset.x)), int(math.Round(s.pos[f.id].y+feetOffset.y))))
					if c.fp.Overlaps(feet) {
						t.Fatalf("paso %d, tick %d: la camilla (%v) pisa los pies de %s (%v), que está quieto", step, tick, c.fp, f.id, feet)
					}
				}
			}
			last = c
		}
	}

	doctors, orderlies := 0, 0
	for patient, cs := range carries {
		if !cs.appearedNear || !cs.rode || !cs.pushedFromBehind || !cs.onBed {
			t.Errorf("%s: apareció la camilla junto a él=%v, fue encima=%v, empujada desde atrás=%v, terminó en la cama=%v",
				patient, cs.appearedNear, cs.rode, cs.pushedFromBehind, cs.onBed)
		}
		if cs.doctor {
			doctors++
		} else {
			orderlies++
		}
	}
	if len(carries) < 4 || doctors == 0 || orderlies == 0 {
		t.Errorf("hubo %d traslados (%d de médicos y %d del camillero); se esperaban los 4 de la demo (pasos 5, 6, 7 y 11), de los dos", len(carries), doctors, orderlies)
	}
}

// La camilla es un obstáculo para las rutas de los demás, no para quien la
// empuja ni para el paciente que va encima.
func TestDemo_OthersAvoidTheStretcher(t *testing.T) {
	s := &DemoScene{}
	if got := s.avoidFor("P-005"); got != nil {
		t.Errorf("sin traslado, P-005 esquiva %v; no había nada que esquivar", got)
	}
	fp := image.Rect(600, 230, 654, 254)
	s.carry = &carrying{order: &carryOrder{pusher: "C-01", patient: "P-004"}, stage: carryPush, fp: fp}
	if got := s.avoidFor("P-005"); len(got) != 1 || got[0] != fp {
		t.Errorf("con la camilla en %v, P-005 esquiva %v; se esperaba la camilla", fp, got)
	}
	for _, id := range []string{"C-01", "P-004"} {
		if got := s.avoidFor(id); got != nil {
			t.Errorf("%s esquiva %v; no se esquiva su propia camilla", id, got)
		}
	}
	s.carry.stage = carryToPickup // la camilla todavía no aparece
	if got := s.avoidFor("P-005"); got != nil {
		t.Errorf("antes de aparecer la camilla, P-005 esquiva %v", got)
	}
}

// Los que están quietos en el camino de una camilla se hacen a un lado
// mientras pasa (a un sitio fuera de su camino) y después vuelven a su
// puesto.
//
// Antes esto se veía en la demo tal cual (alguien que esperaba en la
// recepción). Desde el giro de golpe (2026-10-15, GAME_DESIGN §10.8) ya no
// hace falta "preferir girar donde hay espacio" (la razón era no barrer una
// pared en pleno giro, que ya no existe: el giro no pasa por cuadros
// intermedios), y con eso la camilla de la demo gira en otro sitio y ya no
// le pasa por encima a nadie quieto. El mecanismo sigue haciendo falta para
// el juego (varias camillas, recepción llena), así que se prueba directo
// con un traslado real (pasillo 2 → 101) y alguien de pie a mitad del
// primer tramo.
func TestDemo_PeopleStepAsideForTheStretcher(t *testing.T) {
	needFonts(t)
	plan, ok := planCarry(toVec(floorSpot(zoneHallway2, 0)), zoneRoom101, nil)
	if !ok {
		t.Fatal("no hay ruta con la camilla del pasillo 2 a la 101")
	}
	lane := carryLane(plan)
	blockerAt := vec{(plan.pickup.x + plan.path[0].x) / 2, (plan.pickup.y + plan.path[0].y) / 2}
	if !overlapsAny(feetRectAt(blockerAt), lane) {
		t.Fatal("el sitio elegido para el que espera no queda en el camino de la camilla: el test no prueba nada")
	}
	s := &DemoScene{}
	s.figures = []figure{
		{id: "C-01", target: plan.pickup},
		{id: "P-999", target: plan.anchor()},
		{id: "W-01", target: blockerAt},
	}
	order := &carryOrder{pusher: "C-01", patient: "P-999", room: zoneRoom101, plan: plan}
	dodge := s.dodgesFor(order)
	spot, ok := dodge["W-01"]
	if !ok {
		t.Fatal("W-01 está en el camino de la camilla y no se hizo a un lado")
	}
	if overlapsAny(feetRectAt(spot), lane) {
		t.Errorf("el sitio al que se hizo a un lado (%v) sigue en el camino de la camilla", spot)
	}
}
