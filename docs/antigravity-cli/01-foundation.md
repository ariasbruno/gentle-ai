# 1. Fundamentos, rutas y puesta en marcha

**Esta integración es completamente independiente del adaptador para Antigravity IDE: se enfoca exclusivamente en el ejecutable de terminal `agy` (Antigravity CLI).** Configura `agy` mediante su arquitectura nativa de plugins, hooks y MCP, operando sobre un ejecutable ya instalado sin requerir extensiones gráficas.

[Índice](README.md) · [Arquitectura](SUMMARY.md) · [Fuentes](10-source-index.md)

## Identidad y detección del agente

El adaptador [`antigravitycli`](../../internal/agents/antigravitycli/adapter.go) implementa el soporte para `agy` (Antigravity CLI) bajo el identificador `antigravity-cli` (`model.AgentAntigravityCLI`):

- **Detección del binario:** Busca `agy` en las rutas del sistema mediante `exec.LookPath("agy")`.
- **Detección de configuración:** Comprueba la existencia del directorio de configuración global en `~/.gemini/antigravity-cli/`.
- **Nivel de soporte:** Declara `model.TierFull`, habilitando el despliegue del conjunto completo de componentes (reglas, subagentes nativos, MCP, skills y revisión RDD).
- **Pruebas unitarias:** Los escenarios de detección y resolución de rutas están cubiertos en [`adapter_test.go`](../../internal/agents/antigravitycli/adapter_test.go).

## Tres clases de rutas

### 1. Configuración de aplicación

```text
~/.gemini/antigravity-cli/
└── settings.json                    # configuración de usuario (no inyecta overlay estático)
```

