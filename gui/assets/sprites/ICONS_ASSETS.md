# Íconos sobre el mapa

## `icon_need_doctor.png` — "falta el médico" (GAME_DESIGN §12.8)
- Fuente: `gui/art/icon_need_doctor_a.aseprite` (capa `icono`, tag `flotar`). Aprobado por Samuel el 2026-10-17 (opción A, estetoscopio, con la colita apuntando al paciente).
- Hoja horizontal de 38 × 18: **2 cuadros de 19 × 18**, cada uno de **0,4 s**, en loop (el estetoscopio rebota 1 px: cuadro 1 abajo, cuadro 2 arriba).
- Se dibuja a ×3 (57 × 54 en 1280 × 720), con nearest.
- Posición: esquina de arriba a la izquierda del cuadro en **(+169, +64)** desde la esquina de arriba a la izquierda de la habitación (habitación 101 en (15, 40), y cada habitación siguiente 273 px a la derecha: (184, 104), (457, 104), (730, 104)). Queda a la derecha de la cama, debajo del ID; la punta de la colita (columna 0) apunta arriba a la izquierda, hacia la cabeza del paciente.
- Se muestra mientras el paciente está en la cama sin revisar. **Parpadea desde los 20 s de espera** (lo implementa el builder en código; el parpadeo no está dibujado).
