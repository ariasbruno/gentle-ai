# 1. Fundamentos, rutas y puesta en marcha

**`antigravity-cli` es un adaptador propio para configurar un `agy` ya instalado.** El ejecutable, la configuración de aplicación y el plugin son superficies diferentes; confundirlas produce diagnósticos y expectativas incorrectos.

[Índice](README.md) · [Arquitectura](SUMMARY.md) · [Fuentes](10-source-index.md)

## Identidad y detección

| Pregunta | Respuesta del código | Referencia |
|---|---|---|
| ¿Qué integración selecciono? | `model.AgentAntigravityCLI`, identificador `antigravity-cli` | [`Adapter.Agent`](../../internal/agents/antigravitycli/adapter.go#L72-L74) |
| ¿Cuál es su nivel? | `TierFull` | [`Adapter.Tier`](../../internal/agents/antigravitycli/adapter.go#L76-L78) |
| ¿Cómo detecta el binario? | `lookPath("agy")`, normalmente `exec.LookPath`; no ruta alternativa codificada a `~/.local/bin` | [`NewAdapter`, `Detect`](../../internal/agents/antigravitycli/adapter.go#L60-L91) |
| ¿Cómo detecta configuración? | Inspecciona si `GlobalConfigDir(home)` es un directorio; distingue errores de ausencia | [`Detect`](../../internal/agents/antigravitycli/adapter.go#L77-L89) |
| ¿Instala agy automáticamente? | No: `SupportsAutoInstall` devuelve false e `InstallCommand` un error | [líneas 94–104](../../internal/agents/antigravitycli/adapter.go#L94-L104) |
| ¿Comprueba auth o salud del modelo al detectar? | No; la detección mostrada no ejecuta el modelo | [cuerpo de `Detect`](../../internal/agents/antigravitycli/adapter.go#L77-L91) |

`TestDetect` cubre los escenarios de detección en [adapter_test.go, líneas 19–109](../../internal/agents/antigravitycli/adapter_test.go#L19-L109). El verificador informó PASS del paquete el 2026-10-06, sin salida por-test: no permite contar skips. [Registro completo](09-verification.md).

El catálogo contiene **más de cinco** entradas `TierFull`. Los cinco agentes que compara esta guía corresponden al conjunto cerrado de identidades RDD, no al censo completo: [`allAgents`](../../internal/catalog/agents.go#L19-L43) frente a [`registeredRuntimeIdentities`](../../internal/reviewerprovider/runtimes.go#L11-L17).

## Tres clases de rutas

### 1. Configuración de aplicación

```text
~/.gemini/antigravity-cli/
├── settings.json                    # overlay de permisos
└── gentle-ai/
    └── review-models.json           # asignaciones de revisión del usuario
```

`GlobalConfigDir` y `SettingsPath`: [adapter.go, líneas 112–148](../../internal/agents/antigravitycli/adapter.go#L112-L148). La ruta de modelos se resuelve mediante `os.UserHomeDir`, no el repositorio: [`AntigravityReviewModelsPath`](../../internal/reviewerprovider/antigravitycli_routing.go#L75-L81).

### 2. Árbol canónico escrito por Gentle AI

```text
~/.gemini/config/
├── import_manifest.json
└── plugins/gentle-ai/
    ├── plugin.json
    ├── mcp_config.json
    ├── hooks.json
    ├── hooks/{hook.sh,hook.cmd}
    ├── rules/
    │   ├── AGENTS.md
    │   ├── gentle-ai-routing.md
    │   └── gentle-ai-orchestrator.md
    ├── agents/<nombre>/agent.md
    ├── skills/<directorio>/...
    ├── orchestrator-{delegation,memory,skills,prompts,tracking,verification,writer}.md
    ├── support/{strict-tdd.md,strict-tdd-verify.md}
    └── chains/4r-review.chain.md
```

Las rutas y mappings pertenecen a [`PluginDir`, `BundleAssets`, `SubAgentsDir`](../../internal/agents/antigravitycli/adapter.go#L116-L216). `AGENTS.md`, routing y MCP reciben contenido por componentes separados; no todo el árbol es una copia literal del asset: [finalización de reglas](03-orchestration-and-hooks.md#reglas-y-carga-de-contexto).

### 3. Rutas administradas por el cliente agy

La fuente oficial de plugins distingue instalación manual global `~/.gemini/config/plugins` y proyecto `.agents/plugins`: [Plugins, tab CLI](https://antigravity.google/docs/plugins?tab=cli), contenido recuperado el 2026-10-06. Una observación HTTP anterior recibida por el padre menciona staging administrado por CLI en `~/.gemini/antigravity-cli/plugins`; **no se verificó aquí su equivalencia con el árbol canónico del adaptador**.

Por tanto:

- La ruta canónica de escritura está probada por código.
- La copia temporal pasada a `agy plugin install` está probada por código/tests del import.
- El estado cargado efectivamente por cada versión de agy requiere una observación del cliente. No se deduce solo de que existan carpetas.
- No se recomienda inspeccionar indiscriminadamente configuración de auth o conversaciones. La existencia de `HOME` no autoriza leer secretos.

## Puesta en marcha global

### Antes de autorizar escrituras

1. Confirmar que el `gentle-ai` elegido corresponde al módulo v4: [go.mod](../../go.mod#L1-L3). No usar como receta vigente el checkout antiguo `antigravity-cli-integration`.
2. Confirmar binario/ayuda de agy y requisitos de auth según [Headless](https://antigravity.google/docs/cli/headless/). La documentación oficial exige credenciales interactivas cacheadas; esta guía no automatiza login.
3. Decidir si se acepta un plugin administrado que puede reemplazar assets y podar subdirectorios. Guardar las personalizaciones fuera de ese árbol.
4. Entender que el alcance global es el camino documentado; no sustituirlo por `workspace` suponiendo una instalación local aislada.

### Comandos para el usuario, no ejecutados en esta tarea

Con ambos binarios ya disponibles:

```sh
# Escribe configuración y puede realizar el primer import de plugin.
gentle-ai install --agent antigravity-cli --scope global

# Reconcilia los assets administrados; no es un diagnóstico pasivo.
gentle-ai sync --agents antigravity-cli --scope global
```

Las formas `--agent`/`--agents` y `--scope` están registradas por [`ParseInstallFlags`](../../internal/cli/install.go#L53-L70) y [`ParseSyncFlags`](../../internal/cli/sync.go#L171-L193). `sync --agents antigravity-cli` también está usado por [`antigravityCLISyncArgs`](../../bench/journeys_antigravity_cli.go#L94-L96); el scope global está definido en [`ResolveInstallScope`](../../internal/cli/scope.go#L24-L44). Los comandos no instalan el agente agy: el adaptador no tiene autoinstalación.

La secuencia del paso final es `DeployPluginTree → ensureAntigravityOrchestratorRule → EnsurePluginImported`: [`antigravityCLIPluginImportStep.Run`](../../internal/cli/run.go#L921-L940). Un import ya registrado evita volver a ejecutar `agy plugin install`; no significa que la versión actual del cliente haya recargado cada cambio. Véase [propiedad/import](06-asset-ownership-and-drift.md#primer-import-y-registro).

### Qué comprobar después

Sin tratar ninguna señal como prueba total:

| Señal | Qué aporta | Qué no aporta |
|---|---|---|
| Archivos y frontmatter esperados | Escritura/materialización | Que agy los cargó en contexto |
| Marker de contrato ya sustituido | Finalización del orchestrator | Que un modelo cumplió instrucciones |
| Registro `gentle-ai` en manifest | Registro del import | Salud de MCP ni actualizaciones del cliente |
| Resultado de un test mock | Contrato del producto en fixture | Runtime autenticado |
| Respuesta de modelo que cita instrucciones | Observación de una sesión | Enforcement de restricciones del SO |

No presentar `agy plugin list` como universalmente read-only: el [registro histórico de propiedad](06-asset-ownership-and-drift.md#qué-no-debe-prometernos-el-import) advierte una mutación en 1.2.9. Para verificar activación real hace falta un HOME desechable y permiso explícito; no se ejecutó aquí.

## Alcance workspace no es un camino garantizado

Este es un límite concreto del código, no una preferencia editorial:

```text
componentes/routing:
  scope workspace → workspaceDir usado como home base
  Adapter.PluginDir(base) → <workspace>/.gemini/config/plugins/gentle-ai

paso final de plugin en install y sync:
  homeDir real → ~/.gemini/config/plugins/gentle-ai
  → import con HOME real
```

Fuentes: [`ResolveAgentConfigDir`](../../internal/cli/scope.go#L47-L55), [`componentInjectionDirScoped`, `routingGuidanceDir`](../../internal/cli/run.go#L3694-L3712), programación del import en [install](../../internal/cli/run.go#L904-L905) y [sync](../../internal/cli/sync.go#L662-L664).

El encabezado del orchestrator menciona `.agents/agents/<name>/agent.md` y la página oficial de agentes documenta esa ruta: [asset, líneas 7–9](../../internal/assets/antigravitycli/orchestrator.md#L7-L9), [Agents](https://antigravity.google/docs/cli/commands/agents/). **Eso no demuestra que el instalador Gentle AI entregue allí el bundle**. No se ofrece un happy path workspace ni garantía de que ese scope evite mutaciones en HOME. Resolverlo requiere decisión y prueba de producto, fuera de esta tarea documental.

## Permisos: overlay, no barrera completa

Gentle AI escribe entradas `permissions.deny` y `permissions.allow` en `settings.json`: [`antigravityCLIOverlayJSON`](../../internal/components/permissions/inject.go#L125-L160).

Ejemplos reales del overlay:

```json
{
  "permissions": {
    "deny": ["command(git reset --hard)", "read_file(.env)", "write_file(.ssh)"],
    "allow": ["command(gentle-ai codegraph init)", "command(gentle-ai review assess)", "command(gentle-ai review status)"]
  }
}
```

Este es un **extracto**, no el documento completo ni una recomendación para reemplazar configuración. Los tests comprueban presencia, conservación de una preferencia y segunda inyección idempotente: [`TestInjectAntigravityCLIPreservesUserSettingsAndInjectsDeniedResources`](../../internal/components/permissions/inject_test.go#L957-L1045).

La unión se aplica a arrays `deny`; para `allow`/`ask`, solo dentro del scope de permisos: [`mergeObjectScope`, `unionSlices`](../../internal/components/filemerge/json_merge.go#L1059-L1140). Un JSON base ilegible puede recuperarse desde base vacía por el merge compartido: [`MergeJSONObjects`](../../internal/components/filemerge/json_merge.go#L12-L32). Por ello «preserva configuración» debe calificarse con «base válida» y con el backup de la operación; no significa preservación de bytes o comentarios.

No hay prueba local de matching del cliente para cada entrada, de cobertura exhaustiva de secretos ni de bloqueo de todas las variantes de shell. La documentación oficial headless permite lecturas/escrituras de workspace por defecto y describe soft-denial en herramientas sin aprobación; `--dangerously-skip-permissions` aprueba todas. **No usar ese flag para validar seguridad**. [Fuente y límites](10-source-index.md#fuentes-oficiales-y-observaciones-locales).

## CLI frente a IDE

Ambos adaptadores pueden compartir rutas legacy bajo `.gemini/antigravity-cli`. El IDE elige variante y `GEMINI.md`; el CLI fija su plugin. El canario `.config/antigravity/settings.json` de `j4500` solo protege esa ruta concreta. No sustenta «cero contaminación» universal: [comparación detallada](../antigravity-ide-vs-cli.md).

**Siguiente paso:** [cómo Gentle Shell se convierte en assets agy](02-asset-transpiler.md).
