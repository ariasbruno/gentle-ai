# 4. Integración MCP, CodeGraph y validación de ciclo de vida

**La integración MCP conecta a Antigravity CLI con herramientas especializadas de desarrollo.** Gentle AI gestiona la configuración de servidores MCP como CodeGraph (análisis de grafos de código), Engram (memoria persistente) y Context7 (documentación), garantizando persistencia atómica e idempotencia. Asimismo, el módulo de benchmarks (`j4500`) valida el ciclo de vida de instalación y desinstalación de forma hermética y reproducible.

[Índice](README.md) · [Orquestación](03-orchestration-and-hooks.md) · [Transporte RDD](05-review-transport.md)

## Servidores MCP integrados

Gentle AI define la configuración de herramientas MCP en el archivo `mcp_config.json` dentro del árbol del plugin:
- **Ruta canónica:** `~/.gemini/config/plugins/gentle-ai/mcp_config.json` (declarada en [`MCPConfigPath`](../../internal/agents/antigravitycli/adapter.go#L157-L158)).

| Servidor | Comando / Ejecutable | Propósito |
|---|---|---|
| **CodeGraph** | `codegraph serve --mcp` | Inteligencia estructural de código: grafos de llamadas, dependencias e impacto |
| **Engram** | `engram mcp --tools=agent` | Memoria persistente entre sesiones y tras compactación de contexto |
| **Context7** | `npx -y --package=@upstash/context7-mcp@... -- context7-mcp` | Consulta y resolución de documentación externa contextual |

Estructura canónica de la configuración:

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

Fuentes: [`engramOverlayJSON`](../../internal/components/engram/inject.go#L129-L142), [`defaultContext7OverlayJSON`](../../internal/components/mcp/context7.go#L9-L11), [`injectMCPConfigFile`](../../internal/components/mcp/inject.go#L380-L404).

## Reconciliación y persistencia atómica de CodeGraph

Para asegurar que CodeGraph esté siempre disponible cuando Antigravity CLI está seleccionado, el reconciliador de Gentle AI gestiona `mcp_config.json` respetando las configuraciones preexistentes:

```text
Antigravity CLI no seleccionado ─────────────► No reconciliar
Antigravity CLI seleccionado:
  ├── Archivo ya cuenta con wiring canónico ──► Sin cambios (idempotente)
  ├── Archivo con JSON válido no canónico ────► Preservar otros servidores y hacer upsert de codegraph
  └── Archivo ausente o malformado ───────────► Generar documento canónico nuevo
```

El reconciliador ([`ReconcileAntigravityCLICodeGraphWithAgents`](../../internal/components/communitytool/codegraph_guidance.go#L311-L367)) aplica tres principios clave:
1. **Preservación:** Si el archivo ya define otros servidores (por ejemplo, `engram` o `context7`), los conserva intactos y solo inserta o actualiza la entrada de `codegraph`.
2. **Idempotencia:** Si la entrada canónica de CodeGraph ya está presente, no reescribe el disco (`Changed = false`).
3. **Escritura atómica:** Emplea `WriteFileAtomic` para evitar estados intermedios o archivos corruptos ante fallos imprevistos.

### Controles de prueba del reconciliador

| Test | Aspecto verificado |
|---|---|
| `TestReconcileAntigravityCLICodeGraphCreatesPluginServer` | Generación correcta de ruta, comando y argumentos |
| `TestReconcileAntigravityCLICodeGraphIsIdempotent` | Segunda pasada sin cambios en bytes ni timestamps |
| `TestReconcileAntigravityCLICodeGraphPreservesOtherServers` | Conservación de servidores ajenos y preexistentes |
| `TestReconcileAntigravityCLICodeGraphReplacesMalformedConfig` | Recuperación automática ante un archivo JSON inválido |
| `TestNeedsAntigravityCLICodeGraphReconcileDetection` | Detección precisa de cuándo se requiere intervenir y cuándo no |

Referencia: [`antigravity_cli_codegraph_test.go`](../../internal/components/communitytool/antigravity_cli_codegraph_test.go#L26-L189).

## Uso de CodeGraph en los subagentes

El subagente de exploración (`gentle-ai-explore`) está configurado con `enable_mcp_tools: true` en su definición ([`agent.md`](../../internal/assets/antigravitycli/agents/gentle-ai-explore/agent.md)):

- **Prioridad estructural:** Utiliza la herramienta `codegraph_explore` para mapear callers, dependencias y calcular el radio de impacto (*blast radius*) de los cambios antes de realizar exploraciones ciegas de archivos.
- **Fallback progresivo:** Si CodeGraph no está instalado en la máquina o el repositorio carece de índice, el explorador recurre de forma transparente a las herramientas convencionales del sistema de archivos (`view_file`, `grep_search`, `find_by_name`).

## Validación del ciclo de vida con Benchmarks (`j4500`)

Gentle AI cuenta con un módulo de benchmarks dedicado (`bench/`) diseñado para verificar el comportamiento de extremo a extremo sin depender de llamadas a modelos remotos ni credenciales externas.

Dentro de este conjunto se destaca el journey **`j4500-antigravity-cli-lifecycle-parity`** ([`journeys_antigravity_cli.go`](../../bench/journeys_antigravity_cli.go#L427-L439)):

```text
Entorno aislado (Fixture)
  │
  ├── 1. Instalación inicial (sync --agents antigravity-cli)
  │      └── Valida creación de archivos, reglas, hooks y manifiesto de importación.
  │
  ├── 2. Segunda sincronización (sync --agents antigravity-cli)
  │      └── Comprueba estricta idempotencia (bytes y timestamps intactos).
  │
  └── 3. Desinstalación completa (uninstall --agents antigravity-cli --yes)
         └── Verifica limpieza de componentes de Gentle AI, asegurando que
             las configuraciones externas y canarios del IDE permanezcan inalterados.
```

### Arquitectura hermética del test

Para garantizar reproducibilidad absoluta en cualquier entorno de CI o local:
- **Sandbox aislado:** Crea un directorio temporal con un PATH controlado y sin acceso al entorno global del usuario.
- **Mock de runtime `agy`:** Simula los subcomandos de `agy` (como `plugin install` y validaciones de manifiesto).
- **Canario de coexistencia con el IDE:** Coloca un archivo de configuración testigo en `.config/antigravity/settings.json` y valida que la desinstalación del CLI jamás altere la configuración del entorno de escritorio.

### Estructura de pruebas del proyecto

El repositorio separa deliberadamente las pruebas unitarias del core de los benchmarks de ciclo de vida:

| Suite | Comando | Propósito |
|---|---|---|
| **Unit tests del producto** | `go test ./...` | Validación rápida de lógica interna, adaptadores y transpilación |
| **Benchmarks de ciclo de vida** | `cd bench && go test ./...` | Ejecución de journeys completos (`j4500`) en entornos aislados |
| **Harness CLI de benchmarks** | `gentle-ai-bench run --binary <bin>` | Evaluación automatizada del binario compilado en pipelines de CI |

---

**Siguiente paso:** [Transporte RDD, defaults y límites de aislamiento](05-review-transport.md).
