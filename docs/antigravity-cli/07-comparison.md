# 7. Comparación por capacidad: agy, Claude Code, Codex, OpenCode y Pi

**Estos cinco agentes comparten la identidad de runtime registrada para revisión, no mecanismos idénticos.** Elegir uno implica elegir owners, carriers y límites diferentes. Esta matriz compara el código de Gentle AI v4; no pretende certificar cada producto externo ni demostrar igualdad de modelos.

[Índice](README.md) · [Ecosistema y Adaptadores](generic-vs-dedicated.md) · [Decisiones](08-decisions.md)

## Cómo interpretar la matriz

- **Declarado:** claim compilado de `AgentFeatureClaims` o contrato.
- **Implementado:** path/branch/argv inspeccionado.
- **Observado:** resultado concreto de ejecución recibido.
- **No reclamado por el adaptador:** no significa necesariamente que el producto externo carezca de esa función; puede ser propiedad del paquete host.

Los cinco están en el catálogo `TierFull`, pero hay más entradas con ese tier: [`allAgents`](../../internal/catalog/agents.go#L19-L43). El conjunto de cinco usado aquí viene de [`registeredRuntimeIdentities`](../../internal/reviewerprovider/runtimes.go#L11-L17). Ninguna igualdad de tier decide review transport.

## Claims del adaptador y propiedad de configuración

| Capacidad declarada | agy | Claude Code | Codex | OpenCode | Pi | Fuente/interpretación |
|---|---|---|---|---|---|---|
| `Skills` | Sí, bundle | Sí | Sí | Sí | No declarado por adapter | [`featureClaimsByAgent`](../../internal/agents/capabilitymanifest/manifest.go#L312-L359); Pi depende del paquete host, no de inyección catalog |
| `SystemPrompt` | Sí | Sí | Sí | Sí | False | [Mismo manifest](../../internal/agents/capabilitymanifest/manifest.go#L312-L359); no equiparar false con ausencia de system prompt en Pi |
| `MCP` | Sí | Sí | Sí | Sí | Sí | [Manifest](../../internal/agents/capabilitymanifest/manifest.go#L312-L359); el wiring cambia por adaptador |
| `FileSubAgents` | Sí | Sí | No | No | No | [Manifest](../../internal/agents/capabilitymanifest/manifest.go#L312-L359); OpenCode puede entregar agents en settings, no `SubAgentsDir` |
| `SlashCommands` | No | Sí | No | Sí | No | [Manifest](../../internal/agents/capabilitymanifest/manifest.go#L312-L359); no convierte instrucciones textuales en comandos nativos disponibles |
| `OutputStyles` | No | Sí | No | No | No | [Manifest](../../internal/agents/capabilitymanifest/manifest.go#L312-L359); distinta de persona/prompt |
| Skills del catálogo Gentle AI | Omitido | Elegibles si adapter soporta | Elegibles | Elegibles | Omitidos por capability | [`DeliversFor`, `InjectWithCapability`](../../internal/components/skills/inject.go#L28-L52) y manifest |
| Orchestrator de routing renderizado por Gentle AI | Excepción: transpilado/fill | Sí | Sí | Sí, carrier settings | Excepción: Gentle Shell owner | [`InjectRoutingWithOptions`](../../internal/components/agentguidance/inject.go#L101-L125), [`DeliversThroughOrchestratorPrompt`](../../internal/components/agentguidance/inject.go#L321-L335) |

No se extrapola desde `FileSubAgents=false` a «sin delegación» en OpenCode/Pi/Codex. El claim responde a entrega de archivos por este adaptador; la implementación de delegación del host es otra dimensión.

## Carriers y destinos

| Agente | Root/carry de prompt | Skills/subagents del adaptador | MCP | Referencia |
|---|---|---|---|---|
| agy | App `.gemini/antigravity-cli`; prompt en plugin `rules/AGENTS.md`, routing/orchestrator separados | Plugin skills y `agents/<name>/agent.md` | Plugin `mcp_config.json` | [`Adapter`, paths/strategies](../../internal/agents/antigravitycli/adapter.go#L200-L297) |
| Claude Code | `.claude`, `CLAUDE.md`, estrategia Markdown sections | `.claude/skills`, `.claude/agents` | Adapter expone files por server; wiring efectivo user Claude config para componentes | [`paths/strategy`](../../internal/agents/claude/adapter.go#L137-L170), [`SubAgentsDir`](../../internal/agents/claude/adapter.go#L212-L221), [`injectClaudeUserConfig`](../../internal/components/mcp/inject.go#L284-L301) |
| Codex | `.codex/AGENTS.md`, file replace | `.codex/skills`; no dir de subagents | `.codex/config.toml` | [`Adapter paths/strategies`](../../internal/agents/codex/adapter.go#L84-L150) |
| OpenCode | `ConfigPath`/`AGENTS.md` expuesto; routing always-loaded en orchestrator dentro de settings | `ConfigPath/skills`; no `SubAgentsDir` | Settings `opencode.json/jsonc` | [`Adapter`](../../internal/agents/opencode/adapter.go#L85-L130), [`FileSubAgents`](../../internal/agents/opencode/adapter.go#L196-L205), [carrier efectivo](../../internal/components/agentguidance/inject.go#L283-L295) |
| Pi | `.pi` root global; agent paths resueltos separadamente por `AgentConfigPath` | `SkillsDir` y `SubAgentsDir` vacíos en adapter; host/package owner | `AgentConfigPath` + MCP config | [`Adapter`](../../internal/agents/pi/adapter.go#L300-L348), [canon orchestrator](https://github.com/Gentleman-Programming/gentle-shell/blob/d5d692a6888da9e9844e7e443ee30991fa4b9d14/assets/orchestrator.md#L94-L112) |

El path expuesto por un adaptador no siempre es el **carrier efectivo** de un componente. OpenCode y Claude ilustran por qué hay que leer dispatch/inyector, no solo comparar nombres de métodos. Para agy, la instalación en workspace despliega de forma aislada en `<workspace>/.agents/plugins/gentle-ai/` donde `agy` lo descubre automáticamente sin requerir manifiesto global; ver [Fundamentos](01-foundation.md#puesta-en-marcha-global-y-workspace).

## Revisión e aislamiento

| Dimensión | agy | Claude Code | Codex | OpenCode | Pi | Fuentes |
|---|---|---|---|---|---|---|
| Transporte compilado | `antigravitycli_in_process` | `claude_prompt_carried` | `codex_advisory_scratch_process` | `opencode_provider_injected` | `pi_host_relay` | [`reviewImmutableRuntimeCapability`](../../internal/cli/review_transport_capability.go#L83-L154) |
| Quién lanza/captura | Comando Go | Comando Go | Comando Go | Host/plugin relay | Host Gentle Shell relay | [`CapturesInProcess`](../../internal/reviewerprovider/capture_runtimes.go#L33-L60), [comentarios de transporte](../../internal/cli/review_transport_capability.go#L43-L72) |
| Admisión adicional | Manifest advertise; binario al invocar | Manifest advertise; binario al invocar | Manifest advertise; binario al invocar | Versión detectada y declaración V1/V2 coherentes | Handshake exacto `gentle-pi.review-relay/v1` | [policy](../../internal/cli/review_transport_capability.go#L83-L154) |
| Cwd de subprocess Go | Scratch vacío | Scratch vacío | Scratch vacío | Lanzamiento host, no adapter Go | Lanzamiento host, no adapter Go | [agy](../../internal/reviewerprovider/antigravitycli_adapter.go#L52-L68), [Claude](../../internal/reviewerprovider/claude_adapter.go#L43-L79), [Codex](../../internal/reviewerprovider/codex_adapter.go#L36-L85), [host policy](../../internal/cli/review_transport_capability.go#L43-L72) |
| Desactivación tools/settings explícita | No flags equivalentes en argv | `--tools ""`, `--setting-sources ""`, system prompt explícito | `--ignore-user-config`, `--sandbox read-only` | Contrato de host; no misma receta de flags | Contrato dice discovery disabled por launcher; no argv Go | [agy](../../internal/reviewerprovider/antigravitycli_adapter.go#L78-L90), [Claude](../../internal/reviewerprovider/claude_adapter.go#L27-L40), [Codex](../../internal/reviewerprovider/codex_adapter.go#L88-L95), [Pi contract](../../internal/cli/review_transport_capability.go#L55-L63) |
| Sandbox/read-only solicitado | `--sandbox`, sin modo `read-only` separado | Sin flag sandbox en argv mostrado; tools vacíos | `--sandbox read-only` | Owner host | Owner host | [Tres adapters](../../internal/reviewerprovider/antigravitycli_adapter.go#L78-L90), [Claude argv](../../internal/reviewerprovider/claude_adapter.go#L35-L40), [Codex argv](../../internal/reviewerprovider/codex_adapter.go#L88-L95) |
| Auth/config usuario | HOME temporal; copia solo token e installation_id; sin settings/plugins copiados | Diseño conserva auth normal; settings suprimidos por flags | Config usuario ignorada por flag; auth no investigada aquí | Sesión host | Sesión host | [env agy](../../internal/reviewerprovider/antigravitycli_adapter.go#L106-L139), [Claude explicación](../../internal/reviewerprovider/claude_adapter.go#L27-L34), [Codex argv](../../internal/reviewerprovider/codex_adapter.go#L88-L95), [relays](../../internal/cli/review_transport_capability.go#L43-L72) |
| Resultado del adapter | stdout raw | stdout raw | Archivo `--output-last-message` | Frames raw relay | Bytes raw relay | [agy return](../../internal/reviewerprovider/antigravitycli_adapter.go#L93-L104), [Claude return](../../internal/reviewerprovider/claude_adapter.go#L79-L84), [Codex return](../../internal/reviewerprovider/codex_adapter.go#L67-L85), [relays](../../internal/cli/review_transport_capability.go#L43-L72) |
| Presupuesto de input completo | 200 KiB | 200 KiB | 200 KiB | 200 KiB | 200 KiB | [`RuntimeContextBudget`](../../internal/reviewerprovider/capture_runtimes.go#L5-L30): igualdad de política, no ventana de modelo medida |

La tabla compara **solicitudes y ownership**. No certifica la aplicación de sandbox ni auth del proveedor externo para ninguna de las cinco columnas. Particularmente, agy no es «tan cleanroom como Claude» por usar scratch; sus argv/env son materialmente distintos.

## Modelos: dos planos que no deben mezclarse

| Plano | agy | Claude Code | Codex | OpenCode | Pi | Referencia |
|---|---|---|---|---|---|---|
| Routing RDD leído por core Go | HOME `review-models.json`, strict resolver por key; `inherit` omite `--model` (default del propio AGY en el perfil aislado) y effort opcional validado | Saved assignments válidas de state, fase prioritaria sobre legacy | Saved `rdd-*` válidas de state | Adapter Go no aplica modelos; host-mediated | Adapter Go no aplica modelos; host-mediated | [`reviewProviderAdapter`, modelos guardados](../../internal/cli/review_provider_runtime.go#L18-L106), [agy routing branch](../../internal/cli/review_provider_runtime.go#L127-L157) |
| Modelo del subprocess reviewer | `--model` opcional (inherit por defecto), `--effort` opcional validado | Alias válido `--model` opcional | Modelo válido `--model` opcional | Owner de launcher/task host | Owner de launcher/task host | [agy](../../internal/reviewerprovider/antigravitycli_adapter.go#L78-L84), [Claude](../../internal/reviewerprovider/claude_adapter.go#L63-L66), [Codex](../../internal/reviewerprovider/codex_adapter.go#L64-L66), [host refusal en Go](../../internal/cli/review_provider_runtime.go#L108-L123) |
| Config general de subagentes | Texto adaptado prescribe defaults; profiles Pi eliminados | Fuera de ese resolver RDD | Fuera de ese resolver RDD | Fuera de ese resolver RDD | Canon menciona profiles/runtime propios | [mutaciones agy](../../scripts/transpile-antigravitycli/main.go#L1213-L1256), [writer canónico](https://github.com/Gentleman-Programming/gentle-shell/blob/d5d692a6888da9e9844e7e443ee30991fa4b9d14/assets/orchestrator-writer.md#L1-L7) |

«Host-mediated» no significa que no se puedan seleccionar modelos; significa que este adaptador Go no es el owner de esa selección. No se establece una equivalencia de presets ni de effort entre agentes.

En agy, `inherit` inicia la revisión omitiendo `--model`: el perfil aislado no copia defaults del perfil habitual, así que el reviewer resuelve el default del propio AGY. El resolver malformed falla cerrado (modelo y effort); los helpers Claude/Codex mostrados preservan defaults cuando su state no ofrece una asignación válida. Esa es una diferencia de comportamiento, no una etiqueta cosmética.

## CodeGraph y política de seguridad

| Capacidad | agy | Claude Code | Codex | OpenCode | Pi | Referencia |
|---|---|---|---|---|---|---|
| Estrategia CodeGraph | Reconciliada por Gentle AI | Native target `claude` | Native target `codex` | Reconciliada | Reconciliada | [`codeGraphCompatibilityTable`](../../internal/components/communitytool/codegraph_contract.go#L35-L50) |
| Superficie agy versus otras | Plugin MCP único | `.claude.json` para wiring CodeGraph | MCPPath TOML/default branch | Effective settings/wiring del adapter | Agent MCP/default branch | [`codeGraphToolWiringPaths`](../../internal/components/communitytool/codegraph_contract.go#L94-L119), [Codex MCPPath](../../internal/agents/codex/adapter.go#L117-L121), [Pi MCPPath](../../internal/agents/pi/adapter.go#L326-L328) |
| Overlay de permisos | Overlay nil en settings.json; guardias dinámicas in-process vía `PreToolUse` | `bypassPermissions` más deny list | Sin target de inyección | Granular bash/read, comandos de entrega Ask | No se infiere aquí overlay equivalente | [`TargetPath` y overlays](../../internal/components/permissions/inject.go#L17-L199) |
| Complejidad residual | Plugin discovery/MCP/HOME | Matching propio del cliente | Sandbox/config TOML | Versión/relay/settings JSONC | Launcher/extensiones | [adapter paths](../../internal/agents/antigravitycli/adapter.go#L200-L241), [policy runtime](../../internal/cli/review_transport_capability.go#L83-L154) |

No copiar deny lists o un flag de bypass de una columna a otra como si tuvieran igual semántica. Las fronteras de seguridad precisas requieren ensayos del cliente, no solo JSON escrito.

## Evidencia comparativa: qué falta para decir «paridad»

La verificación recibida del 2026-10-06 cubre paquetes Go completos y bench unitario, no un ensayo autenticado común de los cinco agentes: [registro](09-verification.md). `j4500` cubre agy mock, no otros cuatro modelos.

Una comparación empírica real necesitaría por capacidad:

1. Mismo candidato frozen y mismos criterios de salida, no prompts vagamente similares.
2. Versión de cada cliente y owner de configuración declarados.
3. Ensayo positivo y negativo de discovery, tools, scope, sandbox y modelo.
4. Evidencia de ejecución/captura más admisión, no solo respuesta textual.
5. Métricas de latencia/coste/cobertura con presupuesto y cuenta explicitados.

No está ejecutado ni autorizado en esta tarea. En consecuencia, se documentan **diferencias implementadas y límites**, no porcentajes de paridad ni una clasificación de calidad de agentes.
