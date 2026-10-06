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
# Instalación global (afecta a todas las sesiones de agy del usuario)
gentle-ai install --agent antigravity-cli --scope global
gentle-ai sync --agents antigravity-cli --scope global

# Instalación en workspace (aislada a este repositorio)
gentle-ai install --agent antigravity-cli --scope workspace
gentle-ai sync --agents antigravity-cli --scope workspace
```

Las opciones `--agent`/`--agents` y `--scope` están registradas por [`ParseInstallFlags`](../../internal/cli/install.go#L53-L70) y [`ParseSyncFlags`](../../internal/cli/sync.go#L171-L193). La resolución de alcances está definida en [`ResolveInstallScope`](../../internal/cli/scope.go#L24-L44). Los comandos configuran el entorno para el agente, pero no instalan el binario `agy`: el adaptador no cuenta con autoinstalación de ejecutables externos.

En alcance global, la secuencia del paso final es `DeployPluginTree → ensureAntigravityOrchestratorRule → EnsurePluginImported`: [`antigravityCLIPluginImportStep.Run`](../../internal/cli/run.go#L921-L940). En alcance workspace, se despliega el árbol en `<workspace>/.agents/plugins/gentle-ai/` y se inicializa la regla de orquestador localmente, prescindiendo del registro en el manifiesto global.

### Estructura desplegada en cada alcance

Ambos alcances despliegan el bundle completo de assets:

```text
# En --scope global:
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

# En --scope workspace:
<workspace>/.agents/plugins/gentle-ai/
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

### Particularidades por alcance:
1. **Descubrimiento automático en workspace:** `agy` descubre nativamente los plugins situados en `.agents/plugins/<nombre>/`, por lo que el instalador no modifica `~/.gemini/config/import_manifest.json`.
2. **Aislamiento de `HOME`:** En `--scope workspace`, el directorio de usuario (`HOME`) permanece completamente inalterado.
3. **Precedencia:** Cuando existen ambos, las reglas y configuraciones del workspace tienen prioridad sobre las globales.

### Qué comprobar después de la instalación o sincronización

Sin tratar ninguna señal como prueba total:

| Señal | Qué aporta | Qué no aporta |
|---|---|---|
| Archivos y frontmatter esperados | Escritura/materialización del plugin | Que agy los cargó en contexto |
| Marker de contrato sustituido | Finalización del orchestrator | Que un modelo cumplió instrucciones |
| Registro en `import_manifest.json` (solo global) | Registro del import en agy | Salud de MCP ni actualizaciones del cliente |
| Resultado de tests automatizados | Contrato del producto verificado | Runtime autenticado |
| Respuesta de modelo citando reglas | Observación de una sesión | Enforcement estricto a nivel de SO |

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
