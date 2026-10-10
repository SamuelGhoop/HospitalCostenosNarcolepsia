# GAME_DESIGN.md — Hospital de los Costeños con Narcolepsia (modo juego)

> Especificación del juego que se construye **encima** del modelo de `hospital/`.
> Es el bono de interfaz gráfica (+10 %) del trabajo de POO en Go (Universidad EIA). El bono de concurrencia (+5 %) ya está cubierto por `simulation/`.
> La interfaz se hace con **Ebitengine** (motor 2D en Go, juego de escritorio). El profesor autorizó usarlo para el bono.
> Todos los números de este documento son valores iniciales para balancear; viven en un solo archivo (`game/config.go`) para ajustarlos sin tocar la lógica.

---

## 1. Visión

El jugador administra un hospital en la costa caribe colombiana. Llegan costeños con narcolepsia que se quedan dormidos de repente en cualquier parte. El jugador despacha médicos y camilleros para atenderlos y subirlos a una cama antes de que bloqueen los pasillos, mientras cuida la plata y la reputación del hospital.

Estética: pixel art "chunky" retro pero legible (maquetas de Claude Design en `gui/design/claude-design/`).

### 1.1 Flujo general

1. **Modo demostración** (sección 4): la ventana reproduce, animado y paso a paso, el escenario obligatorio de la Sección 6 del enunciado, con el **despacho automático** del modelo. Termina mostrando las 4 consultas.
2. **Modo juego** (secciones 5 a 9): al terminar la demo aparece **"¡AHORA TE TOCA!"** y empieza una partida donde el jugador toma el control como jefe de turno.

La narrativa: primero se ve el sistema del hospital funcionando solo; después el jugador lo administra.

---

## 2. Restricciones del trabajo (no negociables)

1. **El paquete `hospital/` no se modifica.** La sección 7 del enunciado exige que la lógica no cambie al agregar la interfaz y que `hospital/` no tenga código de presentación.
2. **El `main.go` de la raíz no se toca.** `go run .` sigue ejecutando el escenario de la Sección 6 en consola y la simulación concurrente.
3. `simulation/` no se modifica (es el bono de concurrencia ya entregado).
4. `hospital/`, `simulation/` y `game/` usan solo la librería estándar. **Ebitengine** (`github.com/hajimehoshi/ebiten/v2`) solo se importa dentro de `gui/`, que es un **módulo Go aparte**.
5. Métodos que pueden fallar devuelven `error`; nada de `panic` para situaciones previsibles.
6. `gofmt` y `go vet` limpios; `go test ./...` en verde en la raíz y en `gui/`; `go test -race ./...` limpio en la raíz (vía Docker si no hay gcc de 64 bits, como indica el README).
7. Samuel debe poder explicar y modificar cada línea en la sustentación.

> Si el juego necesita algo que la API de `hospital/` no expone, **no se improvisa**: se le avisa a Samuel antes de tocar nada.

---

## 3. Arquitectura

```
HospitalCostenosNarcolepsia/          ← módulo raíz (go 1.21, solo stdlib)
  go.mod
  main.go                → escenario de la Sección 6 en consola (intacto)
  hospital/              → modelo del enunciado (intacto)
  simulation/            → bono de concurrencia (intacto)
  game/                  → NUEVO: reglas del juego, solo stdlib, con sus tests
    config.go            → constantes de balance
    errors.go            → errores centinela del juego
    game.go              → struct Game, mutex, ciclo de vida, Tick()
    engine.go            → goroutine motor: llama Tick cada 100 ms (sección 11)
    clock.go             → turno (08:00–20:00) y hora del juego
    staff.go             → StaffMember: decorador que cumple hospital.Attender
    dispatch.go          → despacho elegido por el jugador ("de guardia")
    patients.go          → estado de cada paciente en el juego, avanzado por ticks
    beds.go              → sueño en cama, despertar, alta y traslado desde el pasillo
    spawner.go           → llegada de pacientes
    economy.go, reputation.go, hiring.go, events.go, names.go
    snapshot.go          → Snapshot(): copia de solo lectura para la interfaz
    report.go            → Report: las 4 consultas y las vistas (solo valores)
    demo.go              → tipo Demo: guion del modo demostración (Sección 6)
    *_test.go
  gui/                   ← módulo aparte (Ebitengine)
    go.mod               → require ebiten/v2; replace del módulo raíz => ../
    main.go              → crea la App y llama ebiten.RunGame
    ui/                  → escenas, dibujo, animación, entrada
    assets/              → PNG, fuente pixel y sonidos (//go:embed)
    design/claude-design/ → maquetas de Claude Design (solo referencia, no se compila)
```

- `game/` vive en el módulo raíz para que sus tests corran con `go test ./...` desde la raíz y no dependan de Ebitengine.
- `gui/` tiene su propio `go.mod`. Puede declarar la versión de Go que pida Ebitengine sin cambiar el `go 1.21` de la raíz. Para usar el código local se agrega `replace github.com/SamuelGhoop/HospitalCostenosNarcolepsia => ../`.
- El juego se corre con `cd gui && go run .`.

| Capa | Hace | No hace |
|---|---|---|
| `hospital/` | Reglas del dominio: admitir, despachar, asignar cama, registrar episodios, consultas | Nada de juego ni de interfaz |
| `game/` | Tiempo, azar, economía, decisiones del jugador; todo cambio al hospital pasa por la API de `hospital/` | No reimplementa reglas del modelo; no importa Ebitengine |
| `gui/ui/` | Dibuja, anima y traduce clics y teclas en llamadas a `game.Game` | No decide reglas; solo lee `Snapshot()` y llama métodos de `Game` |

### 3.1 El decorador `StaffMember`

`RegisterEpisode` elige por turnos (round-robin) entre el personal cuyo `IsAvailable()` es `true`. `HireStaff` acepta **cualquier** `hospital.Attender`. El juego aprovecha eso: contrata a su personal envuelto en un tipo propio que cumple la interfaz.

```go
// StaffMember cumple hospital.Attender envolviendo al Doctor u Orderly real.
type StaffMember struct {
    inner    hospital.Attender // *hospital.Doctor o *hospital.Orderly
    doctor   *hospital.Doctor  // no nil si es médico (para MyEpisodes)
    speed    int               // 1-5
    skill    int               // 1-5, solo médicos
    salary   int
    look     Appearance
    busy     bool              // caminando o cargando a un paciente
    onCall   bool              // "de guardia": único disponible durante un despacho
}

func (s *StaffMember) ID() string   { return s.inner.ID() }
func (s *StaffMember) Name() string { return s.inner.Name() }
func (s *StaffMember) Attend(p *hospital.Patient, loc string) (hospital.EpisodeRecord, error) {
    return s.inner.Attend(p, loc) // el registro lo crea el Doctor/Orderly real
}
func (s *StaffMember) IsAvailable() bool { return s.onCall }
```

