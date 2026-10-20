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
- **Horas pico** (2026-10-17, §12.9: a Samuel la jugabilidad le pareció "un poco aburrida"; inspirado en Diner Dash). Ese intervalo se multiplica según la hora del turno (`rushBands` en `game/config.go`, fácil de ajustar): **08:00–10:00** tranquilo ×1,3; **10:00–12:00 PICO** ×0,5; **12:00–15:00** normal ×1; **15:00–17:30 PICO** ×0,5; **17:30–20:00** ×0,9. El mínimo de 8 s (arriba) manda siempre, aunque el factor empuje más abajo. `Snapshot.RushHour` dice si es pico ahora mismo, y al **entrar** a uno sale el aviso "¡Hora pico! Llegan más pacientes" (como los demás Notices).
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
   - **Con cama**: lo lleva a la habitación en camilla, en **(8 − rapidez) s** como las caminatas (antes 2 s; decisión de Samuel del 2026-10-11, porque la camilla va por su ruta) y queda libre. El paciente queda **esperando revisión** (en el modelo, `AsleepInBed`).
   - **Sin cama**: el paciente queda `InHallway`, sale el aviso rojo "sin cama disponible" y el personal queda libre.

**Revisar**
3. Clic en un paciente que espera revisión y luego en un **médico** libre. El médico camina hasta la habitación, lo revisa (animación *atender*) y ahí arranca el cronómetro de sueño, con la pericia de ese médico (5.4). Si llega mientras al paciente todavía lo traen en camilla (con un camillero lento, el traslado dura más que su caminata), **espera junto a la cama** y lo revisa al terminar el traslado (2026-10-11).
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
- **Cómo quedó (F4, 2026-10-15)**: la nómina se paga en el `Tick` de las 20:00, justo después de los +500 del día completado, y no al pulsar SIGUIENTE DÍA. Así el reporte del día (`Game.DayReport()`) ya muestra la plata después de pagar cuando sale la pantalla de fin de día.
- **Racha de atención** (2026-10-17, §12.9: pedido de Samuel, inspirado en Diner Dash, porque la jugabilidad le pareció "un poco aburrida"). Cada revisión con menos de `firstWaitPenaltyAt` (30 s) de espera sube una racha en 1. El pago de la revisión se multiplica por `1 + 0,25 × min(racha anterior, 4)` (hasta ×2), redondeado al peso más cercano (`streakStep`/`streakMaxSteps` en `config.go`). La racha vuelve a 0 cuando cae una penalización de espera (5.9), cuando alguien se va enojado o al terminar el día. `Snapshot.Streak` (la racha actual) y `Snapshot.StreakMultiplier` (en centésimas: 100, 125… 200) la muestran; `DayReport.BestStreak` guarda la racha más alta del día, aunque algo la haya roto antes de cerrar el turno.

### 5.9 Reputación (0–5)
- Más de **30 s** de espera (`Collapsed`, `InHallway` o esperando revisión): **−0,5**, y luego **−0,25** cada 10 s más. *(Antes 20 s; subió a 30 el 2026-10-16, §12.8: en la prueba del 16-10 a Samuel "le bajaban las estrellas solas" sin darse cuenta.)* Se mide con el cronómetro de espera (5.4), así que no vuelve a empezar cuando el paciente cambia de estado.
- Se va enojado (45 s sin revisión, sin cambiar): **−0,5**, además de lo anterior.
- Recogido en menos de **10 s** desde el desplome: **+0,1**. Alta: **+0,1**. Tope 5.
- Para la derrota por colapso (5.10) cuentan **solo** los `Collapsed` y los `InHallway`; los que esperan revisión no.
- En el código la reputación se guarda como **entero en centésimas de estrella** (300 = 3 ★). Así sumar 0,1 y restar 0,25 no acumula errores de redondeo.
- **Cada cambio con su motivo** (2026-10-16, §12.8): `game.ReputationChange{Delta, Reason}` (Delta en centésimas, Reason en español con el ID del paciente: "P-003 lleva 30 s esperando revisión", "P-005 se fue enojado", "P-002: lo recogieron rápido"). `Snapshot.ReputationChanges` trae los últimos (como `Notices`, tope `maxReputationChanges` = 5); solo se registra el motivo si `reason != ""` (los llamados internos de los tests que no le importan a la interfaz pasan ""). La interfaz los muestra como avisos cerca de las estrellas, rojos si bajan y verdes si suben, con el signo tipográfico "−".

### 5.10 Derrota y puntaje
- Pierde con: reputación en 0 ("¡PERDIÓ LA LICENCIA!"), quiebra al pagar la nómina ("¡QUIEBRA!") o **5 o más** pacientes `Collapsed` + `InHallway` a la vez ("¡HOSPITAL COLAPSADO!").
- Se revisa al final de cada `Tick`. Al perder, la partida queda **congelada**: `Tick` no hace nada, y `Dispatch` y `WakeEarly` devuelven `ErrGameOver`.
- **Quiebra (F4, 2026-10-15)**: si la nómina de las 20:00 deja la plata en negativo, la derrota es `Bankrupt` ("¡QUIEBRA!"). `Hire` y `StartNextDay` también devuelven `ErrGameOver` con la partida perdida.
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
- **Lógica (hecha en F4, 2026-10-15)**: `Game.Hire(i int) error`, con los errores `ErrNotEnoughMoney`, `ErrStaffFull` (tope 6), `ErrNoCandidate` y `ErrGameOver`. Rapidez y pericia se sortean con pesos 15/30/30/15/10 sobre 1–5. Los médicos llevan una especialidad de la lista `Medicina del sueño`, `Neurología`, `Medicina interna` y `Medicina general` (el médico inicial sigue en Medicina del sueño). `Snapshot().Candidates` trae los 3 candidatos del día como valores (`CandidateView`: nombre, rol, especialidad, edad, rapidez, pericia, sueldo = costo, apariencia y `Affordable` para el botón en gris). El contratado es exactamente el de la hoja: no se vuelve a sortear.
- **Vista aprobada (2026-10-15)**: Figma, sección `7:4` (frames A `7:5`, B `8:64`, C `8:127`, HUD `8:190`; notas `12:219`–`12:222`). Assets y medidas en `gui/assets/hiring/HIRING_ASSETS.md`. Samuel: *"Continúa, con tus recomendaciones para bolsa"*: el portapapeles del HUD de 120 × 68 (la imagen mide 120 × 74 con el gancho), "SIN PERICIA: NO REVISA" para el camillero, TU PLATA y "TE FALTAN $ X" en la hoja, y el tablero como imagen con el texto dibujado por el código. El camillero (`staff_orderly_skin{1,3,4}.png`) es verde, como la maqueta.
- **Interfaz** (decidida el 2026-10-10; se hace en F4):
  - En el HUD hay un **portapapeles chiquito** con la miniatura de la cabeza de un candidato, que va cambiando entre los candidatos disponibles. En F3 ya aparece, pero al darle clic muestra "Próximamente".
  - Al hacerle clic, el portapapeles se anima hasta el centro de la pantalla y se agranda; **el juego se pausa** mientras está abierto.
  - Se ve **un candidato a la vez**, en su hoja. Se cambia de hoja con ←/→ o con dos flechas en pantalla (con clic).
  - La hoja tiene, a la izquierda, el **retrato** del candidato (su sprite grande, con su pelo y su piel, en la animación de quieto) y, a la derecha, su **nombre** en grande, su rol y especialidad, sus estadísticas en puntos (**rapidez** y **pericia**, de 1 a 5), el **sueldo por día** y el **costo de contratarlo**.
  - Abajo, un botón grande **CONTRATAR**, en gris si no alcanza la plata.
  - **Esc** o clic afuera lo cierra y el portapapeles vuelve a su sitio.
  - El candidato es el mismo que después camina en el mapa: su apariencia vive en `game/` (`StaffMember` la guarda y `StaffView` la expone, F3b).
- **Cómo quedó (F4, 2026-10-16)**:
  - `HiringScene` (`gui/ui/hiring_scene.go`) es una capa encima del juego, como la pausa, pero **no implementa `pausable`**: así Esc solo la cierra y no abre además el menú de pausa.
  - Abre con ease-out en 0,22 s desde el portapapeles chiquito del HUD y cierra con ease-in en 0,18 s, escalando el tablero completo como una sola pieza. El cambio de hoja dura 0,10 s, con un salto de 6 px.
  - CONTRATAR llama `Game.Hire(i)`. Si no alcanza la plata, el botón sale en gris con "TE FALTAN $ X" y el clic no hace nada. Si ya se contrató a los 3, sale un aviso simple en el tablero.
  - En el portapapeles chiquito se ve la cabeza ×3 del candidato de turno, que cambia cada 2 s.
  - La flecha presionada (`_1`) se ve 0,10 s después del clic o de la tecla: el juego solo lee el clic recién apretado.
  - La hoja va toda en mayúsculas, como la vista aprobada; la conversión la hace la interfaz al dibujar, y los datos de `game/` no cambian. Si el nombre no cabe a 24 px baja a 16 px, y si tampoco cabe se parte en dos renglones. `CandidateView.RoleLabel` ("Médico", "Médica", "Camillero" o "Camillera") sale del mismo sorteo que pone "Dr."/"Dra." en el nombre (2026-10-16).

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
| Bolsa de empleo | por diseñar (la interfaz está decidida en la sección 6) | Portapapeles que sale del HUD con un candidato por hoja (sección 6) |
| Pausa | assets en `gui/assets/pause/` | Portapapeles derecho y centrado: REANUDAR, AJUSTES, SALIR AL MENÚ, sobre el mapa oscurecido (sección 9.4) |
| Ajustes | por diseñar | Pantalla completa, etiquetas siempre visibles, subtítulos de la demo y volumen si hay sonido (sección 9.5) |
| Shift Report | por diseñar | Las 4 consultas + balance del día; en la demo, botón "¡AHORA TE TOCA!". La 5.4 va rotulada "episodios en esta partida" y los episodios muestran la hora del juego (en la demo, la hora real) |
| Game Over | por diseñar | Motivo, puntaje, días sobrevividos, NUEVA PARTIDA |

