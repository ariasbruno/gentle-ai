# 3. Orquestación, reglas y hooks

**La integración combina política de prompts con guards ejecutables concretos.** El orchestrator describe cómo coordinar; `PreToolUse` valida un formato y detecta repetición; `Stop` puede recordar un preflight. Ninguna de esas piezas debe confundirse con aprobación de entrega.

[Índice](README.md) · [Transpilación](02-asset-transpiler.md) · [RDD](05-review-transport.md)

## Reglas y carga de contexto

| Archivo instalado | Qué lo construye | Qué contiene |
|---|---|---|
| `rules/AGENTS.md` | Componentes mediante secciones administradas | Carrier del system prompt, persona y guidance según selección |
| `rules/gentle-ai-routing.md` | `injectAntigravityCLIRule` | **Solo protocolo ODD**, frontmatter y marker |
| `rules/gentle-ai-orchestrator.md` | Bundle más finalización CLI | Coordinador transpilado y contrato de revisión renderizado |
| Satélites en raíz | `BundleAssets` | Detalle de delegación, memoria, skills, prompts, tracking, verificación y writer; lectura bajo demanda |

Fuentes: [`SystemPromptFile`](../../internal/agents/antigravitycli/adapter.go#L128-L138), [delivery agy](../../internal/components/agentguidance/inject.go#L260-L273), [`BundleAssets`](../../internal/agents/antigravitycli/adapter.go#L192-L207), [`ensureAntigravityOrchestratorRule`](../../internal/cli/antigravity_orchestrator_rule.go#L28-L52).

El contrato implementado antepone `trigger: always_on` a las reglas no-AGENTS. Los comentarios documentan que otras reglas sin trigger eran omitidas por el cliente y que `AGENTS.md` permanece plain Markdown: [inject.go, líneas 297–318](../../internal/components/agentguidance/inject.go#L297-L318). El [registro histórico](../../odd/tasks/agy-rules-always-on.md) describe un diagnóstico con agy 1.2.12. Ese relato no se presenta como ensayo nuevo de 1.3.0 ni como fuente oficial universal.

### Límite de archivo versus límite aggregate

La documentación oficial [Rules](https://antigravity.google/docs/rules/), recuperada por el padre el 2026-10-06, especifica **24 KB (24.000 bytes) tras includes** y un límite agregado de **20.000 tokens**, con democión de los archivos mayores a pointers. Contiene tabs/superficies 2.0, CLI e IDE: hay que comprobar aplicabilidad, no transferir automáticamente todo el documento al binario instalado.

El producto define `antigravityCLIRuleCap = 24000` y los tests exigen tamaños estrictamente menores. Son dos medidas diferentes:

- Bytes embedded/finales locales: comprobables por Go.
- Bytes después de includes y tokens/carga efectiva del cliente: requieren observación agy.

No se afirma que los tres archivos finales de cualquier selección estén siempre bajo ambos límites. `TestAntigravityCLI_RuleSizeLimit` inspecciona el orchestrator embedded; el test CLI comprueba el final con contrato. No son una medición aggregate de cada instalación: [asset test](../../internal/assets/antigravitycli_assets_test.go#L346-L357), [CLI test](../../internal/cli/antigravity_orchestrator_rule_test.go#L13-L60).

### Por qué ya no hay splitting de routing

El writer actual crea un solo archivo de protocolo. Limpia archivos `gentle-ai-routing*.md` fuera del set escrito **solo si contienen marker propio**. Un archivo sin marker sobrevive: [`injectAntigravityCLIRule`](../../internal/components/agentguidance/inject.go#L320-L382), [`TestInjectRoutingAntigravityCLIStalePartCleanup`](../../internal/components/agentguidance/inject_test.go#L1086-L1140).

La decisión histórica «canon-first» evita repetir doctrina ya presente en satélites y conserva el protocolo que los assets no llevan: [registro](../../odd/tasks/agy-canon-first-routing.md). No describir los viejos split parts como estrategia vigente.

## Orchestrator: coordinación escrita

El [asset actual](../../internal/assets/antigravitycli/orchestrator.md#L11-L98) establece coordinación, tamaño por comprensión/riesgo/resumibilidad, preguntas, exploration, verificación independiente en alto riesgo y superficies derivadas por el padre. Son **instrucciones al modelo**.

```text
usuario autoriza objetivo
  → padre deriva alcance y criterios
  → lectura/mapeo acotado si falta comprensión
  → implementación inline o worker por razón explícita
  → checks y verificador cuando corresponde
  → padre sintetiza y conserva decisiones de entrega
```

El asset usa `invoke_subagent` y prescribe foreground. La variante original Pi utiliza mecanismos propios de background/runtime; la mutación no incorpora su implementación. Un texto que diga «runtime-enforced» no basta para probar enforcing de superficies en agy. Véase [diferencias de adaptación](02-asset-transpiler.md#mutaciones-que-marcan-diferencias-con-pi).

La cadena [4r-review.chain.md](../../internal/assets/antigravitycli/chains/4r-review.chain.md#L1-L38) se declara **manual/compat-lane**; native review captures no la cargan. No convertir sus cuatro secciones en una secuencia RDD autorizada por el modelo.

## Finalización del contrato de revisión

El bundle trae `<!-- antigravity-review-execution-contract:insert -->`. El paso CLI:

1. Lee la regla desplegada.
2. Renderiza `reviewassets.ReviewExecutionContractFor(AgentAntigravityCLI)`.
3. Reemplaza el marker una vez, o acepta un contrato ya rellenado.
4. Falla si no hay ninguno de ambos.
5. Anteponer frontmatter y escribir atómicamente.

[`ensureAntigravityOrchestratorRule`](../../internal/cli/antigravity_orchestrator_rule.go#L28-L52). El test comprueba marker ausente, secciones del contrato, tamaño y bytes idénticos tras segunda finalización; otro test rechaza falta de deploy: [tests, líneas 13–68](../../internal/cli/antigravity_orchestrator_rule_test.go#L13-L68).

Esto prueba materialización en fixture, no cumplimiento por el modelo.

## Hooks: contrato registrado

El plugin registra cinco eventos. No registra `SessionStart`. La página oficial [Hooks](https://antigravity.google/docs/hooks/) recuperada el 2026-10-06 enumera esos eventos y `hooks.json`, con la misma cautela de tabs/superficies.

| Evento | Timeout en hooks.json | Handler | Efecto |
|---|---:|---|---|
| `PreInvocation` | 30 s | `handleAntigravityPreInvocation` | Primera invocación: refresh/baseline best-effort; salida `{}` |
| `PreToolUse` | 10 s | `handleAntigravityPreToolUse` | Formato de delegación y circuit breaker; allow/deny |
| `PostToolUse` | 10 s | `handleAntigravityPostToolUse` | No-op portable: `{}` |
| `PostInvocation` | 30 s | `handleAntigravityPostInvocation` | No-op portable: `{}` |
| `Stop` | 60 s | `handleAntigravityStop` | Solo stop idle/model: evaluar recordatorio de revisión |

Definición: [`antigravityCLIPluginHooksJSON`](../../internal/agents/antigravitycli/adapter.go#L407-L470); dispatch y handlers: [antigravity_hook.go, líneas 88–172](../../internal/cli/antigravity_hook.go#L88-L172), [Stop](../../internal/cli/antigravity_hook.go#L212-L282). Timeouts son configuración solicitada al cliente, no medición de ejecución.

El comando registrado llama directamente `gentle-ai hook run --agent=antigravity-cli --event <evento>`. También se escriben `hook.sh` y `hook.cmd`; no afirmar que hooks.json siempre los invoca: [`DeployPluginTree`, líneas 530–556](../../internal/agents/antigravitycli/adapter.go#L530-L556).

## `PreToolUse`: condiciones fail-closed

### Validación del payload

- JSON vacío, null, array o malformed en `PreToolUse` → `deny`.
- `invoke_subagent` debe tener `Args.Subagents` array no vacío.
- Cada entrada requiere strings no vacíos `TypeName`, `Role`, `Prompt`.
- `Workspace`, si aparece, debe ser `inherit`, `branch` o `share`.
- Una herramienta distinta no recibe esa validación especial.

[`RunAntigravityCLIHook`](../../internal/cli/antigravity_hook.go#L88-L108), [`validateAntigravityCLIInvokeSubagentArgs`](../../internal/cli/antigravity_hook.go#L344-L383), [`antigravityCLIValidWorkspace`](../../internal/cli/antigravity_hook.go#L394-L401).

Ejemplo de **forma esperada por el handler**, no prueba oficial de schema nativo completo:

```json
{
  "toolCall": {
    "name": "invoke_subagent",
    "args": {
      "Subagents": [{
        "TypeName": "gentle-ai-explore",
        "Role": "explorer",
        "Prompt": "Mapear solo los archivos autorizados y devolver referencias.",
        "Workspace": "inherit"
      }]
    }
  }
}
```

El handler no demuestra que `TypeName` exista, no analiza si el prompt respeta todas las superficies ni autoriza un rol RDD por recibir ese string. Tests: [`PreToolUseEdgeCases`](../../internal/cli/antigravity_hook_test.go#L341-L375), [`TestAntigravityHookPreToolUseMalformedInvokePayloadFailsClosed`](../../internal/cli/antigravity_hook_test.go#L486-L499), [`TestAntigravityHookPreToolUseInvokeSubagentValidation`](../../internal/cli/antigravity_hook_test.go#L530-L635).

### Circuit breaker

Con `conversationId` y herramienta válida:

```text
(nombre + hash JSON canónico de args) idénticos
  llamada 1 → allow, count 1
  llamada 2 → allow, count 2
  llamada 3 → deny, count 3
  cambio de nombre/args → reinicia count 1
```

[`handleAntigravityPreToolUse`](../../internal/cli/antigravity_hook.go#L405-L476). Se persiste bajo `.gentle-ai/circuit-breaker/v1/<sha256(conversationId)>.json`, con fallback temporal si HOME no está disponible. El orden de claves de JSON no evade identidad: [`PreToolUseKeyOrdering`](../../internal/cli/antigravity_hook_test.go#L301-L339).

Sin conversationId, toolCall o nombre, retorna allow y no puede seguir repetición: [líneas 409–411](../../internal/cli/antigravity_hook.go#L409-L411). No es un detector de todos los loops ni de equivalencia semántica de comandos; solo repetición consecutiva de nombre/args. Persistencia fallida puede debilitar seguimiento entre invocaciones; no se trata de garantía universal anti-loop.

## `PreInvocation`: no siempre es pasivo

Cuando `InvocationNum == 0`, el código intenta refrescar skill registry, asegurar ignore de `.atl` y registrar baseline RDD si puede resolver target. Emite `{}` aun cuando esas operaciones best-effort fallen: [líneas 151–172](../../internal/cli/antigravity_hook.go#L151-L172).

Por tanto, no ejecutar un hook manual de inicio como si fuera un read-only probe. `TestAntigravityHookPreInvocationBaselineAndSilent` utiliza HOME/repositorio fixture y comprueba baseline y silencio: [test, líneas 12–58](../../internal/cli/antigravity_hook_test.go#L12-L58).

## `Stop`: recordatorio condicionado, no gate de entrega

Solo entra en evaluación si `terminationReason == "model_stop"` **y** `fullyIdle == true`. Además debe resolverse repo/modo RDD y status con target no vacío y `Action=start`. Baseline/reminder previo del mismo target suprimen repetición; ciertos errores y ausencia de modo retornan `{}`: [`handleAntigravityStop`](../../internal/cli/antigravity_hook.go#L212-L282).

Salida posible:

```json
{
  "decision": "continue",
  "reason": "Receipt-Driven Development: unreviewed candidate changes detected since session baseline. Please run gentle-ai review status preflight before completing."
}
```

No inicia por sí mismo reviewers ni crea aprobación. El test `TestAntigravityHookStopOutcomes` comprueba idle/model positivo, no-idle/no-model neutros y segundo recordatorio suprimido: [líneas 60–137](../../internal/cli/antigravity_hook_test.go#L60-L137).

## Por qué «todos fail-open» es incorrecto

| Caso | Salida del producto |
|---|---|
| Lectura stdin falla | `{}` |
| `PreToolUse` malformed | `deny` |
| Delegación malformed | `deny` |
| Tercera llamada idéntica | `deny` |
| Post hooks | `{}` |
| Stop no calificante/error de resolución | `{}` |
| Stop calificante con candidato nuevo | `continue` |
| CLI invocada con agente/subcomando incorrecto | Error de parsing/validación |

Fuente completa: [`RunHook`, `runHookCommand`, `RunAntigravityCLIHook`](../../internal/cli/antigravity_hook.go#L52-L137), [circuit breaker](../../internal/cli/antigravity_hook.go#L440-L476). Un proceso que sale exitosamente puede emitir deny. «Exit 0» y «permitido» son preguntas distintas.

**Siguiente paso:** [MCP y la prueba limitada del benchmark](04-codegraph-and-benchmarks.md).