- Todo el personal del modo juego se contrata con `HireStaff(staffMember)`. **Invariante: en el hospital del juego no hay ningún Attender sin envolver.** Un `Orderly` suelto devuelve siempre `IsAvailable() == true` y se robaría los despachos. Un test lo verifica.
- Como `HireStaff` solo registra como médico a un `*hospital.Doctor` real, **los médicos envueltos no aparecen en `Doctors()`**. `Game` guarda su propia lista de `StaffMember` y usa `doctor.MyEpisodes()` para la consulta 5.2.
- Los registros del historial muestran al Doctor/Orderly real como quien atendió, porque `Attend` se delega.
- El `IsAvailable` del decorador **ya ignora** el cupo de 4 pacientes del médico: el `IsAvailable` del Doctor interno nunca se consulta. El juego no llama `DiagnosePatient`: la revisión del médico (5.5) es solo lógica del juego, porque `DiagnosePatient` llena ese cupo y el modelo no lo libera nunca. Por eso en sus registros `TreatingDoctor()` queda vacío ("sin tratante").
- ⚠️ `IsAvailable`, `Attend`, `ID` y `Name` se llaman desde adentro de los métodos del hospital, con su candado tomado y mientras `Game` ya tiene el suyo. **Nunca deben tomar el mutex de `Game`** (sería un deadlock). El orden de candados siempre es `g.mu` → `h.mu`.

### 3.2 Despacho elegido por el jugador ("de guardia")

```go
// Dentro de Game, con g.mu tomado:
s.onCall = true
defer func() { s.onCall = false }() // se apaga aunque RegisterEpisode falle
err := g.h.RegisterEpisode(p, zone) // el round-robin solo encuentra a s
```

El hospital sigue decidiendo "entre los disponibles"; el juego decide quién está disponible. El modelo no se modifica y su lógica (dormir al paciente, buscar la primera cama, crear y guardar el registro) se usa tal cual.

Cuando el despacho crea un episodio, `game/` guarda la **hora del juego** en un `map[string]string` (ID del episodio → `"14:15"`). Es solo para mostrarla en la bitácora y en el Shift Report. El modelo sigue guardando la hora real en `At()`.

### 3.3 Qué método del modelo usa cada acción del juego

| Acción del juego | API de `hospital/` |
|---|---|
| Crear el hospital y las habitaciones | `NewHospital`, `AddRoom(n, 1)` |
| Contratar personal | `NewDoctor` / `NewOrderly` + `HireStaff(&StaffMember{...})` |
| Llega un costeño | `NewPatient` + `AdmitPatient` |
| El camillero (o un médico) recoge al paciente dormido | `RegisterEpisode(p, zona)` con el mecanismo "de guardia" |
| Saber si consiguió cama | `p.Room()` después de `RegisterEpisode` (nil = se queda en el pasillo) |
| Traslado desde el pasillo: el personal llega donde un paciente `InHallway` | `AssignRoom(p)`; su error se muestra como aviso y queda para la consulta 5.3 |
| El médico revisa al paciente en la habitación | **Ninguna**: es solo lógica del juego (5.5) |
| Termina el sueño o el jugador lo despierta | `WakePatient(p)` (libera la cama) |
| Se va enojado desde el pasillo o esperando revisión | `WakePatient(p)`. Si estaba `Collapsed`, ninguna (sección 3.4) |
| Alerta de pasillo / consulta 5.1 | `PatientsInHallway()` |
| Consulta 5.2 | `doctor.MyEpisodes()` de cada médico |
| Consulta 5.3 | `Rooms()` + el último error de `AssignRoom` |
| Consulta 5.4 | `SevereReport()` tal cual, rotulado **"episodios en esta partida"**: el modelo cuenta por día real del calendario y un día de juego dura 5 minutos reales |

`Patient.SufferSleepAttack`, `Patient.WakeUp`, `Room.Occupy`, `Room.Release` y `Attender.Attend` **no** se llaman directamente desde el juego: solo el hospital los usa, con su candado.

### 3.4 Dos capas de estado del paciente

| Momento | Estado en `game/` | Estado en el modelo |
|---|---|---|
| Despierto, deambulando | `Walking` / `Idle` | `Awake` |
| Le dio el ataque y nadie lo ha recogido | `Collapsed` (en el piso, con cronómetro de espera) | `Awake` (el hospital aún no se ha enterado) |
| Recogido y con cama, sin revisar | `AwaitingReview` ("esperando revisión") | `AsleepInBed` |
| Revisado por un médico | `InBed` (con cronómetro de sueño) | `AsleepInBed` |
| Recogido pero sin cama | `InHallway` | `AsleepInHallway` |
| Se fue de alta | `Discharged` (sale del mapa) | `Awake` (sigue en la lista; el modelo no tiene alta) |
| Se cansó de esperar | `LeftAngry` (sale del mapa) | `Awake`: si estaba `Collapsed` nunca se durmió en el modelo; si no, el juego llamó `WakePatient` |

- El episodio entra al historial cuando el personal **lo recoge** (`RegisterEpisode`): "un episodio existe en el sistema cuando el personal lo reporta". La revisión del médico es solo del juego y no toca el modelo (sección 5.5).
- Para la reputación cuentan los `Collapsed`, los `InHallway` y los que esperan revisión (5.9). Para la derrota por colapso cuentan **solo** los `Collapsed` y los `InHallway` (5.10).
- **Nadie espera para siempre.** Si a los **45 s** del desplome nadie lo ha revisado, el paciente se despierta solo y se va enojado (5.4). Si estaba `Collapsed`, el modelo nunca se enteró: no se llama `WakePatient` y no queda episodio. Si estaba `InHallway` o esperando revisión, se llama `WakePatient` (libera la cama si tenía). *(Reemplaza la regla anterior "los `Collapsed` y los `InHallway` no se despiertan solos", 2026-10-10.)*
- Los dados de alta y los que se fueron enojados siguen en el modelo como `Awake`, así que `SevereReport()` puede incluirlos; el Shift Report los marca "(alta)" o "(se fue enojado)".

---

## 4. Modo demostración (Sección 6)

Reproduce en la ventana el mismo escenario de `main.go`, sobre **un hospital propio**, con el personal sin envolver (`HireDoctor` / `HireStaff` reales) para que el despacho sea el round-robin original del modelo.

La demo es un **tipo aparte**, `game.Demo`, con métodos `Next()`, `Skip()`, `Snapshot()` y `Report()`. No vive dentro de `Game` porque tiene su propio hospital y no usa la goroutine motor: solo la usa la goroutine de la interfaz, así que no necesita candado. Las horas de los episodios salen de `At()` (hora real), igual que en la consola de `main.go`.

### 4.1 Guion