### 9.1 HUD
- **Arriba**: plata, estrellas, "DÍA N", reloj con barra de progreso, icono del evento activo. En la demo: "MODO DEMOSTRACIÓN — PASO N/12".
- **Abajo a la derecha**: CONTRATAR (desde F4, el portapapeles chiquito `hud_hire_clipboard.png` con la cabeza ×3 del candidato de turno), REPORTE y PAUSA (ocultos en la demo).
- **Siempre visible**: burbuja roja "Zzz!" con cronómetro (`Collapsed` e `InHallway`; intermitente tras 15 s), burbuja azul "Zzz" en cama, rayitos de nivel solo en dormidos (1 amarillo Mild, 2 naranja Moderate, 3 rojos Severe), punto verde/rojo sobre el personal (libre/ocupado), letreros OCCUPIED/AVAILABLE y nombres de zonas.
- **Etiquetas ocultas** en el modo juego (confirmado el 2026-10-11; en la demo se ven siempre): tooltip al pasar el cursor (ID, nombre, nivel, estado). **Shift** las muestra todas.
- **Barra de arriba del modo juego (F3b)**, según `gui/design/claude-design/Hospital de los Costeños/HUD.dc.html`: los iconos ESTRELLA, VACÍA, PLATA y RELOJ copiados de sus mapas de píxeles a Go con la misma paleta y dibujados ×3; la estrella parcial es la vacía y, encima, la llena recortada a 1 + round(9 × fracción) columnas; textos en Press Start 2P de 16 px (DÍA con el helper de tildes). La reputación en pantalla se mueve 0,1 a la vez hacia el valor real, con un destello de 0,25 s en cada paso: **verde** si sube y **rojo** si baja. La plata cuenta hacia el valor real en ~0,6 s, y el "+$450" verde (#2fb84f) o el "−$X" rojo (#e5484d), a 16 px, sube 24 px y se desvanece en 1 s.
- **Avisos** en la esquina superior: errores del modelo, altas y eventos.

### 9.2 Transiciones
- Inicio → Hospital: el portapapeles se voltea + pixel dissolve.
- Fin del día → Shift Report: las fichas caen y se clavan en un tablero de corcho.

### 9.3 Pantalla de inicio (hecha, 2026-10-10)
- Escena `Title`. Capas, animaciones y portapapeles en `gui/assets/title/`, explicados en `gui/assets/title/TITLE_ASSETS.md`:
  - el paisaje va a **320 × 180** y se dibuja **×4** (escalado entero, *nearest*);
  - el portapapeles va como **imagen ya inclinada**: una por opción seleccionada, 2 cuadros del cursor, la variante con CONTINUAR en gris (`sincontinuar`) y `hitboxes.json` para el mouse.
- El **logo**, la banda "con Narcolepsia", las Zzz del título, "PRESIONA ENTER" y la versión los dibuja el juego con **Press Start 2P**. Referencia: `gui/design/claude-design/Hospital de los Costeños/Title Screen.dc.html`, con los textos en español.
- **Menú**: CONTINUAR, NUEVA PARTIDA, DEMO SECCIÓN 6, AJUSTES, CRÉDITOS y SALIR. CONTINUAR sale en gris si no hay partida guardada (5.11); hasta F6 siempre sale en gris.
- Se navega con **↑/↓** (o **W/S**), **Enter** y el mouse.
- Todo se anima por ticks de `Update`, nunca con `time.Sleep`.
- **Cómo quedó**: NUEVA PARTIDA (hasta F3) y AJUSTES (hasta F6) muestran el aviso "Próximamente" durante 1,5 s en el lugar de "PRESIONA ENTER"; DEMO SECCIÓN 6 abre la demo, y su Shift Report vuelve a la portada con Espacio o clic; CRÉDITOS abre una pantalla con el mismo paisaje (Enter o Esc vuelven); SALIR cierra limpio (`ebiten.Termination`). "HOSPITAL DE LOS" va a 40 px y "COSTEÑOS" a 72; la virgulilla de la Ñ lleva las mismas pasadas que las letras, como en la maqueta: contorno negro, relieve rojo y sombra. **Desde el 2026-10-17 (§12.8) Press Start 2P se usa en TODA la interfaz**, no solo en la portada, los créditos y el aviso: los rótulos del mapa, los IDs, los tooltips, los avisos, los subtítulos y la barra de la demo, los botones REPORTE y PAUSA, el menú de pausa, la bolsa de empleo, y también el reporte del día y el Game Over, en su diseño actual (solo cambió la letra, no el corcho). Detalle de los tamaños en `gui/ui/theme.go`.

### 9.4 Menú de pausa (hecho, 2026-10-10)
- Capa encima de la escena (10.2). Assets y medidas en `gui/assets/pause/PAUSE_ASSETS.md`: portapapeles derecho y centrado con REANUDAR, AJUSTES y SALIR AL MENÚ, sobre el mapa oscurecido.
- **Esc** abre la pausa y, con la pausa abierta, la cierra.
- En el modo juego llama `Game.Pause()` y `Game.Resume()`; en la demo detiene el avance automático.
- **SALIR AL MENÚ** cancela el `context` de la partida y vuelve a la portada. En el modo juego se pierde el día en curso: lo guardado es el final del día anterior (5.11).
- **AJUSTES** abre la pantalla de ajustes (9.5); mientras no exista, muestra el aviso "Próximamente".
- **Cómo quedó**: la pausa es su propia escena (`PauseScene`), que envuelve a la escena pausada: la dibuja quieta, encima pone el negro con alfa 140 y el portapapeles en (442, 213). La abre la App con Esc sobre las escenas que cumplen la interfaz `pausable` (la demo; el juego en F3); la portada, los créditos y el reporte no se pausan. Mientras está abierta, la escena de abajo no recibe `Update`: en la demo se congela todo (avance automático, animaciones, taxi y humo) y al reanudar sigue donde iba. `pausable` trae `onPause`, `onResume` y `onQuit` para lo que corre por fuera de la escena: en F3, `Game.Pause()`, `Game.Resume()` y cancelar el `context`. AJUSTES muestra "Próximamente" 1,5 s debajo del portapapeles. El menú (selección, cursor, confirmación y mouse) es el mismo de la portada, generalizado; los rectángulos de `pause/hitboxes.json` se pasan a 4 vértices para usar la misma prueba de punto.
- ✅ **Animada (hecho 2026-10-17)**: pedido de Samuel ("hazle así animados los menús como el de contratar que me gustó mucho", la bolsa de empleo). Abre con la misma animación de portapapeles que la bolsa (§10.10: ease-out desde el botón PAUSA del HUD, o el centro de la pantalla en la demo, que no tiene HUD) y cierra con ease-in (al mismo punto) con Esc o REANUDAR; SALIR AL MENÚ también cierra animado. Mientras anima no acepta ni teclado ni mouse, y la acción pendiente (reanudar o salir al menú) se hace al terminar de cerrar, no al elegirla.

### 9.5 Ajustes (pendiente: F6)
- Se guardan en un **JSON aparte** de la partida y se abren desde la portada y desde la pausa.
- Opciones: pantalla completa, etiquetas siempre visibles, subtítulos de la demo y volumen (solo si se agrega sonido).
- ✅ **Silencio (hecho 2026-10-17, §10.11)**: mientras no exista la pantalla de Ajustes, el volumen se controla con la tecla **M**, que silencia y vuelve a activar el sonido con un aviso corto "SONIDO: NO"/"SONIDO: SÍ" (1,5 s, arriba al centro). Funciona en cualquier escena, igual que Esc. El backdoor de pruebas (`-agent`, §10.9) nunca suena, con M o sin M.

---

## 10. Interfaz con Ebitengine (`gui/ui/`)

### 10.1 Pantalla
- Resolución lógica **1280 × 720**, igual que la ventana y el mapa de fondo (sección 10.7); en pantalla completa Ebitengine la escala. *(Antes de la tarea A era 640 × 360 con la ventana ×2.)*
- Tiles de **16 × 16 px**; personajes en cuadros de **16 × 24 px** en su hoja, dibujados **×3** (escalado entero, *nearest*): en la pantalla miden 48 × 72 (sección 10.8; antes ×2, 32 × 48).
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
  - **Cómo quedó (F3a, 2026-10-11)**: el paciente se dibuja cuerpo → camisa → pelo → sombrero y el médico con su hoja (`staff_doctor_skin*`, 64 × 120: quieto, caminar abajo, arriba e izquierda, y atender) y el pelo del paciente encima; todas las capas con el mismo cuadro y la misma GeoM (`gui/ui/layers.go`). Camisa y pelo vienen en gris y se tiñen: camisas con los 6 colores del catálogo de sprites (azul, rojo, verde, amarillo, turquesa y naranja) y pelo con negro #2e2a28, castaño oscuro #5b3a24, castaño claro #9c6b3f y canoso #d9d6d0. Calvo y "sin sombrero" no tienen capa. Un tono de piel sin hoja (hoy la piel 2) usa la 1. En el médico el pelo usa el mismo cuadro al estar quieto y caminar, y el cuadro 1 de quieto al atender (la cabeza no se mueve). El sombrero no se dibuja dormido: se cae al desplomarse y vuelve al despertarse (así vienen las hojas). En la demo la apariencia sale de `game.RandomAppearance` con una semilla sacada del ID, y el médico que atiende un episodio hace la animación de atender a su lado antes de llevárselo. El camillero sigue como rectángulo hasta que tenga sprite *(ya lo tiene desde F4: ver abajo)*.
  - **Cómo quedó (F4, 2026-10-16)**: el camillero tiene su hoja (`staff_orderly_skin{1,3,4}.png`, pijama verde como la maqueta, aprobada el 2026-10-15). Tiene el mismo formato que la del médico, pero la fila 5 es **empujar** en vez de atender. Se dibuja igual que el médico (`drawOrderlySprite`/`orderlyLayers`): su hoja con el pelo del paciente encima, y en la fila de empujar el pelo usa la de caminar a la izquierda. Mientras mueve la camilla usa la animación de empujar. Un médico que empuja (porque no había camillero libre) sigue con la de caminar, porque su hoja no tiene fila de empujar.
  - **Cómo quedó (empujar hacia arriba y abajo, 2026-10-17, §12.8)**: la hoja del camillero sumó dos filas y pasó a 64 × 168 (7 filas): fila 5 empujar de lado (como antes), **fila 6 empujar hacia abajo** y **fila 7 empujar hacia arriba** (`staffRect`). `pushAnimation(dx, dy)` (`sprites.go`) elige la fila igual que `walkAnimation` elige la de caminar, según hacia dónde se mueve quien empuja en ese tick (así funciona también corriéndose de costado en el pasillo 1: la fila sale de cómo se mueve de verdad, no de hacia dónde apunta la camilla). El pelo del camillero usa la fila de caminar que corresponda (abajo o arriba) en vez de siempre la de la izquierda. El orden de dibujo (de atrás para adelante según los pies) ya dejaba la camilla delante de quien empuja al subir y detrás al bajar; no hizo falta tocarlo. Visto con el backdoor (demo, paso 7: Wilmer Camargo empuja a Ludys Arrieta hacia la habitación 103): de espaldas al subir, con la camilla y la paciente (ya rotada, cabeza hacia él) por delante.
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
- **Despertarse**: el cuadro 1 va dentro de la cama, rotado como *Dormido en cama*. Desde el cuadro 2 (sentado) el paciente está de pie, sin rotar, donde estaba acostado: se sienta, se estira y se para sobre la cama. Mientras se despierta no se mueve; después camina hasta el costado de la cama (sus pies pueden pisar su propia cama, ningún otro mueble). *(Antes aparecía de golpe al lado de la cama; se cambió el 2026-10-11 por la regla de "nada de teletransporte", sección 10.8.)*
  - **En la demo**: al llegar al costado de la cama hace *Alta* (saluda) una sola vez y queda en *Quieto*. En la demo nadie se va del hospital, porque el modelo no tiene alta.
  - **En el modo juego**: cuando recibe el alta, hace despertarse → alta → camina hasta la puerta del lobby y sale del mapa.
- La burbuja "Zzz" aparece solo cuando el paciente ya está acostado (en el piso o en la cama). El modelo lo marca dormido desde el ataque, pero en pantalla primero camina al sitio y se desploma.
- **Ícono "falta el médico"** (hecho 2026-10-17, §12.8): sobre el paciente en cama sin revisar (`AwaitingReview`), a ×3 en (+169, +64) desde la esquina de arriba a la izquierda de su habitación (`gui/ui/need_doctor_icon.go`, ficha en `gui/assets/sprites/ICONS_ASSETS.md`). Hoja de 2 cuadros de 19×18, 0,4 s cada uno, en bucle; parpadea desde los 20 s de espera.
- El personal y los personajes de eventos usan el mismo formato con sus animaciones (atender, empujar camilla…).
- Las animaciones avanzan por ticks de `Update` (60/s), nunca con `time.Sleep`.

### 10.4 Entrada
- Clic en paciente `Collapsed`, `InHallway` o esperando revisión + clic en personal libre → `Game.Dispatch(patientID, staffID)`. El estado del paciente dice si es recoger, trasladar desde el pasillo o revisar (5.5 y 5.6).
- Clic derecho en paciente en cama → `Game.WakeEarly(patientID)`.
- Hover → tooltip; **Shift** → todas las etiquetas; **Esc** → pausa; flechas + Enter en los portapapeles.

### 10.5 Texto
- Fuente **Press Start 2P** (OFL) en `assets/fonts/`, con `text/v2`. Llega con la pantalla de inicio (9.3); mientras tanto la interfaz usa Go Mono a 16/14/20 px.
- ⚠️ **Tildes**: en Press Start 2P la Ó, É, Í, Ú y Ñ vienen encogidas (para que la tilde quepa en el cuadro de 8 × 8) y parecen minúsculas. Donde se use esa fuente, la tilde se dibuja **a mano encima de la letra normal**, como la virgulilla de COSTEÑOS en la maqueta. Lo hace solo un helper de texto (`accentMarks` en `gui/ui/pixelfont.go`, con su test): cambia Á, É, Í, Ó, Ú y Ñ mayúsculas por la letra normal y dibuja la tilde encima, por arriba de la línea, porque las mayúsculas la llenan entera.
- ✅ **`textFace` vuelve a 16 px (hecho 2026-10-17)**: había bajado a 8 el 2026-10-08 (§12.8) para que los textos largos de la demo y las filas del reporte cupieran enteros en una línea; a 1280 × 720 eso se leía muy chico (prueba de Samuel). Los textos que no caben enteros a 16 px ya no achican toda la letra: los subtítulos de la demo se parten en dos renglones por palabras (`wrapTwoLines`, `gui/ui/pixelfont.go`; también la usa el nombre del candidato de la bolsa de empleo, que antes tenía su propia copia) y las filas largas del reporte (episodios, habitaciones, severidad) se recortan con "…" (`truncateWithEllipsis`). Tests en `gui/ui/text_fit_test.go`.

### 10.6 Comunicación con `game/`
- Cada `Draw` usa `game.Snapshot()`: una copia de solo lectura construida con el mutex de `Game` tomado. `Snapshot()` y `Report()` contienen **solo valores** (textos, números y estados), nunca punteros del modelo como `*hospital.Patient`: la interfaz no puede leer el hospital sin pasar por `Game`.
- Las acciones son métodos de `Game` (`Dispatch`, `WakeEarly`, `Hire`, `Pause`, `Resume`, `Report`). Una partida nueva es un `game.New(rng, looks)` nuevo con su propio `context`, que cancela la anterior: `rng` es el azar de la partida y `looks` el cosmético (la apariencia del personal), aparte para no cambiar la secuencia de los tests con semilla fija. Desde F4 es `game.New(rng, looks, hiring)`: `hiring` sortea la bolsa de empleo (§6), aparte por la misma razón (2026-10-15).
- Para que la interfaz haga caminar a cada uno al ritmo del juego, la foto trae `Left`: lo que le falta a la caminata o al traslado del personal y, en los pacientes, a la llegada, la salida o el cabeceo. El personal trae también su `Look`. La demo usa el tipo `game.Demo` (`Next`, `Skip`, `Snapshot`, `Report`). Los errores se muestran como avisos.
- Los estados se muestran con el `String()` de las constantes tipadas del modelo.

### 10.7 Mapa con imagen de fondo y rutas (hecho, 2026-10-10)
Aplica a la demo y al juego (mismo código de `gui/`). Reemplaza la tarea "rutas por puntos de paso" y el detalle del camillero que atravesaba la 102.
1. **Fondo**: `gui/assets/map/hospital_map_1280x720.png` es la maqueta (`hospital.dc.html`) renderizada a 1280 × 720 (1280/464 px de pantalla por px de mundo, bordes redondeados, sin antialias). Solo trae lo que nunca se mueve: sin personajes, sin taxi, sin humo y sin rayitas de movimiento. Se embebe con `//go:embed` y se dibuja de fondo; la interfaz deja de dibujar pisos, paredes y muebles con rectángulos.
2. **Resolución lógica 1280 × 720** (antes 640 × 360): `layout.go` escala con 1280/464; los personajes se dibujaban ×2 (desde la sección 10.8, ×3; escalado entero, *nearest*), incluidas la rotación en la cama y el volteo; la fuente se ajusta al tamaño nuevo; la ventana sigue en 1280 × 720.
3. **Camas**: el fondo trae las **3 camas libres** (la cobija doblada). `gui/assets/map/bed_blanket.png` (56 × 48, la cobija subida) va **encima del paciente acostado**, en (126, 88), (399, 88) y (672, 88); esas posiciones salen de `layout.go` y las revisa un test. Orden de dibujo: fondo → paciente en la cama → cobija. Las dos imágenes salen de la misma maqueta y se verificaron píxel a píxel contra el mapa original.
4. **Rutas**: eran puntos de paso a los lados de cada puerta con Dijkstra; desde la sección 10.8 son una cuadrícula de piso libre con A*.
5. **Lo que se mueve** va como sprite aparte, ya escalado (se dibuja a 1×, en píxeles de 1280 × 720):
   - el **tráfico** de la calle (desde el 2026-10-11, en lugar de `taxi.png`): ver la sección 10.8, punto 12.
   - `gui/assets/map/steam_pot.png`: humo de la olla de la cafetería, hoja de 3 cuadros de 44 × 36 en fila; esquina superior izquierda en (40, 246), cambia de cuadro cada ~300 ms, en bucle.
   - Todo animado por ticks de `Update`, nunca con `time.Sleep`.
6. **Arreglos del fondo** (2026-10-10, parches de Samuel pegados con un programa temporal y verificados): andén de losas en tono cálido en la acera, la línea verde del pasillo 2 centrada (x = 321 de la maqueta) y un extintor en vez de la pantallita del pasillo 1.
7. **Puestos sobre piso libre** (`gui/ui/spots.go`): los muebles de la maqueta (`put(make(w, h), x, y)`) son obstáculos. Cada puesto deja el cuerpo visible dentro de su zona y sin tocar un mueble, con sus rótulos dentro de la zona y sin tapar a los vecinos. Si así no cabe, un segundo intento deja que los rótulos se salgan unos píxeles, sin tapar otro rótulo, un letrero ni un mueble (hoy solo lo usa el acostado de radiología). Los rótulos de zona tampoco van sobre muebles: el de radiología va abajo a la derecha, detrás del escáner (antes arriba a la izquierda; cambió el 2026-10-11 con el letrero de radiación como piso) y el de la sala del personal, encima de la consola.
   - **Seis puestos para el personal** (2026-10-17, prueba de Samuel: "cuando contrato gente solo hay 3 espacios donde aparece el personal, entonces aparecen 4 en el mismo lugar"). La causa era doble: la sala del personal solo tiene 3 puestos de pie (×3 no da para más: el resto del piso libre lo tapan los armarios y la consola) y, además, `game_world.go` ubicaba a cada uno con `slot(zoneStaffRoom, i)`, con `i` la posición en la foto de `Snapshot().Staff` — no un puesto fijo por persona —, así que a partir del cuarto todos caían en el mismo `slot` de respaldo (el último de la lista). Arreglo (`gui/ui/staff_rest.go`): `staffHomeTarget` le da a cada ID su índice fijo con el mismo `spotBook` (`s.standing`, el que ya usan los pacientes) y `staffHomeSpot` reparte primero los 3 de la sala y, si no alcanzan, el desborde del pasillo 2 junto a su puerta (`staffOverflowSpots`, hasta completar 6: `game.maxStaff`). Ese desborde no le reserva el sitio al paciente que podría desplomarse en el pasillo 2 (sí lo hace el cupo general de la zona, que `spotsOf` reparte entre quienes deambulan ahí): el pasillo es tan chico que, si también se reservara ese espacio, no alcanzaría para el personal. En la práctica casi no se usa (hacen falta más de 4 libres a la vez) y, si alguna vez coincide con un desplome justo ahí, en el peor caso un rótulo se monta sobre el otro, igual que el resto de los puestos relajados del punto 3 de la sección 10.8. Test: `TestStaffHomeSpot_SixDistinctSpotsWithoutOverlap` (6 puestos, ninguno repetido ni encimado).
8. **Escala de los personajes**: ×2. Se probó ×3 (48 × 72) y se descartó: el pasillo 1 mide 66 px de alto, el hueco de las puertas 38 px contra 42 del cuerpo, y en radiología no cabía un acostado con su ayudante. *(El 2026-10-11 Samuel cambió de decisión: ×3 con reglas relajadas, sección 10.8.)*

### 10.8 Movimiento (hecho, 2026-10-11)
Ronda antes de F3b; aplica a la demo y al juego (mismo código de `gui/`).
1. **Escala ×3** (48 × 72), con reglas relajadas: las reglas valen para los **pies**. En el pasillo 1 (66 px de alto) la cabeza de quien camina o está de pie se monta sobre la pared de arriba (vista 3/4), y al cruzar una puerta el cuerpo puede rozar el marco; los pies van por el hueco.
2. **Rutas por cuadrícula** (`gui/ui/nav.go`): la pantalla se parte en celdas de 8 px; una celda está libre si la **huella de los pies** (las piernas, x de 3 a 13 de la hoja en las 3 filas de abajo: 30 × 9 px) no toca una pared ni un mueble. Piso es el de las zonas, los huecos de las puertas y la acera; la calle es pared. El camino sale con **A\* de 4 direcciones** (sin diagonales), que penaliza cada cambio de dirección para preferir pocos tramos largos; el suavizado solo quita puntos colineales, así todo tramo es horizontal o vertical (con test). El punto de partida y el de llegada no siempre caen en el centro de una celda: el primer y el último tramo se parten en escuadra. *(Movimiento "cuadriculado", 2026-10-15, para todos; antes A\* de 8 direcciones con el suavizado que dejaba tramos en cualquier ángulo. Samuel: "el movimiento debe ser mas \"cuadriculado\"".)* Un costo extra aleja a cada uno de las celdas donde el cuerpo quedaría sobre una pared (salvo la cabeza en el pasillo 1 y el roce en las puertas); afuera los manda por la franja baja de la acera, sin montarse en la fachada. Los muebles de la acera (paradero, poste y la base de las palmas) también son obstáculos.
3. **Puestos** (`gui/ui/spots.go`): solo donde se llega caminando desde la puerta principal (los muebles dejan rincones encerrados, como la mitad de radiología detrás del escáner) y nunca frente a la puerta principal. El ayudante de un acostado va a su lado o en diagonal, siempre adentro de la zona del paciente (desde el 2026-10-11; antes podía quedar en el pasillo 1, junto a la puerta). En la recepción, a ×3, caben justo 5 de pie: solo los 2 primeros dejan libre el puesto de piso (cuando Ludys se desploma ahí ya solo quedan 2 de pie). El que se despierta queda al lado de su cama: a la derecha si cabe (cabe sin tapar el letrero ni los rótulos del que llegue a esa cama), si no a la izquierda. ⚠️ **Pendiente del OK de Samuel**: en el segundo intento del buscador, la burbuja "Zzz" y los rayitos de un acostado pueden quedar sobre un mueble (a ×3, en radiología y en la cafetería no caben de otra forma).
4. **Velocidad**: el personal camina a 2 px por tick y los pacientes a 1,5 (constantes de la interfaz; la posición es float y se redondea al dibujar). Caminar va a 8 ticks por cuadro (unos 8 cuadros por segundo). En el modo juego (F3b) la caminata dura lo que diga `game/` ((8 − rapidez) s) y la interfaz solo interpola.
5. **Camilla** (`stretcher.go`, `carry.go`): todo traslado va en camilla, lo haga un camillero o un médico. Quien lleva camina **detrás**, empujando, con su animación de caminar; la camilla va delante, horizontal yendo a los lados y vertical subiendo o bajando (subiendo va 20 px más adelante, para que se vea por encima de la cabeza de quien empuja), y el paciente va encima con sus capas y la animación de dormido en el piso, con la cabeza hacia quien empuja. Al recogerlo, la camilla aparece lo más cerca posible del paciente y él se pasa a ella; al llegar, pasa de la camilla a la cama (la camilla termina pegada al costado derecho de la cama) y la camilla desaparece. Para las rutas cuenta la **huella de la camilla** en el piso (54 × 24 px), no el dibujo del paciente encima: ni ella ni los pies de quien empuja tocan paredes ni muebles. La ruta empujando es un Dijkstra sobre (celda, hacia dónde apunta la camilla): avanzar, girar (la camilla tiene que caber en las dos direcciones; cuesta) o correrse de costado (más caro; hace falta para entrar al pasillo 1, donde la camilla vertical no cabe de ninguna forma de girar, de a poco o de golpe), nunca hacia atrás. La camilla es un obstáculo para las rutas de los demás, y los que están quietos en su camino **se hacen a un lado** mientras pasa y después vuelven a su puesto. Hoy se dibuja con código (marco gris #8a9099, borde #141414, sábana blanca y dos ruedas); el sprite se cambia en `loadStretcher` sin tocar la lógica.
6. **Llegada** (paso 3 de la demo): los 5 pacientes aparecen fuera de la pantalla, a la derecha, y entran caminando por la acera y por la puerta principal hasta su puesto de la recepción, uno cada 1,5 s; el avance automático espera a que llegue el último. La **puerta principal** se dibuja abierta (las dos hojas corridas) mientras haya pies a menos de 48 px de su hueco, y se cierra sola cuando no viene nadie; en el modo juego la abrirá `Snapshot.DoorOpen`.
7. **Variedad en la demo**: la semilla de cada ID lleva una sal fija con la que los 2 médicos se ven con piel distinta y no hay dos pacientes con el mismo sombrero y el mismo color de camisa (un test lo fija).
8. **Tests**: toda la demo, tick a tick: los pies de quien está de pie o caminando, el cuerpo de quien está acostado en el piso y la huella de la camilla no tocan paredes ni muebles (los muebles se miran directo en la maqueta, así que si a la cuadrícula le falta uno, el test lo nota), y los quietos no quedan uno encima del otro. Además: hay ruta entre todas las zonas y desde fuera de la pantalla, la camilla llega desde cada puesto de piso a cada cama, los traslados de la demo van en camilla, la llegada escalonada, la puerta y la variedad.
9. **Puertas anchas** (2026-10-11): Samuel ensanchó las puertas en el mapa de fondo. En coordenadas de la maqueta: abajo de las habitaciones 101, 102 y 103, `hDoor` x = 70, 169 y 268 con ancho 20 (antes 76, 175 y 274, ancho 14); radiología x = 318, ancho 20 (antes 320, ancho 16); arriba de la cafetería x = 87, ancho 20 (antes 90, ancho 14); las laterales cafetería–recepción y pasillo 2–sala del personal, `vDoor` y = 147, alto 20 (antes 150, alto 14). A 1280 × 720 miden 55 px. Las paredes y la cuadrícula salen de `doorways`, y un test mide en el PNG el hueco de cada puerta y lo compara con el código.
10. **De la cafetería a la 101** caminando se sube por la puerta de arriba de la cafetería, sin pasar por la recepción (con test). Con la camilla también desde la ronda de mapa (punto 12).
11. **Nada de teletransporte** (2026-10-11): en toda la demo, tick a tick, nadie se mueve en un tick más que su velocidad, más 1 px de redondeo; el paciente en camilla va a la del personal y, mientras la camilla gira, a la del giro. Las excepciones son aparecer y el tick en que la camilla gira 90° de golpe, en el que el paciente salta de la huella de antes a la de después (2026-10-15; antes era "mientras la camilla gira", porque giraba de a poco). Aparecer: los pacientes nuevos aparecen fuera de la pantalla. Quien lleva a un paciente vuelve caminando a su puesto y el que se despierta camina de la cama a su costado (sección 10.3).
12. **Ronda de mapa** (2026-10-11, segundo mapa de Samuel; un test compara las puertas con el PNG):
    - **Calle**: acera de y 182 a 215 (antes 210), bordillo, **un solo carril** (218–243) con la cebra en x 199–229, bordillo y playa (246–261). El paradero y su poste bajan a y = 184; la base de las palmeras a y = 186. Solo la base es obstáculo: la **copa** (`palm_crown.png`, 104 × 104, en (771, 469) y (1047, 469)) se dibuja **encima** de los personajes, así quien pasa cerca queda debajo de las hojas.
    - **Tráfico** (en lugar del taxi único): carros de 3 capas de 132 × 72 en el mismo punto (`car_details.png` sin teñir, `car_body.png` teñida con `ColorScale` y `car_taxi_sign.png` encima en los taxis). Taxi amarillo (255, 220, 70) con letrero en ~40 % de los carros; los demás rojo, azul, blanco, verde o gris. Carril con la esquina superior izquierda en y = 600, de izquierda a derecha. Entra uno cada 2 a 8 s por la izquierda, fuera de la pantalla, si hay espacio, con su velocidad deseada al azar entre 2,5 y 4,5 px por tick. Acelera y frena a lo sumo 0,05 px por tick²: en cada tick elige la mayor velocidad con la que, si de ahí en adelante él y el de adelante frenaran al máximo, la distancia nunca bajaría de 40 px, así que nunca choca. Base del bus de F5: un carro con `stopAt` frena hasta que su frente queda ahí y los de atrás hacen fila. Azar cosmético propio de la interfaz (semilla fija en los tests). En la demo el carril queda debajo de los subtítulos; se ve con C.
    - **Puertas**: la de arriba de la cafetería pasa a `hDoor(68, 95, 30)`, alineada con la de la 101, y hay una **nueva** entre la recepción y el pasillo 2 (`vDoor(297, 147, 20)`). La **puerta principal** se abre y se cierra suave: las hojas del fondo se corren en 12 ticks y se cierra 0,5 s después de que no haya nadie cerca. El tapete de la entrada se corrió a x = 203 (es piso: no cambia nada en el código).
    - **Muebles**: el mostrador de la cafetería es `make(37, 14)` en (12, 101); el de la recepción es horizontal, `make(32, 10)` en (248, 108), con la silla del recepcionista en (267, 99,5). El **letrero amarillo de radiación es piso** (la maqueta lo dibuja plano, sin sombra, como los tapetes); si fuera mueble, en radiología no cabría un acostado con su ayudante. Los rótulos de zona no lo tapan: el de radiología va abajo a la derecha, detrás del escáner.
    - **Quien atiende** lo hace **adentro de la zona del paciente**, junto a él: a su lado o en diagonal y, si así no cabe en ningún sitio de la zona (radiología), en el sitio libre más cercano, a menos de 140 px. Nunca desde el pasillo ni desde la zona de al lado (con test, para todas las zonas).
    - **Puestos de piso**: además, la camilla tiene que poder aparecer junto al paciente (a 6 celdas como mucho) y sacarlo al pasillo 1 por la puerta de su zona. Así, de la cafetería a la 101 la camilla **sube derecho por las dos puertas alineadas** (con test); el puesto de piso de la cafetería ya no queda debajo de una mesa.
    - **Giro de la camilla** (Samuel, 2026-10-14; hecho el 2026-10-15): gira **90° de golpe**, en el tick en que quien empuja llega a la esquina. La huella en el piso no toca paredes ni muebles ni antes ni después del giro (se revisa por separado en cada dirección); como ya no hay barrido, el A\* ya no prefiere girar donde hay espacio. *(Reemplaza la opción b del 2026-10-11: girar de a poco alrededor de los pies de quien empuja en 18 ticks, que a Samuel le pareció "no muy estilo de un juego 2d".)*
13. **El personal libre deambula** (2026-10-17, solo el modo juego — pedido de Samuel: "que el personal no esté tan quieto"; `gui/ui/staff_rest.go`). Mientras `Activity` es `Free` ("libre" en la foto), cada 8 a 15 s cambia de zona entre las suyas (`restZonesFor`): los médicos, entre la sala del personal, la cafetería ("por un tinto") y el pasillo 2; los camilleros, entre la sala y la recepción, de pie junto a la entrada. El destino es un puesto libre de esa zona (el mismo cupo que usan los pacientes que deambulan ahí, `spotsOf`/`s.standing`, para no encimarse con ellos ni entre sí) y el reloj de deambular queda congelado mientras está ocupado (no se llama a `staffRestSpot` fuera del caso `Free`). El azar es **propio de la interfaz** (`seedSource`, nunca `g.rng`, igual que el tráfico decorado de la sección 10.8 punto 12): no cambia la secuencia de los tests con semilla fija del juego y, con el backdoor de pruebas (`-agent -seed N`), es determinista. Al despachar a alguien libre, sale **desde donde esté** (la posición real, `s.pos`): el juego sigue repartiendo el tiempo de la caminata con `Left`/`timed` sin que la interfaz necesite saber si venía de la sala, la cafetería o la recepción. Test: `TestFreeStaff_WandersAwayFromTheStaffRoomOverTime` (en 20 s libre cambia de zona y nunca comparte el puesto exacto con otro).
14. **Quien camina no choca; quien está quieto, sí** (2026-10-17, afinado el 2026-10-19 por pedido de Samuel; aplica a la demo y al juego).
    - **Caminando** (`nextWaypoint`, `gui/ui/world.go`): nadie tiene cuerpo sólido mientras camina; pasa de largo junto a otro (o a través de él) sin esperarlo ni rodearlo. Se probó antes que cada caminante reservara la celda de adelante y esperara o rodeara al que la ocupaba, pero el mapa tiene pasillos angostos (el pasillo 1 y la recepción quedan a 8 px, menos que el alto de un cuerpo) donde no había otro camino, y el personal se quedaba trabado para siempre con el camino tapado. La camilla de otro sí se esquiva al calcular la ruta (`avoidFor`), y los quietos en el carril de una camilla se hacen a un lado (`dodgesFor`). Test: `TestWalkers_PassThroughEachOtherWithoutWaiting`.
    - **Quieto**: cada quieto reserva un puesto de un `spotBook` y lo suelta al irse (puntos 3, 7 y 13). El desborde del pasillo 2 para el personal sin sitio en la sala (`staffHomeTarget`) pide su puesto al **mismo libro** (`book.take(zoneHallway2, id)`) que usa quien deambula ahí. Los pacientes que llegan juntos esperan fuera de la pantalla cada uno en su puesto de una fila (`arrivalQueueSpot`, `gui/ui/arrival.go`), no todos en el mismo punto. Test: `TestNobodyStandsOnAnother_Demo` y `_Game` (`gui/ui/overlap_test.go`): tick a tick, ningún par de cuerpos quietos se encima más de `allowOverlapPx`.
    - **Al terminar la ruta** (corrección de Samuel, 2026-10-19; `settle`, `gui/ui/world.go`): a 3 celdas de su destino, cada uno decide UNA vez dónde se queda. Si ahí su cuerpo se encimaría con alguien quieto (o con el destino de alguien que todavía camina), se queda en el sitio libre más cercano (`dodgeSpot`); en la acera, hace fila hacia la derecha. Las escenas preguntan `arrived(id, dest)` en vez de comparar la posición con `dest`. Quien empuja la camilla o va encima necesita su punto exacto: ahí son los quietos los que se apartan (`clearSpotFor`), al llegar a recoger, mientras el paciente sube y mientras baja a la cama. Tests: `TestWalker_DoesNotStopOnTopOfSomeoneStill` y `TestNobodyStandsOnAnother_GameSeeds` (semillas 1, 5 y 6). Pendiente: en barridos largos (50 semillas × 20 000 ticks) todavía aparece, rara vez, alguien de un traslado (quien empuja o el paciente encima) quieto junto a otro: por ejemplo, dos camillas que recogen a dos pacientes desplomados uno al lado del otro. Sin contar los traslados, el barrido queda limpio.
    - `patientAnim.step` recibe `arrived` (si de verdad está en su destino) además de `dx,dy`: quedarse quieto un tick a mitad de camino no dispara el desplome ahí mismo.

### 10.9 Backdoor de pruebas (hecho, 2026-10-14)
Aprobado por Samuel el 2026-10-14 ("Hazlo"), para que el agente `tester` llegue a cualquier
momento del juego en segundos, sin una persona jugando, saque capturas y un dump de estado, y
salga. `gui/main.go`, `gui/ui/agent.go`, `gui/ui/agent_hooks.go` y `gui/ui/script.go`.

- **Flags** (`gui/main.go`; el modo normal, `cd gui && go run .`, queda idéntico):

  | Flag | Qué hace |
  |---|---|
  | `-agent` | activa el modo de prueba: sin ventana de juego normal, arranca directo en `-scene`. Nunca escribe archivos del jugador (no hay partida guardada todavía) ni suena nada (hoy no hay sonido). |
  | `-scene <id>` | escena de arranque: ver la tabla de abajo. |
  | `-step N` | con `-scene demo`: adelanta hasta el paso N (1 a 12), con todos ya en su puesto final (sin reproducir las coreografías intermedias). |
  | `-seed N` | semilla fija del azar de la partida/demo y del tráfico decorado (en vez de la hora). |
  | `-input archivo` | guion de instrucciones, una por línea (ver abajo). |
  | `-frames N` | ticks de `Update` sin entrada, después de que se acaba el guion (o directo, si no hay `-input`). |
  | `-shot ruta.png` | captura al terminar (además de los `shot` del guion). |
  | `-dump ruta.json` | dump de estado al terminar (además de los `dump` del guion). |
  | `-quit` | termina solo al acabar; sin ella, la ventana se queda quieta para mirarla a mano. |

- **Tabla de escenas** (`-scene`):

  | id | Qué arma |
  |---|---|
  | `title` | la portada (`NewTitleScene`); es el valor por defecto. |
  | `demo` | `game.NewDemo()` + `NewDemoScene`; con `-step N`, adelantada hasta ese paso. |
  | `game` | una partida nueva (`newGameScene`, **sin** arrancar la goroutine motor: ver determinismo). |
  | `pause` | una partida nueva, con el menú de pausa encima (`newPauseScene`). |
  | `report` | corre la demostración completa (los 12 pasos) y muestra su Shift Report. |
  | `hiring` | una partida nueva con la bolsa de empleo ya abierta (sin la animación de abrir). |

- **El guion** (`-input`), una instrucción por línea (`#` al principio o línea vacía: se ignoran); `gui/ui/script.go` valida todo antes de correr nada, con el número de línea en el error:

  | Instrucción | Qué hace |
  |---|---|
  | `wait N` | N ticks de `Update` sin tocar nada. |
  | `click X Y` | mueve el mouse a (X, Y) y hace clic izquierdo, un tick. |
  | `rclick X Y` | igual, con clic derecho. |
  | `mouse X Y` | solo mueve el mouse (para el tooltip, sin clic). |
  | `key Nombre` | esa tecla, apretada un tick (como `inpututil.IsKeyJustPressed`). |
  | `hold Nombre N` | esa tecla, sostenida N ticks (el primero dispara el "se acaba de apretar"; los demás, no — como de verdad). Hace falta para **Shift** (todas las etiquetas), que se lee continuo. |
  | `shot ruta.png` | captura la pantalla (1280 × 720, PNG a 1×: sin escalar, a diferencia de los sprites) con el cuadro que sigue. |
  | `dump ruta.json` | el estado que dibuja la escena (ver abajo), en ese momento. |
  | `quit` | termina la corrida ahí mismo. |

  Los nombres de tecla válidos son justo las que lee alguna escena del juego (`scriptKeys` en `script.go`): `Escape`, `Space`, `Enter`, `NumpadEnter`, `ArrowUp`, `ArrowDown`, `ArrowLeft`, `ArrowRight`, `W`, `S`, `Shift`, `C`, `A`, `M`. Un nombre que no esté ahí es un error claro al cargar el guion. `M` (silencio, §10.11) no suena nada en `-agent` igual que cualquier otra tecla: el backdoor nunca arma el reproductor.

- **Determinismo** (GAME_DESIGN §11: "los tests llaman `Tick` a mano"; esto es lo mismo). `-agent` reemplaza tres relojes de prueba:
  1. `seedSource` (`gui/ui/seed.go`): en vez de la hora, da `-seed` fija. La usan `NewGameScene`, `newGameScene` (tráfico) y `NewDemoScene` (tráfico).
  2. `newGameSceneFn` (`gui/ui/game_scene.go`): a quién llaman NUEVA PARTIDA (desde la portada o el Game Over). En modo agente arma la partida con `newGameScene` directo, **sin** `Game.Start(ctx)`: así nadie más toca `Game.Tick` por su cuenta.
  3. El propio driver (`agentRunner.driveFrame`, `gui/ui/agent.go`) llama `Game.Tick(100 * time.Millisecond)` cada 6 `Update` (60 Updates/s de Ebitengine ÷ 6 = 10/s = uno cada 100 ms), exactamente como la goroutine motor real y como `play()` en `game_scene_test.go`.

  También fuerza `isFocused` (app.go) a verdadero: sin ventana real con foco, `GameScene.pausesOnBlur()` abriría la pausa en todos los ticks.

- **Cómo lee la entrada cada escena** (`gui/ui/agent_hooks.go`): en vez de llamar directo a `inpututil`/`ebiten`, las escenas pasan por tres variables de paquete (`keyJustPressed`, `clickJustPressed`, `readInputFn`); en producción son la lectura real, y `-agent` las reemplaza por la entrada del guion. Así ninguna escena sabe si la está jugando una persona o un guion — ni `App.Update()` (el Esc), ni `DemoScene.Update()` (C, A, S, Espacio, clic), ni las escenas de portapapeles (`readInput`, `input.go`).

- **El dump** (`dumpState`, `agent.go`): escena actual y tick; en el juego, la foto completa que usaría un tester para no tener que adivinar mirando los píxeles (reloj, día, plata, reputación, avisos, pacientes con ID/estado/posición en pantalla, personal con ID/rol/actividad/posición, camillas activas con su huella y hacia dónde apuntan); en la demo, el paso actual.

- **Probado de verdad** (2026-10-14, `builder`): `go run . -agent -scene game -seed 7 -input guion.txt -quit` con un guion que espera a que P-001 se desplome, hace clic en el paciente y después en el camillero libre (C-01) y saca dos capturas y un dump: el dump mostró `C-01` pasando de `"libre"` a `"va a recogerlo"` y caminando hacia el paciente, y las capturas lo confirmaron a simple vista. `title`, `demo -step N`, `report` y `pause` probados igual.

- **Tests** (`gui/ui/script_test.go`, `agent_test.go`): el parser del guion (líneas válidas, tecla desconocida, número inválido, argumentos de más o de menos, línea y archivo), que la misma semilla + el mismo guion da el mismo dump en dos corridas (sin ventana: arma la escena y llama `agentRunner.step()` a mano, como los demás tests de `gui/ui` llaman `s.update(in)`), `hold` como tecla sostenida de verdad (el borde solo en el primer tick, Shift continuo en todos) y `-step` de la demo. `shot` necesita una ventana real de Ebitengine (la prueba el agente `tester` a mano, no `go test`).

- ⚠️ **Pendiente**: la tabla de `-scene` no tiene entradas para llegar directo al panel de fin de día ni al de Game Over (10.10): hoy solo se ven jugando hasta las 20:00 o hasta perder de verdad. Agregarlas pediría un método de prueba en `game.Game` para forzar esos dos estados sin pasar por `StartNextDay`/las reglas de derrota (que validan el reloj y la reputación de verdad); no se hizo por no abrir ese hueco sin que Samuel lo revise primero.

### 10.10 Paneles animados (hecho, 2026-10-17)
Pedido de Samuel: *"hazle así animados los menús como el de contratar que me gustó mucho"* (la bolsa de empleo, que ya abría con ease-out desde el portapapeles chiquito del HUD y cerraba con ease-in). Se factorizó esa animación en `gui/ui/anim.go` (`easeOutOpenness`, `easeInOpenness`, `clipboardTransform`, el tipo `animPanel` con sus fases `panelOpening`/`panelOpen`/`panelClosing`) y `gui/ui/theme.go` (`drawAnimatedPanel`: oscurece la pantalla, dibuja el contenido del panel en su sitio final sobre un lienzo de pantalla completa y lo pinta escalado y centrado según la *openness* actual), para que la bolsa de empleo, la pausa (9.4) y, en el modo juego, el panel de fin de día (`drawDayOver`) y el de Game Over (`drawGameOver`) la reusen en vez de copiarla.

- **De dónde a dónde**: la pausa abre desde el botón PAUSA del HUD (o el centro, en la demo, que no tiene HUD); el fin de día y el Game Over, desde el botón REPORTE (los dos son "un reporte" de cómo quedó el turno o la partida). Los tres cierran al mismo punto por donde abrieron.
- **Mientras anima no acepta entrada** (ni teclado ni mouse), igual que la bolsa: un clic en SIGUIENTE DÍA, REANUDAR, NUEVA PARTIDA, etc. mientras el panel todavía se mueve no hace nada.
- **La acción pasa al terminar de cerrar**, no al elegirla: `StartNextDay`, `Game.Resume`/`onQuit` o `newGameSceneFn` se llaman en el tick en que `animPanel.step()` devuelve `true`, no en el clic que arrancó el cierre. En el fin de día y el Game Over, `GameScene` guarda la animación en curso (`dayOverAnim`, `gameOverAnim`, nil mientras no hace falta ninguna) y, en el Game Over, la acción pendiente en `gameOverNext` (NUEVA PARTIDA o MENÚ), porque a diferencia del fin de día (que siempre termina en `StartNextDay`) depende de qué botón se usó.
- **Tests**: fases (abriendo/abierto/cerrando), *openness* en 0, a mitad de camino y en 1, que no se acepte entrada mientras anima y que la acción pase justo al terminar de cerrar (`gui/ui/pause_scene_test.go`, `gui/ui/game_scene_overlay_test.go`; estos últimos prueban `updateDayOver`/`updateGameOver` directo, sin pasar por `GameScene.update`, para no depender de hacer que la partida de prueba llegue de verdad a las 20:00 o a una derrota: esas reglas ya se prueban en `game/` y en `game_scene_test.go`).
- ✅ **El Shift Report también (hecho 2026-10-17)**: el reporte de mitad de partida (botón REPORTE del HUD) y el de la demo (`ReportScene`, `gui/ui/report_scene.go`) reusan el mismo `animPanel`: abre desde el botón REPORTE en el modo juego o desde el centro en la demo, y Espacio o clic arrancan el cierre; la vuelta a la portada o a la partida pasa al terminar de cerrar, no al apretar la tecla. Debajo del portapapeles se ve la partida en pausa (o un fondo liso en la demo). Tests de fases iguales a los de la pausa en `gui/ui/report_scene_test.go`.

### 10.11 Sonido (hecho, 2026-10-17)
Pedido de Samuel: *"añade sonidos acordes, que la vida está ahí"*. Generado por código, sin archivos de audio ni licencias que revisar: `gui/sfx` sintetiza cada efecto como PCM de 16 bits a **44100 Hz** con ondas cuadrada, triangular o ruido y una envolvente ADSR corta (`gui/sfx/synth.go`: `Tone`, con `LowPass`, `Concat` y `Mix` para armar los más complejos); `gui/ui/sound.go` es el único archivo de `gui/ui` que importa `ebiten/audio` y los manda a sonar.

- **Tabla de efectos** (50 a 600 ms, volumen moderado):

  | Efecto | Qué es | Cuándo suena |
  |---|---|---|
  | `click` | blip corto y agudo | botón o menú confirmado (portada, pausa, HUD, bolsa de empleo, fin de día, Game Over) |
  | `open` | barrido hacia arriba | se abre un portapapeles (bolsa, pausa, reporte, fin de día, Game Over) |
  | `close` | barrido hacia abajo | se cierra un portapapeles |
  | `hire` | caja registradora: dos blips + ruido corto | al contratar a un candidato |
  | `coin` | dos notas ascendentes; con racha, la 2ª sube un semitono por punto (hasta 4) | cobro de una revisión o alta: `Snapshot.Money` sube |
  | `streakLost` | dos notas descendentes cortas | la racha de atención vuelve a 0 |
  | `starDown` | barrido descendente grave | un `Snapshot.ReputationChanges` nuevo con `Delta` negativo |
  | `starUp` | barrido ascendente | un `ReputationChanges` nuevo con `Delta` positivo |
  | `collapse` | golpe sordo: ruido pasado por un pasabajos + un tono grave | un paciente pasa a `Collapsed` |
  | `rush` | alerta de dos tonos | `Snapshot.RushHour` pasa a cierto |
  | `dayEnd` | fanfarria de 4 notas | `Snapshot.DayOver` pasa a cierto |
  | `gameOver` | 3 notas descendentes tristes | `Snapshot.GameOver` deja de ser `NotOver` |
  | `snore` (muy bajo) | tono grave suave | un paciente pasa a `AwaitingReview` (se acaba de dormir en una cama) |

- **Cuándo suenan**: `gui/ui/sound_events.go` (`gameSoundEvents`, función pura, con tabla de casos en `sound_events_test.go`) compara la foto anterior con la nueva y devuelve qué efectos tocan; `soundTracker` la llama cada tick desde `GameScene.update` y le pasa el resultado a `playSound`/`playCoin` (`sound.go`). Nada de reglas de juego en `gui/ui`: solo se comparan los valores de dos `Snapshot`, igual que ya hacía `hud.update` para animar la barra. `click`, `open`, `hire` y `close` salen de los puntos de la interfaz que los disparan (`menu.update` al confirmar, `newAnimPanel`/`animPanel.startClose` en `anim.go`, los botones de `GameScene` y `HiringScene`), no de la foto.
- **Que no se apile**: cada efecto tiene su propio cupo de `soundCooldownTicks` (3 ticks, ~0,05 s): mientras no se agote, `playSound`/`playCoin` no vuelven a sonar ese mismo efecto, aunque la foto lo pida de nuevo.
- **Silencio** (tecla **M**, §9.5): `soundOn` en falso hace que `playSound` no haga nada, sin tocar el cupo.
- **El backdoor de pruebas (`-agent`) nunca suena** (ya lo decía §10.9): `initSound()` solo la llama `NewApp` (el modo normal); `loadAssets()`, que sí usa `-agent`, no la llama, así `soundPlayer` se queda en `nil` y `playSound` no hace nada (`TestAgentMode_NeverCreatesAPlayer`, `sound_test.go`).
- **Si el dispositivo de audio no se puede abrir** (o algo más sale mal armando el contexto), `initSound()` se recupera del pánico con `recover()` y deja `soundPlayer` en `nil`: el juego sigue, sin sonido y sin caerse.
- **Tests**: la síntesis es pura (`gui/sfx/synth_test.go`, `effects_test.go`: largo correcto, sin recorte, empieza y termina en silencio, el filtro suaviza, la racha sube el tono de `coin` y se topa en 4); el detector de eventos, con una tabla de casos (`gui/ui/sound_events_test.go`); que con el sonido apagado o sin reproductor no sea nada ni truene (`sound_test.go`); que `-agent` nunca arme el reproductor; y la tecla M (`TestApp_MTogglesSound`, `app.go`).
- **Cómo probarlo**: `cd gui && go run .` — contratar, revisar o dar de alta rápido (racha), dejar pasar la hora pico, dejar que alguien se desplome, terminar el día o perder, y apretar M para silenciar. No hay captura que lo muestre: es sonido.

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
- Verificación: `go test -race ./...` limpio en la raíz (incluye `game/`). El test del motor tiene tres goroutines, como el juego real: el motor, la interfaz (que pide `Snapshot()` y `Report()` sin parar) y el jugador (despacha, despierta, pausa y reanuda).

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


### 12.5 F3b: modo juego (hecho, 2026-10-11)
- **NUEVA PARTIDA** abre `GameScene` (`gui/ui/game_scene.go`): `game.New` con el azar de la hora y el motor corriendo (`Game.Start`). Es `pausable`: Esc, el botón PAUSA o **perder el foco** de la ventana abren la pausa (`Game.Pause`); SALIR AL MENÚ cancela el `context` y espera a que el motor termine.
- **Dibujo desde la foto** (`game_world.go`): cada tick la escena lee `Snapshot()` y decide a dónde camina cada uno. Los pacientes, por su etapa: llegan por la acera (al ritmo de `Left`), deambulan entre los puestos de pie de su zona, cabecean, se desploman en un puesto de piso, van en camilla a la cama y se van caminando por la acera hasta salir de la pantalla. El personal, por lo que hace: a recoger o a revisar al ritmo de `Left` (llega cuando el juego dice que llegó), lleva al paciente en camilla repartiendo el camino en el tiempo del traslado, y después de revisar atiende junto a la cama antes de volver a su sala. La demo y el juego comparten el mismo código de movimiento (`world.go`): posiciones, rutas, animaciones y camillas (en el juego puede haber varias a la vez; si la camilla de otro tapa el único paso, la ruta pasa junto a ella en vez de atravesar una pared).
- **Entrada** (`game_input.go`): clic en un paciente desplomado, en el pasillo o esperando revisión (queda con un marco amarillo) y después en alguien del personal → `Dispatch`; clic derecho en uno revisado en su cama → `WakeEarly`. Los errores del juego salen como aviso. Pasar el mouse muestra su ficha (ID y nombre, nivel y estado con el cronómetro; en el personal, rol, rapidez y pericia y lo que hace); **Shift**, todas las etiquetas. Siempre se ven las burbujas ("Zzz! 0:42" roja, que titila desde los 15 s, y "Zzz" azul en la cama), los rayitos y el punto del personal (verde libre, rojo ocupado).
- **Barra de arriba** (`hud.go`), según `HUD.dc.html`: ver 9.1. Un test lee la maqueta y compara los mapas de píxeles y la paleta con los de Go.
- **Abajo a la derecha**: el portapapeles chiquito de CONTRATAR ("Próximamente" hasta F4), REPORTE (el reporte de las 4 consultas con la partida en pausa; al cerrarlo sigue) y PAUSA. Los avisos del juego salen arriba a la izquierda unos segundos.
- **Fin del día y derrota mínimos**: a las 20:00, "FIN DEL DÍA N" con el puntaje y SIGUIENTE DÍA (la nómina y el reporte del día llegan en F4); al perder, el motivo, el puntaje final y los días, con NUEVA PARTIDA y MENÚ.
- ✅ **Decidido (2026-10-14)**: el traslado sigue en (8 − rapidez) s, aunque un jugador automático que atiende primero al que más lleva esperando sobrevive el día 1 en 16 de 20 partidas (con el de 2 s, en 20 de 20). F4 permite contratar personal más rápido.

### 12.6 Plan del 2026-10-14 (después de F3b)
Samuel aprobó las recomendaciones ("Hazlo") y agregó: *"lo que hay de juego en este momento sigue bastante crudo, principalmente en los movimientos de las camillas ya que se sienten irreales"*.
- Se acepta que la burbuja y los rayitos de un acostado queden sobre un mueble en radiología y en la cafetería (a ×3 no caben de otra forma).
- ✅ **Backdoor de pruebas mínimo** en `gui/` (escena, semilla, guion de entrada, captura y dump) para que el agente `tester` pueda revisar las pantallas (hecho 2026-10-14, detalle en §10.9).
- Nueva tarea: **que la camilla se mueva de forma creíble** (pendiente definir qué se siente irreal).
- Orden: F4 (lógica en paralelo con las vistas de Figma de la Bolsa de empleo, el reporte del día y el Game Over) → F5 (los 8 banners los propone el agente `art` en Figma y Samuel los aprueba) → F6 → etapa 8 el 18/10. Si no alcanza, se recortan primero la partida guardada, los ajustes y el sonido.
- Agentes del skill `pixel-game-studio`: `art` dibuja en Aseprite y arma las vistas en Figma; `builder` hace un commit por feature; nadie hace push hasta la última prueba de Samuel.

### 12.7 Decisiones de F4 (2026-10-15)
- La lógica de F4 (candidatos, `Hire`, nómina, quiebra y `DayReport`) está en `game/`, con TDD y 4 contrapruebas.
- La nómina se paga en el `Tick` de las 20:00 y no en `StartNextDay`. Se corrigió el comentario de `game.go` que decía lo contrario.
- Falta en `gui/`: la bolsa de empleo, el reporte del día y el Game Over con la quiebra.

### 12.8 Prueba de Samuel (2026-10-16)
Samuel: *"Las animaciones ya están mejor así, ahora la animación de empujar la camilla es solo lateral entonces cuando se mueven abajo o arribala animación no se ve en la dirección correcta, también no se si estoy jugandolo mal pero me estaban bajando las estrellas solas. Los menus de game over Como el de quiebra o el reporte no mantienen la estética de la letra"*.
- ✅ El movimiento cuadriculado queda aprobado ("ya están mejor así").
- ✅ **Empujar hacia arriba y hacia abajo** (hecho 2026-10-17): la hoja del camillero suma dos filas (6: empujar_abajo, 7: empujar_arriba; pasa a 64 × 168). Detalle en §10.3.
- **Por qué bajan las estrellas** (Samuel eligió las tres opciones):
  - **Aviso con el motivo** cada vez que bajan, por ejemplo "−0,5 ★ P-003 lleva 20 s esperando revisión" o "−0,5 ★ P-005 se fue enojado". `game/` expone los cambios de reputación con su motivo como valores.
  - ✅ **Ícono "falta el médico"** (hecho 2026-10-17): sobre el paciente que ya está en la cama pero sin revisar (`AwaitingReview`). Opción A (estetoscopio en un globo, colita hacia el paciente), aprobada por Samuel; ficha en `gui/assets/sprites/ICONS_ASSETS.md`. Parpadea desde los **20 s** de espera (constante de interfaz en `gui/ui/need_doctor_icon.go`, 10 s antes del −0,5 ★ de `firstWaitPenaltyAt`), reusando el parpadeo de la burbuja roja (`bubbleBlinkAfter`, `game_input.go`). Solo en el modo juego: la demo no distingue revisado de sin revisar.
  - **Más margen**: el primer −0,5 ★ llega a los **30 s** de espera en vez de 20 (§5.9; constante en `config.go`). Lo demás queda igual.
- ✅ **Letra** (hecho 2026-10-17): Press Start 2P en **toda** la interfaz, incluidos los rótulos del mapa, los IDs, los tooltips, los avisos, los subtítulos y la barra de la demo, los botones REPORTE y PAUSA, el menú de pausa, la bolsa, y también el reporte del día y el Game Over (en su diseño actual: solo cambió la letra). Reemplaza la regla de §9.3 de usarla solo en la portada, los créditos y el aviso. Como Press Start 2P ocupa su cuadro completo por letra (más ancha que Go Mono con el mismo tamaño en puntos), `smallFace` bajó de 14 a 8 px, `textFace` de 16 a 8 (los textos largos de la demo y del reporte se salían de su caja a 16) y `boldFace` de 20 a 16; `textBox` (`overlays.go`) subió su relleno vertical de 4 a 12 px porque Press Start 2P no tiene descendente y esa caja más baja dejaba demasiado cerca dos puestos del mapa que antes no se tocaban. Tests de ajuste en `gui/ui/text_fit_test.go`.
- ✅ **Pedido del 2026-10-17**: *"cuando contrato gente solo hay 3 espacios donde aparece el personal, entonces aparecen 4 en el mismo lugar"* (los puestos de la sala, §10.7 punto 7) y *"que el personal no esté tan quieto"* (deambular, §10.8 punto 13). Hecho el mismo día.

### 12.9 Pedido de Samuel (2026-10-17): la jugabilidad "un poco aburrida"
Samuel sintió la jugabilidad **"un poco aburrida"** y pidió dos cambios inspirados en Diner Dash, solo en `game/` (lo visual en la barra queda para otro builder):
- ✅ **Horas pico** (§5.3): el intervalo de llegadas varía según la hora del turno (`rushBands` en `config.go`), con dos picos (10:00–12:00 y 15:00–17:30) que lo reducen a la mitad. `Snapshot.RushHour` y el aviso "¡Hora pico! Llegan más pacientes" al entrar a uno.
- ✅ **Racha de atención** (§5.8): revisar rápido (menos de 30 s de espera) sube una racha que multiplica el pago hasta ×2; se rompe con una penalización de espera, al irse alguien enojado o al cerrar el día. `Snapshot.Streak`, `Snapshot.StreakMultiplier` y `DayReport.BestStreak`.
- ✅ **Lo visual en la barra (hecho 2026-10-17)**: "RACHA ×1,5" junto a la plata (solo con racha ≥ 1, con un salto de unos px cuando sube y un parpadeo corto cuando se pierde) y "HORA PICO" parpadeando suave junto al reloj mientras `RushHour` sea cierto, sin encimarse con el DÍA ni la barra del turno; "Mejor racha: N" en el panel de fin de día (`gui/ui/hud.go`, `gui/ui/game_scene.go`).
- ✅ **Sonido (hecho 2026-10-17, §10.11)**: pedido aparte de Samuel el mismo día ("añade sonidos acordes, que la vida está ahí"); `coin` usa la racha para subir el tono de la segunda nota, así la racha también se "siente" en el oído, no solo en la barra.
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