`GlobalConfigDir` y `SettingsPath`: [adapter.go, líneas 206–225](../../internal/agents/antigravitycli/adapter.go#L206-L225). La ruta de modelos se ubica dentro del plugin (`~/.gemini/config/plugins/gentle-ai/review-models.json`) como configuración opcional del usuario: [`AntigravityReviewModelsPath`](../../internal/reviewerprovider/antigravitycli_routing.go#L75-L83).

### 2. Árbol canónico escrito por Gentle AI

```text
~/.gemini/config/
├── import_manifest.json
└── plugins/gentle-ai/
    ├── plugin.json
    ├── review-models.json           # asignaciones de modelos de revisión (opcional, configurable por el usuario)
    ├── mcp_config.json
    ├── hooks.json
    ├── hooks/{hook.sh,hook.cmd}
    ├── rules/
    │   ├── AGENTS.md
    │   ├── gentle-ai-routing.md
    │   ├── gentle-ai-orchestrator.md
    │   ├── orchestrator-delegation.md
    │   ├── orchestrator-delegation-reference.md
    │   ├── orchestrator-memory.md
    │   ├── orchestrator-skills.md
    │   ├── orchestrator-prompts.md
    │   ├── orchestrator-tracking.md
    │   ├── orchestrator-verification.md
    │   ├── orchestrator-writer.md
    │   ├── strict-tdd.md
    │   └── strict-tdd-verify.md
    ├── agents/<nombre>/agent.md
    ├── skills/<directorio>/...
    ├── chains/4r-review.chain.md
    └── .native-rule-ownership.json
```

Las rutas y mappings pertenecen a [`PluginDir`, `BundleAssets`, `SubAgentsDir`](../../internal/agents/antigravitycli/adapter.go#L202-L297). `AGENTS.md`, routing y MCP reciben contenido por componentes separados; no todo el árbol es una copia literal del asset: [finalización de reglas](03-orchestration-and-hooks.md#reglas-y-carga-de-contexto).

### 3. Rutas de plugins según la especificación de agy

`agy` reconoce dos ubicaciones oficiales para plugins:
- **Global:** `~/.gemini/config/plugins/<nombre>` (gestionado con registro en `~/.gemini/config/import_manifest.json`).
- **Por proyecto (Workspace):** `<workspace>/.agents/plugins/<nombre>` (descubierto automáticamente por `agy` al ejecutarse en el proyecto).

## Puesta en marcha (Global y Workspace)

Gentle AI soporta instalación y sincronización en ambos alcances:
- **Global (`--scope global`)**: Despliega el plugin en `~/.gemini/config/plugins/gentle-ai` y lo registra en el manifiesto global para que esté disponible en cualquier sesión de `agy`.
- **Workspace (`--scope workspace`)**: Despliega el plugin en `<workspace>/.agents/plugins/gentle-ai/`, aislado al repositorio actual. `agy` lo descubre automáticamente sin tocar la configuración de usuario ni mutar `HOME`.

### Antes de autorizar escrituras

1. Confirmar que el binario `gentle-ai` corresponde al módulo v4: [go.mod](../../go.mod#L1-L3).
2. Confirmar binario y disponibilidad de `agy` según [Headless](https://antigravity.google/docs/cli/headless/). La documentación oficial exige credenciales interactivas previas; esta guía no automatiza login.
3. Decidir el alcance deseado (`--scope global` para todas las carpetas o `--scope workspace` para aislarlo al proyecto actual).
4. Guardar personalizaciones propias fuera del árbol del plugin administrado, ya que el sincronizador poda directorios ajenos al bundle.

### Comandos de instalación y sincronización

Con ambos binarios ya disponibles:

```sh
# Instalación global (opcionalmente con --include-theme para configurar StatusLine en settings.json)
gentle-ai install --agent antigravity-cli --scope global --include-theme
gentle-ai sync --agents antigravity-cli --scope global --include-theme

# Instalación en workspace (aislada a este repositorio)
gentle-ai install --agent antigravity-cli --scope workspace
gentle-ai sync --agents antigravity-cli --scope workspace
```

Las opciones `--agent`/`--agents`, `--scope` y `--include-theme` están registradas por [`ParseInstallFlags`](../../internal/cli/install.go#L72-L135) y [`ParseSyncFlags`](../../internal/cli/sync.go#L176-L210). La opción `--include-theme` automatiza la configuración de `statusLine` y `title` en `settings.json` mediante [`EnsureStatusLineSettings`](../../internal/agents/antigravitycli/statusline_settings.go). Si no se pasa `--include-theme`, `settings.json` se preserva intacto sin mutaciones automáticas. La resolución de alcances está definida en [`ResolveInstallScope`](../../internal/cli/scope.go#L24-L44). Los comandos configuran el entorno para el agente, pero no instalan el binario `agy`: el adaptador no cuenta con autoinstalación de ejecutables externos.

En alcance global, la secuencia del paso final es `DeployPluginTreeToDir → ensureAntigravityOrchestratorRuleForScope → EnsurePluginImportedToDir`: [`antigravityCLIPluginImportStep.Run`](../../internal/cli/run.go#L1042-L1073). `EnsureStatusLineSettings` no se ejecuta durante install/sync para preservar el `settings.json` del usuario sin mutaciones automáticas. En alcance workspace, se despliega el árbol en `<workspace>/.agents/plugins/gentle-ai/` y se inicializa la regla de orquestador localmente, prescindiendo del registro en el manifiesto global y del registro de status line.

### Estructura desplegada en cada alcance

Ambos alcances despliegan el bundle completo de assets (13 reglas bajo `rules/`):

```text
# En --scope global:
~/.gemini/config/
├── import_manifest.json
└── plugins/gentle-ai/
    ├── plugin.json
    ├── review-models.json           # asignaciones de modelos de revisión (opcional, configurable por el usuario)
    ├── mcp_config.json
    ├── hooks.json
    ├── hooks/{hook.sh,hook.cmd}
    ├── rules/
    │   ├── AGENTS.md
    │   ├── gentle-ai-routing.md
    │   ├── gentle-ai-orchestrator.md
    │   ├── orchestrator-delegation.md
    │   ├── orchestrator-delegation-reference.md
    │   ├── orchestrator-memory.md
    │   ├── orchestrator-skills.md
    │   ├── orchestrator-prompts.md
    │   ├── orchestrator-tracking.md
    │   ├── orchestrator-verification.md
    │   ├── orchestrator-writer.md
    │   ├── strict-tdd.md
    │   └── strict-tdd-verify.md
    ├── agents/<nombre>/agent.md
    ├── skills/<directorio>/...
    ├── chains/4r-review.chain.md
    └── .native-rule-ownership.json

# En --scope workspace:
<workspace>/.agents/plugins/gentle-ai/
├── plugin.json
├── review-models.json           # asignaciones de modelos de revisión (opcional, configurable por el usuario)
├── mcp_config.json
├── hooks.json
├── hooks/{hook.sh,hook.cmd}
├── rules/
│   ├── AGENTS.md
│   ├── gentle-ai-routing.md
│   ├── gentle-ai-orchestrator.md
│   ├── orchestrator-delegation.md
│   ├── orchestrator-delegation-reference.md
│   ├── orchestrator-memory.md
│   ├── orchestrator-skills.md
│   ├── orchestrator-prompts.md
│   ├── orchestrator-tracking.md
│   ├── orchestrator-verification.md
│   ├── orchestrator-writer.md
│   ├── strict-tdd.md
│   └── strict-tdd-verify.md
├── agents/<nombre>/agent.md
├── skills/<directorio>/...
├── chains/4r-review.chain.md
└── .native-rule-ownership.json
```

### Particularidades por alcance:
1. **Descubrimiento automático en workspace:** `agy` descubre nativamente los plugins situados en `.agents/plugins/<nombre>/`, por lo que el instalador no modifica `~/.gemini/config/import_manifest.json`.
2. **Aislamiento de `HOME`:** En `--scope workspace`, el directorio de usuario (`HOME`) permanece completamente inalterado.
3. **Precedencia:** Cuando existen ambos, las reglas y configuraciones del workspace tienen prioridad sobre las globales.
4. **Status line y settings.json:** El subcomando de status line se ejecuta in-process; Gentle AI preserva `settings.json` sin modificarlo en sync o install. La configuración de usuario en `statusLine`/`title` puede gestionarse manualmente o vía comandos dedicados.

### Verificación automatizada tras la instalación o sincronización

No hace falta inspeccionar carpetas a mano. Podés validar la instalación directamente con las herramientas de línea de comandos:

1. **Verificar que `agy` cargó el plugin y sus componentes:**
   ```bash
   agy plugins list
   ```
   Debe responder confirmando el import de `gentle-ai` con sus 4 componentes activos:
   ```json
   {
     "imports": [
       {
         "name": "gentle-ai",
         "source": "antigravity",
         "components": [
           "skills",
           "agents",
           "mcpServers",
           "hooks"
         ]
       }
     ]
   }
   ```

2. **Verificar la salud del entorno y servicios MCP:**
   ```bash
   gentle-ai doctor
   ```
   Comprueba que el agente `antigravity-cli` esté registrado, el archivo de estado sea consistente y el servidor MCP de memoria (Engram) responda en su canal stdio.

## Permisos: guardias dinámicas in-process sin mutar settings.json

A diferencia de otros agentes, Gentle AI **no inyecta overlays estáticos en `settings.json` para `agy`** ([`internal/components/permissions/inject.go#L172-L176`](../../internal/components/permissions/inject.go#L172-L176) devuelve `nil`). Esto preserva intacta la configuración del usuario y evita dependencias de mutación de archivos de configuración globales.

En su lugar, toda la seguridad preventiva se evalúa **dinámicamente in-process** en cada llamada a través del hook `PreToolUse` ([`handleAntigravityPreToolUse`](../../internal/cli/antigravity_hook.go#L921-L1017)):

- **Bloqueos preventivos de rutas sensibles (`deny`):** Las tools de archivo y búsqueda (`view_file`, `read_file`, `write_file`, `edit_file`, `replace_file_content`, `multi_replace_file_content`, `grep_search`) y comandos de terminal (`run_command`) que referencien secretos (`.env*` a cualquier profundidad, `.ssh`, `.credentials`, `.aws`, tokens de GitHub en `hosts.yml`, certificados `.pem`/`.key`, directorios `secrets/` o Keychains) son bloqueados de inmediato con `deny` antes de ejecutarse.
- **Circuit breaker anti-loops (`deny`):** Tres llamadas consecutivas idénticas de una herramienta con los mismos argumentos en una conversación devuelven `deny`.
- **Comandos destructivos (`force_ask`):** Comandos shell potencialmente peligrosos (`rm -rf`, `sudo rm -rf`, `git reset --hard`, `git push --force`, `git clean -fd`, `chmod -R 777`, `chown -R`) fuerzan una confirmación interactiva en pantalla del usuario (`force_ask`), ignorando cachés previas.
- **Comandos estándar de terminal (`ask`):** Delegan la decisión en el motor nativo de permisos de `agy`.
- **Operaciones de archivo estándar (`allow`):** Operaciones dentro del workspace del proyecto devuelven `allow` (comportamiento estándar del sandbox).

Esta decisión de diseño está respaldada por tests unitarios explícitos: [`TestInjectAntigravityCLISkipsSettingsOverlay`](../../internal/components/permissions/inject_test.go#L753) y [`TestInjectAntigravityCLIPreservesUserSettingsWithoutInjectingOverlay`](../../internal/components/permissions/inject_test.go#L1131), además de [`TestInjectAntigravityCLIDoesNotMutateSettings`](../../internal/components/permissions/antigravity_cli_allow_test.go#L15).

## Status line personalizado: renderizado in-process en Go

`agy` renderiza una status line propia ejecutando el comando del bloque `statusLine` de `~/.gemini/antigravity-cli/settings.json`, al que le pasa el payload de estado JSON por stdin en cada cambio de estado del agente (ANSI completo, multilínea). El usuario puede registrar estos bloques en su `settings.json` apuntando al subcomando in-process [`gentle-ai statusline`](../../internal/cli/statusline.go) (o mediante el helper [`EnsureStatusLineSettings`](../../internal/agents/antigravitycli/statusline_settings.go)):

```json
{
  "statusLine": { "type": "command", "command": "gentle-ai statusline" },
  "title": { "type": "command", "command": "gentle-ai statusline --title" }
}
```

La línea sigue el orden y las prioridades fijadas con el usuario: **pétalo estático** (flor estática `✿` en rosa Gentleman `ansiRose + ansiBold + "✿"` en todos los estados del agente — sin animación y tiempo-independiente) + `modo · modelo` mostrado **verbatim cuando agy lo reporta** (`cycle_mode` en vivo: `plan`, `accept-edits`, `auto` donde la cuenta lo permite — un modo aparte con gating de cuenta; el `execution_mode` de los docs canónicos queda como fallback) y **en silencio en el modo default** — el ausente nunca recibe un label inventado, separador `⟡`, gauge de contexto `▰▱` de 8 celdas (acento <80 / warning ≥80 / error ≥95, marcador `!` ≥90), costo de sesión mostrado **exclusivamente cuando es mayor a cero** (`$X.YZ`), ventanas de quota **solo de la familia del modelo activo** en formato countdown — `4h32m ▰▰▱▱ 38%`: tiempo hasta el reset, barra y % usado (sin countdown cae al label derivado de la key) —, Engram `🧠 proyecto · count`, RDD con **estados de review** (`🌹 Review in progress`/`✓ Approved`/`Fixing findings…`/…; sin autoridad activa muestra `on`/`off` del switch), tasklist ODD (`⋗ 12/17 · T15 in_progress`, `⚠N` con bloqueadas, `⋗ 17/17 ✓` completa; lee `odd/tasks/*.md` del doc más reciente por mtime), cambios git `±N +A/−D` (visible por defecto, subprocess con cache de 5 s por cwd; oculto en repo limpio o no-git) y atajos a la derecha (`? for shortcuts`). Los segmentos de jobs y subagentes se omiten intencionalmente para evitar saturación visual, ya que `agy` los reporta de forma nativa en su interfaz. Si el plugin está desactivado en la configuración de agy (`plugins.gentle-ai.enabled: false`), la StatusLine se **auto-silencia por completo** retornando cadena vacía (`""`). Un payload ausente o roto degrada a `✿` sola con exit 0. El modo `--title` alimenta el título de la terminal.

**Fase ODD (hook PreToolUse):** cada tool call actualiza `~/.gentle-ai/statusline-phase/<hash-conversación>.json` con la fase inferida (reads → `exploring`, edits → `implementing`, corridas de test → `checking`, dispatch de subagentes → `delegating`, resto → `working`), best-effort y sin afectar la decisión del hook; los registros se podan a la hora. El archivo queda como infraestructura (útil para usos futuros): desde la reordenación de la línea el statusline no muestra la fase — el pétalo es solo el pétalo.

**Reglas de propiedad:** Gentle AI administra solo `type`+`command` cuando el `command` está vacío, ya es el subcomando administrado, o es el registro legacy que apuntaba al script viejo; un comando ajeno deja el bloque intacto. Las llaves `enabled`, `padding` y `stack_with_default` son tuyas y se preservan tal cual (`stack_with_default: true` renderiza la línea debajo de la built-in de agy). Un `settings.json` ausente se crea; uno malformado falla cerrado sin modificarse; una configuración ya administrada no se reescribe. `gentle-ai uninstall` retira los bloques cuando apuntan al subcomando o al registro legacy.

**Configuración de usuario** en `~/.gentle-ai/statusline.json` (archivo tuyo; sync nunca lo toca ni lo poda):

```json
{
  "schema": "gentle-ai.statusline/v1",
  "density": "comfortable",
  "visibility": {
    "changes": true,
    "rdd": true,
    "agents": true,
    "usageCost": true,
    "modelDetails": true,
    "contextWarning": true,
    "engram": true,
    "tasklist": true,
    "shortcuts": true
  }
}
```

`density` acepta `comfortable`/`compact`/`minimal` (minimal oculta la marca en reposo). `visibility` oculta segmentos: `modelDetails` (modelo), `usageCost` (contexto+costo+quota), `agents` (jobs+subagentes), `rdd` (interruptor de revisión), `contextWarning` (marcador `!`), `engram` (observaciones de Engram), `tasklist` (avance ODD) y `shortcuts` (atajos de teclado). `changes: true` habilita el segmento de cambios (`±N +A/−D` vía git con deadline de 500 ms). Valores ausentes o archivo malformado → defaults. El interruptor RDD se lee in-process con el mismo resolver read-only del comando `review mode` (deadline 500 ms); cualquier falla muestra `🌹 ?` y nunca bloquea la línea.

**Sin dependencias:** el render es in-process en el binario `gentle-ai` que los hooks ya requieren — no hace falta `jq` ni bash, y funciona igual en Windows. Un payload ausente o ilegible degrada a la línea `✿` sin error.

## CLI frente a IDE

Gentle AI ofrece dos integraciones distintas para el ecosistema de Antigravity:
- **`agy`:** Despliega el plugin completo con reglas, subagentes nativos, MCP servers y revisión RDD vía subproceso Go.
- **Antigravity IDE:** Configura directivas a través de archivos de contexto como `GEMINI.md` en el workspace.

Para una comparación detallada entre ambos enfoques, consultá [El Ecosistema Antigravity: Genérico frente a Dedicado](generic-vs-dedicated.md).

---

**Siguiente paso:** [Cómo Gentle Shell se convierte en assets agy](02-asset-transpiler.md).