| Paso | Qué se ve | Llamada al modelo |
|---|---|---|
| 1 | Se contratan Dra. Karen Ospina (D-01), Dr. Efraín Barraza (D-02) y el camillero Wilmer Camargo (C-01); aparecen en la sala del personal | `HireDoctor`, `HireDoctor`, `HireStaff` |
| 2 | Se habilitan las habitaciones 101, 102 y 103 | `AddRoom` ×3 |
| 3 | Entran por la calle P-001 Yeimy Padilla (Severe), P-002 Kevin Mercado (Moderate), P-003 Ludys Arrieta (Mild), P-004 Wilfrido Berrío (Severe), P-005 Breiner Julio (Severe) | `AdmitPatient` ×5 |
| 4 | Cada uno queda a cargo de su médico tratante | `DiagnosePatient` (como en `main.go`) |
| 5 | Yeimy se duerme en la cafetería; el sistema despacha y la suben a la 101 | `RegisterEpisode` |
| 6 | Kevin, en la fila de radiología → 102 | `RegisterEpisode` |
| 7 | Ludys, en la pista de champeta → 103 | `RegisterEpisode` |
| 8 | Wilfrido, en el pasillo 2 → **no hay cama**; aviso rojo con el error real | `RegisterEpisode` + `AssignRoom` (error) |
| 9 | Panel de la consulta 5.1 con Wilfrido en el pasillo | `PatientsInHallway` |
| 10 | Yeimy se despierta y libera la 101 | `WakePatient` |
| 11 | El camillero lleva a Wilfrido a la 101 | `AssignRoom` |
| 12 | Shift Report con las 4 consultas finales | `PatientsInHallway`, `MyEpisodes`, `Rooms`, `SevereReport` |

- Las ubicaciones de `main.go` se ubican en el mapa así: "cafetería" → Cafetería, "fila de radiología" → Radiología, "pasillo 2" → Hallway 2. **"Pista de champeta del patio" no existe en el mapa**: se ubica en el **Lobby** (decisión de Samuel, 2026-10-08). Es consistente con el evento *Picó de champeta*, que pone los bafles ahí. En la interfaz el Lobby se rotula **RECEPCIÓN**, que además es la ubicación con la que llega todo paciente en el modelo ("recepción").
- El resultado de cada paso sale del modelo real, no de texto fijo.
- Paso 11: `AssignRoom` no involucra a ningún Attender. El camillero se anima llevando a Wilfrido, pero el subtítulo muestra solo la llamada real (`h.AssignRoom(P-004) → habitación 101`).

### 4.2 Controles y subtítulos
- **Espacio / clic**: siguiente paso. **A**: avance automático (un paso cada 3 s). **S**: saltar al final.
- Barra inferior con el subtítulo del paso y la llamada real que se hizo, por ejemplo: `h.AssignRoom(P-004) → error: no hay habitación disponible`. Se oculta y se muestra con **C**. Pensado para la sustentación.
- Al terminar: Shift Report → botón **"¡AHORA TE TOCA!"** → empieza el modo juego. También hay una opción **DEMO SECCIÓN 6** en el menú del portapapeles.

---

## 5. Modo juego: ciclo

### 5.1 Tiempo
- Turno de **08:00 a 20:00** en tiempo de juego, que dura **300 s reales** (1 s real = 2,4 min de juego).
- Tick de simulación: 100 ms. La pausa congela reloj, cronómetros y llegadas.

### 5.2 Inicio de partida
- Plata inicial **$2.000**; reputación **3 estrellas** (escala 0–5).
- Personal: **1 doctor** (rapidez 3, pericia 3) y **1 camillero** (rapidez 3), aleatorios y envueltos en `StaffMember`.
- Habitaciones **101, 102, 103**, capacidad 1.

### 5.3 Llegada de pacientes
- Intervalo día 1: aleatorio entre **18 y 25 s**; cada día ×0,9 (mínimo **8 s**). Máximo **10** pacientes simultáneos en el mapa.
- Nivel: **Mild 40 %**, **Moderate 35 %**, **Severe 25 %**. IDs consecutivos (P-001, P-002…) porque `AdmitPatient` rechaza IDs repetidos.
- Aparecen en la **calle**, caminan unos **4 s** por la acera y `AdmitPatient` se llama al cruzar la puerta del lobby. Hay peatones y carros decorativos.
- Si el mapa está lleno cuando toca una llegada, se sortea otro intervalo.
- **La puerta del hospital se abre sola**: `Snapshot.DoorOpen` vale `true` mientras algún paciente está a 1 s de cruzarla, entrando o saliendo. Lo decide el juego, así la interfaz solo la dibuja.
- Cada paciente llega con una apariencia al azar (sección 7).

### 5.4 Comportamiento del paciente
- Despierto, deambula cada **10–20 s** entre Lobby, Cafetería, Hallway 1, Hallway 2 y Radiología.
- Tiempo despierto antes del ataque: **Mild 60–90 s**, **Moderate 35–55 s**, **Severe 15–30 s**. Arranca **al cruzar la puerta** (después de `AdmitPatient`), nunca en la acera: un paciente en la calle no se desploma.
- ~2 s antes del ataque hace la animación de aviso (cabeceo; estado `Drowsy` del juego, para que la interfaz sepa cuándo animarlo) y luego se desploma: estado `Collapsed`, arranca su **cronómetro de espera**. Es uno solo por episodio: sigue corriendo mientras está `Collapsed`, `InHallway` o esperando revisión, y se detiene con la revisión del médico. De él salen el pago (5.8), la reputación (5.9) y el enojo (abajo).
- Duración del sueño en cama: **Mild 15 s**, **Moderate 22 s**, **Severe 30 s**, con −8 % por punto de pericia **del médico que lo revisó**. El cronómetro de sueño arranca **con la revisión** (5.5), no al llegar a la cama: el que espera revisión o está en el pasillo sigue dormido. Al cumplirse, el juego llama `WakePatient`.
- **Se va enojado**: si a los **45 s** del desplome nadie lo ha revisado, se despierta solo y sale del mapa caminando por la puerta del lobby: **−0,5 ★** y **$0**, además de las penalizaciones de espera que ya corrían (5.9). Qué se le dice al modelo depende de dónde estaba (sección 3.4). Si alguien del personal iba en camino hacia él, ese despacho se cancela (5.5). En el Shift Report sale marcado "(se fue enojado)".
- **Alta**: al despertar después de **1 / 2 / 3** sueños completos (revisado y despertado solo; Mild / Moderate / Severe), sale caminando por la puerta. Si no le toca el alta, vuelve a deambular con un tiempo despierto nuevo.
- **Salir del mapa** (por alta o enojado) dura **4 s** hasta cruzar la puerta; después desaparece del mapa, pero sigue en el modelo como `Awake`. **Quien va saliendo no vuelve a tener ataque.**

### 5.5 Despacho (decisión del jugador)
Cada episodio necesita dos despachos: primero alguien **recoge** al paciente y después un médico lo **revisa** en la habitación. En los dos el personal camina **(8 − rapidez) s** (de 3 a 7 s) y queda `busy` hasta terminar.

