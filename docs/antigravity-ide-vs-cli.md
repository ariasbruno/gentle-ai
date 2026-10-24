# Antigravity IDE frente a Antigravity CLI en Gentle AI v4

**Son adaptadores distintos, pero no se puede afirmar aislamiento universal de rutas.** El CLI configura un plugin agy y admite transporte RDD; el IDE usa paths/estrategias propios con fallback legacy que puede coincidir con el root de aplicación CLI.

Esta comparación describe código vigente. No atribuye ownership organizativo («upstream» frente a «fork») sin prueba histórica ni traslada resultados v3 al árbol v4. [Guía completa agy en español](antigravity-cli/README.md) · [Comparación de los cinco runtimes RDD](antigravity-cli/07-comparison.md).

## Identidad y rutas

| Dimensión | IDE: `model.AgentAntigravity` | CLI: `model.AgentAntigravityCLI` | Fuente |
|---|---|---|---|
| Detección | Evidencia de directorios/marker; no binario agy | `LookPath("agy")` y configDir separado | [IDE Detect](../internal/agents/antigravity/adapter.go#L57-L71), [CLI Detect](../internal/agents/antigravitycli/adapter.go#L82-L95) |
| GlobalConfigDir | `.gemini/config` si migrated; `.gemini/antigravity` si existe; variante si existe; root como default | `.gemini/antigravity-cli` fijo | [IDE](../internal/agents/antigravity/adapter.go#L85-L97), [CLI](../internal/agents/antigravitycli/adapter.go#L112-L114) |
| Variante para settings/skills/MCP | `.gemini/antigravity-desktop` si existe; si no `.gemini/antigravity-cli` | Settings en app root; skills/MCP en plugin | [IDE variant](../internal/agents/antigravity/adapter.go#L29-L35), [CLI paths](../internal/agents/antigravitycli/adapter.go#L116-L158) |
| Prompt | `.gemini/GEMINI.md`, append | Plugin `rules/AGENTS.md`, append más reglas independientes | [IDE](../internal/agents/antigravity/adapter.go#L99-L123), [CLI](../internal/agents/antigravitycli/adapter.go#L128-L158) |
| Skills | `<variant>/skills` | Plugin `skills`, bundle-owned | [IDE](../internal/agents/antigravity/adapter.go#L107-L113), [CLI](../internal/agents/antigravitycli/adapter.go#L136-L138), [skip catálogo](../internal/components/skills/inject.go#L28-L46) |
| SubAgentsDir | Vacío | Plugin `agents/<nombre>/agent.md` | [IDE](../internal/agents/antigravity/adapter.go#L146-L156), [CLI](../internal/agents/antigravitycli/adapter.go#L178-L188) |
| MCP | `<variant>/mcp_config.json` en adapter; CodeGraph contempla config/root | Plugin `mcp_config.json` | [IDE MCP](../internal/agents/antigravity/adapter.go#L124-L126), [CodeGraph paths](../internal/components/communitytool/codegraph_contract.go#L94-L106) |

### Riesgo concreto de coexistencia

El catálogo expone `~/.gemini/antigravity-cli` para ambas entradas: [agents.go, líneas 27–28](../internal/catalog/agents.go#L27-L28). El adapter IDE conserva ese fallback para su variante. Por eso no basta el nombre «IDE/CLI» para inferir superficies disjuntas.

El benchmark CLI crea un canario en `~/.config/antigravity/settings.json` y comprueba sus bytes: [fixture](../bench/journeys_antigravity_cli.go#L78-L86), [assert uninstall](../bench/journeys_antigravity_cli.go#L418-L424). Es una protección de **esa ruta**, no evidencia exhaustiva de que todos los settings/skills de `.gemini` estén aislados. La ejecución driven actual no fue realizada; [registro](antigravity-cli/09-verification.md).

## Capabilities y ownership

| Capacidad | IDE | CLI | Fuente y límite |
|---|---|---|---|
| Skills/SystemPrompt/MCP | Declaradas | Declaradas | [featureClaimsByAgent](../internal/agents/capabilitymanifest/manifest.go#L320-L335) |
| FileSubAgents | No declarado | Sí, diez assets actuales | [manifest](../internal/agents/capabilitymanifest/manifest.go#L320-L335), [census](../internal/assets/antigravitycli_assets_test.go#L38-L96); no prueba runtime discovery |
| Immutable review transport | No en boundary compilada | `antigravitycli_in_process` | [runtime policy](../internal/cli/review_transport_capability.go#L83-L154) |
| Captura Go de reviewer | No adapter en este conjunto | Subprocess agy nuevo, stdin/scratch | [CapturesInProcess](../internal/reviewerprovider/capture_runtimes.go#L50-L60), [adapter](../internal/reviewerprovider/antigravitycli_adapter.go#L42-L104) |
| Permisos inyectados | Overlay nil | Overlay nil (guardias dinámicas in-process vía PreToolUse) | [agentOverlay](../internal/components/permissions/inject.go#L172-L176); guardias evaluadas en tiempo real sin mutar settings.json |
| Context7 | Overlay HTTP `serverUrl` | Overlay default stdio npx | [dispatch MCP](../internal/components/mcp/inject.go#L380-L404), [overlays](../internal/components/mcp/context7.go#L9-L31) |
| CodeGraph | Native compatibility | Reconciled plugin wiring | [tabla](../internal/components/communitytool/codegraph_contract.go#L35-L50) |
| Orchestrator | Routing compartido con renderer, carrier prompt | Canon transpilado, contrato Go filled y regla protocol-only | [branch de render](../internal/components/agentguidance/inject.go#L124-L132), [agy rule](../internal/components/agentguidance/inject.go#L260-L273), [fill](../internal/cli/antigravity_orchestrator_rule.go#L28-L52) |

El IDE no gana capacidad RDD por ser `TierFull` o por compartir un directorio con el CLI. Del mismo modo, instalar el plugin agy no transforma el IDE en `AgentAntigravityCLI`.

## Rules, hooks y ejecución

Para el CLI, `DeployPluginTreeToDir` entrega assets/hook definitions; el paso final completa el contrato y hace import si falta registro: [run.go, líneas 1042–1073](../internal/cli/run.go#L1042-L1073). Hay ocho satélites, dos soportes strict-TDD y una cadena manual 4R (desplegados bajo `rules/`): [BundleAssets](../internal/agents/antigravitycli/adapter.go#L275-L297).

Los hooks CLI no son uniformemente fail-open: `PreToolUse` malformed/delegación inválida/repetición puede denegar; post hooks son no-op y Stop es recordatorio condicionado a idle/model. [Detalle](antigravity-cli/03-orchestration-and-hooks.md).

El reviewer CLI solicita `--sandbox`, no entrega su worktree como cwd y conserva HOME. No hay prueba actual de sandbox SO, supresión total de plugins/rules o tools read-only. [Frontera](antigravity-cli/05-review-transport.md#aislamiento-solicitado-no-cleanroom-demostrado).

Esta página no describe hooks/mecanismos internos del IDE externo que no fueron inspeccionados como parte de la integración Gentle AI. «No hay adapter equivalente aquí» no significa que el producto IDE no tenga su propio lifecycle.

## Import, retirada y datos del cliente

| Superficie | Qué cubre Gentle AI | Qué no queda demostrado |
|---|---|---|
| Plugin CLI administrado | Assets enumerados, agents/skills bundle, markers y primer import | Cada copia/cache que agy genere |
| Manifest import | Merge de root/entries ajenos; rollback de snapshot en fallos | Rollback total de efectos externos |
| Uninstall CLI | Operaciones own paths y registro, dirs si vacíos | Borrado universal de plugin_data, MCP cwd, conversaciones o auth |
| Settings IDE/CLI legacy | Paths elegidos por adapter/operación | Disjunción automática al seleccionar ambos |

Fuentes: [import](../internal/agents/antigravitycli/adapter.go#L785-L871), [uninstall](../internal/components/uninstall/service.go#L1408-L1460). No afirmar una convención de nombres runtime MCP o versión «landed in 1.2.2» sin evidencia cliente fijada; esos datos externos deben verificarse por separado.

## Recomendación operativa

1. Seleccionar explícitamente `antigravity-cli` para el plugin agy; no el identificador IDE.
2. Revisar paths legacy antes de configurar ambas integraciones.
3. Usar el alcance global documentado; workspace tiene destinos mixtos y no es aislamiento local garantizado.
4. Mantener personalizaciones fuera de agents/skills del plugin administrado.
5. Verificar configuración, activation y modelo como capas distintas, con permiso para cualquier ensayo real.

[Fundamentos y ruta segura](antigravity-cli/01-foundation.md) · [Propiedad/import](antigravity-cli/06-asset-ownership-and-drift.md) · [Evidencia y límites](antigravity-cli/09-verification.md).
