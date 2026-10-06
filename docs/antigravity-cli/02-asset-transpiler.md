# 2. De Gentle Shell a assets agy

**El bundle agy deriva de fuentes fijadas de Gentle Shell mediante transformaciones explícitas.** Se adaptan contratos de texto y frontmatter; las extensiones TypeScript del runtime Pi no se transportan por cambiar nombres de herramientas.

[Índice](README.md) · [Propiedad](06-asset-ownership-and-drift.md) · [Decisiones](08-decisions.md)

## Fuente reproducible, no versión inventada

[`PROVENANCE.json`](../../internal/assets/antigravitycli/PROVENANCE.json#L1-L4) contiene:

```json
{
  "source_repo": "https://github.com/Gentleman-Programming/gentle-shell",
  "source_commit": "0da07ce949952268b290bf751f22ad1202a40fc3"
}
```

Durante la lectura documental, `git -C /home/bruno/dev/gentle-shell rev-parse HEAD` devolvió ese mismo SHA. Se inspeccionaron directamente el orchestrator, el explorer, el writer y el skill `gentle-ai` del checkout canónico. No se infiere un semver de Gentle Shell desde ese SHA.

Para referencias inmutables:

- [Orchestrator canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/orchestrator.md#L1-L121).
- [Explorer canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/agents/gentle-ai-explore.md#L1-L25).
- [Writer canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/orchestrator-writer.md#L1-L55).
- [Skill del harness](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/skills/gentle-ai/SKILL.md#L1-L61).

La documentación histórica que menciona otro SHA refleja una etapa anterior, no cambia el pin actual.

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

## Mutaciones que marcan diferencias con Pi

| Contrato de origen | Adaptación agy | Límite de la prueba |
|---|---|---|
| `subagent_run` y background | `invoke_subagent`/`send_message`, trabajo foreground | No porta scheduler ni watchdog TS |
| UI `ask_user_choice` | `ask_question` solo si representa todo el envelope; fallback textual y stop | No demuestra equivalencia de UI ni pérdida cero real |
| RDD host relay Pi | Contrato native review renderizado por Go en regla | No entrega ledger Pi ni authority a prompts |
| Config `subagents.json`/model profiles | Secciones específicas Pi eliminadas; defaults del runtime en genéricos | No demuestra lectura de profiles Pi por agy |
| Tool CodeGraph cwd-scoped init/query/explore | MCP `gentle-ai_codegraph`, `codegraph_explore` | No ofrece operaciones Pi inexistentes |
| Rutas de assets de paquete | Rutas instaladas de plugin | Texto de ruta no cambia el instalador workspace |
| Enforcing runtime de superficies/parallelism | Disciplina escrita y adaptación de satélite | No se identificó aquí equivalente ejecutable agy |

Referencias: [`mutateAgentRuntimeInstructions`](../../scripts/transpile-antigravitycli/main.go#L1092-L1167), [`applyAntigravityCLIParentMutations`](../../scripts/transpile-antigravitycli/main.go#L1170-L1261). El writer canónico contiene background paralelo; la mutación agy lo serializa/condiciona. Debe evaluarse como diferencia, no como «paridad de concurrencia».

## Census actual y lo que realmente cubren los invariantes

El censo fijado por `expectedSubagentNames` es:

| Familia | Nombres | Uso |
|---|---|---|
| Trabajo ODD | `gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify` | Explorar, escribir acotadamente, verificar |
| Judgment Day | `jd-fix-agent`, `jd-judge-a`, `jd-judge-b` | Flujo explícito, no default de todo trabajo |
| Lentes manuales | `review-risk`, `review-readability`, `review-reliability`, `review-resilience` | Definiciones instaladas; no identidad del transporte RDD |

[`expectedSubagentNames`, `Invariant1`](../../internal/assets/antigravitycli_assets_test.go#L38-L96). Los seis roles de routing RDD incluyen refuter y validator sin que por ello existan seis agent.md de revisión: [routing](05-review-transport.md#modelos-de-revisión-y-propiedad-global).

| Test existente | Aserción real | No demuestra |
|---|---|---|
| `Invariant1_Subagents` | Diez directorios y nombres esperados, archivos no triviales | Descubrimiento o ejecución real |
| `Invariant2_Frontmatter` | Fences y ausencia de tools/residuos Pi prohibidos; contrato de lentes | Un parser YAML completo ni enforcement del cliente |
| `Invariant3_Satellites` | Tres nombres históricos, mínimo de longitud | Census exacto de siete satélites; el comentario «exactly 3» no es la entrega actual |
| `Invariant4_SupportContracts` | Dos contratos presentes y longitud mínima | Ejecución de TDD ni validación semántica integral |
| `Invariant5_Chains` | Una cadena presente y longitud mínima | Motor de cadenas nativo ni prueba de cada paso |
| `RuleSizeLimit` | Orchestrator embedded menor que 24.000 bytes | Tamaño final tras rellenar contrato ni presupuesto token aggregate |
| `ExploreAgentUsesMCPForCodeGraph` | Prefijo/operación MCP, ausencia de `call_mcp_tool`/init Pi | MCP conectado en agy real |

Código de tests: [líneas 64–184](../../internal/assets/antigravitycli_assets_test.go#L64-L184), [soporte/cadena/tamaño/explorer](../../internal/assets/antigravitycli_assets_test.go#L306-L381). El tamaño **final** del orchestrator sí tiene test separado: [`TestEnsureAntigravityOrchestratorRuleFillsContractAndFrontmatter`](../../internal/cli/antigravity_orchestrator_rule_test.go#L13-L60).

## Atomicidad y riesgo de adaptación

La generación de subagentes escribe un directorio staged y sustituye destino con backup/rollback si falla el rename: [`replaceStagedAgentDirectory`](../../scripts/transpile-antigravitycli/main.go#L484-L512). No extender esa atomicidad a todo `transpileAll` ni al deployment instalado, que escribe múltiples archivos secuencialmente.

El mayor riesgo de un transpilador textual es **texto sintácticamente válido con significado equivocado**. Los tests de residuos y paths reducen errores conocidos; no prueban que todas las capacidades de cada harness coincidan. Una actualización debe revisar fuente, diff de transformaciones y prueba del runtime requerido, no solo aceptar un SHA nuevo.

**Siguiente paso:** [cómo se entregan reglas y qué hacen los hooks](03-orchestration-and-hooks.md).