**Recoger**
1. Clic en un paciente `Collapsed` y luego en un **camillero** libre. Si no hay ningún camillero libre, también se puede mandar a un **médico** libre; así la consulta 5.2 (`MyEpisodes`) sigue teniendo datos.
2. El personal camina hasta el paciente, lo carga y **ahí** se hace el despacho "de guardia" con `RegisterEpisode` (sección 3.2):
   - **Con cama**: lo lleva a la habitación (animación de 2 s) y queda libre. El paciente queda **esperando revisión** (en el modelo, `AsleepInBed`).
   - **Sin cama**: el paciente queda `InHallway`, sale el aviso rojo "sin cama disponible" y el personal queda libre.

**Revisar**
3. Clic en un paciente que espera revisión y luego en un **médico** libre. El médico camina hasta la habitación, lo revisa (animación *atender*) y ahí arranca el cronómetro de sueño, con la pericia de ese médico (5.4).
4. La revisión es **solo lógica del juego: no llama nada del modelo**. Tampoco llama `DiagnosePatient`, porque llena el cupo de 4 pacientes del médico y el modelo no lo libera nunca.
5. **Pago**: uno solo por episodio, en la revisión (5.8).

**Cancelación**
6. Si el paciente se va enojado (5.4) mientras alguien del personal va en camino hacia él, el despacho se cancela **sin llamar al modelo** y ese miembro del personal queda libre. *(Antes estaba en F5 con los eventos; pasa a F1.3.)*

La demo no cambia: sigue el round-robin real de la Sección 6 (sección 4).

### 5.6 Traslado desde el pasillo
- Cuando una cama se libera, el letrero de la habitación parpadea en verde.
- Clic en un paciente `InHallway` y luego en un **camillero** libre → `Dispatch`: el camillero camina hasta él, lo carga y, **al llegar**, el juego llama `AssignRoom(p)`.
  - **Hay cama**: lo lleva a la habitación (2 s) y el paciente queda **esperando revisión** (5.5).
  - **Ya no hay cama**: sale el aviso con el error del modelo (queda como último error de `AssignRoom` para la consulta 5.3), el paciente sigue `InHallway` y el camillero queda libre.
- Un **médico** también puede hacerlo si no hay camillero libre (la misma regla que para recoger).
- **Ya no hay asignación automática**: el jugador decide a quién manda. *(Reemplaza el clic que llamaba `AssignRoom` directo y el triaje automático a los 10 s, 2026-10-10.)*
- Solo modo juego: la demo no cambia (sigue sola, con el round-robin real de la Sección 6).

### 5.7 Despertar antes de tiempo
- Clic derecho sobre un paciente en cama → **DESPERTAR**: `WakePatient(p)` libera la cama al instante. Cuesta **−0,25 estrellas** y ese paciente vuelve a dormirse en la mitad del tiempo normal.
- Solo sirve con un paciente **revisado** y en cama (`InBed`): el sueño corre desde la revisión. Con otro, `ErrNotInBed`.
- **No cuenta como sueño completo** para el alta, y la mitad del tiempo despierto es solo para esa vez.

### 5.8 Economía
- **Pago por episodio**: uno solo, **en la revisión** del médico (5.5). Depende de los segundos que pasaron desde el desplome hasta la revisión: **$450** hasta 15 s; después **−$15 por cada segundo completo**, con mínimo **$100**. Ejemplos: a los 10 s paga $450; a los 25 s, $300; a los 60 s, $100. *(Reemplaza los $300/$150 por atención, 2026-10-10.)*
- El que se va enojado no paga: **$0**.
- Alta: **+$200**.
- Nómina a las 20:00. Médico: `250 + 50 × (rapidez + pericia)`; camillero: `120 + 60 × rapidez`.
- Contratar cuesta un día de salario por adelantado.
- (Opcional, fase final) **Ampliar**: comprar la habitación 104 por $2.500 con `AddRoom`.

### 5.9 Reputación (0–5)
- Más de **20 s** de espera (`Collapsed`, `InHallway` o esperando revisión): **−0,5**, y luego **−0,25** cada 10 s más. Se mide con el cronómetro de espera (5.4), así que no vuelve a empezar cuando el paciente cambia de estado.
- Se va enojado: **−0,5**, además de lo anterior.
- Recogido en menos de **10 s** desde el desplome: **+0,1**. Alta: **+0,1**. Tope 5.
- Para la derrota por colapso (5.10) cuentan **solo** los `Collapsed` y los `InHallway`; los que esperan revisión no.
- En el código la reputación se guarda como **entero en centésimas de estrella** (300 = 3 ★). Así sumar 0,1 y restar 0,25 no acumula errores de redondeo.

### 5.10 Derrota y puntaje
- Pierde con: reputación en 0 ("¡PERDIÓ LA LICENCIA!"), quiebra al pagar la nómina ("¡QUIEBRA!") o **5 o más** pacientes `Collapsed` + `InHallway` a la vez ("¡HOSPITAL COLAPSADO!").
- Se revisa al final de cada `Tick`. Al perder, la partida queda **congelada**: `Tick` no hace nada, y `Dispatch` y `WakeEarly` devuelven `ErrGameOver`.
- Puntaje: +100 por atención (cada **revisión**), +250 por alta, +500 por día completado (cuando el reloj llega a las 20:00; en F4 la nómina va después); al final, + plata ÷ 10 + estrellas × 200 (en centésimas: `reputación × 2`, aritmética entera).

### 5.11 Partida guardada (pendiente: F6)
- Se guarda **automáticamente al terminar cada día** (después de la nómina), **nunca a mitad del día**: el estado interno del hospital (camas, episodios, quién duerme dónde) no se puede serializar sin tocar `hospital/`.
- Se guarda: día, plata, reputación, puntaje, el siguiente ID de paciente, la cantidad de habitaciones y el personal (nombre, rol, rapidez, pericia, salario y apariencia).
- `game/` expone un struct **de solo valores** con `encoding/json`, que se escribe y se lee con `io.Writer` / `io.Reader`, para probarlo con `bytes.Buffer`. La interfaz decide la ruta del archivo (`os.UserConfigDir()`).
- **Cargar** arma un hospital nuevo con la API pública (`AddRoom`, `NewDoctor` / `NewOrderly` + `HireStaff` envuelto en `StaffMember`) y arranca el día siguiente.
- Tests: guardar y cargar da el mismo estado; un archivo dañado devuelve un error, sin `panic`.
- Salir al menú a mitad del día (9.4) pierde el día en curso: lo guardado es el final del día anterior.

---

## 6. Contratación (Bolsa de empleo)

