# 4. MCP, CodeGraph y la evidencia del benchmark

**Una configuración MCP válida no equivale a un servidor conectado.** Gentle AI reconcilia el wiring CodeGraph de agy dentro del plugin; `j4500` verifica un ciclo del producto con un ejecutable agy falso, no el funcionamiento de un modelo real.

[Índice](README.md) · [Orquestación](03-orchestration-and-hooks.md) · [Verificación](09-verification.md)

## Superficie CodeGraph

`codeGraphToolWiringPaths` considera para `AgentAntigravityCLI` exactamente el `MCPConfigPath` del adaptador: [codegraph_contract.go, líneas 94–106](../../internal/components/communitytool/codegraph_contract.go#L94-L106). La ruta es `~/.gemini/config/plugins/gentle-ai/mcp_config.json`: [`MCPConfigPath`](../../internal/agents/antigravitycli/adapter.go#L157-L158).

Configuración emitida por el reconciliador:

```json
{
  "mcpServers": {
    "codegraph": {
      "command": "codegraph",
      "args": ["serve", "--mcp"]
    }
  }
}
```

[`ReconcileAntigravityCLICodeGraphWithAgents`](../../internal/components/communitytool/codegraph_guidance.go#L311-L367) usa detección/selección, devuelve sin escribir si ya es canonical y añade/reemplaza la entrada `codegraph` si falta.

## Reconciliación y controles negativos

```text
agy no seleccionado/detectado ───────────────► no reconciliar
agy seleccionado + wiring canonical ───────► Changed=false
agy seleccionado + JSON válido no canonical
  → preservar root/otros servers → upsert codegraph → WriteFileAtomic
agy seleccionado + JSON malformed
  → root nuevo → canonical codegraph → bytes anteriores descartados
```

| Control existente | Qué comprueba |
|---|---|
| `TestReconcileAntigravityCLICodeGraphCreatesPluginServer` | Path, comando y dos argumentos |
| `...IsIdempotent` | Segunda pasada sin cambios de bytes |
| `...PreservesOtherServers` | Conservación de context7/engram en base válida |
| `...ReplacesMalformedConfig` | Recuperación con documento nuevo |
| `TestNeedsAntigravityCLICodeGraphReconcileDetection` | No seleccionado no reconcilia; seleccionado sin wiring sí; ya wired no |
| `TestAntigravityCLICodeGraphWiringPathsAndManagedPaths` | Wiring limitado al MCP del plugin |

Tests leídos: [antigravity_cli_codegraph_test.go, líneas 26–189](../../internal/components/communitytool/antigravity_cli_codegraph_test.go#L26-L189). El verificador informó PASS del paquete communitytool el 2026-10-06; no se deduce status individual ni conteo de skips.

**Preservación es condicional a JSON válido.** El reconciliador documenta que backup/rollback del caso malformed corresponde al caller: [comentario, líneas 298–306](../../internal/components/communitytool/codegraph_guidance.go#L298-L306). Escritura atómica reduce archivos parcialmente escritos; no elimina conflictos con escritores concurrentes ni convierte todo sync en una operación indivisible.

## Ruta de herramienta para el explorer

El explorer actual pide usar el servidor namespaced `gentle-ai_codegraph`, herramienta `codegraph_explore`, antes de broad searches. Tiene `enable_mcp_tools: true` y tools built-in en frontmatter: [agent.md, líneas 1–20](../../internal/assets/antigravitycli/agents/gentle-ai-explore/agent.md#L1-L20).

No prescribe `call_mcp_tool` ni el tool Pi cwd-scoped con `operation: "init"`, `query` o `explore`. El test rechaza esas instrucciones residuales: [`TestAntigravityCLI_ExploreAgentUsesMCPForCodeGraph`](../../internal/assets/antigravitycli_assets_test.go#L363-L381).

Ejemplo conceptual de solicitud: «Mapear callers y blast radius de `AntigravityCLIAdapter.Review`». El formato concreto de llamada pertenece a las herramientas expuestas por la sesión agy; no se inventa aquí un envelope de MCP.

### Cuándo usar fallback

La instrucción del explorer permite `view_file`, `grep_search` y `find_by_name` solo cuando CodeGraph está indisponible o falla. Es política escrita; no prueba que cada modelo la siga. El servidor mantiene su índice; no ejecutar operaciones administrativas o uninit para «arreglar» una ausencia de herramientas.

## Engram y Context7: ownership y disponibilidad

| Servidor | Construcción de configuración | Límite |
|---|---|---|
| Engram | Overlay `mcpServers.engram`, comando resuelto, args `mcp --tools=agent` | Configuración no demuestra memoria activa ni equivalencia de herramientas con versiones previas |
| Context7 | Default stdio con `npx -y --package=@upstash/context7-mcp@<versión> -- context7-mcp` | Puede depender de Node/red/cache; no se midió arranque |
| CodeGraph | `codegraph serve --mcp` reconciliado | Requiere binario/servidor/índice accesibles al cliente |

Fuentes: [`engramOverlayJSON`](../../internal/components/engram/inject.go#L129-L142), [`defaultContext7OverlayJSON`](../../internal/components/mcp/context7.go#L9-L11), [`injectMCPConfigFile`](../../internal/components/mcp/inject.go#L380-L404) (la rama HTTP es IDE, no `AgentAntigravityCLI`), [reconciliador](../../internal/components/communitytool/codegraph_guidance.go#L350-L353).

No afirmar que el `PreInvocation` CLI «fuerza 14 herramientas Engram»: el handler CLI actual refresca registry/baseline. Hay superficies Engram adicionales en el componente, incluidas ramas IDE, que no equivalen automáticamente a la conducta de ese handler: [hook CLI](../../internal/cli/antigravity_hook.go#L151-L172). No llamar `agy mcp list` una prueba suficiente del wiring de un plugin; discovery global y plugin son preguntas distintas.

## `j4500`: contrato de ciclo de vida con mock

La definición se llama `j4500-antigravity-cli-lifecycle-parity`. «Parity» es parte del identificador; **no es una conclusión de paridad general**.

### Fixture

[`antigravityCLIRuntimeFixture`](../../bench/journeys_antigravity_cli.go#L18-L86) crea:

- Repositorio fixture y PATH aislado.
- Script `agy` falso que informa versión `1.2.9` y simula `plugin install`.
- Error cuando source y destination son iguales o falta plugin.json staged.
- Manifest inicial con plugin/root ajenos.
- Simulación de rewrite destructivo de manifest por agy.
- Canario de IDE en `.config/antigravity/settings.json`.

No usa el `agy 1.3.0` local ni llama a modelos.

### Secuencia conducida

```text
fixture
  → sync --agents antigravity-cli
     → assert archivos, reglas y contrato/import
  → segundo sync
     → assert idempotencia y timestamp/import bytes
  → uninstall --agents antigravity-cli --yes
     → assert limpieza administrada y conservación de import ajeno/canario
```

[`antigravityCLILifecycleJourneys`](../../bench/journeys_antigravity_cli.go#L427-L439). Los argumentos públicos se definen en [líneas 89–100](../../bench/journeys_antigravity_cli.go#L89-L100). No conserva la receta obsoleta `--components sdd` de la documentación anterior.

### Qué cubre y qué no

| Frontera | Cubierta por diseño de journey | No cubierta |
|---|---|---|
| CLI Gentle AI | Invocaciones públicas de sync/uninstall | Todas las combinaciones de componentes/scopes |
| Bundle | Archivos, reglas, idempotencia según assertions | Carga efectiva en contexto real |
| Import | Staging, preservación de registro ajeno, no reinstall | Semántica de staging/runtime actual de agy |
| Coexistencia IDE | Bytes de un canario específico | Todas las rutas compartidas legacy `.gemini` |
| RDD | Guidance/contrato instalado | Captura autenticada o lineage real |
| Seguridad | Fixture aislada del bench | Enforcement del sandbox/permissions del cliente |

## Unit, driven y CI no son sinónimos

`bench/` es un módulo separado. [`bench/README.md`](../../bench/README.md#L24-L75) distingue pruebas del harness de ejecución `run --binary ...`; la raíz no lo incluye en `go test ./...`.

En CI se declaran pasos diferentes:

1. `go test ./...` del producto.
2. Build/vet/test dentro de bench.
3. Build del producto y `gentle-ai-bench run --binary ... --out ...`.

[ci.yml, líneas 144–171](../../.github/workflows/ci.yml#L144-L171). Una declaración YAML no es resultado ejecutado de esta tarea. La lista mostrada de `completed_once(...)` no acredita por sí misma `j4500`.

**Estado actual:** tests del paquete bench PASS recibidos (4.787 s, 2026-10-06); ejecución driven actual `j4500` NO EJECUTADA; runtime real/autenticado no autorizado. [Registro reproducible](09-verification.md#benchmark-driven-no-equivale-a-go-test).

**Siguiente paso:** [transporte RDD, defaults y límites de aislamiento](05-review-transport.md).
