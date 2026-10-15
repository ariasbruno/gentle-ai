# Mapa técnico de la integración v4

**Hay dos procesos distintos:** configurar el plugin que acompaña una sesión agy y capturar una revisión RDD mediante un subprocess dirigido por Go. Comparten identificador de runtime, pero no deben confundirse sus mecanismos de carga, modelos ni autoridad.

[Índice](README.md) · [Comparación](07-comparison.md) · [Decisiones](08-decisions.md) · [Verificación](09-verification.md)

## Mapa de propietarios

```text
Gentle Shell @ SHA fijado
  assets/{agents,orchestrator*,support,chains} + skills/
                    │
                    ▼ adaptación textual/frontmatter
scripts/transpile-antigravitycli
                    │
                    ▼ bytes versionados + PROVENANCE.json
internal/assets/antigravitycli ────── embed ───────► binario Gentle AI v4
                                                       │
       componentes (persona/MCP/permisos/routing) ──────┤
                                                       ▼ install/sync
                                  árbol canónico del plugin en HOME
                                  ├─ rules + contrato renderizado
                                  ├─ agents/skills/satélites
                                  └─ hooks/MCP
                                                       │
                                  staging temporal + primer import
                                                       ▼
                                               cliente agy
```

Fuentes: [`transpileAll`](../../scripts/transpile-antigravitycli/main.go#L224-L280), [`transpileSkills`](../../scripts/transpile-antigravitycli/main.go#L634-L679), [`BundleAssets`/despliegue](../../internal/agents/antigravitycli/adapter.go#L192-L207), [`antigravityCLIPluginImportStep.Run`](../../internal/cli/run.go#L921-L940). Es un diagrama de implementación, no prueba de carga del cliente.

## Secuencia de configuración global

| Paso | Responsable | Resultado esperado en código | Control o límite |
|---|---|---|---|
| Detectar | Adaptador | Binario y directorio reportados por separado | No prueba auth/modelos |
| Inyectar componentes y routing | Pipeline CLI/componentes | Configuración según selección y reglas administradas | Workspace presenta destinos divergentes |
| Desplegar bundle | `DeployPluginTree` | Copiar assets, agentes, hooks, skills; podar elementos stale | Escrituras atómicas por archivo, no transacción única de árbol |
| Finalizar orchestrator | CLI | Sustituir marker con contrato agy, anteponer `always_on` | Error si no hay marker ni contrato ya rellenado |
| Importar si falta registro | Adaptador | Copia temporal, `agy plugin install`, validación de registro | Timeout 20 s, restauración del manifest en fallos |
| Próxima sesión agy | Cliente | Descubrir/configurar plugin | Requiere verificación real de versión/activación |

Referencias: [detección](../../internal/agents/antigravitycli/adapter.go#L77-L103), [despliegue](../../internal/agents/antigravitycli/adapter.go#L474-L638), [finalización](../../internal/cli/antigravity_orchestrator_rule.go#L28-L52), [import](../../internal/agents/antigravitycli/adapter.go#L643-L732).

## Secuencia de captura RDD

```text
padre/controlador
  │ operación exacta devuelta por el contrato Go
  ▼
Go: validar rol/capacidad/routing y construir Invocation opaca
  │ stdin: prompt materializado por proveedor
  ▼
agy --sandbox --output-format text [--model <asignación>] [--effort <asignación>]
  cwd: workspace vacío; HOME: perfil auth-only temporal; contexto cancelable
  │ stdout: bytes sin interpretación en el adaptador
  ▼
Go: admisión de esquema/binding/completitud
  ├─ transporte falla → error, no admisión
  ├─ resultado rechazado → corrección acotada si corresponde
  └─ resultado admitido → almacenamiento y siguiente transición nativa
```

Fuentes: [`reviewProviderAdapterWithRouting`](../../internal/cli/review_provider_runtime.go#L127-L157), [`Review`](../../internal/reviewerprovider/antigravitycli_adapter.go#L39-L78), [`Contract`](../../internal/reviewerprovider/contract.go#L112-L143), [`reviewProviderCaptureRetry`](../../internal/cli/review_provider_correction.go#L143-L183).

**«In-process» describe quién inicia y captura dentro del comando Go; agy sigue siendo un subprocess separado.** `--sandbox` es una solicitud al runtime, no una comprobación de confinamiento. El stdout no se transforma en recibo por sí solo. [Detalle de frontera](05-review-transport.md).

## Lo compartido y lo específico

| Compartido | Específico de agy |
|---|---|
| Contratos de roles y esquemas del proveedor Go | Traducción de herramientas/frontmatter |
| Política de presupuesto completo de contexto | Plugin bajo `.gemini/config/plugins/gentle-ai` |
| Admisión y lifecycle fuera del adaptador | Import nativo staged, hooks JSON |
| Vocabulario ODD/RDD y disciplina del padre | Routing protocol-only más canon transpilado |
| Intención de exploración/worker/verificador de Gentle Shell | Acceso CodeGraph MCP con prefijo agy |

[`ContractFor`](../../internal/reviewerprovider/contract.go#L154-L161), [`RuntimeContextBudget`](../../internal/reviewerprovider/capture_runtimes.go#L5-L30), [`applyAntigravityCLIParentMutations`](../../scripts/transpile-antigravitycli/main.go#L1170-L1261), [`injectAntigravityCLIRule`](../../internal/components/agentguidance/inject.go#L320-L382).

No se ha probado que Pi background, watchdog, UI de elección, enforcing de superficies o ledger TS estén portados a agy. La adaptación elimina o cambia texto de esas superficies; no transfiere el código que las ejecuta.

## Estado de evidencia al redactar

| Nivel | Disponible | Pendiente |
|---|---|---|
| Fuente actual | Código, assets y tests inspeccionados; SHA Gentle Shell comprobado | Actualizar rangos si cambia la rama |
| Fuente oficial | Contenido recuperado por el padre el 2026-10-06, tab/superficie cualificados | Fijar comportamiento a cada versión del cliente |
| Observación local | `agy 1.3.0`, ayuda y Go `1.27.1` recibidos | Activación real del plugin |
| Tests de producto | 17 paquetes PASS, exit 0 recibidos del verificador, 2026-10-06 | Conteo por-test/skips no visible; transcripción de comandos en registro |
| Driven | `j4500` definido con mock agy | Evidencia de ejecución actual y JSON de journey |
| Orgánica | Test opt-in existente de transporte sintético | No autorizado/no ejecutado; tampoco prueba lineage real |

El [registro de verificación](09-verification.md) es el lugar para incorporar resultados, no para copiarlos de documentación antigua. El [índice de fuentes](10-source-index.md) permite empezar una revisión por la afirmación, en vez de reconstruir todo el repositorio.
