package game

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SamuelGhoop/HospitalCostenosNarcolepsia/hospital"
)

// DemoSteps es la cantidad de pasos del guion de la Sección 6.
const DemoSteps = 12

// DemoStep describe un paso ya ejecutado de la demostración: lo que
// necesita la barra de subtítulos de la interfaz.
type DemoStep struct {
	Number   int    // 1 a 12
	Subtitle string // qué está pasando, en palabras
	Call     string // la llamada real que se le hizo al modelo
	Result   string // lo que respondió el modelo (no es texto fijo)
	Err      error  // error PREVISTO del modelo (solo el paso 8); nil en los demás
}

// DemoSnapshot es la foto de la demo en un momento dado. Solo valores.
type DemoSnapshot struct {
	Step, TotalSteps int      // pasos ejecutados y total
	Last             DemoStep // el último paso ejecutado (vacío al inicio)
	Patients         []PatientView
	Staff            []StaffView
	Rooms            []RoomView
}

// Demo reproduce, paso a paso, el escenario obligatorio de la Sección 6
// (el mismo de main.go) sobre su propio hospital.
//
// El personal NO va envuelto en StaffMember: se contrata con HireDoctor y
// HireStaff reales para que el despacho sea el round-robin original.
//
// Demo no tiene candado: no tiene goroutine motor y solo la usa la
// goroutine de la interfaz, así que nunca hay dos goroutines tocándola.
type Demo struct {
	h *hospital.Hospital

	karen, efrain *hospital.Doctor
	wilmer        *hospital.Orderly

	yeimy, kevin, ludys, wilfrido, breiner *hospital.Patient

	// steps es el guion: una función por paso. Next ejecuta steps[done].
	steps         []func() (DemoStep, error)
	done          int      // cuántos pasos se han ejecutado
	last          DemoStep // el último paso ejecutado
	lastAssignErr string   // último error de AssignRoom (consulta 5.3)
}

// NewDemo prepara la demostración. Crea a las personas, pero el hospital
// arranca vacío: contratar, habilitar y admitir son los primeros pasos.
func NewDemo() *Demo {
	d := &Demo{
		h:        hospital.NewHospital("Hospital de los Costeños con Narcolepsia"),
		karen:    hospital.NewDoctor("D-01", "Dra. Karen Ospina", 45, "Neurología del sueño"),
		efrain:   hospital.NewDoctor("D-02", "Dr. Efraín Barraza", 52, "Medicina interna"),
		wilmer:   hospital.NewOrderly("C-01", "Wilmer Camargo", 29),
		yeimy:    hospital.NewPatient("P-001", "Yeimy Padilla", 34, hospital.Severe),
		kevin:    hospital.NewPatient("P-002", "Kevin Mercado", 27, hospital.Moderate),
		ludys:    hospital.NewPatient("P-003", "Ludys Arrieta", 58, hospital.Mild),
		wilfrido: hospital.NewPatient("P-004", "Wilfrido Berrío", 61, hospital.Severe),
		breiner:  hospital.NewPatient("P-005", "Breiner Julio", 22, hospital.Severe),
	}

	// El guion. d.hireStaff (sin paréntesis) es un "method value": guarda el
	// método ya atado a d, listo para llamarse después como cualquier función.
	// Los ataques usan funciones anónimas porque necesitan argumentos.
	d.steps = []func() (DemoStep, error){
		d.hireStaff,
		d.addRooms,
		d.admitPatients,
		d.diagnose,
		func() (DemoStep, error) {
			return d.attack(d.yeimy, "cafetería, después del sancocho", "Yeimy se duerme en la cafetería, después del sancocho")
		},
		func() (DemoStep, error) {
			return d.attack(d.kevin, "fila de radiología", "Kevin se duerme en la fila de radiología")
		},
		func() (DemoStep, error) {
			return d.attack(d.ludys, "pista de champeta del patio", "Ludys se duerme en la pista de champeta")
		},
		d.noBedForWilfrido,
		d.showHallway,
		d.wakeYeimy,
		d.bedForWilfrido,
		d.shiftReport,
	}
	return d
}

// Next ejecuta el siguiente paso del guion y lo devuelve.
// Devuelve ErrDemoFinished si ya se ejecutaron los 12. Cualquier otro error
// es inesperado (el escenario está pensado para no fallar) y la demo no avanza.
func (d *Demo) Next() (DemoStep, error) {
	if d.Done() {
		return DemoStep{}, ErrDemoFinished
	}
	step, err := d.steps[d.done]()
	if err != nil {
		return DemoStep{}, fmt.Errorf("paso %d de la demo: %w", d.done+1, err)
	}
	d.done++
	step.Number = d.done
	d.last = step
	return step, nil
}

