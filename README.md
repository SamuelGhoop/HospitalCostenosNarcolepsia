# Hospital de los Costeños con Narcolepsia

Sistema de gestión del hospital escrito en Go. Registra dónde está cada paciente, qué médico lo atiende y cada episodio de sueño repentino, para asignarle cama rápido antes de que bloquee un pasillo.

Ejercicio aplicado de **Programación Orientada a Objetos**, Universidad EIA (profesor Sebastián Zapata Ramírez).

| | |
|---|---|
| **Integrante** | Samuel Giraldo Jiménez |
| **Lenguaje** | Núcleo: Go 1.21 o superior (`go.mod` declara `go 1.21`), solo librería estándar. Interfaz gráfica: **Go 1.26** o superior (ver abajo) |
| **Estado** | Modelo, consultas, escenario y bonus de concurrencia completos. Interfaz gráfica: la demostración de la Sección 6 funciona; el modo juego está en construcción |

## Cómo correrlo

```bash
go run .          # escenario de la Sección 6, las cuatro consultas y, al final, la simulación concurrente (2 s)
go test ./...     # pruebas de la raíz: hospital, simulation y game
go vet ./...      # análisis estático: no reporta nada
gofmt -l .        # formato: no lista ningún archivo
```

### Interfaz gráfica (bonus)

```bash
cd gui
go run .          # abre la ventana con la demostración de la Sección 6
go test ./...     # pruebas de la interfaz
```

> **La interfaz gráfica necesita Go 1.26 o superior.** Está en un módulo aparte (`gui/go.mod`) porque Ebitengine v2.10 y `golang.org/x/image` exigen esa versión; el núcleo sigue en Go 1.21. Si tu Go es más viejo, `go` descarga solo el toolchain 1.26 (`GOTOOLCHAIN=auto`, el valor por defecto), así que **la primera vez se necesita internet**, también para descargar Ebitengine. Después funciona sin conexión.
>
> En Windows y macOS no hace falta nada más. En Linux, Ebitengine necesita cgo y las librerías de desarrollo de X11 y OpenGL (ver la guía de instalación de Ebitengine).

Controles de la demostración:

| Tecla | Acción |
|---|---|
| Espacio o clic | Siguiente paso |
| A | Avance automático (un paso cada 3 s) |
| S | Saltar al final |
| C | Mostrar u ocultar los subtítulos |

Cada paso muestra el subtítulo, la **llamada real** al modelo (por ejemplo `h.AssignRoom(P-004)`) y lo que respondió. El paso 8 muestra en rojo el error real de "no hay habitación disponible". Al final aparece el Shift Report con las cuatro consultas.

**Detector de carreras.** `-race` necesita cgo y un gcc de 64 bits. En Windows sin ese compilador, se corre en Docker con la versión mínima de Go:

```bash
docker run --rm -v "<ruta-del-repo>:/src" -w /src golang:1.21 sh -c "go test -race ./... && go run -race ."
```

Resultado verificado con Go 1.21.13: tests en verde y `go run -race .` termina sin ningún `DATA RACE`.

## Estructura

```
main.go          escenario de demostración + impresión (sin lógica de negocio)
hospital/        el modelo: structs, métodos, interfaz, consultas y sus tests
  person.go        Person: identidad compartida (se embebe)
  patient.go       Patient + NarcolepsyLevel + PatientState
  doctor.go        Doctor (cumple Attender)
  orderly.go       Orderly, el camillero (cumple Attender)
  attender.go      interfaz Attender
  room.go          Room + RoomState
  episode.go       EpisodeRecord
  hospital.go      Hospital: despacho, candado y las cuatro consultas
  errors.go        errores centinela
  *_test.go        tests (los 3 obligatorios están al inicio de hospital_test.go)
simulation/      (bonus) simulación concurrente: una goroutine por paciente
game/            reglas de la demostración y del juego, encima de hospital (solo stdlib)
  demo.go          los 12 pasos de la Sección 6 sobre su propio hospital
  report.go        las cuatro consultas copiadas a vistas (solo valores)
gui/             (bonus) interfaz en Ebitengine: módulo Go aparte (go 1.26)
  main.go          abre la ventana
  ui/              escenas (demostración, Shift Report), mapa y dibujo
  design/          maquetas de Claude Design (solo referencia visual, no se compilan)
GAME_DESIGN.md   especificación del modo demostración y del juego
```

## Cómo se cumple el enunciado

