# 5. Transporte RDD, modelos y fronteras de prueba

**Go conserva el contrato de revisión; agy aporta bytes de una respuesta.** El adaptador no decide aprobación, recibos, correcciones ni entrega. El reviewer exige un modelo explícito y usa un perfil temporal con autenticación mínima, sin copiar reglas, settings, plugins ni MCP del usuario. Esto no equivale a un sandbox read-only ni a desactivar todas las herramientas.

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
  1. resolver agy desde PATH; modelo vacío/`inherit` omite `--model`
  2. crear perfil temporal y workspace vacío separados
  3. copiar solo antigravity-oauth-token y, si existe, installation_id
  4. argv = --sandbox --output-format text [--model <modelo>] [--effort <effort>]
  5. cwd = workspace; HOME y locators Windows apuntan al perfil temporal
  6. stdin = Invocation.Prompt()
  7. ejecutar con context cancellation y WaitDelay 5 s
  8. failure → error y nil bytes
  9. stdout whitespace-only → error y nil bytes
 10. éxito → stdout original, sin parseo de JSON
 11. al terminar: destruir credenciales best-effort y remover scratch; si la remoción falla, error de transporte con warning si la destrucción no se confirmó
```

Argumentos del transporte, **no comando de revisión completo para copiar**:

```text
agy --sandbox --output-format text [--model <modelo>] [--effort <effort>]
```

El prompt viaja por stdin y no en argv. El comentario del código explica la omisión deliberada de `-p/--print`: evitar que headless consuma flags posteriores como texto del prompt cuando detecta input pipeado, con input-format text por defecto. Es **intención de diseño documentada**, no prueba autónoma de cómo v1.3.0 parsea todas las variantes: [líneas 34–38](../../internal/reviewerprovider/antigravitycli_adapter.go#L34-L38).

No añadir `--dangerously-skip-permissions`, `--agent` ni `--json-schema` a esta descripción: el help local los declara, pero el adaptador no los utiliza. `--model` se emite solo con una asignación válida distinta de `inherit`; `--effort` solo con un tier válido (low, medium, high, xhigh, max) distinto de `inherit`. `--output-format text` transporta texto; el proveedor exige dentro de ese texto un objeto JSON de resultado, no el wrapper JSON de salida del cliente.

### Cancelación y espera

`exec.CommandContext` usa el contexto recibido. `WaitDelay = 5s` limita la espera de pipes después de cancelación/terminación relevante; **no es un timeout total de revisión de cinco segundos**. El adaptador no crea un deadline global propio. Si el contexto está cancelado devuelve su error tipado en lugar de solo `signal: killed`: [líneas 61–74](../../internal/reviewerprovider/antigravitycli_adapter.go#L61-L74).

Los 20 segundos del import del plugin son otro timeout y otro proceso: [adapter de instalación](../../internal/agents/antigravitycli/adapter.go#L38-L43).

## Aislamiento solicitado, no cleanroom demostrado

| Mecanismo observado en código | Qué restringe/solicita | Qué sigue sin probar |
|---|---|---|
| Scratch vacío como cwd | No entrega el worktree como cwd del reviewer | Acceso por rutas absolutas: demostrado posible (probe diferencial de herramientas) |
| `--sandbox` | Solicita sandbox al cliente | Archivos: demostrado que NO bloquea lecturas/escrituras fuera del scratch; red, MCP y plataformas siguen sin medir |
| Prompt por stdin | Evita material del proveedor en argv | Persistencia/logs del runtime o servicio remoto |
| Perfil auth-only y allowlist | HOME aislado; token e installation_id copiados a archivos 0600, directorios 0700 | Lecturas absolutas por herramientas, comportamiento por plataforma y auth remota |
| Sin `--dangerously-skip-permissions` | No solicita bypass total | Las herramientas autoallow de headless |
| Context cancellation | Proceso cancelable por caller | Comportamiento de descendientes fuera de pipes/context |

[`prepareAntigravityCLIProfile`](../../internal/reviewerprovider/antigravitycli_profile.go) copia únicamente los dos archivos de autenticación admitidos desde `~/.gemini/antigravity-cli`; rechaza enlaces, incluidos los directorios intermedios, y archivos no regulares. No copia configuración ni historial. El original no se escribe. Las regresiones cubren el contenido del perfil, la limpieza en éxito y error y la ausencia opcional de `installation_id`. Antes de remover el scratch, el transporte destruye best-effort las copias de credenciales; si la remoción del árbol falla y esa destrucción no se confirma, el error añade `; isolated credentials may remain on disk under <scratch>` — fallo de transporte, no éxito, con señal observable para el operador. [`antigravityCLIRuntimeEnvironment`](../../internal/reviewerprovider/antigravitycli_adapter.go) cambia HOME y los locators Windows al perfil temporal; los extras de test no pueden restaurar esos locators. No porta los flags Claude que vacían tools/settings ni el `--ignore-user-config` de Codex. [Comparación](07-comparison.md#revisión-e-aislamiento).

La documentación oficial headless describe workspace reads/writes autoallowed, shell Ask, soft-denial cuando no puede aprobar y sandbox default false. Es consistente con tratar el flag como contrato solicitado, no evidencia de que una revisión sea «tool-free». El probe diferencial de herramientas de archivo —control sin `--sandbox` primero mediante un shim de PATH que elimina solo ese flag del adapter real, fase sandboxed condicionada a que el control demuestre efectos, canaries aleatorios frescos por fase en un directorio neutral— demostró que el reviewer **lee y crea archivos fuera de su scratch en ambas fases** (read_canary y write_canary exactos, transporte ok, 2 llamadas): el flag no bloquea el acceso a archivos en este escenario.

**Evidencia:** con AGY 1.3.5, el adaptador que conservaba HOME devolvió un marcador aleatorio de un `AGENTS.md` global que no estaba en stdin; el perfil auth-only eliminó esa vía y la repetición orgánica del señuelo devolvió exactamente `NO_GLOBAL_TAG` con la credencial fuente intacta. Las herramientas de archivo, en cambio, NO están confinadas: el probe diferencial demostró lectura y escritura fuera del scratch con y sin `--sandbox`. No afirmar «cero acceso al usuario» ni «SO read-only»; hoy se sabe que el reviewer puede tocar archivos del usuario si su prompt se lo pide (el prompt es propiedad de Go, no del candidato revisado).

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

Ejemplo conservador, no escrito por esta documentación. Sin archivo o con todos `inherit`, el transporte omite `--model` y AGY resuelve su propio default en el perfil aislado:

```json
{
  "review-risk": "gemini-3.8-flash-low",
  "review-readability": {"model": "gemini-3.8-flash-low", "effort": "low"},
  "review-reliability": {"effort": "medium"},
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
| String `inherit` (case-insensitive) u objeto `{}` | Model vacío; el transporte omite `--model` |
| String de modelo válida u objeto `{"model":"..."}` | Trim y validación de charset; se pasa `--model` |
| Objeto con `effort` | Effort trim y validado (low, medium, high, xhigh, max; `inherit` = unset); se pasa `--effort`. `{"effort":"..."}` sin model es válido: modelo inherit + effort |
| String vacío, null, array, JSON malformed, campo desconocido/identificador inválido | `InvalidReviewRoutingError` |

[`ResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing.go#L364-L425), [test table](../../internal/reviewerprovider/antigravitycli_routing_test.go#L15-L68), [ausencia](../../internal/reviewerprovider/antigravitycli_routing_test.go#L143-L157), [diagnóstico caller](../../internal/cli/review_provider_runtime.go#L143-L155).

Sin archivo, con key ausente o con `inherit`, el transporte omite `--model` y el reviewer aislado resuelve el default del propio AGY: el perfil temporal no copia defaults del perfil habitual ni de la sesión interactiva. Configurá un modelo explícito por rol solo si lo preferís; el ejemplo no garantiza disponibilidad ni recomienda ese modelo para toda revisión. No hay fallback al repositorio ni selección silenciosa de otro modelo. El effort solo se emite con un tier válido; un valor inválido falla cerrado con `InvalidReviewRoutingError`.

### Presets y picker

Los cuatro presets son Default, Recommended, Performance y Economy; Custom es un modo de selección por rol. Los slugs hardcoded son política/catalog del producto, no modelos disponibles garantizados para toda cuenta: [`AntigravityPresetConstructors` y presets](../../internal/reviewerprovider/antigravitycli_routing.go#L141-L192).

| Preset | Asignación implementada |
|---|---|
| Default | Todos vacíos → se omite `--model`; AGY resuelve su propio default en el perfil aislado |
| Recommended | Opus risk/refuter, Pro reliability/validator, Flash medium resilience/readability |
| Performance | Opus risk/refuter/reliability, Pro validator/resilience, Sonnet readability |
| Economy | Flash high risk/refuter, medium reliability/validator, low resilience/readability |

No inferir benchmarks de precio/calidad de sus nombres. `DiscoverAntigravityModels` usa `agy models` y un fallback local si falla; descubrir puede contactar servicios del cliente. No se ejecutó. La navegación del picker ofrece presets, roles, búsqueda de modelo y selección de effort por rol; [estado/navigation](../../internal/tui/screens/antigravity_review_model_picker.go#L45-L208). La disponibilidad del runtime requiere comprobación separada.

**Diferencia importante:** routing RDD por rol no es configuración general de todos los subagentes agy. El transpilador elimina las secciones de profiles Pi y prescribe defaults del runtime para genéricos: [mutación](../../scripts/transpile-antigravitycli/main.go#L1170-L1228). No proponer `.pi/subagents.json` como mecanismo agy.

## Evidencia del adaptador

| Test leído | Evidencia que aportaría ejecutado | Límite |
|---|---|---|
| `TestAntigravityCLIAdapterUsesStdinLockedDownArgumentsAndReturnsUntouchedRawOutput` | stdin exacto, argv sin prompt/print, raw bytes y WaitDelay | Helper process, no agy |
| `...PassesOnlyRuntimeEnvironment` | Variables ajenas no heredadas | No inspecciona discovery dentro de HOME |
| `...FailsClosedOnProcessFailureAndEmptyOutput` | Failure y empty devuelven nil bytes | No auth/model real |
| `...RoutingArguments` | Modelo/effort explícitos conservados en argv; inherit omite `--model` | No disponibilidad del slug |
| `...FakeRoundTrip` | Script POSIX comprueba flags y hash de prompt sintético | No lifecycle/lineage real |
| `TestAntigravityCLIAdapterOrganicTransport` | Si se autorizara: agy real, respuesta sintética y modelo inválido | Tag opt-in; no prueba autoridad real ni confinamiento |

Referencias: [adapter_test.go](../../internal/reviewerprovider/antigravitycli_adapter_test.go#L21-L203), [fake test](../../internal/reviewerprovider/antigravitycli_fake_test.go#L91-L150), [organic test](../../internal/reviewerprovider/antigravitycli_organic_test.go#L1-L102).

`TestAdapterMinimalityGuard` protege por AST la separación de parsing/budgets/admisión/lifecycle fuera del adaptador: [guard](../../internal/reviewerprovider/adapter_minimality_guard_test.go#L18-L40). No certifica aislamiento del cliente externo.

**Estado:** paquetes reviewerprovider/CLI/TUI/screens PASS recibidos del verificador el 2026-10-06, sin output por-test para contar skips; orgánica no autorizada/no ejecutada. [Registro](09-verification.md).
