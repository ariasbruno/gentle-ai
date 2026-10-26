# Resolución de Problemas y Diagnóstico (Troubleshooting)

Esta guía documenta problemas reales, causas raíz empíricas y soluciones verificadas para la integración de Antigravity CLI (`agy`) con Gentle AI.

---

## 1. Herramientas de Diagnóstico Integradas

### Diagnóstico General de Gentle AI
Comprueba dependencias del sistema (`git`, `curl`, `node`, `go`), espacio en disco y conectividad con la memoria persistente Engram:
```bash
gentle-ai doctor
```

### Validación Estricta del Plugin de `agy`
Verifica manifiestos, esquemas, los 10 subagentes, reglas (límite de 24.000 bytes) y hooks sin errores:
- **Instalación Global:**
  ```bash
  agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai
  ```
- **Instalación en Workspace:**
  ```bash
  agy plugin validate --strict .agents/plugins/gentle-ai
  ```
Ambos comandos deben retornar exit code 0 (`0 errors, 0 warnings`).

### Comprobación de la StatusLine Directa en Consola
Podés probar la respuesta del renderizador de statusline directamente fuera de `agy`:
```bash
echo '{}' | gentle-ai statusline
```
O el título de ventana:
```bash
echo '{}' | gentle-ai statusline --title
```

---

## 2. Problemas Reales y Soluciones Verificadas

### A. La StatusLine no aparece en la terminal de `agy`

- **Causa 1: La terminal no refrescó la barra al iniciar sesión.**
  - **Solución:** Dentro del prompt de `agy`, ejecutá el comando slash `/statusline` para forzar la inicialización y recarga de la barra.

- **Causa 2: No se configuró el bloque de tema en `settings.json`.**
  - **Explicación:** Para no mutar configuraciones sin tu consentimiento, la instalación no inyecta la statusline a menos que uses `--include-theme`.
  - **Solución:** Reconfigurá el tema ejecutando:
    ```bash
    gentle-ai sync --agents antigravity-cli --scope global --include-theme
    ```
    Y comprobá que `~/.gemini/antigravity-cli/settings.json` contenga:
    ```json
    {
      "statusLine": { "type": "command", "command": "gentle-ai statusline" },
      "title": { "type": "command", "command": "gentle-ai statusline --title" }
    }
    ```

- **Causa 3: El ejecutable `gentle-ai` no está disponible en el `$PATH` de la terminal.**
  - **Explicación:** Go compila los binarios en `~/go/bin` (o `$GOPATH/bin`). Si esa ruta no está exportada en el entorno interactivo de tu shell, `agy` no podrá encontrar el comando.
  - **Solución:**
    - En Linux / macOS: Agregá a tu `~/.bashrc` o `~/.zshrc`:
      ```bash
      export PATH="$HOME/go/bin:$PATH"
      ```
    - En Windows: Agregá `%USERPROFILE%\go\bin` a la variable de entorno `PATH` de tu usuario.
    - Verificá la disponibilidad con `command -v gentle-ai` (o `where gentle-ai` en Windows).

---

### B. La validación falla con `agy plugin validate --strict`
- **Causa 1: Se validó la ruta incorrecta según el alcance de instalación.**
  - **Explicación:** Si instalaste con `--scope workspace`, el plugin reside en `.agents/plugins/gentle-ai` y la ruta global `~/.gemini/config/plugins/gentle-ai` no existirá o estará desactualizada.
  - **Solución:**
    - Si instalaste con `--scope global`: `agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai`
    - Si instalaste con `--scope workspace`: `agy plugin validate --strict .agents/plugins/gentle-ai`

- **Causa 2: Una regla en `rules/` excede el límite de tamaño de Antigravity CLI.**
  - **Explicación:** `agy` impone un límite estricto de 24.000 bytes por archivo de regla. Si modificaste manualmente alguna regla y superó ese tope, el validador estricto fallará.
  - **Solución:** Ejecutá `gentle-ai sync --agents antigravity-cli` para restaurar los assets.

---

### C. La revisión RDD en modo headless falla o no emite veredicto ("produced no final message")
- **Causa: El modelo configurado en el rol de revisión intenta ejecutar herramientas interactivas.**
  - **Explicación:** Durante el ciclo de revisión automatizada (RDD con lentes 4R), los subagentes revisores corren en un sandbox estricto de solo lectura. Algunos modelos (como `gemini-3.8-flash` con esfuerzo `high`) intentan generar llamadas a herramientas de ejecución que el sandbox bloquea.
  - **Solución:** Configurá explícitamente modelos verificados para roles de revisión creando o editando `~/.gemini/config/plugins/gentle-ai/review-models.json`:
    ```json
    {
      "review-risk": "gemini-3.8-flash-low",
      "review-readability": "gemini-3.8-flash-low",
      "review-reliability": "gemini-3.8-flash-low",
      "review-resilience": "gemini-3.8-flash-low",
      "review-refuter": "gemini-3.8-flash-low",
      "review-validator": "claude-sonnet-4-6"
    }
    ```
    Para más detalles sobre la sintaxis de modelos y esfuerzo, consultá la [Guía de Transporte RDD (05-review-transport.md)](05-review-transport.md#modelos-de-revisión-y-propiedad-global).

---

### D. Colisiones, skills duplicadas o prompts redundantes (Conflicto con Upstream)
- **Causa: Coexistencia del adaptador genérico `antigravity` de upstream.**
  - **Explicación:** Si anteriormente utilizaste la versión oficial de Gentle AI, quedó instalado el adaptador genérico `antigravity`, el cual escribe directivas conflictivas en `.gemini/` y `GEMINI.md`.
  - **Solución:** Desinstalá el adaptador genérico viejo y asegurá el nuevo:
    ```bash
    # 1. Desinstalar el adaptador genérico viejo:
    gentle-ai uninstall --agent antigravity

    # 2. Re-sincronizar el adaptador dedicado:
    gentle-ai sync --agents antigravity-cli --scope global
    ```

---

## 3. Reinstalación Limpia (Clean Rollback & Reinstall)

Si necesitás reiniciar toda la integración a un estado completamente nuevo y limpio:

```bash
# 1. Desinstalar la integración (limpia plugins, manifiestos y settings.json):
gentle-ai uninstall --agent antigravity-cli

# 2. Reinstalar el adaptador de forma global con la StatusLine:
gentle-ai install --agent antigravity-cli --scope global --include-theme

# 3. Comprobar la validez del plugin:
agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai

# 4. Verificar salud del sistema:
gentle-ai doctor
```

---

## 4. Enlaces de Referencia

- [**README Principal**](README.md)
- [**Guía de Instalación (installation.md)**](installation.md)
- [**StatusLine y Experiencia de Terminal (statusline.md)**](statusline.md)
- [**Transporte RDD In-Process (05-review-transport.md)**](05-review-transport.md): Arquitectura de revisión sandboxed y configuración de `review-models.json`.
