# 8. Decisiones, alternativas y costes

**Esta página distingue decisiones históricamente documentadas de explicaciones inferidas del código actual.** Un beneficio razonable no demuestra que el equipo lo haya elegido por ese motivo. Los registros ODD aportan contexto histórico; la funcionalidad vigente se verifica contra implementación y tests, no contra casillas marcadas.

[Índice](README.md) · [Comparación](07-comparison.md) · [Fuentes](10-source-index.md)

## Etiquetas de procedencia

| Etiqueta | Significado |
|---|---|
| **Registrada** | Existe un documento de decisión que declara motivo/objetivo |
| **Intención en código** | Comentario explica el diseño; no sustituye prueba del cliente |
| **Inferida** | Trade-off derivado del mecanismo actual, sin evidencia histórica suficiente del motivo |
| **Pendiente de producto** | Inconsistencia o elección no resuelta por esta documentación |

No se fabrican una cronología de «18 commits» ni resultados históricos a partir de títulos de documentación anterior.

## Tabla de decisiones y alternativas

| Decisión actual | Procedencia del motivo | Alternativa considerada en este análisis | Ventaja/coste verificable | Fuentes |
|---|---|---|---|---|
| Canon Gentle Shell + transpilador, no diez prompts mantenidos manualmente | **Registrada**: bundle-provenance-skills fija un único owner y pin | Copias manuales o importar literalmente Pi | Reduce fuentes de doctrina; añade riesgo de mutaciones textuales y pin obsoleto | [Registro](../../odd/tasks/agy-bundle-provenance-skills.md), [`transpileSkills`](../../scripts/transpile-antigravitycli/main.go#L667-L713) |
| Routing canon-first con protocolo ODD único | **Registrada**: canon no incluye esa inyección runtime Pi, resto de doctrina ya existe | Duplicar RenderRouting completo o mantener split rules | Reduce duplicación; exige preservar routing y canon como dos piezas complementarias | [Registro](../../odd/tasks/agy-canon-first-routing.md), [`injectAntigravityCLIRule`](../../internal/components/agentguidance/inject.go#L320-L382) |
| Orchestrator en rules con always_on y fill | **Registrada**: el registro rules-always-on describe caída de reglas sin trigger y asset root no cargado | Root-only, inlining íntegro en AGENTS.md, o loading manual | Hace explícita superficie cargable; presión de bytes/tokens y dependencia de client behavior | [Registro](../../odd/tasks/agy-rules-always-on.md), [`ensureAntigravityOrchestratorRule`](../../internal/cli/antigravity_orchestrator_rule.go#L28-L52) |
| Satélites lazy fuera de rules | **Registrada**: rules-always-on señala tamaño de delegation y lectura on-demand | Todo siempre en contexto o splitting exhaustivo | Reduce coste inicial; modelo debe leer el satélite correcto, no garantía automática | [Registro](../../odd/tasks/agy-rules-always-on.md), [`BundleAssets`](../../internal/agents/antigravitycli/adapter.go#L192-L207) |
| Plugin propietario con pruning de agents/skills | **Registrada** para skills, **inferida** para beneficio de árbol exacto de agentes | Mezclar contenido administrado y personalizado sin poda | Retira stale assets; puede borrar personalizaciones dentro del namespace propio | [Registro](../../odd/tasks/agy-bundle-provenance-skills.md), [`DeployPluginTree`](../../internal/agents/antigravitycli/adapter.go#L589-L625) |
| Primer import staged; registro existente evita reimport | **Intención en código**: source distinto porque agy rechaza source=dest; snapshot preserva rewrite | Import directo sobre mismo path o reimport en cada sync | Evita caso source=dest y churn de timestamp; deja pendiente semántica de recarga del cliente | [`EnsurePluginImported`](../../internal/agents/antigravitycli/adapter.go#L640-L732), [test merge](../../internal/agents/antigravitycli/adapter_test.go#L268-L343) |
| Adaptador mínimo, autoridad compartida Go | **Intención/guard de arquitectura** | Lifecycle/admisión dentro de adapter o autoridad en prompts | Menor duplicación de dominio; requiere leer capas compartidas para comprender fallos | [`TestAdapterMinimalityGuard`](../../internal/reviewerprovider/adapter_minimality_guard_test.go#L18-L40), [`Contract`](../../internal/reviewerprovider/contract.go#L112-L143) |
| `--sandbox`, workspace vacío y HOME auth-only temporal | **Decisión explícita del usuario** tras recuperar un marcador global ajeno a stdin | Conservar HOME o esperar una opción nativa de no-discovery | Excluye configuración copiada; modelo/effort quedan en default del propio AGY si no se configuran; confinamiento de herramientas de archivo demostrado AUSENTE (probe diferencial), red/MCP sin medir | [`Review`, env](../../internal/reviewerprovider/antigravitycli_adapter.go#L33-L103) |
| Stdin sin `-p/--print`, salida text raw | **Intención en comentario** sobre consumo de flags | Prompt en argv, salida JSON wrapper, `--json-schema` | Evita material en argv y parseo en adapter; core debe extraer/admitir resultado | [`Review`](../../internal/reviewerprovider/antigravitycli_adapter.go#L33-L78), [`reviewProviderExtractRoleRaw`](../../internal/cli/review_provider.go#L81-L99) |
| Modelos/effort HOME-owned; `inherit` por defecto (decisión explícita del usuario), explícitos opcionales, malformed cerrado | **Decisión explícita del usuario** (Log L1 en [odd/tasks/agy-reviewer-inherit-effort.md](../../odd/tasks/agy-reviewer-inherit-effort.md)); repo no prescribe modelos canónicos | Repo fallback, defaults silenciosos ante corruption, modelos fijos por prompt o exigir modelo explícito | Preserva control del usuario; el default aislado es el del propio AGY, no el del perfil habitual; corruption bloquea | [`ResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing.go#L359-L425) |
| Hooks neutros selectivamente, deny en pre-tool | **Inferida** de branches actuales; comentario genérico fail-open no manda | Fallar siempre abierto o abortar toda sesión ante cualquier error | Evita bloqueo en post/resolución stop; protege formato/loops conocidos en pre | [`RunAntigravityCLIHook`](../../internal/cli/antigravity_hook.go#L88-L148), [pre-tool](../../internal/cli/antigravity_hook.go#L405-L476) |
| Evidencia separada en fuente/unit/mock/driven/orgánica | **Decisión documental de esta tarea**, basada en fixtures | Tratar cualquier PASS o help como proof de paridad | Evita falsa confianza; mantiene explícitos límites no medidos | [Mock](../../bench/journeys_antigravity_cli.go#L27-L67), [organic](../../internal/reviewerprovider/antigravitycli_organic_test.go#L1-L102), [registro](09-verification.md) |
| Los guards de assets por runtime excluyen el bundle transpilado y delegan su cobertura en el gate de drift y en el transpilador | **Decisión documental de esta tarea**, derivada del mecanismo actual | Incluir `antigravitycli/` en esos guards, o sembrar a mano el placeholder de secciones ODD en el core | El bundle no se puede editar sin romper el drift gate, así que un guard ahí fija una revisión ajena; a cambio, el core queda fuera de esas aserciones y su cobertura debe leerse en el mapa de la sección 8 | [`isTranspiledBundleAsset`](../../internal/assets/planned_path_gate_test.go#L15-L46), [`inject.go`](../../internal/components/agentguidance/inject.go#L138-L151) |
| `antigravity-cli` convive con `antigravity` en vez de reemplazarlo | **Decisión explícita del usuario** durante esta remediación | Renombrar la entrada heredada, fusionar ambos agentes en uno, o mover el agente nuevo a otro id | Ninguna instalación existente cambia de destino y el PR puede añadir sin romper; a cambio quedan dos entradas seleccionables que resuelven su configuración bajo el mismo árbol `~/.gemini`, y esa convivencia hay que documentarla | [Catálogo](../../internal/catalog/agents.go#L26-L27), [sección 9](#9-antigravity-cli-convive-con-antigravity-en-vez-de-reemplazarlo) |

Las alternativas de la tercera columna son comparación técnica **de esta guía**. Salvo que el registro lo diga, no se afirma que históricamente se probaron y descartaron.

## 1. Un canon compartido no implica un runtime compartido

La fuente canónica separa disciplina del harness de contratos dinámicos RDD y memory. La adaptación agy mantiene doctrina pero elimina secciones Pi y cambia rutas/primitivas: [`applyAntigravityCLIParentMutations`](../../scripts/transpile-antigravitycli/main.go#L1341-L1452).

### Frente a edición manual

**Beneficio inferido:** cuando cambia una instrucción de canon, el pin y un diff de regeneración permiten revisar de dónde proviene. La edición manual en assets pierde esa relación y hace que `--check` falle en HEAD.

**Coste observable:** una sustitución textual puede dejar una instrucción incoherente aun con Markdown válido. Los tests negativos conocidos protegen residuos, paths y fallback, no equivalencia total de funciones.

### Frente a copiar Pi literalmente

El ledger compartido Pi describe SessionManager/relay/runtime específicos y se excluye: [`transpileSharedDirExcludingReviewLedger`](../../scripts/transpile-antigravitycli/main.go#L714-L737). Copiarlo completo introduciría un segundo lifecycle textual contradictorio. El contrato agy correcto se materializa desde Go, no desde ese fragmento.

## 2. Límite de archivo frente a splitting

El diseño vigente no corta todo en trozos arbitrarios. Mantiene el protocolo ODD en un archivo pequeño y deja doctrina/satélites en su owner canónico. El registro histórico canon-first dice explícitamente retirar machinery de splitting y conservar stale cleanup.

**Beneficio inferido:** menos versiones simultáneas del mismo routing y menos archivos always-on.

**Coste:** el orchestrator final sigue añadiendo un contrato sustancial; su test de bytes no mide includes ni tokens aggregate de agy. [Detalle de límites](03-orchestration-and-hooks.md#límite-de-archivo-versus-límite-aggregate).

No reemplazar este hecho por «Gentle AI mitiga cualquier truncación»: solo se protege el layout/test conocido.

## 3. Ownership/pruning frente a extensibilidad dentro del plugin

El producto usa un namespace concreto y poda directorios que no encuentra embedded. Es una decisión coherente con bundle-owned, pero no una garantía de preservar extensiones personales allí.

**Alternativa inferida:** preservar todo directorio ajeno facilita customización, pero retiene nombres obsoletos y reduce la posibilidad de describir un set instalado exacto.

**Regla operativa:** no convertir el árbol administrado en un workspace del usuario. La preservación marker-based de routing y la conservación de JSON import ajeno son contratos diferentes del pruning de directorios. [Matriz de propiedad](06-asset-ownership-and-drift.md).

## 4. Captura directa frente a relay del host

Agy se invoca dentro del comando Go; no precisa el relay OpenCode/Pi para devolver raw bytes. [`reviewImmutableRuntimeCapability`](../../internal/cli/review_transport_capability.go#L83-L154).

**Beneficio inferido:** menos protocolo entre host y capture para este runtime.

**Costes:** el Go caller necesita resolver `agy`, credenciales/config desde HOME y ejecutar un proceso externo; sus restricciones no son las del host parent. La simplicidad del adapter desplaza parsing, retry y binding a capas compartidas, no los elimina. Un error local/auth puede fallar antes de admisión.

No inferir «review manual chain es el executor» por la presencia de cuatro lenses en assets: la cadena declara manual/compat.

## 5. Lectura/sandbox y el compromiso con auth

El código usa HOME temporal con una copia mínima del token OAuth y el installation_id opcional, sin copiar reglas/settings/plugins/MCP. La autenticación con esos archivos se observó en AGY 1.3.5; el perfil original no se escribe. Los defaults de modelo no se copian: el usuario autorizó exigir modelo explícito.

**Hecho:** `--sandbox` está siempre solicitado por el adapter.

**Intención:** comments describen aislamiento y stdin sin flags print.

**Pendiente:** efectividad por plataforma (Windows no probado), red y MCP. File tools: el probe diferencial demostró que `--sandbox` NO confina lecturas/escrituras de archivos fuera del scratch ([L10](../../odd/tasks/agy-reviewer-inherit-effort.md)); el contexto global sí quedó cerrado y verificado. Las regresiones cubren el contenido del perfil, la limpieza en éxito y error (con destrucción best-effort de credenciales y warning observable si no se confirma), el rechazo de enlaces en directorios intermedios y la ausencia opcional de `installation_id`; no reemplazan la repetición orgánica del señuelo global. [Frontera](05-review-transport.md#aislamiento-solicitado-no-cleanroom-demostrado).

## 6. Defaults de modelos y alcance

El resolver establece una preferencia global de usuario, fuera del candidato. Malformed no vuelve silenciosamente a un modelo o effort diferente. Archivo ausente, key ausente o `inherit` dejan el rol sin configurar: el transporte omite `--model` (y `--effort` si no hay tier) y el reviewer aislado resuelve el default del propio AGY; key ausente en archivo existente añade diagnóstico.

**Beneficio inferido:** no sorprender con un fallback controlado por el repositorio revisado.

**Coste:** con `inherit`, el default efectivo es el del propio AGY en el perfil aislado — no el del perfil habitual ni el de la sesión interactiva — y no se elige por rol explícitamente. Un effort inválido bloquea la revisión (fail-closed). Presets hardcoded pueden caducar o no estar disponibles para una cuenta. No se midió relación coste/calidad de Recommended/Economy.

**Pendiente de producto:** el flag workspace no tiene una entrega agy local coherente demostrada; documentación oficial de `.agents` no corrige las rutas implementadas. [Análisis concreto](01-foundation.md#alcance-workspace-no-es-un-camino-garantizado).

## 7. Fail-open frente a fail-closed: decidir por frontera

No hay una única política de fallo para toda la integración:

- Manifest malformed → detener antes del comando externo.
- Review routing malformed → error tipado.
- Review subprocess failure/empty → no bytes.
- Provider output inválido → admisión/corrección controlada.
- PreToolUse malformed o loop → deny.
- Post hooks → neutral.
- Stop sin condiciones o con error de status → neutral, no gate fuerte.
- MCP malformed → documento nuevo, caller debe respaldar.

Esta asimetría está en el código. Presentarla explícitamente es más útil que «resiliente y fail-open»: dice al operador cuál señal buscar y qué no esperar. [Hooks](03-orchestration-and-hooks.md#por-qué-todos-fail-open-es-incorrecto), [MCP](04-codegraph-and-benchmarks.md#reconciliación-y-controles-negativos), [import](06-asset-ownership-and-drift.md#primer-import-y-registro).

## 8. Los guards de assets por runtime excluyen el bundle transpilado, y por qué

`internal/assets/antigravitycli/` es el único directorio de runtime que este repositorio no escribe: `scripts/transpile-antigravitycli` lo genera desde la fuente de Gentle Shell, y el gate de drift compara el árbol commiteado contra una transpilación nueva **byte a byte en las dos direcciones** ([`compareTrees`](../../scripts/transpile-antigravitycli/main.go#L1505)). Un archivo escrito a mano ahí no se corrige: falla CI, así que un guard que lo inspeccione no puede arreglar nada que encuentre.

Tres guards de assets por runtime lo excluyen de forma nombrada y explicada, no por descuido:

| Guard | Qué verifica en los runtimes que este repositorio escribe | Por qué no aplica al bundle |
|---|---|---|
| [`TestEveryODDOrchestratorDoesNotRejectPlannedPaths`](../../internal/assets/planned_path_gate_test.go#L52) | Que cada `orchestrator.md` lleve `GENTLE_AI_ODD_SECTION:` y no rechace rutas planificadas sin el carve-out; además cierra el inventario en doce runtimes | El core de agy es compacto y rutea a módulos por regla en vez de llevar las secciones ODD compartidas: no hay placeholder que afirmar, y la expansión de placeholders solo corre para los runtimes a los que `agentguidance` les renderiza orchestrator, exclusión que hoy incluye a Antigravity CLI ([`inject.go`](../../internal/components/agentguidance/inject.go#L146), [`expandSharedOrchestratorSections`](../../internal/components/agentguidance/orchestrator.go#L285)) |
| [`orchestratorSectionVariants`](../../internal/assets/orchestrator_drift_ratchet_test.go#L33) | Anclas de seguridad de prompts («Lossless Blocking Prompts», handoff de defectos) y ratchet de variantes por sección | Fijaría en este repositorio la redacción de un asset que pertenece a Gentle Shell |
| [`allSDDOrchestratorAssetPaths`](../../internal/assets/language_contract_test.go#L312) | Contrato de idioma, alcance del reporte de defectos y orden de formateo sobre los doce orquestadores escritos acá | El texto es transpilado; su idioma se decide en la fuente |

**Qué cubre al bundle, entonces:** el gate de drift (bytes y conjunto de archivos contra una transpilación nueva), los 15 tests del propio transpilador ([`main_test.go`](../../scripts/transpile-antigravitycli/main_test.go)) y los del adaptador, que verifican el despliegue de `rules/` y satélites sin depender de su redacción ([`renderNativeRule`](../../internal/agents/antigravitycli/native_rules.go#L31), que solo reescribe rutas y no expande secciones).

**Límite explícito:** el contenido del core por secciones no lo cubre nadie hoy. Una aserción module-aware que recorra los módulos por regla del bundle es una propuesta para upstream, no un cambio de este fork; mientras tanto, este mapa es la declaración de dónde mirar cuando la doctrina de agy cambie.

## 9. `antigravity-cli` convive con `antigravity` en vez de reemplazarlo

**Decisión explícita del usuario durante esta remediación:** la integración de Antigravity CLI **no reemplaza** a la entrada heredada; ambas conviven a propósito. El catálogo quedó con dos agentes que comparten ruta de configuración ([`agents.go`](../../internal/catalog/agents.go#L26-L27)):

| Agente | Nombre en el catálogo | Qué escribe | Dónde |
|---|---|---|---|
| `antigravity` (heredada, de upstream) | Google Antigravity | Prompt de sistema y settings de la app/CLI de Antigravity | `~/.gemini/GEMINI.md`; settings en el variant dir, que es `~/.gemini/antigravity-desktop` si existe y si no `~/.gemini/antigravity-cli`; pasa a `~/.gemini/config` cuando existe su marcador de migración ([`adapter.go`](../../internal/agents/antigravity/adapter.go#L29-L35), [`SystemPromptFile` y `SettingsPath`](../../internal/agents/antigravity/adapter.go#L100-L110)) |
| `antigravity-cli` (la que agrega el fork) | Antigravity CLI | El bundle transpilado de Gentle Shell como plugin y sus reglas nativas | `~/.gemini/config/plugins/gentle-ai` en global y `<workspace>/.agents/plugins/gentle-ai` en workspace ([`PluginDir`](../../internal/agents/antigravitycli/adapter.go#L116-L196)) |

**Por qué se documenta en vez de unificarse:** las dos entradas resuelven su configuración bajo el mismo árbol `~/.gemini`, así que la duplicación salta a la vista y es el tipo de cosa que alguien «arregla» renombrando o fusionando entradas. Renombrar la heredada cambiaría el destino de instalaciones existentes, y fusionar los dos ids obligaría a adivinar cuál de los dos harness quiso seleccionar el usuario. La regla de esta remediación también lo pide: un cambio que afecta a agy y a otro agente se trata como que afecta a los dos, y acá el «otro» es la entrada heredada de upstream, que este fork **no toca**.

**Consecuencia operativa:** cada id tiene su propio camino de limpieza. El desinstalador trabaja por agente y reporta por agente qué completó y qué no ([`Result`](../../internal/components/uninstall/service.go#L48-L62)), y el arreglo de permisos de esta remediación trata a `antigravity-cli` como un caso aparte del de la entrada heredada.

**Límite no medido:** no se probó una instalación cruzada (instalar una entrada, desinstalar la otra y medir el árbol resultante). Queda anotado en la misma línea que el resto de esta página: no se afirma paridad ni aislamiento entre ambas.

## 10. El protocolo ODD no viene del canon: exclusión aceptada por runtime

**Decisión ratificada por el usuario** en la remediación de canon-first: agy es canon-first para la doctrina de routing, y el protocolo ODD obligatorio —los siete pasos más la línea de recuperación— es la **única** pieza que el canon no carga, porque es un contrato de runtime y no texto de doctrina. La exclusión está aceptada y es explícita, no accidental.

**Mapa verificado por grep sobre el pin `d5d692a6`:**

| Pieza | Dónde vive | ¿Lleva el protocolo ODD? |
|---|---|---|
| Canon de Shell (escalera de routing, delegación, TDD, RDD, feature-doc) | `gentle-shell/assets/` → transpilado a `internal/assets/antigravitycli/` | **No**: 0 ocurrencias en todo `assets/` |
| Protocolo ODD de Pi | `gentle-shell/extensions/gentle-ai.ts:1336` (`ODD_PROTOCOL_HEADING`) y `:1462` (la inyección en vivo); desde 4.2.0 espejado del bundle del contrato (`routing/pi.md`, `:1355`) | **Sí** |
| Protocolo ODD del resto de runtimes | El renderer del binario, [`writeODDWorkflowProtocol`](../../internal/components/agentguidance/routing.go#L50) | **Sí** |
| Regla slim de agy | `rules/gentle-ai-routing.md`, escrita por [`injectAntigravityCLIRule`](../../internal/components/agentguidance/inject.go#L351) sobre la ruta canónica de [`inject.go`](../../internal/components/agentguidance/inject.go#L285) | **Sí**, y sólo eso |

**Por qué agy no puede recibir el ODD de Pi.** El bloque que upstream 4.2.0 añadió como `piODDSteps` ([`routing.go`](../../internal/components/agentguidance/routing.go#L23), commit `ef7ff753e`) no es la redacción genérica: es la que describe el harness de Shell. Nombra el `todo` visible que el harness proyecta (*"then create or rebuild the visible `todo` list"*), el label de fase que el harness infiere de la actividad de la sesión (*"the Gentle Shell prompt label is inferred automatically from the primary session's tool activity"*) y, según el propio mensaje de upstream, el enrutado por los *Mechanisms* del orquestador, el cierre del paso 7 y los punteros a assets. **En agy nada de eso existe**: no hay `todo` proyectado, ni label de fase, ni Mechanisms, ni punteros a assets del harness. Entregar ese texto a agy sería ordenarle al agente que ejecute cosas que no puede, y el modelo obedecería con fictionalidad. Por eso el renderer entrega a agy la redacción genérica y la de Pi queda detrás de `if agent == model.AgentPi`.

**Efecto operativo:** el bundle de agy entrega trece reglas, todas del canon salvo una: `gentle-ai-routing.md` (6,9 KB) lleva exactamente el protocolo. Presupuesto always-on: AGENTS.md 17,4 KB + orchestrator 22,5 KB + slim ~4 KB ≈ 44 KB.

### Frente a darle a agy el ODD de Pi

**Beneficio inferido:** una sola redacción del protocolo en todo el ecosistema, y el texto de Pi y el de agy no pueden divergir porque salen del mismo binario.

**Coste observable:** el texto de Pi describe capacidades del harness de Shell que agy no tiene; el agente en agy recibiría instrucciones sobre un `todo` visible y un label de fase que nadie produce.

### Frente a meter el protocolo dentro del canon

**Beneficio inferido:** agy lo recibiría sin regla propia y el bundle no tendría una pieza que no viene de Shell.

**Coste observable:** el canon es doctrina compartida por runtime y el protocolo es contrato de runtime: el propio upstream lo mantiene fuera de los assets y lo espeja desde el bundle del contrato. Meterlo en el canon convertiría una exclusión declarada en una convención silenciosa, y `gentle-shell` sigue siendo la fuente de la que el bundle se transpila byte a byte.

### Frente a duplicar el protocolo en el canon y en la regla slim

**Beneficio inferido:** ninguna de las dos fuentes depende de la otra.

**Coste observable:** dos redacciones del mismo procedimiento que divergen con el próximo cambio del canon, y el bundle deja de ser una transcripción fiel de Shell.

**Quién lo fija:** el contrato protocol-only de agy está pineado por [`TestInjectRoutingAntigravityCLIWritesProtocolOnlyRule`](../../internal/components/agentguidance/inject_test.go#L1003) y [`TestInstallAndSyncKeepAntigravityCLIRoutingGuidanceRoutingOnly`](../../internal/cli/orchestrator_install_test.go#L202), y la frontera desplegada por [`assertAntigravityCanonicalRouting`](../../internal/components/agentguidance/routing_test.go#L1123), que despliega el plugin y verifica las obligaciones reales en `rules/`. La decisión de diseño, con su mapa verificado, está en [`odd/tasks/agy-canon-first-routing.md`](../../odd/tasks/agy-canon-first-routing.md).

## Decisiones no resueltas por esta documentación

| Límite | Qué falta para resolverlo | Qué no se hizo |
|---|---|---|
| Workspace/hogar mezclados | Diseño de destinos y test de no-mutation HOME | Cambiar rutas o recomendar happy path local |
| Sandbox y discovery del reviewer | Ensayo real positivo/negativo agy, por versión/plataforma | Auth, llamadas remotas o pruebas destructivas |
| Copia administrada del cliente/import recarga | Observar dónde carga y actualiza el plugin agy actual | Reinstalar para forzar un resultado |
| Presets y defaults efímeros | Modelo disponible y resolución del proceso nuevo | Descubrir catálogo remoto o inventar precios |
| Total token budget de reglas | Medición de carga/includes de selección real | Deducir tokens desde bytes |
| Semver de Gentle Shell | Tag/release verificable asociado al SHA | Inventar versión desde package name o fecha |

**Siguiente paso:** [registro reproducible con resultados y controles negativos](09-verification.md).