- Botón **CONTRATAR** → portapapeles con **3 candidatos** que cambian cada día (≈ 66 % médicos, 34 % camilleros).
- Estadísticas de 1 a 5, con más probabilidad de 2–3. Cada ficha: retrato, nombre, rol, barras de **Rapidez** y **Pericia** (solo médicos) y salario diario.
- Botón deshabilitado si no alcanza la plata. Máximo **6** miembros del personal.
- Contratar crea el `*hospital.Doctor` o `*hospital.Orderly`, lo envuelve en `StaffMember` y llama `HireStaff`.

---

## 7. Generación aleatoria

- **Nombres** costeños. Nombres: Wilfrido, Yeimy, Dairo, Yuleidis, Éder, Keyner, Nayibe, Yeferson, Ledys, Aníbal, Rosiris, Hernando, Yorledis, Dagoberto, Marelvis, Ronaldo. Apellidos: Berrío, Padilla, Barrios, Arrieta, Mendoza, Julio, Polo, Pertuz, Cassiani, Ospina, Castro, Herrera, Altamar, Cantillo. Médicos con "Dr." o "Dra.".
- **Edad**: pacientes 18–80; personal 25–60.
- **Apariencia** (`Appearance`, con `RandomAppearance(rng)`): `skinTone` (4), `shirtColor`, `shirtPattern` (liso, floreada, rayas), `hat` (vueltiao, gorra, ninguno; solo pacientes), `hair` (corto, afro, trenzas, calvo) y `hairColor`. El personal usa uniforme por rol.
  - **Cada campo se sortea por separado, con la misma probabilidad y sin restricciones entre campos** (por ejemplo, afro con o sin sombrero).
  - `shirtColor` es un índice en una paleta de **6** colores y `hairColor`, un índice en una paleta propia de **4** (negro, castaño oscuro, castaño claro y canoso). El juego solo guarda el índice; **la interfaz decide qué color es cada uno**. Con pelo calvo, `hairColor` se sortea igual y simplemente no se usa.
- El azar se inyecta en `Game` como `*rand.Rand` para que los tests usen semilla fija.

---

## 8. Eventos aleatorios

- Día 1 sin eventos; días 2–3, **1** por día; desde el día 4, **2** por día, a una hora aleatoria entre 10:00 y 18:00.
- Se anuncian con un **banner** de 3 s (nombre, descripción breve y decoración temática) y un icono en el HUD mientras están activos. Varios cambian el mapa (bafles, el bus en la calle…).
- ⚠️ Pendiente: Samuel quiere definir en detalle cada banner y decoración antes de pedirlos en Claude Design.

| # | Evento | Efecto | Duración |
|---|---|---|---|
| 1 | 🎉 Carnaval | Intervalo de llegadas ÷ 2 | 60 s |
| 2 | 🍲 Día de sancocho | Los que están en la Cafetería se duermen el doble de rápido | 60 s |
| 3 | ⚾ Final de béisbol | Nadie se duerme; al terminar, cada despierto se duerme en 3–8 s | 30 s |
| 4 | 📋 Visita del Ministerio | Si al irse el inspector no hay nadie `Collapsed`/`InHallway`: +$800 y +0,5 ★. Si hay 2 o más: −0,5 ★ | 30 s |
| 5 | 👑 Paciente VIP: el alcalde | Llega un Severe con banda tricolor. Atendido en menos de 10 s: +$1.000 y +1 ★. Más de 20 s sin cama: −1 ★ | Hasta que se va |
| 6 | 🤢 Doctor enguayabado | Un médico al azar pierde 2 de rapidez (mínimo 1) | Resto del día |
| 7 | 🚌 Bus de excursión de Cartagena | Llegan 4 pacientes de una (respetando el máximo) | Instantáneo |
| 8 | 🔊 Picó de champeta | Todo el personal +2 de rapidez (máximo 5); nadie recibe el alta mientras suene | 45 s |

---

## 9. Pantallas

| Pantalla | Maqueta (Claude Design) | Contenido |
|---|---|---|
| Inicio | `Hospital de los Costeños/Title Screen.dc.html` | Paisaje de playa animado, logo y menú en portapapeles: CONTINUAR, NUEVA PARTIDA, DEMO SECCIÓN 6, AJUSTES, CRÉDITOS, SALIR (sección 9.3) |
| Hospital | `hospital.html` | Mapa top-down (habitaciones, lobby, pasillos, cafetería, radiología, sala del personal) y calle con paradero |
| Catálogo de sprites | `sprites.html` | Exportador de spritesheets PNG |
| Bolsa de empleo | por diseñar | Portapapeles con 3 fichas |
| Pausa | assets en `gui/assets/pause/` | Portapapeles derecho y centrado: REANUDAR, AJUSTES, SALIR AL MENÚ, sobre el mapa oscurecido (sección 9.4) |
| Ajustes | por diseñar | Pantalla completa, etiquetas siempre visibles, subtítulos de la demo y volumen si hay sonido (sección 9.5) |
| Shift Report | por diseñar | Las 4 consultas + balance del día; en la demo, botón "¡AHORA TE TOCA!". La 5.4 va rotulada "episodios en esta partida" y los episodios muestran la hora del juego (en la demo, la hora real) |
| Game Over | por diseñar | Motivo, puntaje, días sobrevividos, NUEVA PARTIDA |

### 9.1 HUD
- **Arriba**: plata, estrellas, "DÍA N", reloj con barra de progreso, icono del evento activo. En la demo: "MODO DEMOSTRACIÓN — PASO N/12".
- **Abajo a la derecha**: CONTRATAR, REPORTE, PAUSA (ocultos en la demo).
- **Siempre visible**: burbuja roja "Zzz!" con cronómetro (`Collapsed` e `InHallway`; intermitente tras 15 s), burbuja azul "Zzz" en cama, rayitos de nivel solo en dormidos (1 amarillo Mild, 2 naranja Moderate, 3 rojos Severe), punto verde/rojo sobre el personal (libre/ocupado), letreros OCCUPIED/AVAILABLE y nombres de zonas.
- **Etiquetas ocultas**: tooltip al pasar el cursor (ID, nombre, nivel, estado). **Shift** las muestra todas.
- **Avisos** en la esquina superior: errores del modelo, altas y eventos.

### 9.2 Transiciones
- Inicio → Hospital: el portapapeles se voltea + pixel dissolve.
- Fin del día → Shift Report: las fichas caen y se clavan en un tablero de corcho.

### 9.3 Pantalla de inicio (pendiente: junto con el mapa de fondo, 10.7, antes de F3)
- Escena `Title`. Capas, animaciones y portapapeles en `gui/assets/title/`, explicados en `gui/assets/title/TITLE_ASSETS.md`:
  - el paisaje va a **320 × 180** y se dibuja **×4** (escalado entero, *nearest*);
  - el portapapeles va como **imagen ya inclinada**: una por opción seleccionada, 2 cuadros del cursor, la variante con CONTINUAR en gris (`sincontinuar`) y `hitboxes.json` para el mouse.
