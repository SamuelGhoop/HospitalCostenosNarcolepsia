# Reporte del día

El tablero de corcho del Shift Report diario (§9, §5.8): el fin del turno de
las 20:00, con el balance del día, la nómina, los contratados y las cuatro
consultas de la Sección 5. Todo en píxeles de pantalla de 1280×720, dibujado
a **1×**. Vista aprobada el 2026-10-19 (Figma, sección "Hospital – vistas",
"Reporte A – día con ganancia" 17:224; "Reporte B – día con pérdida" 17:225,
vacío salvo el mapa: usa el mismo diseño de A, con los negativos en rojo).

## Imágenes

| Archivo | Tamaño | Dónde | Usado |
|---|---|---|---|
| `report_corkboard.png` | 1226×693 | **(32, 16)** | Sí: fondo del panel (`drawDayOverPanel`, `gui/ui/report_panel.go`) |
| `report_cork_tile.png` | 32×32 | — | **No todavía**. El agente de arte lo exportó como una textura de corcho suelta, pero la vista de Figma usa `report_corkboard.png` entero, no un mosaico: se dejó embebido por si hace falta un fondo tileable más adelante (por ejemplo, un tablero de otro tamaño) |
| `report_pin_amarillo.png`, `report_pin_azul.png`, `report_pin_rojo.png`, `report_pin_verde.png` | 8×8 cada una | — | **No todavía**: en Figma cada chincheta ("chincheta roja x3", etc.) quedó como un nodo sin su propia posición (x=0, y=0 en los cuatro), así que no hay dónde ponerla sin inventar el número. Quedan embebidas, listas para usarse el día que Figma las posicione de verdad |

Fuente: Press Start 2P (`drawPixelText`), igual que el resto de la interfaz
desde el 2026-10-17. Tamaños 24 (títulos grandes: "REPORTE DEL DÍA N",
"AL FINAL", "PUNTAJE", "SIGUIENTE DÍA"), 16 (etiquetas y filas normales) y
8 (las cuatro consultas de la Sección 5, más densas). Las tildes y la Ñ con
`accentMarks`, como en el resto.

## Por qué el panel no reusa `panel()` igual que Figma pixel a pixel
Cada "ficha" de Figma es: un rectángulo de contorno (negro), uno de papel
más chico adentro (3 px de margen) y una sombra de 4 px pegada abajo del
papel. El código usa el `panel()` que ya usa el resto del juego (fondo +
contorno de 2 px, sin la sombra de 3D): mismo lenguaje visual que la pausa,
la bolsa de empleo y el Shift Report de mitad de partida, en vez de una
pieza de dibujo nueva solo para esto. Es una simplificación a propósito,
no un olvido.

## Colores "sin estilar" en el Figma aprobado
Dos elementos del frame 17:224 quedaron en el color por defecto de una
figura nueva de Figma (gris `#d9d9d9` de fondo, texto negro puro `#000000`,
no el `#141414` de la paleta): la ficha PUNTAJE y el botón SIGUIENTE DÍA.
Se nota porque es EXACTAMENTE el mismo par de colores que aparece en el
bloque duplicado del Game Over C2 (quiebra), que GAME_DESIGN ya marca como
error de Figma ("17:227 tiene un bloque duplicado al final: ignóralo"). Acá
se dibujaron con el estilo normal del resto de la interfaz: la ficha
PUNTAJE como una ficha de papel más (etiqueta gris, valor en tinta) y el
botón en azul con texto blanco (el mismo azul que ya tenía el código viejo
para "SIGUIENTE DÍA"), en vez de copiar un gris que con toda seguridad nadie
eligió a propósito.

## Qué datos usa cada ficha (`game.DayReport`, `game/report.go`)
Todo sale de `Game.DayReport()` (el Shift Report del último día que cerró)
y del `Snapshot.Score` actual para la ficha PUNTAJE (no el final: ese solo
existe con la partida perdida, ver `gui/assets/gameover/GAMEOVER_ASSETS.md`).

| Ficha | Campos |
|---|---|
| Título | `Day` |
| BALANCE DEL DÍA | `StartMoney`, `ReviewIncome`, `DischargeIncome`, la suma de `Hires[].Cost`, `PayrollTotal`, `EndMoney` |
| NÓMINA | `Payroll[]` (`Name`, `RoleLabel`, `Salary`), `PayrollTotal` |
| CONTRATADOS | `Hires[]` (`Name`, `RoleLabel`, `Cost`) |
| 5.1 PASILLO | `Report.Hallway` |
| 5.2 MÉDICOS | `Report.Episodes` (hasta 2 episodios por médico se muestran enteros; el resto, "+ N MÁS") |
| 5.3 HABITACIONES | `Report.Rooms`, `Report.LastAssignError` |
| 5.4 SEVEROS | `Report.Severe` (con su `Note`: "(ALTA)"/"(SE FUE ENOJADO)") |
| PUNTAJE | `Snapshot.Score` |

`RoleLabel` (en `PayrollLine` y `HireLine`) es nuevo en `game/` (2026-10-19,
TDD): antes solo `CandidateView` tenía el rol con género ("Médico"/"Médica"/
"Camillero"/"Camillera"); ahora la nómina y los contratados del día dicen lo
mismo, no un texto recalculado aparte.

## Listas largas: "+ N MÁS"
Las cuatro consultas de la Sección 5 (que pueden crecer sin límite con la
partida) se recortan a lo que quepa en su ficha (9 renglones de 8 px,
`maxQueryCardLines`, `gui/ui/report_panel.go`): si sobra algo, el último
renglón dice "+ N MÁS" en vez de cortar un registro a la mitad (`fitBlocks`).
NÓMINA y CONTRATADOS no se recortan: están acotadas por las reglas del
juego (`maxStaff` = 6, `candidatesPerDay` = 3), así que siempre caben
enteras en su ficha.
