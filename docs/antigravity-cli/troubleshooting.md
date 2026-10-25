# Resolución de Problemas y Diagnóstico (Troubleshooting)

Esta guía te ayuda a diagnosticar y solucionar rápidamente cualquier inconveniente común con la integración de `agy` y Gentle AI.

---

## 1. Herramientas de Diagnóstico Integradas

### Diagnóstico Completo del Sistema
Ejecutá el comando `doctor` para comprobar el estado de los componentes, rutas de configuración y servidores MCP:

```bash
gentle-ai doctor
```

### Validación Estricta del Plugin
Verificá que el plugin instalado cumpla con las especificaciones del validador de `agy`:

```bash
# Para instalación global:
agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai

# Para instalación en workspace:
agy plugin validate --strict .agents/plugins/gentle-ai
```
Debe retornar exit 0 con cero errores y cero advertencias.

---

## 2. Problemas Comunes y Soluciones

### A. La StatusLine no aparece en la terminal
- **Causa 1**: No se pasó `--include-theme` al instalar.
  - **Solución**: Ejecutá `gentle-ai sync --agents antigravity-cli --scope global --include-theme`.
- **Causa 2**: El binario `gentle-ai` no está en tu `$PATH` del sistema.
  - **Solución**: Comprobá `command -v gentle-ai`. En `settings.json`, `agy` busca el comando en el `$PATH` del entorno de la terminal. Asegurate de que `~/.local/bin` o `/usr/local/bin` formen parte del `$PATH` de tu shell.
- **Causa 3**: El plugin está desactivado en la configuración.
  - **Solución**: Revisá `~/.gemini/config/config.json`. Si `plugins.gentle-ai.enabled` está en `false`, la StatusLine se auto-silencia por diseño. Cambialo a `true`.

### B. El hook de `Stop` bloquea el cierre de la interacción
- **Causa**: Realizaste modificaciones sustanciales de código que califican como de alto riesgo (`review_due`) y aún no fueron revisadas bajo RDD.
  - **Solución**: Ejecutá el preflight de revisión indicado en el mensaje:
    ```bash
    gentle-ai review status --cwd . --contract gentle-ai.review-integration/v2 --agent antigravity-cli --next-transition
    ```
    Y luego ejecutá el comando `review.start` devuelto para completar la revisión.

### C. Conflicto entre `antigravity` (IDE) y `antigravity-cli` (CLI)
- **Causa**: Se intentó sincronizar el adaptador genérico `antigravity` en lugar del específico `antigravity-cli`.
  - **Solución**: Para la terminal CLI, siempre utilizá el nombre formal de agente:
    ```bash
    gentle-ai sync --agents antigravity-cli
    ```
    El identificador `antigravity` sin sufijo `-cli` está reservado exclusivamente para la integración con el IDE gráfico.

### D. En Windows: problemas de resolución de `gentle-ai.exe`
- **Causa**: Go instala binarios en `%USERPROFILE%\go\bin` y este directorio no siempre está en el `%PATH%` de usuario.
  - **Solución**: Agregá `%USERPROFILE%\go\bin` a las variables de entorno de Windows o copiá `gentle-ai.exe` a una carpeta incluida en `%PATH%` (ej. `C:\Windows\System32` o tu carpeta de herramientas).

---

## 3. Restablecimiento Completo (Clean Reinstall)

Si necesitás reiniciar la configuración a un estado limpio:

```bash
# 1. Desinstalar el plugin actual
gentle-ai uninstall --agent antigravity-cli --scope global

# 2. Reinstalar con configuración limpia de tema
gentle-ai install --agent antigravity-cli --scope global --include-theme
gentle-ai sync --agents antigravity-cli --scope global --include-theme

# 3. Comprobar salud
gentle-ai doctor
```
