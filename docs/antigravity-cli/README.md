# Antigravity CLI (`agy`) con Gentle AI v4

**Gentle AI configura `agy` mediante un plugin y adapta los contratos de Gentle Shell; no convierte agy en Pi ni asume paridad total entre runtimes.** Esta documentación detalla la instalación, arquitectura, decisiones de diseño, revisión RDD, cobertura de pruebas y límites del soporte. Su propósito es orientar a los desarrolladores sobre qué componentes se configuran y permitir a los mantenedores auditar cada integración.

> [!NOTE]
> La actualización local de esta rama fija **Gentle AI 4.2.0, Gentle Shell 4.3.0 y AGY 1.4.2**. Consultá el [registro incremental](11-upgrade-ai-4.2.0-shell-4.3.0.md) para commits, checks y límites; el [anterior](11-upgrade-ai-4.1.1-shell-4.2.0.md) queda como historia de la rama. No implica publicación de la rama, suite global verde ni certificación de modelos autenticados; los comandos de instalación publicados siguen siendo una referencia, no pasos ejecutados en esta tarea.

## Inicio Rápido (Quick Start)

Si querés instalar y empezar a usar Gentle AI con `agy`:

### 1. Prerrequisitos e instalación del binario `gentle-ai`

Necesitás tener `agy` (con login interactivo completado) y el binario `gentle-ai` en tu `$PATH`.

Si todavía no tenés `gentle-ai`, podés instalarlo con Go (1.25+):

