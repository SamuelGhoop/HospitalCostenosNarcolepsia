package hospital

import "errors"

// Errores centinela del hospital.
//
// En Go los errores son valores, no excepciones: un método que puede fallar
// devuelve un error como último resultado y quien lo llama decide qué hacer.
// Estos son valores fijos ("centinelas") para que quien llama pueda
// preguntar QUÉ falló con errors.Is, aunque el mensaje traiga más detalle:
//
//	room, err := h.AssignRoom(p)
//	if errors.Is(err, hospital.ErrNoRoomAvailable) {
//		// el paciente se queda en el pasillo; el programa sigue normal
//	}
//
// Para agregar contexto sin perder el centinela se envuelve con %w:
//
//	fmt.Errorf("%w: %s se queda en el pasillo", ErrNoRoomAvailable, p)
//
// Ninguna situación prevista (como "no hay cama") usa panic.
var (
	// Admisión y personal.
	ErrNilPatient       = errors.New("el paciente es nil")
	ErrAlreadyAdmitted  = errors.New("el paciente ya fue admitido")
	ErrNotAdmitted      = errors.New("el paciente no está admitido en este hospital")
	ErrNilAttender      = errors.New("el miembro del personal es nil")
	ErrAlreadyHired     = errors.New("esa persona ya trabaja en el hospital")
	ErrNoStaffAvailable = errors.New("no hay personal disponible para atender")
	ErrDoctorFull       = errors.New("el médico ya tiene el cupo de pacientes lleno")
	ErrAlreadyHasDoctor = errors.New("el paciente ya tiene otro médico tratante")

	// Estados del paciente.
	ErrAlreadyAsleep = errors.New("el paciente ya está dormido")
	ErrAlreadyAwake  = errors.New("el paciente ya está despierto")
	ErrNotAsleep     = errors.New("el paciente no está dormido")
	ErrAlreadyInBed  = errors.New("el paciente ya está en una cama")

	// Habitaciones.
	ErrInvalidCapacity = errors.New("la capacidad de la habitación debe ser mayor que cero")
	ErrDuplicateRoom   = errors.New("ya existe una habitación con ese número")
	ErrRoomFull        = errors.New("la habitación está llena")
	ErrNotInRoom       = errors.New("el paciente no está en esta habitación")
	ErrNoRoomAvailable = errors.New("no hay habitación disponible")
)
