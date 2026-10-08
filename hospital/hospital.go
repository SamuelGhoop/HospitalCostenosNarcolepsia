package hospital

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Hospital es el sistema central: conoce a sus pacientes, su personal, sus
// habitaciones y el historial de todos los episodios.
//
// # Concurrencia
//
// Un solo candado (mu) protege TODO el estado: el del hospital y el de los
// pacientes, habitaciones y doctores que maneja. Cada método exportado lo
// toma al empezar (Lock) y lo suelta al terminar (defer Unlock), así que dos
// goroutines nunca modifican el hospital al mismo tiempo.
//
// Patient, Room y Doctor por sí solos NO son seguros para uso concurrente:
// el código con goroutines debe pasar siempre por los métodos de Hospital.
//
// Convención de nombres: los métodos en minúscula que terminan en "Locked"
// asumen que el candado YA está tomado. Existen porque el Mutex de Go no es
// reentrante: si RegisterEpisode (que ya tiene el candado) llamara a
// AssignRoom (que lo pide otra vez), la goroutine se quedaría esperándose a
// sí misma para siempre (deadlock). Por eso ambos usan assignRoomLocked.
type Hospital struct {
	mu sync.Mutex

	name         string
	doctors      []*Doctor
	staff        []Attender // polimorfismo: *Doctor y *Orderly en el mismo slice
	patients     []*Patient // slice y no map: conserva el orden de admisión
	rooms        []*Room    // en orden de registro: "la primera libre" es la de número más bajo registrado
	history      []EpisodeRecord
	nextAttender int // posición del próximo turno en staff (round-robin)
}

// NewHospital crea un hospital vacío.
func NewHospital(name string) *Hospital {
	return &Hospital{name: name}
}

// Name devuelve el nombre del hospital. No usa el candado porque el nombre
// nunca cambia después de crearlo: leer algo que nadie escribe no es una
// carrera.
func (h *Hospital) Name() string { return h.name }

// ─── Admisión, personal y habitaciones ────────────────────────────────

// AdmitPatient registra un paciente. Falla si es nil o si ya hay un
// paciente con el mismo ID.
func (h *Hospital) AdmitPatient(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	h.mu.Lock()
	defer h.mu.Unlock() // defer: se ejecuta al salir del método, por cualquier return

	for _, admitted := range h.patients {
		if admitted.ID() == p.ID() {
			return fmt.Errorf("%w: %s", ErrAlreadyAdmitted, p)
		}
	}
	h.patients = append(h.patients, p)
	return nil
}