| Requisito | Dónde |
|---|---|
| 5 structs con campos en minúscula y métodos | `Hospital`, `Patient`, `Doctor`, `Room`, `EpisodeRecord` en `hospital/` |
| Estados como constantes tipadas | `PatientState`, `NarcolepsyLevel`, `RoomState` con `iota` + `String()` |
| `Person` embebida, sin campos duplicados | `Patient`, `Doctor` y `Orderly` embeben `Person` |
| Interfaz usada polimórficamente | `Hospital.staff` es un `[]Attender` con `*Doctor` y `*Orderly`; `pickAttenderLocked` despacha sin saber el tipo |
| 5.1 Pacientes en el pasillo | `func (h *Hospital) PatientsInHallway() []*Patient` |
| 5.2 Historial por doctor (sin recibir el hospital) | `func (d *Doctor) MyEpisodes() []EpisodeRecord` |
| 5.3 Disponibilidad de camas | `func (h *Hospital) AssignRoom(p *Patient) (*Room, error)` |
| 5.4 Reporte de severidad | `func (h *Hospital) SevereReport() map[*Patient]int` |
| "Sin cama" devuelve error, sin `panic` | `AssignRoom` envuelve `ErrNoRoomAvailable`; `main` lo imprime y sigue |
| 3 tests obligatorios | `hospital_test.go`, marcados `[Obligatorio 1/2/3]` |
| Bonus: goroutines + `sync.Mutex`, limpio con `-race` | `simulation/simulation.go` + el candado de `Hospital`; sección final de `go run .` |
| Bonus: interfaz gráfica separada de la lógica | `gui/` (Ebitengine, módulo aparte). `hospital/` no cambió al agregarla: `git diff --stat modelo-cerrado -- hospital/` sale vacío |

## Decisiones de diseño

**División en paquetes.** `hospital` tiene todo el modelo y no imprime nada. `main` solo arma el escenario y formatea la salida. `game` tiene las reglas de la demostración y del juego, y solo cambia el modelo a través de los métodos de `Hospital`. La interfaz gráfica está en `gui/`, un **módulo Go aparte** (con su propio `go.mod`): `go test ./...` desde la raíz no entra a módulos anidados, así que los tests del núcleo no dependen de Ebitengine ni de las librerías gráficas del sistema. La interfaz no decide nada: solo dibuja las vistas que le entrega `game` (copias con valores, nunca punteros del modelo) y llama sus métodos. El tag `modelo-cerrado` marca el modelo antes de la interfaz.

**Composición en vez de herencia.** `Person` (id, nombre, edad) se embebe **por valor** en `Patient`, `Doctor` y `Orderly`. Sus métodos se promueven (`paciente.Name()`) y le dan a `Doctor` y `Orderly` dos de los cuatro métodos de `Attender`. Se embebe por valor y no como `*Person` porque cada persona tiene su propia identidad y el valor cero es seguro. Con `*Person`, dos copias compartirían los datos y un `Patient{}` sin inicializar tendría `Person == nil`.

**Interfaz `Attender` y despacho por turnos.** El hospital guarda a todo el personal en un `[]Attender` y reparte las emergencias por turnos (round-robin), saltando a quien no esté disponible. Solo usa los métodos de la interfaz, nunca el tipo concreto. `HireStaff` acepta cualquier `Attender`, así que un tipo nuevo (por ejemplo, una enfermera) se agrega sin cambiar `Hospital`. `attender.go` verifica en compilación que `*Doctor` y `*Orderly` cumplen la interfaz.

**Por qué `Doctor.IsAvailable` depende del cupo.** Cada médico tiene un cupo de 4 pacientes a cargo (los que diagnosticó). Con el cupo lleno no puede salir a una emergencia de pasillo sin desatender a los suyos, así que el despachador lo salta y busca al siguiente. El camillero siempre está disponible: un traslado es rápido y no le deja pacientes a cargo. Atender una emergencia (`Attend`) **no** agrega al paciente a cargo; eso solo lo hace `DiagnosePatient`.

**`RegisterEpisode` y la falta de cama.** `RegisterEpisode` elige a quién despachar, duerme al paciente, busca la primera cama libre, crea el registro y lo guarda. Que no haya cama **no** es un error de este método: el episodio queda registrado sin habitación y el paciente sigue `AsleepInHallway`, como pide la consulta 5.3. El error legible lo produce `AssignRoom` cuando `main` vuelve a pedir la cama (Sección 6). `RegisterEpisode` solo devuelve error en fallas reales (paciente no admitido, ya dormido, nadie disponible), y en ese caso no cambia nada.

**El camillero sin cama.** Si el paciente no consiguió cama, `Orderly.Attend` no tiene a dónde llevarlo: lo deja vigilado en el pasillo, el registro queda con habitación `nil` y no suma un traslado.

**Registros inmutables.** `EpisodeRecord` guarda la habitación y el médico tratante **del momento** del episodio. `Summary()` muestra quién atendió y quién es el tratante. Si después el paciente se despierta, el registro no cambia. Los IDs (`EP-001`…) salen de un contador atómico del paquete, porque quien crea el registro es el `Attender` y este no conoce al hospital.

