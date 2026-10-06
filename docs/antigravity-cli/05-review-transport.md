# 5. Transporte RDD, modelos y fronteras de prueba

**Go conserva el contrato de revisión; agy aporta bytes de una respuesta.** El adaptador no decide aprobación, recibos, correcciones ni entrega. Su aislamiento está solicitado parcialmente mediante flags y scratch; no se ha medido aquí el enforcement de agy.

[Índice](README.md) · [Comparación](07-comparison.md) · [Verificación](09-verification.md)

## Identidad y admisión del runtime

Son tres preguntas diferentes:

1. **Identidad registrada:** `RegisteredRuntime("antigravity-cli")` pertenece al conjunto cerrado de cinco identidades: [runtimes.go, líneas 11–35](../../internal/reviewerprovider/runtimes.go#L11-L35).
2. **Transporte elegible en el proceso actual:** `reviewImmutableRuntimeCapability` cruza policy compilada y publicidad `ContractImmutableReviewExecutorV1`: [review_transport_capability.go, líneas 83–154](../../internal/cli/review_transport_capability.go#L83-L154).
3. **Captura ejecutada dentro del comando Go:** `CapturesInProcess` incluye Claude Code, Codex y agy, no OpenCode/Pi: [capture_runtimes.go, líneas 33–60](../../internal/reviewerprovider/capture_runtimes.go#L33-L60).

Para agy el nombre del transporte es `antigravitycli_in_process`. **El término no dice que el modelo corra en memoria Go:** el comando Go lanza un proceso `agy` nuevo. OpenCode y Pi usan relays del host; agy no requiere ese relay para capturar.

La elegibilidad compilada no verifica credenciales, existencia del binario, disponibilidad de modelos ni efecto del sandbox. El error por `LookPath` sucede en el adaptador al ejecutar.

## Qué hace exactamente `AntigravityCLIAdapter.Review`

[`antigravitycli_adapter.go`, líneas 33–78](../../internal/reviewerprovider/antigravitycli_adapter.go#L33-L78):

```text
Review(ctx, Invocation)
  1. resolver agy desde PATH
  2. crear scratch temporal vacío
  3. argv = --sandbox --output-format text
  4. añadir --model solo si Model no es vacío
  5. cwd = scratch; env = allowlist + extras de test
  6. stdin = Invocation.Prompt()
  7. ejecutar con context cancellation y WaitDelay 5 s
  8. failure → error y nil bytes
  9. stdout whitespace-only → error y nil bytes
 10. éxito → stdout original, sin parseo de JSON
```

Argumentos del transporte, **no comando de revisión completo para copiar**:

```text
agy --sandbox --output-format text [--model <modelo>]
```

El prompt viaja por stdin y no en argv. El comentario del código explica la omisión deliberada de `-p/--print`: evitar que headless consuma flags posteriores como texto del prompt cuando detecta input pipeado, con input-format text por defecto. Es **intención de diseño documentada**, no prueba autónoma de cómo v1.3.0 parsea todas las variantes: [líneas 34–38](../../internal/reviewerprovider/antigravitycli_adapter.go#L34-L38).

No añadir `--dangerously-skip-permissions`, `--agent`, `--json-schema` ni `--effort` a esta descripción. El help local declara algunos de esos flags, pero el adaptador no los utiliza. `--output-format text` transporta texto; el proveedor exige dentro de ese texto un objeto JSON de resultado, no el wrapper JSON de salida del cliente.

### Cancelación y espera

`exec.CommandContext` usa el contexto recibido. `WaitDelay = 5s` limita la espera de pipes después de cancelación/terminación relevante; **no es un timeout total de revisión de cinco segundos**. El adaptador no crea un deadline global propio. Si el contexto está cancelado devuelve su error tipado en lugar de solo `signal: killed`: [líneas 61–74](../../internal/reviewerprovider/antigravitycli_adapter.go#L61-L74).

Los 20 segundos del import del plugin son otro timeout y otro proceso: [adapter de instalación](../../internal/agents/antigravitycli/adapter.go#L38-L43).

## Aislamiento solicitado, no cleanroom demostrado

| Mecanismo observado en código | Qué restringe/solicita | Qué sigue sin probar |
|---|---|---|
| Scratch vacío como cwd | No entrega el worktree como cwd del reviewer | No bloquea acceso por rutas absolutas |
| `--sandbox` | Solicita sandbox al cliente | Enforcement SO, red, herramientas y diferencias por plataforma |
| Prompt por stdin | Evita material del proveedor en argv | Persistencia/logs del runtime o servicio remoto |
| Allowlist de entorno | PATH y HOME; locators Windows adicionales | Configuración/credenciales resueltas desde HOME |
| Sin `--dangerously-skip-permissions` | No solicita bypass total | Las herramientas autoallow de headless |
| Context cancellation | Proceso cancelable por caller | Comportamiento de descendientes fuera de pipes/context |

[`antigravityCLIRuntimeEnvironment`](../../internal/reviewerprovider/antigravitycli_adapter.go#L81-L103) conserva `HOME`; en Windows también SYSTEMROOT y locators del perfil si existen. No hereda variables API arbitrarias de la shell por defecto, pero **no elimina auth/config del usuario**. No porta los flags Claude que vacían tools/settings ni el `--ignore-user-config` de Codex. [Comparación de aislamiento](07-comparison.md#revisión-e-aislamiento).

La documentación oficial headless describe workspace reads/writes autoallowed, shell Ask, soft-denial cuando no puede aprobar y sandbox default false. Es consistente con tratar el flag como contrato solicitado, no evidencia de que una revisión sea «tool-free». Ninguna prueba orgánica de confinamiento fue autorizada aquí.

**Riesgo residual:** reglas, plugins, MCP o permisos del HOME podrían intervenir según discovery del cliente. La inspección del adaptador no demuestra su supresión. La red/auth sigue siendo necesaria para una ejecución de modelo real. No afirmar «cero acceso al usuario», «sin contaminación de contexto» ni «SO read-only garantizado».

## Contrato del proveedor: roles y límites

[`reviewerprovider.Contract`](../../internal/reviewerprovider/contract.go#L112-L143) es la autoridad compartida para schema, storage slot, capacidades requeridas, prompt y límite de resultado:

| Rol compilado | Slot | Resultado esperado |
|---|---|---|
| `lens` | `selected-lens` | `subject_hash`, `inspection`, `findings`, `evidence` |
| `refuter` | `transaction-refuter-batch` | Hash de request y outcome por claim |
| `targeted-validator` | `correction-targeted-validator` | Hash/target de corrección, criterios/regresión/follow-ups |

Los schemas no autorizan al modelo a inventar lineage, revision o continuación. Las definiciones de `agent.md` manuales y las keys `review-*` de modelos no son el registro de roles compilados.

El presupuesto de entrada completo es **200 KiB = 204.800 bytes** de política del producto; incluye bloque materializado, wrappers y schema. No son 200.000 tokens ni capacidad medida de Gemini/Claude: [`ApprovedRuntimeContextBudget`, `RuntimeContextBudget`](../../internal/reviewerprovider/capture_runtimes.go#L5-L30). Cada contrato declara límite de resultado de `4 << 20` bytes. El adaptador acumula stdout en buffer antes de admisión; el límite de resultado del proveedor no equivale a un límite de memoria/streaming implementado en ese buffer.

## Del stdout a un resultado admitido

[`reviewProviderMaterialize`](../../internal/cli/review_provider.go#L63-L78) obtiene autoridad/binding del contexto de lens y crea `Invocation`. [`reviewProviderExtractRoleRaw`](../../internal/cli/review_provider.go#L81-L99) rechaza malformed, oversized y múltiples objetos antes del decode específico.

[`reviewProviderAdmitRaw`](../../internal/cli/review_provider.go#L111-L164) comprueba arrays explícitos, `subject_hash`, canonización y admisión con frozen context e inspección/causalidad. Durabilidad/captura queda fuera de esa función: no publicar bytes ni mutar autoridad durante la mera admisión.

```text
salida de proceso          ≠ resultado válido
objeto JSON válido        ≠ binding correcto
binding admitido          ≠ aprobación de candidato
aprobación de candidato   ≠ permiso humano para commit/push
```

No se ofrece una secuencia manual fija de comandos RDD en esta guía. El padre/controlador debe usar operaciones y bindings emitidos por el contrato nativo vigente; un ejemplo antiguo de START/CAPTURE no autoriza repetir tokens en otra transacción.

### Corrección acotada de salida del proveedor

La capa compartida, no el adaptador, permite una re-invocación correctiva ante rechazo de admisión cuando el rol la considera retryable. Añade feedback exacto al prompt original, preserva rechazos fuera del authority store y respeta el presupuesto de contexto: [`reviewProviderCaptureRetry`](../../internal/cli/review_provider_correction.go#L143-L183).

- Error de transporte → no retry de admisión.
- Primer rechazo admisible para retry → conservar resultado y medir prompt correctivo.
- Correctivo sobre presupuesto → no lanzar segunda invocación.
- Segundo rechazo → refusal explícito; no convierte resultados en aprobación.
- Targeted-validator puede excluir errores no retryable de esa rama.

Existe además una corrección compartida estrecha de `subject_hash` cuando la salida in-process copia exactamente target_identity de la transacción en vez del subject esperado; otros hashes siguen rechazados. No ocurre dentro del adaptador ni se aplica a toda entrada externa: [`reviewProviderCorrectSubjectHashEcho`](../../internal/cli/review_provider_correction.go#L44-L88).

Tests leídos de la capa compartida: [`TestProviderCaptureRetriesOnceWithAdmissionFeedback`](../../internal/cli/review_provider_correction_test.go#L122-L165), [`...RefusesAfterTwoRejectedResultsAndPreservesBoth`](../../internal/cli/review_provider_correction_test.go#L167-L225), [`...TransportErrorSkipsCorrectionAndPreservation`](../../internal/cli/review_provider_correction_test.go#L227-L246). Usan adapter seam y runtime Claude en esos ejemplos: prueban el core compartido, **no un modelo agy real**.

## Fallos: transporte versus admisión versus autoridad

| Fallo | Responsable | Señal | Consecuencia acotada |
|---|---|---|---|
| Binario no disponible/scratch no creable | Adaptador | `transport unavailable` | No bytes de revisión |
| Exit no cero/auth/modelo fallido | Adaptador | `transport failed`, detalle stderr o stdout | No admisión de esos bytes |
| Cancelación | Adaptador/caller | Error de contexto | No confundir con verdict |
| Stdout vacío/espacios | Adaptador | `produced no final message` | No admisión |
| Stdout no vacío pero objeto incorrecto | Core proveedor | Rechazo de admisión | Posible corrección acotada |
| Binding/inspection incorrectos | Core proveedor | Rechazo | No acreditar inspección por prose |
| Routing JSON malformed | Resolver de modelos | `InvalidReviewRoutingError` | Fallar antes de crear reviewer con default silencioso |
| Aprobación/delivery no resuelta | Controlador nativo | Estado/operación del lifecycle | Fuera de responsabilidad del adaptador |

[`Review`](../../internal/reviewerprovider/antigravitycli_adapter.go#L39-L78), [`ResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing.go#L364-L425), [admisión](../../internal/cli/review_provider.go#L81-L164).

No equiparar «stdout devuelto intacto» con «core no canonicaliza»: el adaptador no interpreta; las capas posteriores sí.

## Modelos de revisión y propiedad global

Ruta exclusiva del usuario:

```text
~/.gemini/antigravity-cli/gentle-ai/review-models.json
```

Hay seis keys: risk, readability, reliability, resilience, refuter y validator con prefijo `review-`: [`CanonicalReviewRoles`](../../internal/reviewerprovider/antigravitycli_routing.go#L103-L113).

Ejemplo conservador, no escrito por esta documentación:

```json
{
  "review-risk": "inherit",
  "review-readability": "inherit",
  "review-reliability": "inherit",
  "review-resilience": "inherit",
  "review-refuter": "inherit",
  "review-validator": "inherit"
}
```

### Gramática de resolución

| Entrada/estado | Resultado |
|---|---|
| Archivo ausente | Model vacío, sin diagnóstico de key missing ni escritura |
| Archivo válido pero key ausente | Model vacío, `assignmentKeyMissing=true`; caller emite diagnóstico |
| String `inherit` (case-insensitive) u objeto `{}` | Model vacío |
| String de modelo válida u objeto `{"model":"..."}` | Trim y validación de charset; se pasa `--model` |
| Objeto con modelo y legacy `effort` | Se tolera effort, no se emite |
| String vacío, null, array, JSON malformed, campo desconocido/identificador inválido | `InvalidReviewRoutingError` |

[`ResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing.go#L364-L425), [test table](../../internal/reviewerprovider/antigravitycli_routing_test.go#L15-L68), [ausencia](../../internal/reviewerprovider/antigravitycli_routing_test.go#L143-L157), [diagnóstico caller](../../internal/cli/review_provider_runtime.go#L143-L155).

Sin `--model` el **nuevo proceso** usa defaults que agy resuelva desde su configuración/runtime. No se ha demostrado que herede una selección efímera del proceso padre interactivo. No hay fallback al repositorio: evita que el candidato dicte modelos canónicos de roles.

### Presets y picker

Los cuatro presets son Default, Recommended, Performance y Economy; Custom es un modo de selección por rol. Los slugs hardcoded son política/catalog del producto, no modelos disponibles garantizados para toda cuenta: [`AntigravityPresetConstructors` y presets](../../internal/reviewerprovider/antigravitycli_routing.go#L141-L192).

| Preset | Asignación implementada |
|---|---|
| Default | Todos vacíos → serializados como `inherit` |
| Recommended | Opus risk/refuter, Pro reliability/validator, Flash medium resilience/readability |
| Performance | Opus risk/refuter/reliability, Pro validator/resilience, Sonnet readability |
| Economy | Flash high risk/refuter, medium reliability/validator, low resilience/readability |

No inferir benchmarks de precio/calidad de sus nombres. `DiscoverAntigravityModels` usa `agy models` y un fallback local si falla; descubrir puede contactar servicios del cliente. No se ejecutó. La navegación del picker ofrece presets, roles, búsqueda e inherit; [estado/navigation](../../internal/tui/screens/antigravity_review_model_picker.go#L45-L208). La disponibilidad del runtime requiere comprobación separada.

**Diferencia importante:** routing RDD por rol no es configuración general de todos los subagentes agy. El transpilador elimina las secciones de profiles Pi y prescribe defaults del runtime para genéricos: [mutación](../../scripts/transpile-antigravitycli/main.go#L1170-L1228). No proponer `.pi/subagents.json` como mecanismo agy.

## Evidencia del adaptador

| Test leído | Evidencia que aportaría ejecutado | Límite |
|---|---|---|
| `TestAntigravityCLIAdapterUsesStdinLockedDownArgumentsAndReturnsUntouchedRawOutput` | stdin exacto, argv sin prompt/print, raw bytes y WaitDelay | Helper process, no agy |
| `...PassesOnlyRuntimeEnvironment` | Variables ajenas no heredadas | No inspecciona discovery dentro de HOME |
| `...FailsClosedOnProcessFailureAndEmptyOutput` | Failure y empty devuelven nil bytes | No auth/model real |
| `...RoutingArguments` | Con/sin `--model` | No disponibilidad del slug |
| `...FakeRoundTrip` | Script POSIX comprueba flags y hash de prompt sintético | No lifecycle/lineage real |
| `TestAntigravityCLIAdapterOrganicTransport` | Si se autorizara: agy real, respuesta sintética y modelo inválido | Tag opt-in; no prueba autoridad real ni confinamiento |

Referencias: [adapter_test.go](../../internal/reviewerprovider/antigravitycli_adapter_test.go#L21-L203), [fake test](../../internal/reviewerprovider/antigravitycli_fake_test.go#L91-L150), [organic test](../../internal/reviewerprovider/antigravitycli_organic_test.go#L1-L102).

`TestAdapterMinimalityGuard` protege por AST la separación de parsing/budgets/admisión/lifecycle fuera del adaptador: [guard](../../internal/reviewerprovider/adapter_minimality_guard_test.go#L18-L40). No certifica aislamiento del cliente externo.

**Estado:** paquetes reviewerprovider/CLI/TUI/screens PASS recibidos del verificador el 2026-10-06, sin output por-test para contar skips; orgánica no autorizada/no ejecutada. [Registro](09-verification.md).