// HireDoctor contrata un médico: queda en la lista de médicos y también en
// el personal que atiende emergencias.
func (h *Hospital) HireDoctor(d *Doctor) error {
	// Se revisa nil aquí, ANTES de convertirlo a Attender: un *Doctor nil
	// metido en una interfaz ya no es "nil" para la interfaz (es una
	// interfaz con tipo *Doctor y valor nil), y esa trampa no se detectaría.
	if d == nil {
		return ErrNilAttender
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.hireLocked(d); err != nil {
		return err
	}
	h.doctors = append(h.doctors, d)
	return nil
}

// HireStaff contrata a cualquier Attender (un camillero, o una enfermera si
// se crea ese tipo). El hospital no necesita cambiar para aceptar tipos nuevos.
func (h *Hospital) HireStaff(a Attender) error {
	if a == nil {
		return ErrNilAttender
	}
	// Type assertion "coma ok": ¿el valor dentro de la interfaz es un *Doctor?
	// Si lo es, se contrata como médico para que también salga en Doctors().
	// Se llama a HireDoctor ANTES de tomar el candado (HireDoctor lo toma).
	if d, ok := a.(*Doctor); ok {
		return h.HireDoctor(d)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hireLocked(a)
}

// hireLocked agrega a alguien al personal si su ID no está repetido.
func (h *Hospital) hireLocked(a Attender) error {
	for _, member := range h.staff {
		if member.ID() == a.ID() {
			return fmt.Errorf("%w: %s (%s)", ErrAlreadyHired, a.Name(), a.ID())
		}
	}
	h.staff = append(h.staff, a)
	return nil
}

// AddRoom registra una habitación nueva. Falla si la capacidad no es válida
// o si ya existe una habitación con ese número.
func (h *Hospital) AddRoom(number, capacity int) error {
	r, err := NewRoom(number, capacity)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, existing := range h.rooms {
		if existing.number == number {
			return fmt.Errorf("%w: %d", ErrDuplicateRoom, number)
		}
	}
	h.rooms = append(h.rooms, r)
	return nil
}

// ─── Episodios ─────────────────────────────────────────────────────────

// RegisterEpisode registra que un paciente se quedó dormido en location:
//
//  1. elige por turnos a alguien del personal que esté disponible;
//  2. el paciente sufre el ataque (queda AsleepInHallway);
//  3. se busca la primera habitación libre;
//  4. el elegido lo atiende y se crea el registro;
//  5. el registro se guarda en el historial.
//
// Que no haya cama NO es un error de este método: el episodio se registra
// sin habitación y el paciente se queda AsleepInHallway (consulta 5.3). El
// error legible de "sin cama" lo da AssignRoom cuando se vuelve a intentar.
//
// Devuelve error solo en fallas reales (no admitido, ya dormido, nadie
// disponible), y en ese caso NO cambia nada.
func (h *Hospital) RegisterEpisode(p *Patient, location string) error {
	if p == nil {
		return ErrNilPatient
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isAdmittedLocked(p) {
		return fmt.Errorf("%w: %s", ErrNotAdmitted, p)
	}

	// 1. Se elige ANTES de tocar al paciente: si no hay nadie, todo queda igual.
	attender, next, err := h.pickAttenderLocked()
	if err != nil {
		return err
	}

	// 2. El ataque. Si ya estaba dormido falla aquí, sin haber cambiado nada.
	if err := p.SufferSleepAttack(location); err != nil {
		return err
	}

	// 3. Cama. ErrNoRoomAvailable se ignora a propósito: se queda en el pasillo.
	if _, err := h.assignRoomLocked(p); err != nil && !errors.Is(err, ErrNoRoomAvailable) {
		return err
	}

	// 4. Atención. El hospital no sabe si attender es Doctor u Orderly:
	//    llama Attend y cada tipo hace lo suyo (polimorfismo).
	rec, err := attender.Attend(p, location)
	if err != nil {
		return err
	}

	// 5. Todo salió bien: se confirma el turno y se guarda el registro.
	h.nextAttender = next
	h.history = append(h.history, rec)
	return nil
}

// pickAttenderLocked busca por turnos (round-robin) al primer miembro del
// personal disponible, empezando donde quedó el turno anterior, para
// repartir el trabajo. Solo usa los métodos de la interfaz Attender: nunca
// pregunta si es Doctor u Orderly.
//
// Devuelve también la posición del siguiente turno, pero no la guarda:
// RegisterEpisode la guarda solo si el episodio se registra completo.
func (h *Hospital) pickAttenderLocked() (Attender, int, error) {
	n := len(h.staff)
	for i := 0; i < n; i++ {
		idx := (h.nextAttender + i) % n // % hace que después del último se vuelva al primero
		if a := h.staff[idx]; a.IsAvailable() {
			return a, (idx + 1) % n, nil
		}
	}
	return nil, 0, ErrNoStaffAvailable
}

// WakePatient despierta a un paciente del hospital; si estaba en cama, la
// habitación queda libre. Es la entrada segura (con candado) a Patient.WakeUp.
func (h *Hospital) WakePatient(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isAdmittedLocked(p) {
		return fmt.Errorf("%w: %s", ErrNotAdmitted, p)
	}
	return p.WakeUp()
}

// ─── Consultas (Sección 5) ─────────────────────────────────────────────

// PatientsInHallway devuelve los pacientes dormidos en un pasillo, para
// despacharles un camillero (consulta 5.1).
func (h *Hospital) PatientsInHallway() []*Patient {
	h.mu.Lock()
	defer h.mu.Unlock()

	var inHallway []*Patient // empieza nil; append lo crea si hace falta
	for _, p := range h.patients {
		if p.state == AsleepInHallway {
			inHallway = append(inHallway, p)
		}
	}
	return inHallway
}

// AssignRoom busca la primera habitación libre y acuesta ahí al paciente:
// la habitación queda Occupied y el paciente AsleepInBed (consulta 5.3).
//
// Si no hay ninguna libre devuelve un error que envuelve ErrNoRoomAvailable,
// el paciente sigue AsleepInHallway y no se inventa ninguna habitación.
func (h *Hospital) AssignRoom(p *Patient) (*Room, error) {
	if p == nil {
		return nil, ErrNilPatient
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isAdmittedLocked(p) {
		return nil, fmt.Errorf("%w: %s", ErrNotAdmitted, p)
	}
	return h.assignRoomLocked(p)
}

// assignRoomLocked es el trabajo de AssignRoom, sin tomar el candado, para
// que RegisterEpisode también lo pueda usar.
func (h *Hospital) assignRoomLocked(p *Patient) (*Room, error) {
	switch p.state {
	case Awake:
		return nil, fmt.Errorf("%w: %s está despierto, no necesita cama", ErrNotAsleep, p)
	case AsleepInBed:
		return nil, fmt.Errorf("%w: %s ya está en la habitación %d", ErrAlreadyInBed, p, p.room.number)
	}
	for _, r := range h.rooms {
		if r.IsAvailable() {
			if err := r.Occupy(p); err != nil {
				return nil, err
			}
			return r, nil
		}
	}
	return nil, fmt.Errorf("%w: %s se queda dormido en %s", ErrNoRoomAvailable, p, p.currentLocation)
}

// SevereReport cruza dos fuentes: los pacientes con narcolepsia severa y
// los episodios de HOY que hay en el historial. Devuelve cuántos episodios
// tuvo cada paciente severo; los que no tuvieron ninguno aparecen con 0
// (consulta 5.4).
//
// Es un map porque responde "¿cuántos tuvo ESTE paciente?" en un paso.
// Ojo: un map no tiene orden; quien imprime debe ordenarlo.
func (h *Hospital) SevereReport() map[*Patient]int {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Fuente 1: los pacientes severos, todos arrancan en 0.
	report := make(map[*Patient]int)
	for _, p := range h.patients {
		if p.level == Severe {
			report[p] = 0
		}
	}

	// Fuente 2: el historial. Solo cuentan los episodios de hoy y de
	// pacientes que estén en el reporte ("coma ok" sobre el map).
	today := time.Now()
	for _, rec := range h.history {
		if _, isSevere := report[rec.patient]; isSevere && sameDay(rec.at, today) {
			report[rec.patient]++
		}
	}
	return report
}

// ─── Getters (devuelven copias) ────────────────────────────────────────

// Patients devuelve una copia de la lista de pacientes, en orden de admisión.
func (h *Hospital) Patients() []*Patient {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Patient(nil), h.patients...) // copia en una línea
}

// Doctors devuelve una copia de la lista de médicos.
func (h *Hospital) Doctors() []*Doctor {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Doctor(nil), h.doctors...)
}

// Staff devuelve una copia de todo el personal (médicos y camilleros).
func (h *Hospital) Staff() []Attender {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Attender(nil), h.staff...)
}

// Rooms devuelve una copia de la lista de habitaciones.
func (h *Hospital) Rooms() []*Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Room(nil), h.rooms...)
}

// History devuelve una copia del historial de episodios, en orden.
func (h *Hospital) History() []EpisodeRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]EpisodeRecord(nil), h.history...)
}

// ─── Ayudantes internos ────────────────────────────────────────────────

// isAdmittedLocked dice si p (este mismo objeto) fue admitido aquí.
func (h *Hospital) isAdmittedLocked(p *Patient) bool {
	for _, admitted := range h.patients {
		if admitted == p {
			return true
		}
	}
	return false
}

// sameDay dice si dos instantes caen en el mismo día del calendario.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date() // Date devuelve tres valores: año, mes y día
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
