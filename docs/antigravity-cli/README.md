# Google Antigravity CLI (`agy`) con Gentle AI

Integración nativa de primer nivel de **Google Antigravity CLI (`agy`)** en **Gentle AI**. Diseñada específicamente para aprovechar al máximo las capacidades de `agy` como runtime autónomo: arquitectura modular de plugins, 10 subagentes nativos especializados, ciclo de vida con hooks, statusline y título dinámicos, evaluación en memoria de permisos y revisión adversarial automatizada basada en recibos inmutables (RDD con lentes 4R).

> [!IMPORTANT]
> **Versión fijada actual**: **v4.2.1** (basada en upstream Gentle AI v4.2.0, Gentle Shell v4.3.0 y Antigravity CLI v1.4.4).

---

## 1. Inicio Rápido (Quick Start)

### Paso 1: Prerrequisitos
Asegurate de haber abierto `agy` e iniciado sesión al menos una vez y tener Go 1.25+ en tu sistema:
```bash
agy --version
go version
```

### Paso 2: Instalación del binario `gentle-ai`

#### Opción A: Descarga desde GitHub Releases (Recomendado)
Descargá el binario compilado y firmado desde las [Releases del repositorio](https://github.com/ariasbruno/gentle-ai/releases) y colocalo en tu `$PATH` (ej. `~/.local/bin` o `/usr/local/bin`).

#### Opción B: Compilación local desde la rama de desarrollo
```bash
git clone -b agy/ai-v4.2.0_shell-v4.3.0 https://github.com/ariasbruno/gentle-ai.git
cd gentle-ai
go install ./cmd/gentle-ai
```

### Paso 3: Instalación y sincronización

Elegí el alcance según tu flujo de trabajo (consultá la [Guía de Instalación Detallada](installation.md)):

- **Instalación Global (Recomendada para uso personal en todos tus proyectos):**
  ```bash
  gentle-ai install --agent antigravity-cli --scope global
  gentle-ai sync --agents antigravity-cli --scope global
  ```

- **Instalación por Workspace (Aislada exclusivamente al repositorio actual):**
  ```bash
  gentle-ai install --agent antigravity-cli --scope workspace
  gentle-ai sync --agents antigravity-cli --scope workspace
  ```

*(Opcional: Si querés que además configure la StatusLine y el título dinámico en `settings.json`, sumá el flag `--include-theme` a los comandos anteriores).*

### Paso 4: Verificación inmediata del estado
```bash
# Diagnóstico integral del sistema y servidores MCP
gentle-ai doctor

# Validación estricta mediante el validador oficial de Antigravity CLI
agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai
```

---

## 2. ¿Por qué existe esta integración? (La Realidad vs Upstream)

En upstream Gentle AI existe un único adaptador genérico llamado `antigravity` que comete el error de intentar abarcar **tres entornos completamente distintos (Antigravity CLI, Antigravity 2.0 y Antigravity IDE) en una sola bolsa**:

- **El problema de upstream**: Trata a Antigravity como si fuera un editor desktop tradicional. Concatena un único prompt plano en `.gemini/GEMINI.md`, **no provee subagentes** (`SubAgentsDir` está vacío en el código), **carece de transporte RDD** (los usuarios de Antigravity no pueden ejecutar revisiones Receipt-Driven Development), no utiliza el sistema oficial de plugins de `agy`, no implementa hooks de ciclo de vida, no ofrece statusline nativa, no soporta aislamiento por workspace en `.agents/`, y mezcla rutas entre `.gemini/antigravity-desktop` y `.gemini/antigravity-cli`, provocando colisiones.
- **Solución**: Un adaptador dedicado y desacoplado (`model.AgentAntigravityCLI`). En lugar de emparchar las diferencias con el IDE, explota **el 100% de las posibilidades que Antigravity CLI ofrece como runtime moderno de agentes**.

### Cuadro Comparativo: Upstream (Antigravity) vs Este Fork (Antigravity CLI)

| Capacidad | Upstream (`antigravity` genérico) | Esta Integración (`antigravity-cli` dedicada) | Evidencia en Código |
|---|---|---|---|
| **Arquitectura de carga** | Concatena texto plano en `.gemini/GEMINI.md` | Plugin nativo con `plugin.json`, `rules/`, `agents/`, `skills/` y `hooks.json` | [`internal/agents/antigravitycli/adapter.go`](../../internal/agents/antigravitycli/adapter.go#L275-L295) |
| **Subagentes nativos** | ❌ Ninguno (`SubAgentsDir: ""` vacío) | ✅ **10 subagentes nativos** en formato Markdown listos para delegación | [`internal/assets/antigravitycli/agents/`](../../internal/assets/antigravitycli/agents) |
| **Revisión RDD (4R)** | ❌ No soportada (`transport: nil`) | ✅ **Transporte RDD in-process** con subprocesos sandboxed y árboles Git inmutables | [`internal/reviewerprovider/antigravitycli_adapter.go`](../../internal/reviewerprovider/antigravitycli_adapter.go) |
| **Ciclo de vida con Hooks** | ❌ No implementado | ✅ **5 hooks declarativos** (`PreInvocation`, `PreToolUse`, `PostToolUse`, `PostInvocation`, `Stop`) | [`internal/cli/antigravity_hook.go`](../../internal/cli/antigravity_hook.go#L122-L136) |
| **StatusLine y Título** | ❌ Sin soporte | ✅ **Statusline in-process y título dinámico** con fases ODD, RDD, cuotas y memoria Engram | [`internal/cli/statusline.go`](../../internal/cli/statusline.go), [`internal/statusline`](../../internal/statusline) |
| **Políticas de Seguridad** | ❌ No existen | ✅ **Guardia dinámica en memoria** (bloquea `.env*`, `.ssh`, `cat .env`, y fuerza `force_ask` en comandos destructivos) | [`internal/cli/antigravity_hook.go`](../../internal/cli/antigravity_hook.go#L921-L943) |
| **Registro de Plugins** | ❌ Manual | ✅ Automático en `import_manifest.json` vía `agy plugin install` en staging | [`internal/agents/antigravitycli/adapter.go`](../../internal/agents/antigravitycli/adapter.go#L820-L865) |
| **Soporte Multi-Scope** | ⚠️ Riesgo de colisión en `$HOME` | ✅ Separación estricta: `--scope global` vs `--scope workspace` (`.agents/plugins/gentle-ai`) | [`internal/cli/run.go`](../../internal/cli/run.go#L1053-L1071) |
| **Aislamiento de Git** | ❌ Riesgo de ensuciar `.gitignore` | ✅ Gestión limpia en `.git/info/exclude` sin alterar el working tree del repositorio | [`internal/components/localexclude`](../../internal/components/localexclude) |

---

## 3. Mapa Completo de Componentes y Superficies Desplegadas

El siguiente mapa describe todos los componentes instalados, registros de integración y almacenamiento en runtime:

```text
Entorno del Sistema y Repositorio
│
├── 1. Plugin Oficial de Gentle AI
│   (Global: ~/.gemini/config/plugins/gentle-ai/  |  Workspace: <repo>/.agents/plugins/gentle-ai/)
│   ├── plugin.json                                      # Manifiesto del plugin para agy
│   ├── mcp_config.json                                  # Servidores MCP (CodeGraph, Context7, Engram)
│   ├── hooks.json                                       # Definición de hooks de ciclo de vida
│   ├── hooks/                                           # Ejecutables puente (hook.sh / hook.cmd)
│   ├── rules/                                           # Reglas operativas (AGENTS.md, orchestrator, routing ODD, strict-tdd)
│   ├── agents/                                          # 10 Subagentes nativos Markdown
│   ├── skills/                                          # Catálogo completo de skills transpiladas
│   ├── chains/                                          # Flujos compuestos (4r-review.chain.md)
│   └── .native-rule-ownership.json                      # Registro de propiedad y digests de reglas
│
├── 2. Integración de StatusLine y Título (~/.gemini/antigravity-cli/settings.json)
│   ├── "statusLine": {"type": "command", "command": "gentle-ai statusline"}
│   └── "title": {"type": "command", "command": "gentle-ai statusline --title"}
│       (Configuración de usuario; gentle-ai preserva settings.json sin mutaciones automáticas en sync).
│
├── 3. Registro Global de Plugins (~/.gemini/config/import_manifest.json)
│   └── Registra el plugin de Gentle AI para su descubrimiento y carga inmediata en el arranque de agy.
│       (Solo en instalación global; en workspace se omite para aislamiento total).
│
├── 4. Aislamiento Limpio del Repositorio Local (<repo>/.git/info/exclude)
│   └── Excluye automáticamente artefactos internos (.atl, carpetas scratch) sin modificar el .gitignore del proyecto.
│
└── 5. Configuración y Estado en Tiempo de Ejecución (~/.gentle-ai/)
    ├── statusline.json                                  # Configuración de usuario para statusline (densidad y visibilidad)
    ├── statusline-phase/<conv_hash>.json                # Detección de fase de trabajo inferida vía PreToolUse
    ├── statusline-engram-state.json                     # Estado dinámico de observaciones Engram capturadas
    ├── statusline-rdd-cache.json                        # Caché del estado de revisión RDD
    ├── statusline-changes-cache.json                    # Caché de diferencias Git
    └── antigravity-mutations/v1/<conv_hash>.jsonl       # Ledger inmutable de mutaciones de archivos por sesión
```

### Barra de Estado y Título Dinámicos (`gentle-ai statusline`)

Gentle AI provee una statusline rica y nativa ejecutada in-process sin dependencias externas (`jq` o `bash`), totalmente compatible con Linux, macOS y Windows:

- **Pétalo estático**: Muestra el glifo de rosa estático `✿` en rosa Gentle Shell (`ansiRose + ansiBold + "✿"`), consistente y sin animación en todos los estados del agente.
- **Modo y modelo**: Refleja el modo de ciclo activo (ej. `auto · Gemini 3.8 Flash`).
- **Gauges de contexto y cuotas**: Muestra el consumo de la ventana de contexto (`ctx ▰▰▰▱▱▱▱▱ 43%`) y el tiempo restante de cuota.
- **Memoria Engram reactiva**: Indica el estado del proyecto (`🧠 <proyecto> · ready` o `✓ saved #id` al registrar observaciones).
- **Autoridad RDD**: Estado en vivo de revisiones de código (`🌹 Review in progress`, `✓ Approved`, `Fixing findings…`).
- **Métricas Git y Tareas ODD**: Contador de cambios (`±N +A/−D`) y avance de tareas del documento ODD más reciente (`⋗ 12/17 · T15 in_progress`).
- **Atajos alineados a la derecha**: Muestra `? for shortcuts` (o `?` en terminales compactas) alineado al margen derecho.
- **Auto-silenciado inteligente**: Si el plugin se desactiva en `config.json`, la barra se apaga automáticamente para no ensuciar la salida.
- **Personalización del usuario**: Mediante `~/.gentle-ai/statusline.json` podés ajustar la densidad (`comfortable`, `compact`, `minimal`) y alternar la visibilidad de cualquier segmento.

---

## 4. Arquitectura y Flujo de Trabajo

### Diagrama 1: Cadena de Transpilación y Despliegue
Gentle Shell es la fuente de verdad canónica. El transpilador adapta y valida los assets, los embebe en el binario Go y el comando `install/sync` despliega el plugin y el registro:

```mermaid
flowchart TD
    subgraph Sources["1. Fuentes de Verdad"]
        GS["Gentle Shell (commit fijado en PROVENANCE.json)<br/>Assets, Prompts, Canons y Skills"]
        GA["Gentle AI Core (Motor Go)<br/>CLI, RDD, Políticas y Lifecycle"]
    end

    subgraph Build["2. Transpilación y Compilación"]
        TR["Transpilador Go<br/>(scripts/transpile-antigravitycli)"]
        DG{{"Drift Gate CI<br/>(Verificación estricta de paridad)"}}
        Bundle["Asset Bundle Embebido<br/>(internal/assets/antigravitycli)"]
        Binary["Binario gentle-ai<br/>(v4.2.1 compilado)"]
    end

    subgraph Deploy["3. Despliegue en Sistema"]
        CLI["gentle-ai install / sync"]
        PluginFS["Plugin de agy<br/>~/.gemini/config/plugins/gentle-ai/<br/>o .agents/plugins/gentle-ai/"]
        SettingsFS["Configuración agy<br/>~/.gemini/antigravity-cli/settings.json<br/>(statusLine y title)"]
        ManifestFS["Registro de Plugins<br/>~/.gemini/config/import_manifest.json"]
        ExcludeFS["Exclusiones Git<br/>.git/info/exclude"]
    end

    subgraph Runtime["4. Ejecución en Antigravity CLI"]
        AGY["Runtime agy"]
    end

    GS --> TR
    TR --> DG
    DG --> Bundle
    GA --> Binary
    Bundle --> Binary
    Binary --> CLI
    CLI --> PluginFS
    CLI --> SettingsFS
    CLI --> ManifestFS
    CLI --> ExcludeFS
    PluginFS --> AGY
    SettingsFS --> AGY
    ManifestFS --> AGY
```

---

### Diagrama 2: Ciclo de Vida de una Sesión con Hooks y RDD
Durante una sesión interactiva, los hooks de Antigravity CLI colaboran con el motor en Go para auditar mutaciones, proteger archivos sensibles y exigir revisiones antes de cerrar cambios de alto riesgo:

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Desarrollador
    participant AGY as Antigravity CLI (agy)
    participant HookPre as Hook PreInvocation
    participant HookPreTool as Hook PreToolUse
    participant Orch as Orquestador Gentle AI
    participant Worker as Subagente Worker
    participant HookPost as Hook PostToolUse
    participant HookStop as Hook Stop
    participant RDD as Motor RDD (Go Engine)

    Dev->>AGY: Inicia sesión con su requerimiento
    AGY->>HookPre: Evento PreInvocation
    HookPre-->>AGY: Inyecta contexto y asegura exclusiones en .git/info/exclude
    AGY->>Orch: Inicia sesión con la regla ODD (Organic Driven Development)
    Orch->>Worker: Delega implementación con superficie de edición acotada
    Worker->>HookPreTool: Solicita ejecutar herramienta de edición o comando
    alt Comando destructivo o archivo sensible (.env, .ssh)
        HookPreTool-->>AGY: Bloquea con deny o exige force_ask
    else Operación segura
        HookPreTool-->>AGY: Autoriza ejecución
        Worker->>Worker: Aplica cambios en código y corre tests
        Worker->>HookPost: Evento PostToolUse (registra mutaciones en el ledger)
    end
    Worker-->>Orch: Finaliza tarea
    Orch->>AGY: Intenta cerrar la sesión
    AGY->>HookStop: Evento Stop (evalúa si autoriza el cierre)
    alt Cambio de Alto Riesgo Detectado
        HookStop->>RDD: Exige revisión de código obligatoria
        RDD->>RDD: Ejecuta lentes 4R en árbol Git inmutable
        RDD-->>HookStop: Emite veredicto (Approved / Correction)
        HookStop-->>AGY: Quema el recibo y autoriza el cierre
        AGY-->>Dev: Respuesta final verificada
    else Cambio de Bajo Riesgo o Lectura
        HookStop-->>AGY: Cierre autorizado de inmediato
        AGY-->>Dev: Respuesta directa
    end
```

---

## 5. Matriz de Subagentes Nativos Incluidos

Gentle AI dota a `agy` de 10 subagentes nativos configurados en formato Markdown:

| Categoría | Subagente | Rol y Responsabilidad | Herramientas |
|---|---|---|---|
| **Ciclo ODD** | `gentle-ai-explore` | Exploración de arquitectura, codebase e investigación previa | Solo lectura |
| | `gentle-ai-worker` | Implementación con superficie de edición estrictamente acotada | Edición + Tests |
| | `gentle-ai-verify` | Verificación técnica de cumplimiento de requisitos y pruebas | Solo lectura |
| **Judgment Day** | `jd-judge-a` | Primer juez ciego adversarial e independiente | Solo lectura |
| | `jd-judge-b` | Segundo juez ciego adversarial e independiente | Solo lectura |
| | `jd-fix-agent` | Agente quirúrgico de resolución para hallazgos confirmados | Edición + Tests |
| **Revisión 4R (RDD)** | `review-risk` | Lente R1: Seguridad, privilegios, fuga de datos y credenciales | Solo lectura |
| | `review-readability`| Lente R2: Nombres, complejidad, claridad de contratos y mantenibilidad | Solo lectura |
| | `review-reliability`| Lente R3: Pruebas behavior-first, cobertura, determinismo y regresiones | Solo lectura |
| | `review-resilience` | Lente R4: Recuperación de fallos, retries, backoff y carga | Solo lectura |

---

## 6. Documentación Modularizada

La documentación se organiza según la audiencia y el objetivo técnico:

### 📘 Guías Operativas para el Usuario
- [**Guía de Instalación Paso a Paso**](installation.md): Opciones global y workspace, flag opcional `--include-theme` y qué se copia en disco.
- [**StatusLine y Experiencia de Terminal**](statusline.md): Explicación de cada segmento, atajos de teclado, cuotas de modelos y auto-silenciado.
- [**Seguridad, Jerarquías y Sandbox**](security-and-sandbox.md): Confinamiento del sandbox, capas de jerarquía, prevención de lectura de secretos (`.env`) e higiene con `.git/info/exclude`.
- [**Resolución de Problemas (Troubleshooting)**](troubleshooting.md): Diagnóstico rápido con `gentle-ai doctor` y respuestas a preguntas frecuentes.
- [**Antigravity IDE frente a CLI (ide-vs-cli.md)**](ide-vs-cli.md): Comparativa directa y sin tecnicismos entre el IDE desktop y el runtime de terminal.

### 🏛️ Auditoría Técnica y Fundamentos de la Integración
- [**Antigravity IDE frente a CLI (ide-vs-cli.md)**](ide-vs-cli.md): Comparativa directa entre el IDE desktop y el runtime de terminal.
- [**Fundamentos y Puesta en Marcha (01-foundation.md)**](01-foundation.md): Despliegue de assets y directivas del plugin.
- [**Transpilador Determinista (02-asset-transpiler.md)**](02-asset-transpiler.md): Transpilación desde Gentle Shell y drift gate `--check`.
- [**Orquestación y Hooks (03-orchestration-and-hooks.md)**](03-orchestration-and-hooks.md): Hooks de ciclo de vida y ledger de mutaciones.
- [**Transporte RDD In-Process (05-review-transport.md)**](05-review-transport.md): Arquitectura de revisión sandboxed y esquemas 4R.
- [**Comparativa frente a otros Agentes (07-comparison.md)**](07-comparison.md): Tabla comparativa con Claude, Codex, OpenCode y Pi.
- [**Registro de Decisiones (08-decisions.md)**](08-decisions.md): Decisiones arquitectónicas, tradeoffs y alternativas descartadas.
- [**Registro de Verificación (09-verification.md)**](09-verification.md): Evidencia de suites de prueba, benchmarks y validación estricta.
- [**Mapa Técnico Completo (SUMMARY.md)**](SUMMARY.md): Diagramas Mermaid de secuencias y flujo transaccional.

### 🌐 Propuesta de Integración Upstream (Para Maintainers de Gentle AI)
Toda la justificación técnica, matriz de no-regresión y especificaciones detalladas para mergear esta integración al repositorio oficial se encuentran en:
- [**Propuesta de Integración Upstream (propuesta-upstream.md)**](propuesta-upstream.md)