#### Opción A: Descargando el binario del release de este fork (Recomendado)
Tomá el artefacto de la [página de releases de este fork](https://github.com/ariasbruno/gentle-ai/releases) y ponelo en tu `$PATH`.

*(`go install github.com/ariasbruno/gentle-ai/...@<tag>` no funciona: el `go.mod` declara el módulo de upstream, así que apuntaría al repositorio original.)*

#### Opción B: Clonando y compilando desde la rama de desarrollo
```bash
git clone -b agy/ai-v4.2.0_shell-v4.3.0 https://github.com/ariasbruno/gentle-ai.git
cd gentle-ai
go install ./cmd/gentle-ai
```

Asegurate de tener `$GOPATH/bin` (o `~/go/bin`) incluido en tu variable `$PATH` y verificá que ambos ejecutables respondan:
```sh
command -v agy && command -v gentle-ai
```

### 2. Instalación del plugin

Elige el alcance según tu preferencia:

- **Para todas tus sesiones de usuario (Global):**
  ```sh
  gentle-ai install --agent antigravity-cli --scope global
  gentle-ai sync --agents antigravity-cli --scope global
  ```

- **Solo para tu proyecto actual (Workspace / Aislado):**
  ```sh
  gentle-ai install --agent antigravity-cli --scope workspace
  gentle-ai sync --agents antigravity-cli --scope workspace
  ```

### 3. ¿Qué se instala y dónde?

Dependiendo del alcance seleccionado, Gentle AI despliega el bundle completo del plugin en:
- **Global:** `~/.gemini/config/plugins/gentle-ai/` (más el registro en `import_manifest.json` y, en `~/.gemini/antigravity-cli/settings.json`, el overlay de permisos y la status line administrada).
- **Workspace:** `<tu-proyecto>/.agents/plugins/gentle-ai/` (aislado al repositorio, sin tocar `$HOME`).

**Árbol completo de archivos desplegados:**
```text
├── plugin.json                                                                      # Manifiesto del plugin para agy
├── mcp_config.json                                                                 # Configuración de MCP (Context7, Engram, CodeGraph)
├── hooks.json                                                                      # Definición de hooks de eventos para agy
├── hooks/                                                                          # Scripts ejecutables de hooks (hook.sh / hook.cmd)
├── rules/
│   ├── AGENTS.md                                                                   # Reglas y persona base
│   ├── gentle-ai-orchestrator.md                                                   # Regla principal de orquestación
│   ├── gentle-ai-routing.md                                                        # Protocolo de enrutamiento ODD
│   ├── orchestrator-{delegation,memory,skills,prompts,tracking,verification,writer}.md # Reglas condicionales de fase
│   └── strict-tdd{,-verify}.md                                                     # Dos reglas condicionales de testing
├── agents/                                                                         # 10 subagentes nativos (.md)
│   ├── gentle-ai-explore, gentle-ai-worker, gentle-ai-verify                       # Subagentes de ciclo ODD
│   ├── jd-judge-a, jd-judge-b, jd-fix-agent                                        # Subagentes de Judgment Day
│   └── review-risk, review-resilience, review-readability, review-reliability       # Subagentes de revisión 4R
├── skills/                                                                         # Skills del bundle transpilado
└── chains/                                                                         # Cadenas de flujos compuestos (4r-review.chain.md)
```

---

## Leer según la necesidad

| Necesidad | Lectura |
|---|---|
| Configurar por primera vez, conocer rutas y precauciones | [Fundamentos y puesta en marcha](01-foundation.md) |
| Entender el sistema de extremo a extremo | [Mapa de arquitectura](SUMMARY.md) |
| Saber qué se adapta de Gentle Shell y qué no | [Transpilador y contratos](02-asset-transpiler.md) |
| Entender reglas, delegación y hooks | [Orquestación y hooks](03-orchestration-and-hooks.md) |
| Evaluar MCP y la evidencia de `j4500` | [CodeGraph y benchmarks](04-codegraph-and-benchmarks.md) |
| Auditar subprocess, modelos, admisión y autoridad | [Transporte RDD](05-review-transport.md) |
| Mantener o retirar el plugin sin confundir propietarios | [Propiedad y deriva](06-asset-ownership-and-drift.md) |
| Comparar Claude Code, Codex, OpenCode, Pi y agy | [Matriz por capacidad](07-comparison.md) |
| Revisar motivos y alternativas sin inventar historia | [Decisiones](08-decisions.md) |
| Reproducir controles o conocer resultados pendientes | [Registro de verificación](09-verification.md) |
| Localizar afirmaciones, símbolos, tests y fuentes oficiales | [Índice de fuentes](10-source-index.md) |
| Distinguir CLI de la integración IDE | [Antigravity IDE frente a CLI](../antigravity-ide-vs-cli.md) |
| Actualizar incrementalmente a AI 4.2.0 / Shell 4.3.0 / AGY 1.4.2 | [Inputs, checks, reviews y límites de esta actualización](11-upgrade-ai-4.2.0-shell-4.3.0.md) |
| Actualizar incrementalmente a AI 4.1.1 / Shell 4.2.0 / AGY 1.3.4 | [Actualización anterior, ya aceptada](11-upgrade-ai-4.1.1-shell-4.2.0.md) |

## Requisitos previos y verificación de entorno

Antes de instalar o modificar la configuración, se recomienda realizar una comprobación pasiva del entorno (comandos de solo lectura que no requieren autenticación ni llamadas a modelos):

```sh
command -v agy
agy --version
agy --help
go version
```

### Entorno de referencia validado

Esta documentación y la suite de pruebas toman como base el siguiente entorno de referencia:

- **Sistema operativo:** Linux `amd64`.
- **Referencia histórica de Antigravity CLI:** `agy 1.3.0`.
- **Actualización de esta rama:** `agy 1.4.2` observado en checks estructurales aislados y en una instalación real bajo `HOME` aislado; no se ejecutaron modelos AGY ni una instalación real de Pi. [Evidencia y límites](11-upgrade-ai-4.2.0-shell-4.3.0.md).
- **Entorno Go:** El proyecto declara compatibilidad base con Go `1.25.10` ([go.mod, líneas 1–3](../../go.mod#L1-L3)) y fue validado con cadenas de herramientas Go 1.25+.

> [!NOTE]
> La comprobación pasiva anterior únicamente verifica la presencia de los binarios en tu `PATH`. No implica que el plugin de Gentle AI esté activo ni evalúa el aislamiento del sandbox. Para aplicar la configuración en el sistema o en tu proyecto, consulta los pasos en [Puesta en marcha (Global y Workspace)](01-foundation.md#puesta-en-marcha-global-y-workspace).

## Consideraciones importantes

- **Personalizaciones propias fuera del árbol administrado:** No guardes skills o subagentes manuales dentro de las carpetas `agents/` o `skills/` del plugin. `gentle-ai sync` reconcilia el bundle y poda automáticamente cualquier archivo ajeno al árbol oficial: [`DeployPluginTree`](../../internal/agents/antigravitycli/adapter.go#L589-L625).
- **Alcance por proyecto vs global:** Si usás `--scope workspace`, el plugin se ubica en `.agents/plugins/gentle-ai/` de tu repositorio actual y tiene prioridad sobre tu configuración global, sin tocar `$HOME`.
- **Precedencia de configuraciones:** `agy` lee la configuración local de `.agents/` por sobre la global de `~/.gemini/config/`.

---

¿Querés profundizar en la arquitectura o la auditoría de código? Consultá el [Índice de fuentes](10-source-index.md) o el [Mapa de arquitectura](SUMMARY.md).
