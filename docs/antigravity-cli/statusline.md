# StatusLine y Experiencia de Terminal

Gentle AI provee una barra de estado dinámica y de alto rendimiento renderizada directamente en Go in-process (`gentle-ai statusline`) y un título de ventana (`gentle-ai statusline --title`) diseñado específicamente para la interfaz de `agy`.

---

## 1. Activación Opcional (`--include-theme`)

La StatusLine no se activa automáticamente para no modificar tu configuración sin tu consentimiento. Si deseás activarla, podés pasar la bandera opcional `--include-theme` durante la sincronización o instalación:

```bash
gentle-ai sync --agents antigravity-cli --scope global --include-theme
```

Esto registrará los comandos en `~/.gemini/antigravity-cli/settings.json`:
```json
{
  "statusLine": { "type": "command", "command": "gentle-ai statusline" },
  "title": { "type": "command", "command": "gentle-ai statusline --title" }
}
```

---

## 2. Anatomía de la StatusLine

La línea de estado muestra información contextual en tiempo real con una jerarquía visual limpia:

![StatusLine en agy](../assets/features/statusline-agy.png)

```text
✿ accept-edits · gemini-3.8-flash ▰▰▰▱▱▱▱▱ 38% 4h12m ▰▰▰▱▱ 60% $0.14 🧠 gentle-ai · 24 🌹 ✓ Approved ⋗ 3/5 · T2 in_progress ±2 +14/−3                   ? for shortcuts
```

### Segmentos y su Significado

1. **Pétalo Estático (`✿`)**: Flor estática en color rosa Gentleman (`ansiRose + bold`). Se mantiene sobria y sin animaciones que distraigan.
2. **Modelo y Nivel de Esfuerzo (`accept-edits · Gemini 3.8 Flash · high`)**: Modelo reportado por `agy` junto a su nivel de thinking/effort (`high`, `low`, etc.) o modo de ciclo de vida (`plan`, `accept-edits`, `auto`).
3. **Gauge de Contexto (`ctx ▰▰▱▱▱ 17%`)**: Barra indicando el consumo de la ventana de contexto. Cambia de color dinámicamente (<80% sobrio, ≥80% amarillo de advertencia, ≥95% rojo crítico).
4. **Ventanas de Cuota (`3h ▰▱▱▱▱ 5% ◈ 5d ▰▰▰▱▱ 57%`)**: Countdown en tiempo real hasta el próximo reinicio de cuota del modelo activo (por horas y días), junto al porcentaje consumido.
5. **Costo de Sesión (`$0.14`)**: Se muestra **exclusivamente cuando es mayor a cero** (`$X.YZ`), evitando ruidos innecesarios en cuentas de cuota por suscripción o sesiones recién iniciadas.
6. **Memoria Engram (`🧠 gentle-ai · ready`)**: Proyecto activo de memoria persistente Engram y su estado dinámico (`ready` o `✓ saved #id` al registrar observaciones).
7. **Estado de Revisión RDD (`🌹 off` / `🌹 ✓ Approved` / `🌹 Review in progress`)**: Estado en vivo del motor de Receipt-Driven Development (`off`/`on` en reposo, o veredicto de la revisión activa).
8. **Progreso ODD (`⋗ 3/5 · T2 in_progress`)**: Tareas completadas vs totales según el archivo de seguimiento en `odd/tasks/*.md` (se oculta automáticamente cuando no hay tareas activas).
9. **Cambios en Git (`±2 +13/-70`)**: Archivos modificados, inserciones y eliminaciones en el repositorio actual (con caché eficiente de 5 s).
10. **Atajos a la Derecha (`? for shortcuts`)**: Recordatorio interactivo alineado al margen derecho para consultar atajos de teclado sin ocupar espacio central.

---

## 3. Comportamientos Inteligentes y Silenciado

- **Omisión de Ruido Redundante**: Los jobs encolados y subagentes se omiten de la StatusLine, ya que `agy` los muestra de forma nativa en su propia interfaz superior.
- **Auto-silenciado Automático**: Si el plugin `gentle-ai` está desactivado en la configuración (`plugins.gentle-ai.enabled: false` en `~/.gemini/config/config.json`), la StatusLine **se auto-silencia por completo** devolviendo una cadena vacía (`""`).
- **Resiliencia ante Fallos**: Si el payload JSON recibido desde stdin es nulo, está roto o la terminal no soporta escape ANSI, degrada a una flor básica `✿` con código de salida 0 sin romper el prompt del usuario.

---

## 4. Personalización del Usuario

Podés configurar la apariencia y visibilidad creando o editando `~/.gentle-ai/statusline.json` (Gentle AI nunca sobreescribe ni poda este archivo):

```json
{
  "schema": "gentle-ai.statusline/v1",
  "density": "comfortable",
  "visibility": {
    "changes": true,
    "rdd": true,
    "usageCost": true,
    "modelDetails": true,
    "contextWarning": true,
    "engram": true,
    "tasklist": true,
    "shortcuts": true
  }
}
```

### Opciones de Densidad (`density`)
- `comfortable` (predeterminado): Formato completo con gauges de 8 celdas y todos los indicadores.
- `compact`: Gauges de 6 celdas y etiquetas abreviadas para pantallas angostas.
- `minimal`: Muestra únicamente flor, modelo, tasklist ODD, estado RDD y memoria Engram.