// Skip ejecuta todos los pasos que faltan (tecla S de la interfaz).
func (d *Demo) Skip() error {
	for !d.Done() {
		if _, err := d.Next(); err != nil {
			return err
		}
	}
	return nil
}

// Done dice si ya se ejecutaron todos los pasos.
func (d *Demo) Done() bool { return d.done >= len(d.steps) }

// Snapshot devuelve la foto actual de la demo (solo valores).
func (d *Demo) Snapshot() DemoSnapshot {
	snap := DemoSnapshot{Step: d.done, TotalSteps: len(d.steps), Last: d.last}
	for _, p := range d.h.Patients() {
		snap.Patients = append(snap.Patients, patientViewOf(p))
	}
	for _, a := range d.h.Staff() {
		// Type assertion "coma ok": en la demo el personal no va envuelto,
		// así que basta preguntar si es un *hospital.Doctor.
		_, isDoctor := a.(*hospital.Doctor)
		snap.Staff = append(snap.Staff, StaffView{ID: a.ID(), Name: a.Name(), IsDoctor: isDoctor})
	}
	for _, r := range d.h.Rooms() {
		snap.Rooms = append(snap.Rooms, roomViewOf(r))
	}
	return snap
}

// Report devuelve las cuatro consultas con el estado actual de la demo.
// Las horas de los episodios son las reales, como en la consola.
func (d *Demo) Report() Report {
	return buildReport(d.h, d.h.Doctors(), d.lastAssignErr, realTime)
}

// ─── Los pasos del guion ───────────────────────────────────────────────
// Cada uno hace las llamadas reales al modelo y arma el subtítulo con lo
// que el modelo respondió.

// Paso 1.
func (d *Demo) hireStaff() (DemoStep, error) {
	for _, err := range []error{d.h.HireDoctor(d.karen), d.h.HireDoctor(d.efrain), d.h.HireStaff(d.wilmer)} {
		if err != nil {
			return DemoStep{}, err
		}
	}
	var names []string
	for _, a := range d.h.Staff() {
		names = append(names, a.Name())
	}
	return DemoStep{
		Subtitle: "Se contratan dos médicos y un camillero",
		Call:     "h.HireDoctor(D-01) · h.HireDoctor(D-02) · h.HireStaff(C-01)",
		Result:   "personal: " + strings.Join(names, ", "),
	}, nil
}

// Paso 2.
func (d *Demo) addRooms() (DemoStep, error) {
	var rooms []string
	for _, number := range []int{101, 102, 103} {
		if err := d.h.AddRoom(number, 1); err != nil {
			return DemoStep{}, err
		}
	}
	for _, r := range d.h.Rooms() {
		rooms = append(rooms, fmt.Sprintf("%d (%v)", r.Number(), r.State()))
	}
	return DemoStep{
		Subtitle: "Se habilitan tres habitaciones de una cama",
		Call:     "h.AddRoom(101, 1) · h.AddRoom(102, 1) · h.AddRoom(103, 1)",
		Result:   strings.Join(rooms, ", "),
	}, nil
}

// Paso 3.
func (d *Demo) admitPatients() (DemoStep, error) {
	for _, p := range []*hospital.Patient{d.yeimy, d.kevin, d.ludys, d.wilfrido, d.breiner} {
		if err := d.h.AdmitPatient(p); err != nil {
			return DemoStep{}, err
		}
	}
	severe := 0
	for _, p := range d.h.Patients() {
		if p.Level() == hospital.Severe {
			severe++
		}
	}
	return DemoStep{
		Subtitle: "Llegan cinco costeños con narcolepsia",
		Call:     "h.AdmitPatient(P-001) … h.AdmitPatient(P-005)",
		Result:   fmt.Sprintf("%d pacientes admitidos, %d con narcolepsia severa", len(d.h.Patients()), severe),
	}, nil
}

// Paso 4: el mismo reparto de médicos tratantes que main.go.
func (d *Demo) diagnose() (DemoStep, error) {
	for _, p := range []*hospital.Patient{d.yeimy, d.kevin, d.breiner} {
		if err := d.karen.DiagnosePatient(p); err != nil {
			return DemoStep{}, err
		}
	}
	for _, p := range []*hospital.Patient{d.ludys, d.wilfrido} {
		if err := d.efrain.DiagnosePatient(p); err != nil {
			return DemoStep{}, err
		}
	}
	return DemoStep{
		Subtitle: "Cada paciente queda a cargo de su médico tratante",
		Call:     "karen.DiagnosePatient(P-001, P-002, P-005) · efrain.DiagnosePatient(P-003, P-004)",
		Result: fmt.Sprintf("%s: %d pacientes · %s: %d pacientes",
			d.karen.Name(), len(d.karen.Patients()), d.efrain.Name(), len(d.efrain.Patients())),
	}, nil
}