- El **logo**, la banda "con Narcolepsia", las Zzz del título, "PRESIONA ENTER" y la versión los dibuja el juego con **Press Start 2P**. Referencia: `gui/design/claude-design/Hospital de los Costeños/Title Screen.dc.html`, con los textos en español.
- **Menú**: CONTINUAR, NUEVA PARTIDA, DEMO SECCIÓN 6, AJUSTES, CRÉDITOS y SALIR. CONTINUAR sale en gris si no hay partida guardada (5.11); hasta F6 siempre sale en gris.
- Se navega con **↑/↓** (o **W/S**), **Enter** y el mouse.
- Todo se anima por ticks de `Update`, nunca con `time.Sleep`.

### 9.4 Menú de pausa (pendiente: junto con la pantalla de inicio)
- Capa encima de la escena (10.2). Assets y medidas en `gui/assets/pause/PAUSE_ASSETS.md`: portapapeles derecho y centrado con REANUDAR, AJUSTES y SALIR AL MENÚ, sobre el mapa oscurecido.
- **Esc** abre la pausa y, con la pausa abierta, la cierra.
- En el modo juego llama `Game.Pause()` y `Game.Resume()`; en la demo detiene el avance automático.
- **SALIR AL MENÚ** cancela el `context` de la partida y vuelve a la portada. En el modo juego se pierde el día en curso: lo guardado es el final del día anterior (5.11).
- **AJUSTES** abre la pantalla de ajustes (9.5); mientras no exista, muestra el aviso "Próximamente".

### 9.5 Ajustes (pendiente: F6)
- Se guardan en un **JSON aparte** de la partida y se abren desde la portada y desde la pausa.
- Opciones: pantalla completa, etiquetas siempre visibles, subtítulos de la demo y volumen (solo si se agrega sonido).

---

## 10. Interfaz con Ebitengine (`gui/ui/`)

### 10.1 Pantalla
- Resolución lógica **1280 × 720**, igual que la ventana y el mapa de fondo (sección 10.7); en pantalla completa Ebitengine la escala. *(Antes de la tarea A era 640 × 360 con la ventana ×2.)*
- Tiles de **16 × 16 px**; personajes en cuadros de **16 × 24 px** en su hoja, dibujados **×2** (escalado entero, *nearest*): en la pantalla miden 32 × 48.
- **Mapa**: `hospital.dc.html` dibuja un mundo de **464 × 261** (también 16:9). `gui/ui/layout.go` copia sus coordenadas tal cual (para poder compararlas con la maqueta) y las escala a 1280 × 720 multiplicando por 1280/464 y redondeando, igual que se renderizó el mapa de fondo.
- **Idioma**: todos los textos de la interfaz van en español. Las maquetas están en inglés, pero solo son referencia visual; los textos se escriben en el código, y los estados salen del `String()` del modelo ("disponible", "ocupada", "dormido en el pasillo").

### 10.2 Escenas (otra interfaz)
```go
type Scene interface {
    Update() (Scene, error) // devuelve la escena que sigue: ella misma si no cambia
    Draw(screen *ebiten.Image)
}
```
`ui.App` implementa `ebiten.Game` y delega en la escena actual: Title, Demo, Hospital, Hiring, Report, GameOver. Pausa y Settings son capas encima. Como `Update` devuelve la escena siguiente, `App` cambia de pantalla sin un `switch` y las escenas no necesitan guardar un puntero a la `App`.

### 10.3 Sprites y animaciones
- Personajes por capas: cuerpo (tono de piel), pelo, camisa y sombrero. La camisa va en escala de grises y se tiñe con `ColorScale`.
- Spritesheet PNG: **una fila por animación, una columna por cuadro**, cuadros de 16 × 24 px. La animación hacia la derecha es la de la izquierda volteada con `GeoM.Scale(-1, 1)`.
- Primer sprite: `gui/assets/sprites/patient_body_skin1.png` (cuerpo base), de 64 × 240 px = 4 columnas × 10 filas. Las filas van en el orden de la tabla: quieto, caminar abajo, caminar arriba, caminar izquierda, aviso de sueño, desplomarse, dormido en el piso, dormido en cama, despertarse y alta. Las animaciones de 2 cuadros dejan vacías las columnas 3 y 4.

| Animación | Cuadros | Duración por cuadro |
|---|---|---|
| Quieto respirando | 2 | 500 ms |
| Caminar abajo / arriba / izquierda | 4 | 150 ms |
| Aviso de sueño (cabeceo) | 4 | 250 ms |
| Desplomarse | 4 | 120 ms (no se repite) |
| Dormido en el piso | 2 | 600 ms |
| Dormido en cama | 2 | 600 ms |
| Despertarse | 4 | 150 ms (no se repite) |
| Alta (saluda) | 4 | 150 ms |

- **Cuadros de personaje acostado**: los cuadros 3 y 4 de *Desplomarse*, todos los de *Dormido en el piso* y *Dormido en cama*, y el cuadro 1 de *Despertarse* ya vienen dibujados **en horizontal dentro del mismo cuadro de 16 × 24 px**, con la cabeza a la izquierda. Así todos los cuadros miden lo mismo y se recortan de la hoja igual que los demás.
  - **En el piso** se dibujan tal cual, sin rotar.
  - **En la cama** la cama sigue vertical, como en la maqueta, y solo se rotan 90° en sentido horario (`GeoM.Rotate`) los cuadros de *Dormido en cama* y el cuadro 1 de *Despertarse* si arranca en la cama. Así la cabeza queda hacia la cabecera. Un test verifica que la rotación solo se aplica en la cama.
  - Mientras lo cargan hacia la cama, el paciente se dibuja con *Dormido en el piso*, sin rotar.
  - Las capas de camisa, pelo y sombrero se dibujan con **la misma GeoM** que el cuerpo, así rotan y se voltean juntas.
- **Despertarse**: el cuadro 1 va dentro de la cama, rotado como *Dormido en cama*. Desde el cuadro 2 (sentado) el paciente aparece de pie al lado de la cama, sin rotar, y ahí termina la animación. Un test verifica las dos posiciones.
  - **En la demo**: después de despertarse hace *Alta* (saluda) una sola vez y queda en *Quieto* al lado de la cama. En la demo nadie se va del hospital, porque el modelo no tiene alta.
  - **En el modo juego**: cuando recibe el alta, hace despertarse → alta → camina hasta la puerta del lobby y sale del mapa.
- La burbuja "Zzz" aparece solo cuando el paciente ya está acostado (en el piso o en la cama). El modelo lo marca dormido desde el ataque, pero en pantalla primero camina al sitio y se desploma.
- El personal y los personajes de eventos usan el mismo formato con sus animaciones (atender, empujar camilla…).
- Las animaciones avanzan por ticks de `Update` (60/s), nunca con `time.Sleep`.

