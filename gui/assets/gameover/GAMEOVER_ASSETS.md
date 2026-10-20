# Game Over

La pantalla de derrota (§5.10, §9): motivo, ícono, días sobrevividos y el
desglose del puntaje final. Todo en píxeles de pantalla de 1280×720,
dibujado a **1×**. Vista aprobada el 2026-10-19 (Figma, sección
"Hospital – vistas", "Game Over C1 – perdió la licencia" 17:226, "C2 –
quiebra" 17:227, "C3 – hospital colapsado" 18:348).

## Imágenes

| Archivo | Tamaño | Dónde | Usado por |
|---|---|---|---|
| `gameover_icon_licencia.png` | 32×32 (×3 en pantalla = 96×96) | **(592, 196)** | `game.LostLicense` |
| `gameover_icon_quiebra.png` | 32×32 (×3 = 96×96) | **(592, 196)** | `game.Bankrupt` |
| `gameover_icon_colapsado.png` | 32×32 (×3 = 96×96) | **(592, 196)** | `game.HospitalCollapsed` |

El portapapeles de fondo **no tiene un PNG propio**: la capa de Figma de
las tres vistas se llama "portapapeles (hiring_board.png 730×556)" — el
mismo archivo de la bolsa de empleo (`gui/assets/hiring/hiring_board.png`,
`gui/assets/hiring/HIRING_ASSETS.md`), en la misma posición (280, 81). El
código lo reusa de `hiringImgs["hiring_board.png"]` (ya cargado por
`loadHiring`, `gui/ui/hiring_scene.go`): no hace falta embeberlo de nuevo.

Fuente: Press Start 2P (`drawPixelText`). Tamaños 24 (el sello del motivo,
"PUNTAJE FINAL" y los botones), 16 (el resto: encabezado, días
sobrevividos, puntaje, el desglose). Las tildes y la Ñ con `accentMarks`.

## Colores del frame C2 (quiebra): se usan los de C1/C3, no los de Figma
De la ficha PUNTAJE para abajo, el frame 17:227 (quiebra) tiene todo en gris
plano (`#d9d9d9`) y texto negro puro (`#000000`): el mismo color "recién
creado, sin estilo" de Figma que ya aparece en el bloque duplicado que
GAME_DESIGN marca como error de ese mismo frame ("17:227 tiene un bloque
duplicado al final: ignóralo"). Los frames C1 (licencia) y C3 (colapsado),
que SÍ están bien estilados, coinciden entre sí: etiquetas en gris
(`colMuted`), el signo del botón NUEVA PARTIDA en verde (`colGreen`/
`colGreenDark` de sombra) y MENÚ en azul (`colBlue`/`colBlueDark`). El
código usa esos mismos colores para los tres motivos, incluida la quiebra:
no tendría sentido que justo el motivo de la plata se vea distinto por un
olvido de estilo en Figma, no por una decisión de diseño.

## Qué datos usa cada parte (`game.Snapshot`, `game/snapshot.go`)

| Parte | Campo / función |
|---|---|
| Ícono | `Snapshot.GameOver` (`gameOverIconName`, `gui/ui/gameover_panel.go`) |
| "DÍA N" | `Snapshot.Day` |
| Sello (motivo) | `Snapshot.GameOver.String()` (ya trae el texto exacto: "¡PERDIÓ LA LICENCIA!", "¡QUIEBRA!", "¡HOSPITAL COLAPSADO!") |
| Renglón del motivo | `gameOverReason(over, money)`: para `Bankrupt` usa `Snapshot.Money` (la plata real, negativa); para `HospitalCollapsed` usa `game.CollapseLimit` (nuevo, exportada en `game/config.go`: el mismo 5 de la regla real, no un número repetido a mano); para `LostLicense` es fijo, siempre pasa por la misma razón |
| DÍAS SOBREVIVIDOS | `daysSurvived(Snapshot.Day, Snapshot.GameOver)`: la quiebra se dispara en la nómina de las 20:00 (ese día sí terminó completo), pero la reputación en 0 y el colapso pueden pasar a mitad del día (ese no cuenta como sobrevivido) |
| PUNTAJE | `Snapshot.Score` (el acumulado, sin el desglose final) |
| PLATA ÷ 10 | `Snapshot.MoneyScore` (nuevo en `game/`, TDD: antes solo existía la suma dentro de `FinalScore`, no esta parte sola) |
| ESTRELLAS × 200 | `Snapshot.StarScore` (igual, nuevo) |
| PUNTAJE FINAL | `Snapshot.FinalScore` (ya existía) |

`Snapshot.MoneyScore`/`StarScore` quedan en 0 mientras `GameOver ==
NotOver`, igual que `FinalScore` (no se dibujan nunca en ese caso: el panel
solo se muestra con la partida perdida).

## El renglón del motivo puede partirse en dos
La vista aprobada solo muestra el caso de una plata modesta ("-$ 340"), a
una línea. Con una quiebra de números grandes el texto no entra en los
612 px de la columna: se parte en hasta dos renglones con `wrapTwoLines`
(igual que los subtítulos de la demo), cada uno centrado por su cuenta.