// attack es el ataque de sueño de los pasos 5 a 8: el hospital despacha a
// quien toque por turno y, si hay, le da la primera cama libre.
func (d *Demo) attack(p *hospital.Patient, location, subtitle string) (DemoStep, error) {
	if err := d.h.RegisterEpisode(p, location); err != nil {
		return DemoStep{}, err
	}
	// Quién atendió: el último registro del historial.
	history := d.h.History()
	attendedBy := history[len(history)-1].AttendedBy().Name()

	destination := "pasillo (sin cama)"
	if r := p.Room(); r != nil {
		destination = fmt.Sprintf("habitación %d", r.Number())
	}
	return DemoStep{
		Subtitle: subtitle,
		Call:     fmt.Sprintf("h.RegisterEpisode(%s, %q)", p.ID(), location),
		Result:   fmt.Sprintf("%s → %s · atendió: %s", p, destination, attendedBy),
	}, nil
}

// Paso 8: Wilfrido se duerme y ya no hay cama. Se vuelve a pedir con
// AssignRoom para mostrar el error real del modelo.
func (d *Demo) noBedForWilfrido() (DemoStep, error) {
	step, err := d.attack(d.wilfrido, "pasillo 2, segundo piso", "Wilfrido se duerme en el pasillo 2… y no quedan camas")
	if err != nil {
		return DemoStep{}, err
	}
	_, assignErr := d.h.AssignRoom(d.wilfrido)
	if !errors.Is(assignErr, hospital.ErrNoRoomAvailable) {
		// El escenario exige que aquí falte cama; cualquier otra cosa es un bug.
		return DemoStep{}, fmt.Errorf("se esperaba que no hubiera cama y AssignRoom respondió: %v", assignErr)
	}
	d.lastAssignErr = assignErr.Error()

	step.Call += " · h.AssignRoom(P-004)"
	step.Result = "error: " + assignErr.Error()
	step.Err = assignErr
	return step, nil
}

// Paso 9: consulta 5.1.
func (d *Demo) showHallway() (DemoStep, error) {
	var lines []string
	for _, p := range d.h.PatientsInHallway() {
		lines = append(lines, fmt.Sprintf("%s (%s)", p, p.Location()))
	}
	result := "nadie"
	if len(lines) > 0 {
		result = strings.Join(lines, ", ")
	}
	return DemoStep{
		Subtitle: "Consulta 5.1: ¿quién está dormido en el pasillo?",
		Call:     "h.PatientsInHallway()",
		Result:   result,
	}, nil
}

// Paso 10: Yeimy se despierta y su cama queda libre.
func (d *Demo) wakeYeimy() (DemoStep, error) {
	room := d.yeimy.Room() // se guarda antes de despertarla: después queda en nil
	if room == nil {
		return DemoStep{}, fmt.Errorf("se esperaba que %s estuviera en una cama", d.yeimy)
	}
	if err := d.h.WakePatient(d.yeimy); err != nil {
		return DemoStep{}, err
	}
	return DemoStep{
		Subtitle: "Yeimy se despierta y libera su cama",
		Call:     "h.WakePatient(P-001)",
		Result:   fmt.Sprintf("%s %v · la %d queda %v", d.yeimy, d.yeimy.State(), room.Number(), room.State()),
	}, nil
}

// Paso 11: ahora sí hay cama para Wilfrido. (En la interfaz se anima al
// camillero, pero AssignRoom no involucra a ningún Attender.)
func (d *Demo) bedForWilfrido() (DemoStep, error) {
	room, err := d.h.AssignRoom(d.wilfrido)
	if err != nil {
		return DemoStep{}, err
	}
	return DemoStep{
		Subtitle: "Wilfrido por fin consigue cama",
		Call:     "h.AssignRoom(P-004)",
		Result:   fmt.Sprintf("%s → habitación %d", d.wilfrido, room.Number()),
	}, nil
}

// Paso 12: fin del turno, el Shift Report con las cuatro consultas.
func (d *Demo) shiftReport() (DemoStep, error) {
	r := d.Report()
	occupied := 0
	for _, room := range r.Rooms {
		if room.State == hospital.Occupied {
			occupied++
		}
	}
	return DemoStep{
		Subtitle: "Fin del turno: Shift Report con las cuatro consultas",
		Call:     "h.PatientsInHallway() · d.MyEpisodes() · h.Rooms() · h.SevereReport()",
		Result: fmt.Sprintf("en el pasillo: %d · habitaciones ocupadas: %d de %d · pacientes severos: %d",
			len(r.Hallway), occupied, len(r.Rooms), len(r.Severe)),
	}, nil
}
