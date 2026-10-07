# 2. De Gentle Shell a assets agy

**El bundle agy deriva de fuentes fijadas de Gentle Shell mediante transformaciones explícitas.** Se adaptan contratos de texto y frontmatter; las extensiones TypeScript del runtime Pi no se transportan por cambiar nombres de herramientas.

[Índice](README.md) · [Propiedad](06-asset-ownership-and-drift.md) · [Decisiones](08-decisions.md)

## Fuente reproducible y procedencia

Para asegurar compilaciones deterministas y trazables, los assets transpilados están anclados a un commit inmutable de Gentle Shell registrado en [`PROVENANCE.json`](../../internal/assets/antigravitycli/PROVENANCE.json#L1-L4):

```json
{
  "source_repo": "https://github.com/Gentleman-Programming/gentle-shell",
  "source_commit": "0da07ce949952268b290bf751f22ad1202a40fc3"
}
```

Fuentes canónicas de referencia en dicho commit:
- [Orchestrator canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/orchestrator.md#L1-L121)
- [Explorer canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/agents/gentle-ai-explore.md#L1-L25)
- [Writer canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/orchestrator-writer.md#L1-L55)
- [Skill del harness](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/skills/gentle-ai/SKILL.md#L1-L61)

## Entradas y salidas

| Clase | Entrada Gentle Shell | Salida embedded | Tratamiento |
|---|---|---|---|
| Subagentes | `assets/agents/*.md` | `agents/<nombre>/agent.md` | Planning, descripción/tools, frontmatter nativo, mutaciones |
| Orchestrator | `assets/orchestrator.md` | `orchestrator.md` | Encabezado agy, rutas instaladas, marker de contrato |
| Satélites | Siete `assets/orchestrator-*.md` | Raíz del bundle | Strip de inyecciones y mutaciones del padre |
| Soporte | `assets/support/strict-tdd*.md` | `support/` | Misma familia de transformaciones |
| Cadena | `assets/chains/4r-review.chain.md` | `chains/4r-review.chain.md` | Elimina preflight Pi; mantiene vía manual |
| Skills | Todos los directorios de `skills/` | `skills/<directorio>/...` | Markdown transformado; otros archivos copiados |
| `_shared` | Fragmentos comunes | `skills/_shared/` | Excluye `review-ledger-contract.md` específico Pi |
| Manifiesto/procedencia | Generados por herramienta | `plugin.json`, `PROVENANCE.json` | Datos deterministas |

Implementación: [`transpileSubagents`](../../scripts/transpile-antigravitycli/main.go#L443-L512), [`transpileTopLevelOrchestrator`](../../scripts/transpile-antigravitycli/main.go#L568-L612), [`transpileSkills`](../../scripts/transpile-antigravitycli/main.go#L634-L704), [`transpileSatellites`, soporte/cadenas](../../scripts/transpile-antigravitycli/main.go#L768-L866).

No se aplica a agy la antigua regla «SKILL.md fork-owned del catálogo interno» como mecanismo de entrega. El componente de skills omite agy: [`DeliversFor`, `InjectWithCapability`](../../internal/components/skills/inject.go#L28-L46). Esto conserva el catálogo para otros adaptadores y establece un único owner del `skills/` del plugin.

## Resolución de fuente para generación

`resolveSourceRootWithProvenance` aplica este orden: [main.go, líneas 140–210](../../scripts/transpile-antigravitycli/main.go#L140-L210).

1. `--source-dir` válido; lee su `HEAD`.
2. `GENTLE_SHELL_DIR`, después `GENTLE_PI_DIR`.
3. Checkouts relativos `../gentle-shell`, `../gentle-pi` y los equivalentes dos niveles arriba.
4. Clon remoto temporal de `defaultRemoteRepo` con `--depth 1`.

La generación normal toma la fuente resuelta, no fuerza por sí sola el pin anterior. El modo `--check` tiene otro contrato: compara la regeneración en el pin contra el bundle **committed en HEAD**. No confundir ambos modos.

El verificador informó PASS de los paquetes transpilador/assets el 2026-10-06; eso no ejecuta el gate completo upstream. No se ejecutó generación en esta tarea. Los comandos del [registro de mantenimiento](09-verification.md#controles-de-mantenimiento-no-ejecutados) requieren autorización separada por sus escrituras/clones.

## Traducción de primitivas

El mapa real es una tabla de Go, no un compilador semántico del sistema de agentes: [`piToAntigravityToolMap`, `canonicalToolOrder`](../../scripts/transpile-antigravitycli/main.go#L44-L76).

| Primitiva de origen | Componentes emitidos |
|---|---|
| `read` | `view_file`, `list_dir` |
| `grep` | `grep_search` |
| `find` | `find_by_name` |
| `edit` | `replace_file_content` |
| `write` | `write_to_file` |
| `bash` | `run_command` |
| `web_search` | `search_web`, `read_url_content` |
| `web_fetch`, `fetch_content`, `get_search_content`, `source_check` | `read_url_content` |
| `ask_user_question`, `ask_user_choice` y variante de paquete | `ask_question` |
| `subagent_run` | `invoke_subagent`, `send_message` |

`glob` no es la clave de discovery en este mapa. Las herramientas `mem_*` y CodeGraph son tratadas como conocidas/ignoradas a nivel de lista built-in porque tienen ruta MCP; `gentle_*` se clasifica como directiva interna: [`isKnownIgnoredPiTool`](../../scripts/transpile-antigravitycli/main.go#L868-L887). **Ignorar un nombre no implementa una herramienta equivalente**.

## Frontmatter y capacidades solicitadas

`transpileSubagentMarkdown` exige fences y descripción, traduce tools, añade `mainAgent: false`, añade `excludeDefaultComponents: true` para los agentes clasificados como aislados y `enable_mcp_tools: true` cuando la fuente contiene CodeGraph o `mem_save`: [líneas 515–566](../../scripts/transpile-antigravitycli/main.go#L515-L566), [lista de aislados](../../scripts/transpile-antigravitycli/main.go#L78-L85).

Ejemplo actual, no esquema exhaustivo del cliente:

```yaml
---
name: gentle-ai-explore
description: Read-only exploration and mapping for generic ODD work.
tools: [view_file, list_dir, grep_search, find_by_name]
mainAgent: false
enable_mcp_tools: true
---
```

Fuente: [explorer emitido](../../internal/assets/antigravitycli/agents/gentle-ai-explore/agent.md#L1-L7).

La página oficial [Agents](https://antigravity.google/docs/cli/commands/agents/) documenta `agent.md` con `name` y `description` en rutas global/proyecto. **No prueba por sí sola el enforcement completo de `tools`, `mainAgent`, `excludeDefaultComponents` o `enable_mcp_tools` en v1.3.0.** Los comentarios del transpilador sobre versiones antiguas son intención/experiencia registrada, no un ensayo nuevo del cliente.

`--strict` falla si se acumulan tools sin mapping; los errores estructurales de la transformación también abortan. No llamarlo «validación completa del AST de agy»: la implementación usa strings/regex para gran parte de los contratos: [`main`](../../scripts/transpile-antigravitycli/main.go#L90-L137), [`transpileSubagentMarkdown`](../../scripts/transpile-antigravitycli/main.go#L515-L566).

## Adaptaciones de runtime frente a Pi

Dado que `agy` no ejecuta el runtime TypeScript de Pi ni sus extensiones propietarias, el transpilador aplica mutaciones específicas para adaptar las directivas a las capacidades reales de `agy`:

| Directiva en Gentle Shell (Pi) | Adaptación para Antigravity CLI | Motivo técnico |
|---|---|---|
| `subagent_run` en background | `invoke_subagent` / `send_message` | `agy` ejecuta subagentes mediante su sistema nativo de mensajería interactiva |
| Menús interactivos `ask_user_choice` | `ask_question` nativo | Se mapea al modal de preguntas nativo de `agy` |
| Relay de sockets para RDD | Contrato nativo de Go en reglas | En `agy`, la revisión RDD se controla directamente vía subproceso Go con `--sandbox` |
| Perfiles en `subagents.json` | Secciones específicas removidas | `agy` resuelve modelos vía flags de CLI o configuración estándar |
| Operaciones CodeGraph built-in | Servidor MCP `gentle-ai_codegraph` | CodeGraph se expone como servidor MCP estándar vía stdio |
| Rutas canónicas de Pi (`.pi/`) | Rutas de plugin (`.agents/` o `.gemini/`) | Ajuste automático al destino seleccionado (global o workspace) |

Referencias: [`mutateAgentRuntimeInstructions`](../../scripts/transpile-antigravitycli/main.go#L1092-L1167) y [`applyAntigravityCLIParentMutations`](../../scripts/transpile-antigravitycli/main.go#L1170-L1261).

## Subagentes incluidos y validación por invariantes

El bundle transpilado incluye 10 definiciones nativas de subagentes ([`expectedSubagentNames`](../../internal/assets/antigravitycli_assets_test.go#L38-L96)):

| Familia | Subagentes | Función |
|---|---|---|
| **Ciclo ODD** | `gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify` | Exploración de código de solo lectura, escritura acotada y verificación |
| **Judgment Day** | `jd-fix-agent`, `jd-judge-a`, `jd-judge-b` | Revisión adversarial ciega en dos rondas y corrección quirúrgica |
| **Revisión 4R** | `review-risk`, `review-readability`, `review-reliability`, `review-resilience` | Lentes especializados de seguridad, legibilidad, confiabilidad y resiliencia |

### Cobertura de invariantes en tests automatizados

La integridad del transpilador está asegurada mediante tests de invariantes en [`antigravitycli_assets_test.go`](../../internal/assets/antigravitycli_assets_test.go):

- **Invariante 1 (Estructura):** Valida la presencia de las 10 carpetas y archivos `agent.md` obligatorios.
- **Invariante 2 (Frontmatter):** Asegura que no queden residuos de sintaxis o herramientas prohibidas de Pi y valida el frontmatter YAML para `agy`.
- **Invariante 3 (Satélites):** Comprueba que los satélites del orquestador contengan las directivas vigentes.
- **Invariante 4 y 5 (Soporte y Cadenas):** Verifica la integridad de los contratos de TDD (`strict-tdd*.md`) y las cadenas de revisión (`4r-review.chain.md`).
- **Límite de tamaño:** Asegura que el prompt base del orquestador respete el límite máximo de bytes permitido para no desbordar el contexto inicial.

---

**Siguiente paso:** [Cómo se entregan reglas y qué hacen los hooks](03-orchestration-and-hooks.md).
