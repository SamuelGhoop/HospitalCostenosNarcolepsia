package game

import (
	"fmt"
	"math/rand"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// candidate es una ficha de la bolsa de empleo (GAME_DESIGN §6), antes de
// contratarla. Cuando el jugador la contrata, estos mismos datos (nombre,
// edad, especialidad, estadísticas y apariencia) pasan tal cual al
// *hospital.Doctor/Orderly real y al StaffMember que lo envuelve: el
// candidato que se ve en la hoja es el mismo que después camina en el mapa.
type candidate struct {
	isDoctor  bool
	name      string
	age       int
	specialty string     // "" en los camilleros
	speed     int        // 1 a maxStat
	skill     int        // 1 a maxStat; 0 en los camilleros
	salary    int        // sueldo diario (la nómina de las 20:00)
	cost      int        // lo que cuesta contratarlo hoy: el sueldo encarecido según el día (hireCost)
	look      Appearance // sorteada con el azar cosmético, como el resto del personal
	female    bool       // mismo sorteo que decide "Dr."/"Dra." en el nombre (§6, F4): decide "Médica"/"Camillera" en RoleLabel
}

// roleLabel decide cómo se llama el rol en la hoja del candidato (§6, F4):
// la vista aprobada usa "MÉDICA" para una doctora, así que el rol lleva
// género, con el mismo dato que ya puso "Dra." en el nombre. gui/ui solo
// dibuja esta palabra (en mayúsculas): no vuelve a decidir el género.
func roleLabel(isDoctor, female bool) string {
	switch {
	case isDoctor && female:
		return "Médica"
	case isDoctor:
		return "Médico"
	case female:
		return "Camillera"
	default:
		return "Camillero"
	}
}

// view copia el candidato a una vista de solo valores para el Snapshot.
// money es la plata actual del hospital, para que la interfaz sepa si
// alcanza sin tener que repetir la cuenta.
func (c *candidate) view(money int) CandidateView {
	return CandidateView{
		IsDoctor:   c.isDoctor,
		Name:       c.name,
		RoleLabel:  roleLabel(c.isDoctor, c.female),
		Specialty:  c.specialty,
		Age:        c.age,
		Speed:      c.speed,
		Skill:      c.skill,
		Salary:     c.salary,
		Cost:       c.cost, // §5.8: un día de sueldo adelantado, encarecido según el día (hireCost)
		Look:       c.look,
		Affordable: money >= c.cost,
	}
}

// HireLine es una contratación hecha durante un día (§6); queda en el
// DayReport (report.go) del día en que se hizo.
type HireLine struct {
	Name      string
	IsDoctor  bool
	Cost      int
	RoleLabel string // "Médico"/"Médica"/"Camillero"/"Camillera" (§9, CONTRATADOS): el mismo de CandidateView
}

// CandidateView es la ficha de un candidato que dibuja la bolsa de empleo
// (§6): solo valores, como el resto de las vistas de GAME_DESIGN §3.
type CandidateView struct {
	IsDoctor   bool
	Name       string
	RoleLabel  string // "Médico"/"Médica"/"Camillero"/"Camillera" (§6, F4): ya decidido por el juego, gui/ui solo lo dibuja
	Specialty  string // "" en los camilleros
	Age        int
	Speed      int
	Skill      int // 0 en los camilleros
	Salary     int // sueldo por día
	Cost       int // costo de contratarlo hoy: el sueldo encarecido según el día (hireCost)
	Look       Appearance
	Affordable bool // si la plata del hospital alcanza para contratarlo
}

// weightedStat sortea la rapidez o la pericia (1 a maxStat) con los pesos de
// statWeights: más probabilidad de salir 2 o 3 (§6).
func weightedStat(rng *rand.Rand) int {
	n := rng.Intn(100)
	acc := 0
	for i, w := range statWeights {
		acc += w
		if n < acc {
			return i + 1
		}
	}
	return maxStat // no debería llegar aquí: los pesos suman 100
}

// generateCandidatesLocked sortea los candidatesPerDay candidatos del día
// (§6): al crear la partida y cada vez que empieza un día nuevo. Sale de
// g.hiring, no de g.rng: la bolsa de empleo es un sorteo aparte (como la
// apariencia con g.looks) para que generarla no mueva la secuencia de rng
// que usa el resto de la partida (llegadas, ataques, nombres del personal
// inicial) en los tests con semilla fija.
func (g *Game) generateCandidatesLocked() {
	g.candidates = make([]*candidate, candidatesPerDay)
	for i := range g.candidates {
		c := g.newCandidateLocked()
		c.cost = hireCost(c.salary, g.clock.day)
		g.candidates[i] = c
	}
}

// hireCost es lo que cuesta contratar a alguien de sueldo salary el día day:
// cada día un hireCostStepPercent más caro, hasta hireCostMaxPercent
// (config.go). Aritmética entera, como el resto de la plata.
func hireCost(salary, day int) int {
	percent := 100 + hireCostStepPercent*(day-1)
	if percent > hireCostMaxPercent {
		percent = hireCostMaxPercent
	}
	return salary * percent / 100
}

// newCandidateLocked sortea un candidato: ≈ 66 % médico, el resto camillero.
func (g *Game) newCandidateLocked() *candidate {
	age := randomAge(g.hiring, staffMinAge, staffMaxAge)
	look := RandomAppearance(g.looks) // azar cosmético: no toca la secuencia de g.rng ni la de g.hiring
	if g.hiring.Intn(100) < doctorChancePercent {
		speed, skill := weightedStat(g.hiring), weightedStat(g.hiring)
		name, female := doctorNameAndGender(g.hiring)
		return &candidate{
			isDoctor:  true,
			name:      name,
			female:    female,
			age:       age,
			specialty: doctorSpecialties[g.hiring.Intn(len(doctorSpecialties))],
			speed:     speed,
			skill:     skill,
			salary:    doctorSalary(speed, skill),
			look:      look,
		}
	}
	// El camillero no lleva título, pero su nombre sale de la misma lista
	// con género (§7): female decide "Camillera" en RoleLabel.
	name, female := randomName(g.hiring)
	speed := weightedStat(g.hiring)
	return &candidate{
		isDoctor: false,
		name:     name,
		female:   female,
		age:      age,
		speed:    speed,
		salary:   orderlySalary(speed),
		look:     look,
	}
}

// hireFromCandidateLocked crea el StaffMember de c, con el ID siguiente del
// esquema de siempre (D-XX o C-XX). A diferencia de newDoctorLocked y
// newOrderlyLocked, NO sortea nada: usa los datos exactos de la ficha que
// vio el jugador, para que el candidato contratado sea el mismo que camina
// en el mapa (§6).
func (g *Game) hireFromCandidateLocked(c *candidate) *StaffMember {
	if c.isDoctor {
		g.doctorCount++
		d := hospital.NewDoctor(fmt.Sprintf("D-%02d", g.doctorCount), c.name, c.age, c.specialty)
		m := newDoctorMember(d, c.speed, c.skill, c.female)
		m.look = c.look
		m.salary = c.salary
		return m
	}
	g.orderlyCount++
	o := hospital.NewOrderly(fmt.Sprintf("C-%02d", g.orderlyCount), c.name, c.age)
	m := newOrderlyMember(o, c.speed, c.female)
	m.look = c.look
	m.salary = c.salary
	return m
}

// Hire contrata al candidato i de la bolsa de empleo del día (§6): cuesta un
// día de salario adelantado, crea el Doctor u Orderly real envuelto en
// StaffMember y lo contrata con HireStaff (invariante de §3.1: ningún
// Attender sin envolver). El candidato contratado sale de la lista del día.
func (g *Game) Hire(i int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.over != NotOver {
		return ErrGameOver
	}
	if i < 0 || i >= len(g.candidates) {
		return fmt.Errorf("%w: %d", ErrNoCandidate, i)
	}
	if len(g.staff) >= maxStaff {
		return ErrStaffFull
	}
	c := g.candidates[i]
	if g.money < c.cost {
		return ErrNotEnoughMoney
	}

	m := g.hireFromCandidateLocked(c)
	if err := g.hireLocked(m); err != nil {
		// No debería pasar (los IDs son consecutivos y nuevos), pero si
		// HireStaff fallara no se le cobra ni sale de la lista.
		return err
	}
	g.money -= c.cost
	g.dayHires = append(g.dayHires, HireLine{Name: m.Name(), IsDoctor: c.isDoctor, Cost: c.cost, RoleLabel: roleLabel(c.isDoctor, c.female)})
	// Sale de la lista del día (§6): se recorta el slice en el índice i.
	g.candidates = append(g.candidates[:i], g.candidates[i+1:]...)
	return nil
}