**Slice o map.** Los pacientes, habitaciones y el personal se guardan en **slices** porque importa el orden: el de admisión, y "la primera habitación libre". Con 5 pacientes, recorrer la lista es instantáneo. `SevereReport` devuelve un **map** porque responde "¿cuántos episodios tuvo este paciente?" en un paso, y así lo pide el enunciado. Como un map no tiene orden, `main` lo ordena por ID antes de imprimir. El reporte cruza dos fuentes: los pacientes `Severe` (todos aparecen, aunque tengan 0) y los episodios de hoy en el historial.

**Errores.** Los métodos que pueden fallar devuelven `error` como último valor. Se usan errores centinela (`ErrNoRoomAvailable`, `ErrNotAsleep`…) envueltos con `%w` para dar contexto, y se comparan con `errors.Is`. Ninguna situación prevista usa `panic`.

**Concurrencia a nivel de `Hospital`.** Un solo `sync.Mutex` protege todo el estado del hospital, y con él el de sus pacientes, habitaciones y doctores. `Patient`, `Room` y `Doctor` por sí solos **no** son seguros para uso concurrente: el código con goroutines debe entrar siempre por los métodos de `Hospital`, y los getters de las entidades se leen **después de `wg.Wait()`**, cuando ya no hay goroutines escribiendo. El `Mutex` de Go no es reentrante. Por eso los métodos públicos toman el candado y delegan en helpers `...Locked` (por ejemplo `assignRoomLocked`): si `RegisterEpisode` llamara a `AssignRoom`, se bloquearía esperándose a sí mismo. Un test lanza 10 goroutines por 3 camas y pasa con `-race`.

**Bonus: simulación concurrente.** `simulation.Run` lanza una goroutine por paciente. Cada una repite un ciclo: despierto un rato al azar, ataque de sueño (`RegisterEpisode`), dormido un rato (si quedó en el pasillo, reintenta `AssignRoom` y compite con las demás por las camas que se liberan) y despertar (`WakePatient`). Las goroutines **solo** llaman métodos de `Hospital`. Para saber si el paciente quedó en cama le preguntan al hospital (`AssignRoom` responde `ErrAlreadyInBed`), no al paciente. Los contadores son `atomic.Int64`. La simulación se detiene con `context.WithTimeout`, y `Run` espera a todas las goroutines (`sync.WaitGroup`) antes de devolver, así que `main` lee los resultados cuando ya nadie escribe.

**Go 1.21.** `go.mod` declara la versión mínima que acepta el enunciado. En esa versión la variable de un `for` es la misma en cada vuelta, así que antes de lanzar una goroutine se copia explícitamente (`p := p`).

**Saltos de línea.** `.gitattributes` fuerza LF. Sin eso, Git para Windows descarga los `.go` con CRLF y `gofmt -l` los marca a todos.

## Salida del escenario (resumida)

```
== Ataques de sueño ==
  P-001 Yeimy Padilla        dormido en cama        habitación 101
  P-002 Kevin Mercado        dormido en cama        habitación 102
  P-003 Ludys Arrieta        dormido en cama        habitación 103
  P-004 Wilfrido Berrío      dormido en el pasillo  pasillo 2, segundo piso

== Consulta 5.3 — Asignar cama a P-004 Wilfrido Berrío ==
  ✗ no hay habitación disponible: P-004 Wilfrido Berrío se queda dormido en pasillo 2, segundo piso
...
== Consulta 5.3 — Asignar cama a P-004 Wilfrido Berrío (otra vez) ==
  ✓ P-004 Wilfrido Berrío → habitación 101
...
== Consulta 5.4 — Reporte de severidad (pacientes Severe) ==
  P-001 Yeimy Padilla        1 episodio(s) hoy
  P-004 Wilfrido Berrío      1 episodio(s) hoy
  P-005 Breiner Julio        0 episodio(s) hoy
```

## Avance

- [x] Fase 0: repositorio, módulo y esqueleto
- [x] Fase 1: `Person`, estados tipados y errores
- [x] Fase 2: `Patient` y `Room`
- [x] Fase 3: `Attender`, `Doctor`, `Orderly` y `EpisodeRecord`
- [x] Fase 4: `Hospital` y las cuatro consultas
- [x] Fase 5: escenario de demostración en `main.go`
- [x] Fase 6: bonus de simulación concurrente
- [ ] Fase 7: bonus de interfaz gráfica (ver `GAME_DESIGN.md`)
  - [x] Demostración de la Sección 6 paso a paso, con subtítulos y Shift Report
  - [ ] Modo juego
- [ ] Fase 8: revisión final y `AI_USAGE.md`
