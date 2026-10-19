# 9. Registro de verificación y reproducción

**El 2026-10-06 el verificador independiente informó exit 0 y PASS en 17 paquetes, sin indicadores `(cached)` en la salida recibida.** Eso acredita pruebas Go de la revisión indicada; no acredita ejecución driven, modelo autenticado ni enforcement de sandbox. La redacción documental no ejecutó tests de producto.

[Índice](README.md) · [Índice de fuentes/tests](10-source-index.md)

## Identidad de la evidencia recibida

| Campo | Valor |
|---|---|
| Fecha reportada | 2026-10-06 |
| HEAD de producto | `628bc61ec2851040f2c95de150d8c9e62883b17d` |
| Módulo | `github.com/gentleman-programming/gentle-ai/v4` |
| Go observado | `go1.27.1 linux/amd64` |
| agy observado | `1.3.0`, binario `$HOME/.local/bin/agy` |
| Fuente de resultado | Handoff del verificador independiente transmitido por el padre |
| Verbosidad | Salida no verbose: no acredita conteo por-test de ejecuciones/skips |
| Cambio de esta tarea | Documentación pasiva; sin RED/GREEN significativo |

El SHA de Gentle Shell del bundle es otro campo, no el HEAD del producto: [procedencia](../../internal/assets/antigravitycli/PROVENANCE.json#L1-L4).

## Resultado por grupo recibido

El verificador ejecutó **cuatro grupos de producto y uno de bench**, además de dos comandos de identificación/control del worktree. Todos los siete comandos terminaron con exit 0. Esta es la transcripción literal recibida, en orden, desde `$HOME/dev/gentle-ai`:

```sh
# 1. Identidad y estado inicial.
git status --short; git rev-parse HEAD; go version; agy --version

# 2. Registro y adaptadores: cuatro paquetes PASS.
go test ./internal/catalog ./internal/agents ./internal/agents/antigravitycli ./internal/agents/capabilitymanifest

# 3. Generación y assets: dos paquetes PASS.
go test ./scripts/transpile-antigravitycli ./internal/assets

# 4. Componentes: seis paquetes PASS.
go test ./internal/components/agentguidance ./internal/components/persona ./internal/components/skills ./internal/components/permissions ./internal/components/uninstall ./internal/components/communitytool

# 5. Proveedor, CLI y TUI: cuatro paquetes PASS.
go test ./internal/reviewerprovider ./internal/cli ./internal/tui ./internal/tui/screens

# 6. Módulo bench: un paquete PASS.
(cd bench && go test ./...)

# 7. Estado final.
git status --short
```

Los comentarios explican el orden; no forman parte de las líneas ejecutadas. No se agregaron flags `-v`, `-count`, filtros ni globs a la evidencia. La primera salida de estado contenía solamente el archivo de seguimiento untracked; la última contenía seis documentos tracked modificados por el trabajo documental concurrente. El verificador no atribuyó esas mutaciones y no limpió archivos: el padre confirmó que las rutas pertenecían al alcance del escritor autorizado. Este control no equivale a una ejecución sin escritor concurrente.

| Grupo | Paquetes reportados y tiempo | Resultado | Nivel |
|---|---|---|---|
| 1 — Registro/adaptador | catalog 0.010 s; agents 0.031 s; antigravitycli 0.108 s; capabilitymanifest 0.016 s | Exit 0, cuatro PASS | Tests Go del grupo, sin proof del cliente |
| 2 — Generación/assets | transpiler 0.526 s; assets 8.368 s | Exit 0, dos PASS | Tests/invariantes con fixtures; no gate upstream real |
| 3 — Componentes | agentguidance 5.683 s; persona 0.256 s; skills 0.452 s; permissions 0.251 s; uninstall 0.277 s; communitytool 0.975 s | Exit 0, seis PASS | Escritura/reglas/MCP/cleanup en fixtures |
| 4 — Proveedor/CLI/TUI | reviewerprovider 0.442 s; cli 323.877 s; tui 5.351 s; screens 3.134 s | Exit 0, cuatro PASS | Core/adapter helper-fake/UI; no modelo orgánico |
| 5 — Bench | bench 4.787 s | Exit 0, un PASS | Tests del módulo/harness, no ejecución driven |

No se reciclan las salidas v3 de la colección anterior. Ningún test individual se marca PASS por nombre solo porque su paquete salió ok: pueden existir skips y condiciones de plataforma sin detalle en la salida recibida.

### Comprobaciones documentales del escritor

Los únicos comandos de validación autorizados aquí son:

```sh
git diff --check
git diff --stat
```

Resultado observado por el escritor el 2026-10-06: `git diff --check` terminó con exit 0 y sin salida. `git diff --stat` terminó con exit 0; el corte anterior al registro de esta evidencia mostraba 10 archivos tracked, 950 inserciones y 1.161 eliminaciones. Los cuatro documentos nuevos (comparación, decisiones, verificación e índice de fuentes) están fuera de ese stat por ser untracked; el stat definitivo se entrega en el handoff, sin staging.

Son controles de whitespace/tamaño, no verificación automática de todos los enlaces ni de comportamiento. La lectura directa de símbolos/tests sirve de auditoría de trazabilidad; no reemplaza ejecución de tests.

## Auditoría documental final

El 2026-10-06 un verificador independiente leyó los doce documentos de esta colección, la comparación IDE/CLI y la entrada agy del README raíz. Contrastó directamente quince áreas de funcionamiento y decisiones con código, tests y registros históricos; no certificó automáticamente la semántica de todas las citas.

| Control observado | Resultado | Qué acredita |
|---|---|---|
| `git diff --check` | Sin diagnósticos | Whitespace del diff tracked |
| Checker local en Python, 14 archivos | 521 enlaces locales, cero incidencias | Destinos existentes, rangos dentro del archivo y anchors de encabezados |
| Correcciones revisadas contra fuente | 30 citas aceptadas y precisión de merge corregida | Resolución de los 29 rangos inválidos iniciales y de una cita que omitía el retorno descrito |
| Acceso HTTP con Python, sin fragments | Nueve URLs únicas, todas HTTP 200; cero errores | Accesibilidad, no contenido completo ni aplicabilidad a una versión del cliente |
| Preservación del registro de producto | Siete comandos literales intactos | Sin flags, ejecuciones o resultados agregados a la evidencia previa |

El checker local resolvió los links relativos desde cada documento, verificó existencia del destino, comprobó límites de `#L…` contra la cantidad de líneas y comparó anchors Markdown con encabezados. Encontró 24 ocurrencias de enlaces externos, que el segundo control deduplicó por URL sin fragment. El control HTTP leyó solo los primeros 1.024 bytes de cada respuesta y no comprobó anchors remotos. Los scripts se ejecutaron en memoria, sin generar archivos de producto ni alterar documentación.

La auditoría inicial encontró 29 rangos fuera de archivo. Se corrigieron y se repitió el mismo checker, que pasó de 29 incidencias a cero. La revalidación también inspeccionó las referencias corregidas, no solamente sus límites. Estos resultados no amplían la evidencia runtime de agy ni convierten los 17 paquetes PASS en pruebas orgánicas.

## Reproducción focal para un mantenedor

**Esta sección es receta, no evidencia de ejecución de comandos nuevos.** Ejecutar tests puede crear temporales/cache y fixtures Git. El mantenedor debe autorizar runner/entorno y conservar resultado exacto.

```sh
# Adaptador y contrato de import (helpers, no agy real).
go test ./internal/agents/antigravitycli

# Transpilador e invariantes embedded con fixtures.
go test ./scripts/transpile-antigravitycli ./internal/assets

# Routing/reglas/permisos/MCP bajo fixtures.
go test ./internal/components/agentguidance ./internal/components/permissions ./internal/components/communitytool

# Transporte/helper-fake/resolver y frontera compartida de captura.
go test ./internal/reviewerprovider ./internal/cli

# Bench tiene módulo propio; no incluirlo como ./bench/... en el módulo raíz.
(cd bench && go test ./...)
```

Los comandos focales no sustituyen la verificación completa requerida por una futura tarea. La transcripción de la ejecución independiente está en la sección anterior; esta receta no agrega resultados observados.

Registrar para cada ejecución: fecha, HEAD, command literal, cwd, Go/OS, exit, salida completa o artefacto reproducible, y skips visibles. Si no se ve `(cached)`, decir «ninguno reportado», no afirmar que se forzó cache off sin `-count=1` explícito.

## Controles negativos existentes y qué protegen

| Frontera | Test/source inspeccionado | Caso negativo real | Nivel de demostración |
|---|---|---|---|
| Import gate | [`TestEnsurePluginImportedManifestGate`](../../internal/agents/antigravitycli/adapter_test.go#L345-L380) | JSON inválido, imports no array, nombre ausente; cero comandos/mutaciones | Runner/mock |
| Import failure | [`TestEnsurePluginImportedFailures`](../../internal/agents/antigravitycli/adapter_test.go#L382-L427) | Falta binario/árbol, command failure, falta registro, timeout | Runner falso |
| Generación staged | [`TestTranspileSubagentsFailureLeavesDestinationUnchanged`](../../scripts/transpile-antigravitycli/main_test.go#L48-L87) | Último agente fuente inválido no refresca parcialmente ni poda destino | Fuente fixture, filesystem temporal |
| Drift bytes | [`TestCheckGateContentDriftFails`](../../scripts/transpile-antigravitycli/main_test.go#L685-L722) | Contenido plugin.json corrupto provoca diff | Árbol fuente mínimo fixture |
| Drift set | [`TestCheckGateMissingFileFails`, `ExtraFileFails`](../../scripts/transpile-antigravitycli/main_test.go#L622-L798) | Falta/extra de archivo detectados | No canon upstream completo |
| HEAD frente a dirty | [`TestCheckGateComparesAgainstCommittedTree`](../../scripts/transpile-antigravitycli/main_test.go#L622-L684) | Dirty no altera comparación; hand edit committed falla | Git fixture; no gate actual del checkout |
| Hook wiring de pipeline | [`TestStagePlansKeepAntigravityCLIPluginImportStep`](../../internal/cli/antigravity_cli_plugin_import_guard_test.go#L19-L72) | Selección no-agy no tiene paso, agy sí en install/sync | Plan, no ejecución real |
| Pre-tool transport | [`TestAntigravityHookPreToolUseEdgeCases`](../../internal/cli/antigravity_hook_test.go#L337-L369) | Empty/null/array/malformed deny | Handler fixture |
| Loop | [`TestAntigravityHookPreToolUseCircuitBreaker`](../../internal/cli/antigravity_hook_test.go#L183-L299) | Tercera llamada idéntica deny; args/tool distintos reinician | Persistencia fixture |
| Stop | [`TestAntigravityHookStopOutcomes`](../../internal/cli/antigravity_hook_test.go#L60-L137) | No idle/no model_stop neutro; recordatorio repetido suprimido | Repo fixture |
| MCP | [`TestReconcileAntigravityCLICodeGraphReplacesMalformedConfig`](../../internal/components/communitytool/antigravity_cli_codegraph_test.go#L124-L148) | Config inválida sustituida por canonical | No servidor real |
| Routing personal | [`TestResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing_test.go#L15-L68) | Syntax/null/campo desconocido/slug inválido fallan | HOME fixture |
| Adapter process | [`TestAntigravityCLIAdapterFailsClosedOnProcessFailureAndEmptyOutput`](../../internal/reviewerprovider/antigravitycli_adapter_test.go#L120-L137) | Exit failure/empty stdout → error/nil bytes | Helper Go |
| Fake round-trip | [`...FakeRoundTrip`, `FakeFailsClosedOnMissingBinary`](../../internal/reviewerprovider/antigravitycli_fake_test.go#L91-L150) | Con/sin modelo, hash de stdin; missing binary cerrado | POSIX fake, no modelo |
| Admisión/retry | [`TestProviderCaptureTransportErrorSkipsCorrectionAndPreservation`](../../internal/cli/review_provider_correction_test.go#L227-L246) | Error de transporte no intenta correction | Core compartido con seam |

Estas referencias sirven para seleccionar controles ya existentes, no para afirmar que se agregaron tests en la tarea documental.

## Benchmark driven no equivale a `go test`

**Estado: NO EJECUTADO en la verificación actual.** La definición de journey y PASS del paquete bench no demuestran que un binario haya completado `j4500`.

Receta con binarios de producto y bench ya construidos, únicamente bajo autorización de un entorno de fixtures:

```sh
gentle-ai-bench run --binary /path/to/gentle-ai --only j4500-antigravity-cli-lifecycle-parity --out /path/to/report.json
```

Sintaxis: [`bench/README.md`, Driven](../../bench/README.md#L59-L80); journey: [`antigravityCLILifecycleJourneys`](../../bench/journeys_antigravity_cli.go#L427-L439).

`/path/to` son placeholders documentales, no comandos ejecutados ni targets aprobados. Aportar evidencia solo si el JSON tiene la journey exacta con `status: completed`, exit del run y hashes/versiones de los binarios. `unsupported` no es PASS de esa capacidad.

El mock agy del journey no es una autorización para usar HOME real. Incluso con journey completed seguiría faltando runtime real, auth, carga de MCP/reglas y sandbox.

## Controles de mantenimiento no ejecutados

| Control | Comando de referencia | Estado/precaución |
|---|---|---|
| Gate contra pin upstream | `go run ./scripts/transpile-antigravitycli --check` | **NO EJECUTADO**. Puede clonar y crear temporales; compara committed HEAD |
| Gate con fuente local fijada | `go run ./scripts/transpile-antigravitycli --check --source-dir /path/to/gentle-shell` | **NO EJECUTADO**. HEAD local debe coincidir con pin o se usa remoto |
| Regeneración | `go run ./scripts/transpile-antigravitycli --strict --source-dir /path/to/gentle-shell` | **NO EJECUTADO**. Escribe assets/procedencia; no validación pasiva |
| Orgánica del adapter | `go test -tags organic -run TestAntigravityCLIAdapterOrganicTransport -v ./internal/reviewerprovider/` | **NO EJECUTADO/NO AUTORIZADO**. Modelo real/autenticación/coste |

La existencia de tests `TestCheckGate*` ejecutados dentro del paquete no permite marcar el gate completo upstream como ejecutado: utilizan fuentes fixtures. La prueba orgánica existente utiliza un prompt sintético sin lineage real: [organicAntigravityCLILensPrompt](../../internal/reviewerprovider/antigravitycli_organic_test.go#L85-L102). Aunque se ejecutara y pasara, no demostraría autoridad RDD o entrega.

## Matriz de límites pendientes

| Afirmación potencial | Evidencia que falta | Formulación permitida ahora |
|---|---|---|
| Sandbox impide escrituras/red | Probe real acotado, por cliente/plataforma | El adapter solicita `--sandbox` |
| Todos los frontmatter fields son enforcing | Ensayo positivo/negativo cliente y contrato oficial por versión | El transpilador los emite |
| Plugin cargado y recargado por agy1.3.0 | Observación real de discovery/context en HOME desechable | Árbol y registro implementados |
| Workspace local no toca HOME | Test end-to-end de destinos tras decisión de producto | No ofrecer ruta garantizada |
| Rules bajo budget agregado | Medición tras includes/tokenización del cliente | Tests de bytes locales específicos |
| Preset disponible/óptimo | Catálogo y mediciones de cuenta/modelo | Slugs/política hardcoded |
| Paridad de los cinco agentes | Ensayo homogéneo de capacidades | Matriz de diferencias de código |
| Todos los tests relevantes ejecutados sin skips | Salida por-test y runner exacto | 17 paquetes PASS, skips no contados |

## Cómo incorporar nueva evidencia sin sobreafirmar

1. Copiar el comando literal y su resultado, no reconstruirlo a partir de los paquetes.
2. Identificar si es fixture/mock, driven contra binario o cliente autenticado.
3. Registrar qué negativos se observaron, incluido skip/no binary/auth error.
4. Cambiar solo la afirmación cubierta por esa evidencia. Un PASS de entorno no arregla scope, un PASS de flags no prueba sandbox.
5. Mantener histórico con fecha/versión; no trasladar mediciones de v3 al v4 actual.
