# 10. Índice de afirmaciones, código, tests y fuentes

**Empezar por la afirmación que se quiere comprobar.** Los enlaces relativos apuntan al árbol de esta revisión; el SHA remoto fija el canon Gentle Shell. Los rangos `#L` se interpretan en GitHub; un lector local puede buscar el símbolo indicado.

[Índice](README.md) · [Registro de resultados](09-verification.md) · [Decisiones](08-decisions.md)

## Convenciones de evidencia

| Código | Nivel |
|---|---|
| **I** | Implementación/asset inspeccionado directamente |
| **T** | Fuente del test inspeccionada, con alcance de fixture explícito |
| **V** | Resultado de paquete recibido del verificador el 2026-10-06; no status por-test |
| **O** | Observación local/oficial recibida del padre, no repetida por el escritor |
| **H** | Registro histórico; informa intención, no funcionalidad actual por sí solo |
| **L** | Límite/no demostrado, sin resultado runtime actual |

Todos los PASS corresponden al HEAD `628bc61ec2851040f2c95de150d8c9e62883b17d`. El [registro](09-verification.md) separa scopes y resultados. «T+V» significa test source leído y paquete PASS recibido, no que se haya observado ese nombre particular sin skip.

## Identidad, rutas y entrega

| ID / afirmación | Implementación y símbolo | Tests relevantes | Nivel/límite |
|---|---|---|---|
| F1 — TierFull no equivale a cinco únicos agentes | [`catalog.allAgents`](../../internal/catalog/agents.go#L19-L43) | Grupo catalog recibido | I+V; conjunto RDD separado |
| F2 — Detecta agy desde PATH y configDir aparte | [`Adapter.Detect`](../../internal/agents/antigravitycli/adapter.go#L77-L91) | [`TestDetect`](../../internal/agents/antigravitycli/adapter_test.go#L19-L109) | I+T+V; no auth/health |
| F3 — No auto-instala agy | [`SupportsAutoInstall`, `InstallCommand`](../../internal/agents/antigravitycli/adapter.go#L94-L104) | [`TestSupportsAutoInstall`, `TestInstallCommand`](../../internal/agents/antigravitycli/adapter_test.go#L110-L135) | I+T+V |
| F4 — App root, plugin y settings diferentes | [`GlobalConfigDir`, `PluginDir`, `SettingsPath`, `MCPConfigPath`](../../internal/agents/antigravitycli/adapter.go#L112-L158) | [`TestConfigPaths`](../../internal/agents/antigravitycli/adapter_test.go#L137-L171) | I+T+V; managed paths del cliente por separado |
| F5 — Workspace usa base workspace, import usa HOME real | [`ResolveAgentConfigDir`](../../internal/cli/scope.go#L47-L55), [`componentInjectionDirScoped`](../../internal/cli/run.go#L3708-L3712), [plan install](../../internal/cli/run.go#L904-L905), [plan sync](../../internal/cli/sync.go#L662-L664) | [`TestStagePlansKeepAntigravityCLIPluginImportStep`](../../internal/cli/antigravity_cli_plugin_import_guard_test.go#L19-L72) verifica selección, no no-mutation workspace | I+T+V+L |
| F6 — Diez agents embedded | [`expectedSubagentNames`, `Invariant1`](../../internal/assets/antigravitycli_assets_test.go#L38-L96) | Mismo test; [`TestDeployPluginTree`](../../internal/agents/antigravitycli/adapter_test.go#L612-L725) | I+T+V; no runtime census |
| F7 — Siete satélites, dos soportes y una cadena | [`BundleAssets`](../../internal/agents/antigravitycli/adapter.go#L192-L207) | [`TestBundleAssets`](../../internal/agents/antigravitycli/adapter_test.go#L236-L266), [Invariant3](../../internal/assets/antigravitycli_assets_test.go#L146-L167), [Invariant4–5](../../internal/assets/antigravitycli_assets_test.go#L306-L343) | I+T+V; Invariant3 solo comprueba tres satélites |
| F8 — Permissions overlay conserva base válida y union | [`antigravityCLIOverlayJSON`](../../internal/components/permissions/inject.go#L125-L160), [`mergeObjectScope`, `unionSlices`](../../internal/components/filemerge/json_merge.go#L1059-L1140) | [`TestInjectAntigravityCLIPreservesUserSettingsAndInjectsDeniedResources`](../../internal/components/permissions/inject_test.go#L957-L1045) | I+T+V; no matching cliente medido |

## Canon y transpilación

| ID / afirmación | Implementación y símbolo | Tests relevantes | Nivel/límite |
|---|---|---|---|
| A1 — Fuente fijada por SHA, no semver demostrado | [`PROVENANCE.json`](../../internal/assets/antigravitycli/PROVENANCE.json#L1-L4), [`resolveSourceRootWithProvenance`](../../scripts/transpile-antigravitycli/main.go#L147-L219) | [`TestProvenanceEmissionDeterministic`](../../scripts/transpile-antigravitycli/main_test.go#L457-L526) | I+T+V; HEAD local canónico observado por escritor |
| A2 — Map de tools built-in | [`piToAntigravityToolMap`](../../scripts/transpile-antigravitycli/main.go#L44-L76) | [`Invariant2_Frontmatter`](../../internal/assets/antigravitycli_assets_test.go#L98-L159) | I+T+V; no equivalencia de ejecución |
| A3 — mainAgent/MCP/exclude solicitados | [`transpileSubagentMarkdown`](../../scripts/transpile-antigravitycli/main.go#L540-L597), [explorer](../../internal/assets/antigravitycli/agents/gentle-ai-explore/agent.md#L1-L7) | Frontmatter/residuos [test](../../internal/assets/antigravitycli_assets_test.go#L98-L159) | I+T+V+L; enforcement cliente pendiente |
| A4 — Staging conserva destino ante fuente inválida | [`transpileSubagents`, `replaceStagedAgentDirectory`](../../scripts/transpile-antigravitycli/main.go#L469-L508) | [`TestTranspileSubagentsFailureLeavesDestinationUnchanged`](../../scripts/transpile-antigravitycli/main_test.go#L48-L87) | I+T+V; atomicidad solo subtree agents |
| A5 — Skills agy vienen del bundle, no catálogo | [`transpileSkills`](../../scripts/transpile-antigravitycli/main.go#L667-L713), [`DeliversFor`](../../internal/components/skills/inject.go#L28-L46) | [`TestDeployPluginTreePrunesStaleSkills`](../../internal/agents/antigravitycli/adapter_test.go#L727-L805) | I+T+V |
| A6 — Ledger Pi excluido | [`transpileSharedDirExcludingReviewLedger`](../../scripts/transpile-antigravitycli/main.go#L714-L737), [prune legacy](../../internal/agents/antigravitycli/adapter.go#L610-L614) | Test de deploy + sources del bundle | I+T+V; contrato agy por fill Go |
| A7 — Mutaciones no portan scheduler/runtime Pi | [`mutateAgentRuntimeInstructions`, `applyAntigravityCLIParentMutations`](../../scripts/transpile-antigravitycli/main.go#L1275-L1452) | [`TestAntigravityCLI_ParentSatelliteSurfaceHasNoExecutablePiResidue`](../../internal/assets/antigravitycli_assets_test.go#L169-L212), [`TestAntigravityCLI_AssetsUseInstalledRuntimePaths`](../../internal/assets/antigravitycli_assets_test.go#L252-L272) | I+T+V+L |
| A8 — Gate contra committed HEAD y set/bytes | [`runCheckGate`](../../scripts/transpile-antigravitycli/main.go#L1571-L1667) | [`TestCheckGateComparesAgainstCommittedTree`, `TestCheckGateContentDriftFails`, `TestCheckGateMissingFileFails`, `TestCheckGateExtraFileFails`](../../scripts/transpile-antigravitycli/main_test.go#L622-L798) | I+T+V; gate completo upstream NO EJECUTADO |
| A9 — Nombre de skill = carpeta en el plugin emitido | [`normalizeEmittedSkillNames`](../../scripts/transpile-antigravitycli/main.go#L810-L874), [`frontmatterName`, `replaceSkillName`](../../scripts/transpile-antigravitycli/main.go#L875-L920) | [`TestEmittedSkillNamesMatchTheirFolders`](../../scripts/transpile-antigravitycli/main_test.go#L312-L368) | I+T+V; `agy plugin validate --strict` sin findings sobre un plugin desplegado con AGY 1.4.2 (antes: 11 warnings y exit 1) |

## Orquestación y hooks

| ID / afirmación | Implementación y símbolo | Tests relevantes | Nivel/límite |
|---|---|---|---|
| H1 — Routing protocol-only, single file y cleanup | [`injectAntigravityCLIRule`](../../internal/components/agentguidance/inject.go#L320-L382) | [`TestInjectRoutingAntigravityCLIWritesProtocolOnlyRule`, `TestInjectRoutingAntigravityCLIStalePartCleanup`](../../internal/components/agentguidance/inject_test.go#L1003-L1140) | I+T+V; no split actual |
| H2 — Orchestrator final con contrato y frontmatter | [`ensureAntigravityOrchestratorRule`](../../internal/cli/antigravity_orchestrator_rule.go#L28-L52) | [`TestEnsureAntigravityOrchestratorRuleFillsContractAndFrontmatter`, `TestEnsureAntigravityOrchestratorRuleRejectsMissingDeploy`](../../internal/cli/antigravity_orchestrator_rule_test.go#L13-L68) | I+T+V; test final <24.000 B |
| H3 — Política de coordinación en prompt | [orchestrator asset](../../internal/assets/antigravitycli/orchestrator.md#L11-L98) | Tests textuales/residuos | I+T+L; no enforcement integral |
| H4 — Cinco hooks, sin SessionStart | [`antigravityCLIPluginHooksJSON`](../../internal/agents/antigravitycli/adapter.go#L407-L470) | [`TestAntigravityHookHelpNamesOfficialEvents`](../../internal/cli/antigravity_hook_test.go#L478-L484) | I+T+V+O; contrato hooks multi-surface |
| H5 — PreInvocation primera invocación refresh/baseline | [`handleAntigravityPreInvocation`](../../internal/cli/antigravity_hook.go#L151-L172) | [`TestAntigravityHookPreInvocationBaselineAndSilent`](../../internal/cli/antigravity_hook_test.go#L12-L58) | I+T+V; puede escribir, no passive probe |
| H6 — Post hooks no-op | [handlers](../../internal/cli/antigravity_hook.go#L139-L148) | [`TestAntigravityHookPostEvents`](../../internal/cli/antigravity_hook_test.go#L404-L461) | I+T+V |
| H7 — PreToolUse malformed/delegación deny | [decoder](../../internal/cli/antigravity_hook.go#L88-L108), [`validateAntigravityCLIInvokeSubagentArgs`](../../internal/cli/antigravity_hook.go#L344-L383) | [`TestAntigravityHookPreToolUseEdgeCases`](../../internal/cli/antigravity_hook_test.go#L337-L369), [`TestAntigravityHookPreToolUseMalformedInvokePayloadFailsClosed`](../../internal/cli/antigravity_hook_test.go#L486-L499), [`TestAntigravityHookPreToolUseInvokeSubagentValidation`](../../internal/cli/antigravity_hook_test.go#L530-L635) | I+T+V; no validación semántica de todo prompt |
| H8 — Tercera llamada consecutiva idéntica deny | [`handleAntigravityPreToolUse`](../../internal/cli/antigravity_hook.go#L405-L476) | [`TestAntigravityHookPreToolUseCircuitBreaker`, `TestAntigravityHookPreToolUseKeyOrdering`](../../internal/cli/antigravity_hook_test.go#L175-L335) | I+T+V; depende de estado/ID |
| H9 — Stop idle/model y Action=start recuerda preflight | [`handleAntigravityStop`](../../internal/cli/antigravity_hook.go#L212-L282) | [`TestAntigravityHookStopOutcomes`](../../internal/cli/antigravity_hook_test.go#L59-L136) | I+T+V; no authority/gate fuerte |

## MCP, import y limpieza

| ID / afirmación | Implementación y símbolo | Tests relevantes | Nivel/límite |
|---|---|---|---|
| P1 — CodeGraph en plugin, canonical serve --mcp | [`codeGraphToolWiringPaths`](../../internal/components/communitytool/codegraph_contract.go#L94-L106), [`ReconcileAntigravityCLICodeGraphWithAgents`](../../internal/components/communitytool/codegraph_guidance.go#L311-L367) | [create/idempotent/preserve/malformed/detection](../../internal/components/communitytool/antigravity_cli_codegraph_test.go#L26-L189) | I+T+V; no servidor conectado |
| P2 — Explorer MCP, no tool Pi/bridge ficticio | [agent.md](../../internal/assets/antigravitycli/agents/gentle-ai-explore/agent.md#L1-L20) | [`TestAntigravityCLI_ExploreAgentUsesMCPForCodeGraph`](../../internal/assets/antigravitycli_assets_test.go#L363-L381) | I+T+V |
| P3 — Engram args agent; Context7 agy stdio | [`engramOverlayJSON`](../../internal/components/engram/inject.go#L129-L142), [`injectMCPConfigFile`](../../internal/components/mcp/inject.go#L380-L404), [default overlay](../../internal/components/mcp/context7.go#L9-L11) | Componentes seleccionados en suites, no probe runtime aquí | I; disponibilidad pendiente |
| P4 — Import staged y snapshot/restore | [`EnsurePluginImported`](../../internal/agents/antigravitycli/adapter.go#L643-L732) | [merge/gate/failures](../../internal/agents/antigravitycli/adapter_test.go#L268-L427) | I+T+V; agy mock |
| P5 — Pruning agents/skills propios | [`DeployPluginTree`](../../internal/agents/antigravitycli/adapter.go#L589-L625) | [deploy/prune](../../internal/agents/antigravitycli/adapter_test.go#L612-L805) | I+T+V; custom dirs no protegidos |
| P6 — Uninstall enumera assets/markers/entries | [`retainedAntigravityCLIPluginOperations`](../../internal/components/uninstall/service.go#L1408-L1460) | [`j4500`](../../bench/journeys_antigravity_cli.go#L427-L439), paquete uninstall PASS | I+V; driven no ejecutado |
| P7 — IDE/CLI pueden compartir fallback legacy | [`antigravityVariantDir`, `SettingsPath`](../../internal/agents/antigravity/adapter.go#L29-L35), [paths IDE](../../internal/agents/antigravity/adapter.go#L101-L133) | Canario j4500 solo `.config/antigravity/settings.json` | I+L; no universal isolation |

## RDD y evidencia de ejecución

| ID / afirmación | Implementación y símbolo | Tests relevantes | Nivel/límite |
|---|---|---|---|
| R1 — Agente registrado y captura in-process | [`runtimes.go`](../../internal/reviewerprovider/runtimes.go#L11-L35), [`CapturesInProcess`](../../internal/reviewerprovider/capture_runtimes.go#L50-L60), [policy](../../internal/cli/review_transport_capability.go#L83-L154) | [`TestAntigravityCLIReviewRuntimeBoundaryIsFrozen`](../../internal/reviewerprovider/antigravitycli_boundary_test.go#L21-L38) | I+T+V |
| R2 — Stdin, scratch, flags, raw stdout y cancelación | [`AntigravityCLIAdapter.Review`](../../internal/reviewerprovider/antigravitycli_adapter.go#L33-L78) | [stdin/raw/failure helper tests](../../internal/reviewerprovider/antigravitycli_adapter_test.go#L85-L137) | I+T+V; sandbox solicitado, no medido |
| R3 — HOME/PATH conservados, resto env limitado | [`antigravityCLIRuntimeEnvironment`](../../internal/reviewerprovider/antigravitycli_adapter.go#L81-L103) | [`TestAntigravityCLIAdapterPassesOnlyRuntimeEnvironment`](../../internal/reviewerprovider/antigravitycli_adapter_test.go#L20-L75) | I+T+V; HOME discovery no suprimida |
| R4 — Modelos user-owned/missing defaults/malformed cerrado | [`ResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing.go#L364-L425) | [resolver table/absence](../../internal/reviewerprovider/antigravitycli_routing_test.go#L15-L168) | I+T+V; sin prueba de default efímero del parent |
| R5 — Seis roles/presets y picker | [`CanonicalReviewRoles`, presets](../../internal/reviewerprovider/antigravitycli_routing.go#L103-L192), [picker](../../internal/tui/screens/antigravity_review_model_picker.go#L45-L208) | Paquetes routing/TUI PASS; [`TestAntigravityReviewPresets`](../../internal/reviewerprovider/antigravitycli_routing_test.go#L250-L288) | I+T+V; slugs no disponibilidad garantizada |
| R6 — Core posee schema/binding/admisión | [`Contract`](../../internal/reviewerprovider/contract.go#L112-L143), [`reviewProviderAdmitRaw`](../../internal/cli/review_provider.go#L111-L164) | [`TestAdapterMinimalityGuard`](../../internal/reviewerprovider/adapter_minimality_guard_test.go#L18-L40) | I+T+V; no authority por stdout |
| R7 — Presupuesto completo 200 KiB | [`ApprovedRuntimeContextBudget`](../../internal/reviewerprovider/capture_runtimes.go#L5-L30) | Budget/correction tests del core | I+V; policy no capacidad model medida |
| R8 — Una correction acotada en core, no transport retry | [`reviewProviderCaptureRetry`](../../internal/cli/review_provider_correction.go#L143-L183) | [retry/refuse/transport error](../../internal/cli/review_provider_correction_test.go#L122-L246) | I+T+V; seam compartido, no agy orgánico |
| R9 — Fake round-trip prueba contrato de proceso | [fake script/test](../../internal/reviewerprovider/antigravitycli_fake_test.go#L28-L150) | `TestAntigravityCLIAdapterFakeRoundTrip`, `TestAntigravityCLIAdapterFakeFailsClosedOnMissingBinary` | T+V; no auth/lineage |
| R10 — Orgánica existente es sintética, opt-in | [`TestAntigravityCLIAdapterOrganicTransport`, prompt](../../internal/reviewerprovider/antigravitycli_organic_test.go#L1-L102) | Misma fuente, no ejecutada | I+T+L |
| R11 — j4500 conduce mock, no modelo | [fixture](../../bench/journeys_antigravity_cli.go#L18-L86), [journey](../../bench/journeys_antigravity_cli.go#L427-L439) | Bench paquete PASS, driven no ejecutado | I+V+L |
| R12 — CI separa unit y run contra binario | [workflow](../../.github/workflows/ci.yml#L144-L171) | Declaración, no resultado de CI actual | I+L |

## Fuentes oficiales y observaciones locales

**Origen:** el padre recuperó contenido legible completo el **2026-10-06**, no resúmenes de buscador. El escritor no hizo llamadas web adicionales. Son documentos vivos; no se supone que correspondan exactamente a la ayuda de agy 1.3.0. Las citas cortas siguientes son nombres de campos/flags y literales reportados; las explicaciones son paráfrasis, no citas de frases reconstruidas.

| Fuente / sección temática | Citas cortas y contrato recibido | Uso permitido y cautela |
|---|---|---|
| [Headless — invocation/output/auth](https://antigravity.google/docs/cli/headless/) | `"--print"`, `"--prompt"`, `"text"`, `"json"`, `"stream-json"`; `-p` alias; respuesta stdout y diagnósticos stderr; credenciales interactivas cacheadas | Sintaxis declarada oficial; no prueba que auth esté disponible aquí |
| [Headless — JSON output](https://antigravity.google/docs/cli/headless/) | `"status"`, `"response"`, `"structured_output"` | Wrapper de salida cliente, distinto de JSON result del proveedor Go bajo output text |
| [Headless — models/failures](https://antigravity.google/docs/cli/headless/) | `"ERROR"`: modelo desconocido produce exit no cero; approval-unavailable tools soft-denied y run puede terminar exit 0 | Exit 0 no implica todas las tools completadas |
| [Headless — permissions/sandbox/timeouts](https://antigravity.google/docs/cli/headless/) | `"--sandbox"` default false; `"--dangerously-skip-permissions"` aprueba todas; `"--print-timeout"` default 5m; workspace read/write autoallow, shell Ask | Contrato oficial vivo; adapter no usa bypass ni print-timeout; no enforcement medido |
| [Headless — streaming input](https://antigravity.google/docs/cli/headless/) | `"stream-json"` en stdin exige mismo formato en stdout | No se usa en el adapter text actual |
| [Agents — custom agent definitions](https://antigravity.google/docs/cli/commands/agents/) | `"agent.md"`, `"name"`, `"description"`; `.agents/agents/<name>/agent.md`, `~/.gemini/config/agents/<name>/agent.md` | No demuestra todos los campos emitidos ni destino de install Gentle AI |
| [Rules — budgets/includes](https://antigravity.google/docs/rules/) | `"24,000 bytes"`, 24 KB tras includes; 20.000 tokens aggregate, mayores demoted a pointers | Tabs 2.0/CLI/IDE: cualificar superficie antes de extrapolar |
| [Hooks — lifecycle events/config](https://antigravity.google/docs/hooks/) | `"PreToolUse"`, `"PostToolUse"`, `"PreInvocation"`, `"PostInvocation"`, `"Stop"`, `"hooks.json"`; sin SessionStart en contenido recibido | Cautela multi-surface; handlers Go observados por separado |
| [Plugins — tab CLI, structure/manual install](https://antigravity.google/docs/plugins?tab=cli) | `"plugin.json"` obligatorio; agents/skills/rules/MCP/hooks opcionales; roots manuales `.agents/plugins`, `~/.gemini/config/plugins` | No confundir con staging/copia CLI-managed ni recarga probada |

### Fuente de releases de AGY

Hasta el 2026-10-19 este repositorio no nombraba **ninguna** fuente para las releases de Antigravity CLI. La fuente autoritativa es el repositorio `google-antigravity/antigravity-cli`: las releases se leen con `gh release list --repo google-antigravity/antigravity-cli` y el changelog vive en `CHANGELOG.md` en `main`; el CLI además trae `agy changelog` y el comando `/changelog`. De ahí salen las reglas del validador de plugins que rigen el bundle emitido: desde `1.4.0` `agy plugin validate` aplica las reglas del loader, con `--strict` y `--json`; desde `1.4.1` avisa cuando el `name` de una skill no coincide con su carpeta, cuando el cuerpo supera 24 KB, o cuando una skill queda salteada por repetir nombre, y exige `name` y `description` en los subagentes Markdown. Nivel **O** para el nombre del repositorio, **I** para el comportamiento observado del validador 1.4.2 sobre un plugin desplegado.

### Ayuda local frente a docs vivas

El padre informó estos comandos de inspección: `command -v agy`, `agy --version`, `agy --help`, `go version`. Resultado: `$HOME/.local/bin/agy`, v1.3.0 y Go1.27.1 linux/amd64. Help enumera `--agent`, `--print`, `--model`, `--sandbox`, `--output-format`, `--input-format`, `--json-schema`, `--dangerously-skip-permissions`.

Esta observación **O** prueba superficie anunciada de flags, no cumplimiento de sandbox/permisos ni contrato del frontmatter. La mención recibida de staging CLI-managed `~/.gemini/antigravity-cli/plugins` es observación externa separada, no cambio a `PluginDir` del código.

## Fuentes históricas e inmutables

| Fuente | Qué fundamenta | Qué no fundamenta |
|---|---|---|
| [agy-canon-first-routing](../../odd/tasks/agy-canon-first-routing.md) | Decisión de conservar protocolo único y retirar split doctrine | Estado actual de tests por sí solo |
| [agy-bundle-provenance-skills](../../odd/tasks/agy-bundle-provenance-skills.md) | Decisión de pin/gate y owner de skills | Pin actual si diverge del registro antiguo |
| [agy-rules-always-on](../../odd/tasks/agy-rules-always-on.md) | Rationale de carrier rules/fill y diagnóstico registrado de cliente antiguo | Prueba nueva de agy1.3.0 |
| [Orchestrator canónico fijado](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/orchestrator.md#L1-L121) | Doctrina fuente inspeccionada en checkout del mismo SHA | Runtime Pi portable completo |
| [Explorer canónico fijado](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/agents/gentle-ai-explore.md#L1-L25) | Tool Pi original y policy de fallback | Implementación del servidor agy |
| [Writer canónico fijado](https://github.com/Gentleman-Programming/gentle-shell/blob/0da07ce949952268b290bf751f22ad1202a40fc3/assets/orchestrator-writer.md#L1-L55) | Diferencia de background/profiles/superficies | Enforcing de esas reglas por agy |

## Ruta de auditoría sugerida

1. Revisar F4/F5 y P4/P5: rutas y efectos sobre disco.
2. Revisar A1/A5/A7: ownership y adaptación sin inventar runtime features.
3. Revisar H1/H2/H7/H9: contexto escrito frente a guard ejecutable.
4. Revisar R2/R3/R6/R8: aislamiento, admisión y autoridad.
5. Cotejar resultados en [verificación](09-verification.md), sin confundir paquete PASS con prueba orgánica.