### 10.4 Entrada
- Clic en paciente `Collapsed`, `InHallway` o esperando revisión + clic en personal libre → `Game.Dispatch(patientID, staffID)`. El estado del paciente dice si es recoger, trasladar desde el pasillo o revisar (5.5 y 5.6).
- Clic derecho en paciente en cama → `Game.WakeEarly(patientID)`.
- Hover → tooltip; **Shift** → todas las etiquetas; **Esc** → pausa; flechas + Enter en los portapapeles.

### 10.5 Texto
- Fuente **Press Start 2P** (OFL) en `assets/fonts/`, con `text/v2`. Llega con la pantalla de inicio (9.3); mientras tanto la interfaz usa Go Mono a 16/14/20 px.
- ⚠️ **Tildes**: en Press Start 2P la Ó, É, Í, Ú y Ñ vienen encogidas (para que la tilde quepa en el cuadro de 8 × 8) y parecen minúsculas. Donde se use esa fuente, la tilde se dibuja **a mano encima de la letra normal**, como la virgulilla de COSTEÑOS en la maqueta. Lo hace solo un helper de texto.

### 10.6 Comunicación con `game/`
- Cada `Draw` usa `game.Snapshot()`: una copia de solo lectura construida con el mutex de `Game` tomado. `Snapshot()` y `Report()` contienen **solo valores** (textos, números y estados), nunca punteros del modelo como `*hospital.Patient`: la interfaz no puede leer el hospital sin pasar por `Game`.
- Las acciones son métodos de `Game` (`Dispatch`, `WakeEarly`, `Hire`, `Pause`, `Resume`, `Report`). Una partida nueva es un `game.New(rng)` nuevo con su propio `context`, que cancela la anterior. La demo usa el tipo `game.Demo` (`Next`, `Skip`, `Snapshot`, `Report`). Los errores se muestran como avisos.
- Los estados se muestran con el `String()` de las constantes tipadas del modelo.

### 10.7 Mapa con imagen de fondo y rutas (hecho, 2026-10-10)
Aplica a la demo y al juego (mismo código de `gui/`). Reemplaza la tarea "rutas por puntos de paso" y el detalle del camillero que atravesaba la 102.
1. **Fondo**: `gui/assets/map/hospital_map_1280x720.png` es la maqueta (`hospital.dc.html`) renderizada a 1280 × 720 (1280/464 px de pantalla por px de mundo, bordes redondeados, sin antialias). Solo trae lo que nunca se mueve: sin personajes, sin taxi, sin humo y sin rayitas de movimiento. Se embebe con `//go:embed` y se dibuja de fondo; la interfaz deja de dibujar pisos, paredes y muebles con rectángulos.
2. **Resolución lógica 1280 × 720** (antes 640 × 360): `layout.go` escala con 1280/464; los personajes se dibujan ×2 (escalado entero, *nearest*), incluidas la rotación en la cama y el volteo; la fuente se ajusta al tamaño nuevo; la ventana sigue en 1280 × 720.
3. **Camas**: el fondo trae las **3 camas libres** (la cobija doblada). `gui/assets/map/bed_blanket.png` (56 × 48, la cobija subida) va **encima del paciente acostado**, en (126, 88), (399, 88) y (672, 88); esas posiciones salen de `layout.go` y las revisa un test. Orden de dibujo: fondo → paciente en la cama → cobija. Las dos imágenes salen de la misma maqueta y se verificaron píxel a píxel contra el mapa original.
4. **Rutas**: los puntos de paso son los dos lados de cada puerta de la maqueta (`hDoor`, `vDoor` y la puerta principal), y la ruta más corta sale con Dijkstra. Las rutas son para los **pies** del personaje (abajo al centro de su cuadro). Un test verifica que ningún tramo entre dos puntos conectados cruza una pared (las paredes salen de los mismos rectángulos de la maqueta), y otro recorre toda la demo tick a tick: nadie pisa una pared y los que están quietos no quedan uno encima del otro.
5. **Lo que se mueve** va como sprite aparte, ya escalado (se dibuja a 1×, en píxeles de 1280 × 720):
   - `gui/assets/map/taxi.png`: pasa por el carril de abajo, de izquierda a derecha, con la esquina superior izquierda en y = 651; sale por la derecha y vuelve a aparecer por la izquierda cada cierto tiempo (constante en la interfaz).
   - `gui/assets/map/steam_pot.png`: humo de la olla de la cafetería, hoja de 3 cuadros de 44 × 36 en fila; esquina superior izquierda en (40, 246), cambia de cuadro cada ~300 ms, en bucle.
   - Todo animado por ticks de `Update`, nunca con `time.Sleep`.

---

## 11. Concurrencia

> Decisión del 2026-10-08: **un solo reloj (`Tick`) y una sola goroutine motor**, en lugar de una goroutine por paciente.

- El bono de concurrencia ya está cubierto por `simulation/`. En el juego hay **una única fuente de tiempo**: `Game.Tick(dt)`, que avanza en un solo paso el reloj, los cronómetros de cada paciente (máquina de estados en `patients.go`), las llegadas, la reputación y la derrota.
- **Dos goroutines tocan `Game`:**
  - la **goroutine motor** (`Start(ctx)`, en `engine.go`): un `time.Ticker` que llama `Tick` cada 100 ms hasta que se cancela el `context` de la partida;
  - la **goroutine de Ebitengine**: llama `Snapshot()` en cada `Draw` y las acciones del jugador (`Dispatch`, `WakeEarly`…).
  
  Por eso el mutex de `Game` es necesario, y `-race` lo verifica.
- Los tests no lanzan el motor: llaman `Tick` a mano con un `*rand.Rand` de semilla fija, así que son deterministas. Un test aparte arranca el motor y lee `Snapshot()` en paralelo para que `-race` revise la concurrencia real.
- La pausa es una bandera bajo `g.mu`: `Tick` no hace nada mientras está activa. Así quedan congelados el reloj, los cronómetros y las llegadas, sin coordinar varias goroutines.
- `*rand.Rand` no es seguro entre goroutines: solo se usa con `g.mu` tomado, y como todo pasa dentro de `Tick` o de las acciones, se cumple solo.
- `Hospital` tiene su propio candado, pero los getters de `Patient`, `Room` y `Doctor` **no son seguros con goroutines** (lo dice el README). Por eso **todo** acceso al hospital (escrituras y lecturas, incluido `Snapshot()`) pasa por métodos de `Game` que toman `g.mu`.
- Orden de candados siempre: primero `g.mu` (Game), después `h.mu` (Hospital, adentro de sus métodos). Los métodos de `StaffMember` nunca toman `g.mu` (sección 3.1).
- El `sync.Mutex` no es reentrante: los métodos públicos de `Game` no se llaman entre sí, sino que delegan en helpers `...Locked`. Es la misma convención que usa `hospital/`.
- Nunca se usa `time.Sleep` mientras se tiene `g.mu`: el motor espera al `Ticker` sin candado y solo lo toma dentro de `Tick`.
- Verificación: `go test -race ./...` limpio en la raíz (incluye `game/`).

