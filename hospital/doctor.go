package hospital

import "fmt"

// maxPatientsPerDoctor es el cupo de pacientes a cargo de cada médico.
// Con el cupo lleno el médico deja de estar disponible para emergencias.
const maxPatientsPerDoctor = 4

// Doctor es un médico del hospital. Embebe Person (igual que Patient) y
// cumple la interfaz Attender.
//
// Maneja dos relaciones distintas:
//   - patients: pacientes A CARGO, los que diagnosticó (es su médico tratante).
//   - episodes: emergencias de sueño que ATENDIÓ, de cualquier paciente.
//
// El doctor guarda sus propios episodios; por eso MyEpisodes no necesita
// recibir el hospital (lo exige la consulta 5.2).
type Doctor struct {
	Person

	specialty string
	patients  []*Patient
	episodes  []EpisodeRecord
}

// NewDoctor crea un médico sin pacientes ni episodios.
func NewDoctor(id, name string, age int, specialty string) *Doctor {
	return &Doctor{
		Person:    NewPerson(id, name, age),
		specialty: specialty,
	}
}

// Specialty devuelve la especialidad del médico.
func (d *Doctor) Specialty() string { return d.specialty }

// Patients devuelve una copia de la lista de pacientes a cargo.
func (d *Doctor) Patients() []*Patient {
	out := make([]*Patient, len(d.patients))
	copy(out, d.patients)
	return out
}

// DiagnosePatient toma al paciente a cargo: el médico pasa a ser su médico
// tratante. Diagnosticar dos veces al mismo paciente no hace nada. Falla si
// el paciente ya tiene OTRO médico tratante o si el cupo está lleno.
func (d *Doctor) DiagnosePatient(p *Patient) error {
	switch {
	case p == nil:
		return ErrNilPatient
	case p.assignedDoctor == d:
		return nil // ya es su paciente
	case p.assignedDoctor != nil:
		return fmt.Errorf("%w: %s está con %s", ErrAlreadyHasDoctor, p, p.assignedDoctor.Name())
	case len(d.patients) >= maxPatientsPerDoctor:
		return fmt.Errorf("%w: %s ya tiene %d pacientes", ErrDoctorFull, d.Name(), len(d.patients))
	}
	d.patients = append(d.patients, p)
	p.assignedDoctor = d
	return nil
}

// AttendSleepEmergency es la atención clínica de un ataque de sueño:
// confirma que el paciente de verdad esté dormido. No lo toma a cargo (eso es
// DiagnosePatient) ni crea el registro (eso lo hace Attend).
func (d *Doctor) AttendSleepEmergency(p *Patient) error {
	if p == nil {
		return ErrNilPatient
	}
	if p.state == Awake {
		return fmt.Errorf("%w: %s está despierto, no hay emergencia", ErrNotAsleep, p)
	}
	return nil
}

// Attend cumple la interfaz Attender: atiende la emergencia, crea el
// registro del episodio y lo guarda en el historial propio del médico.
func (d *Doctor) Attend(p *Patient, location string) (EpisodeRecord, error) {
	if err := d.AttendSleepEmergency(p); err != nil {
		// Con error se devuelve el valor cero del registro; quien llama
		// debe mirar primero err (convención de Go).
		return EpisodeRecord{}, err
	}
	rec := newEpisodeRecord(p, d, location)
	d.episodes = append(d.episodes, rec)
	return rec, nil
}

// MyEpisodes devuelve una copia de los episodios que atendió este médico
// (consulta 5.2). No recibe el hospital: la relación vive en el Doctor.
func (d *Doctor) MyEpisodes() []EpisodeRecord {
	out := make([]EpisodeRecord, len(d.episodes))
	copy(out, d.episodes)
	return out
}

// IsAvailable cumple la interfaz Attender: el médico puede salir a una
// emergencia mientras no tenga el cupo de pacientes a cargo lleno.
func (d *Doctor) IsAvailable() bool { return len(d.patients) < maxPatientsPerDoctor }