---

## 12. Fases de implementación

Cada fase termina con `gofmt`, `go vet`, tests en verde, un commit y una explicación en español para Samuel.

| Fase | Entregable | Criterio de aceptación |
|---|---|---|
| F0 | Modelo cerrado (ya hecho) | Tag `modelo-cerrado` sobre el commit actual. Desde aquí `hospital/` y `simulation/` no cambian |
| F1 | `game/` núcleo | `StaffMember`, despacho "de guardia", traslado desde el pasillo, despertar, guion de la demo, reloj, llegadas, plata (ingresos), reputación y derrota por licencia o colapso, con tick manual y tests con semilla fija. La quiebra llega con la nómina en F4 |
| F2 | `gui/` + **modo demostración** | La ventana reproduce los 12 pasos de la Sección 6 con subtítulos y termina en el Shift Report. **Con esto ya está el bono de interfaz** |
| F3 | Modo juego jugable | Despachar, trasladar y despertar con clics; HUD; goroutine motor + interfaz con `-race` limpio |
| F4 | Contratación + nómina + Shift Report diario | Bolsa de empleo funcional |
| F5 | Eventos aleatorios | Los 8 eventos con banner y su test |
| F6 | Pulido | README actualizado (cómo correr la demo y el juego), Game Over, partida guardada (5.11) y ajustes (9.5), sonido opcional, `AI_USAGE.md` |

Prioridad si el tiempo no alcanza: F1 → F2 → F3 → F4 → F5 → F6. **F2 es la meta mínima.**

**Orden real (2026-10-08), para asegurar primero el bono de interfaz:**
1. F1.5: `game.Demo` + `Report`, que no dependen del resto de F1.
2. F2 mínima: la demo de los 12 pasos en Ebitengine, con rectángulos de colores en lugar de sprites, subtítulos y Shift Report.
3. F1.1–F1.4: reloj y motor, `StaffMember` + despacho, pacientes, llegadas y camas, y economía, reputación y derrota. F1.3 se parte en dos (2026-10-10):
   - **F1.3a**, el paciente vive solo: apariencia, llegadas por la calle, puerta, deambular, ataque, cronómetro de espera, sueño según la pericia, despertar y alta;
   - **F1.3b**, la espera tiene límite: el que se va enojado a los 45 s, la cancelación del despacho y el traslado desde el pasillo.
4. Mapa con imagen de fondo y rutas (sección 10.7), pantalla de inicio (9.3) y menú de pausa (9.4), antes de F3 porque el juego también los necesita.
5. F3 → F6.

### 12.1 Decisiones tomadas antes de F2 (2026-10-08)
- ✅ **Idioma de la interfaz**: español en todo (sección 10.1). Las maquetas en inglés son solo referencia visual.
- ✅ **"Pista de champeta del patio"**: va en el Lobby, rotulado RECEPCIÓN (sección 4.1).
- ✅ El "despierto" en masculino que aparece con pacientes mujeres viene del `String()` del modelo congelado, y se deja así.
- Velocidad del tiempo (×2, ×3): fuera de alcance por ahora; si se quiere, va como constante en `config.go`.

### 12.2 Cambio de diseño del modo juego (2026-10-10)
- **Episodio en dos despachos**: el camillero recoge (ahí va el `RegisterEpisode` "de guardia") y el médico revisa en la habitación (solo lógica del juego). Estado nuevo: **esperando revisión** (secciones 3.4 y 5.5).
- **Pago único en la revisión**, según el tiempo de espera: $450 → −$15/s → mínimo $100 (5.8). Reemplaza los $300/$150 por atención.
- **Se va enojado a los 45 s** sin revisión: reemplaza "los `Collapsed` y los `InHallway` no se despiertan solos" (3.4 y 5.4).
- La **cancelación** de un despacho en camino pasa de F5 a **F1.3**.
- "Esperando revisión" baja reputación igual que `InHallway`, pero no cuenta para la derrota por colapso (5.9).
- La demo no cambia.

### 12.3 Decisiones de F1.3 (2026-10-10)
- El paciente `InHallway` se despacha: el personal camina, lo carga y al llegar se llama `AssignRoom`. Sin asignación automática (5.6).
- La puerta la decide el juego (`DoorOpen`); salir del mapa dura 4 s; si el mapa está lleno, se sortea otro intervalo (5.3 y 5.4).
- El tiempo despierto arranca al cruzar la puerta; quien va saliendo no vuelve a tener ataque; el cabeceo es el estado `Drowsy` (5.4).
- Para el alta cuentan los sueños completos (5.4).
- `Appearance` suma `hairColor`; `shirtColor` y `hairColor` son índices de paleta que decide la interfaz (sección 7).
- `WakeEarly` (5.7) va en F1.4, con su costo de reputación.
- Tareas anotadas para después: mapa de fondo y rutas (10.7), pantalla de inicio (9.3) y menú de pausa (9.4), antes de F3; partida guardada (5.11) y ajustes (9.5), en F6.

### 12.4 Decisiones de F1.4 (2026-10-10)
- El alta paga +$200 (5.8); "+100 por atención" es por revisión; los +500 del día se suman al llegar a las 20:00 (5.10).
- `WakeEarly`: solo con pacientes revisados en cama, no cuenta como sueño completo y la mitad del tiempo despierto es solo esa vez (5.7).
- Al perder, la partida queda congelada; el puntaje final se calcula con aritmética entera (5.10).
- La penalización por espera sigue el cronómetro de espera y se detiene con la revisión; para el colapso cuentan solo desplomados y del pasillo (5.9).
- La reputación sale en la foto en centésimas (la interfaz la muestra como "2,75 ★"); no hay aviso por cada pago, sí al perder.

---

## 13. Notas para la sustentación

Por cada fase, Claude Code debe explicarle a Samuel:
- Por qué `StaffMember` es un **decorador**: cumple `hospital.Attender`, envuelve al Doctor/Orderly real y cambia la disponibilidad sin modificar el hospital.
- Cómo el mecanismo "de guardia" deja que el jugador elija mientras el round-robin del modelo sigue funcionando.
- Por qué los médicos envueltos no salen en `Doctors()` (la aserción de tipo de `HireStaff`) y cómo se resuelve la consulta 5.2.
- Por qué la revisión del médico no llama `DiagnosePatient` (el cupo de 4 se llena y el modelo no lo libera).
- Por qué el mutex de `Game` es necesario aunque `Hospital` ya tenga el suyo, y por qué `StaffMember` no puede tomarlo (deadlock).
- Cómo agregar una `Nurse` en vivo: un tipo nuevo que cumpla `Attender`, envuelto en `StaffMember` y contratado con `HireStaff`.
- Cómo `ui.App` cumple `ebiten.Game` y cómo la interfaz `Scene` cambia de pantalla sin `switch` gigantes.
- Por qué `game/` está en el módulo raíz y `gui/` en un módulo aparte.
