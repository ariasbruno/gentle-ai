# Antigravity CLI (`agy`) con Gentle AI v4

**Gentle AI configura `agy` mediante un plugin y adapta los contratos de Gentle Shell; no convierte agy en Pi ni demuestra paridad entre runtimes.** Esta guía en español explica instalación, fuentes, decisiones, revisión RDD, cobertura y límites verificables. Está destinada a usuarios que necesitan saber qué se modifica y a mantenedores que deben poder auditar cada afirmación.

## Leer según la necesidad

| Necesidad | Lectura |
|---|---|
| Configurar por primera vez, conocer rutas y precauciones | [Fundamentos y puesta en marcha](01-foundation.md) |
| Entender el sistema de extremo a extremo | [Mapa de arquitectura](SUMMARY.md) |
| Saber qué se adapta de Gentle Shell y qué no | [Transpilador y contratos](02-asset-transpiler.md) |
| Entender reglas, delegación y hooks | [Orquestación y hooks](03-orchestration-and-hooks.md) |
| Evaluar MCP y la evidencia de `j4500` | [CodeGraph y benchmarks](04-codegraph-and-benchmarks.md) |
| Auditar subprocess, modelos, admisión y autoridad | [Transporte RDD](05-review-transport.md) |
| Mantener o retirar el plugin sin confundir propietarios | [Propiedad y deriva](06-asset-ownership-and-drift.md) |
| Comparar Claude Code, Codex, OpenCode, Pi y agy | [Matriz por capacidad](07-comparison.md) |
| Revisar motivos y alternativas sin inventar historia | [Decisiones](08-decisions.md) |
| Reproducir controles o conocer resultados pendientes | [Registro de verificación](09-verification.md) |
| Localizar afirmaciones, símbolos, tests y fuentes oficiales | [Índice de fuentes](10-source-index.md) |
| Distinguir CLI de la integración IDE | [Antigravity IDE frente a CLI](../antigravity-ide-vs-cli.md) |

## Camino inicial seguro

Primero, inspeccionar versiones y ayuda; estos comandos no autentican ni llaman al modelo:

```sh
command -v agy
agy --version
agy --help
go version
```

La observación local recibida para esta documentación informa `agy 1.3.0` y Go `1.27.1 linux/amd64`. No acredita que el plugin esté activado ni que el sandbox impida escrituras. El módulo del producto es v4 y declara Go `1.25.10`: [go.mod, líneas 1–3](../../go.mod#L1-L3).

Después, con `agy` y el binario v4 de `gentle-ai` ya disponibles, el usuario puede optar por la configuración global descrita en [puesta en marcha](01-foundation.md#puesta-en-marcha-global). **No se ejecutaron instalación, sync, autenticación ni modelos durante la redacción.** Instalar/configurar sí escribe archivos y puede invocar `agy plugin install`; no es una comprobación pasiva.

## Qué está implementado y qué no está demostrado

| Afirmación acotada | Base | Límite |
|---|---|---|
| `antigravity-cli` está registrado y su adaptador declara `TierFull` | [`catalog.allAgents`](../../internal/catalog/agents.go#L19-L34), [`Adapter.Tier`](../../internal/agents/antigravitycli/adapter.go#L76-L78) | El nivel no promete igualdad de capacidades |
| El bundle contiene diez definiciones de subagentes ODD/JD/4R | [`expectedSubagentNames`, `Invariant1`](../../internal/assets/antigravitycli_assets_test.go#L38-L96) | Archivos presentes no prueban ejecución ni enforcement del frontmatter por agy |
| La captura RDD ejecuta un subprocess nuevo con stdin y `--sandbox` | [`AntigravityCLIAdapter.Review`](../../internal/reviewerprovider/antigravitycli_adapter.go#L33-L78) | Solicitud de sandbox, no medición de confinamiento del SO |
| Hay cinco hooks y respuestas distintas por evento | [Definición](../../internal/agents/antigravitycli/adapter.go#L407-L470), [handler](../../internal/cli/antigravity_hook.go#L88-L148) | No son todos fail-open; `PreToolUse` puede denegar |
| El origen del bundle está fijado a un SHA de Gentle Shell | [`PROVENANCE.json`](../../internal/assets/antigravitycli/PROVENANCE.json#L1-L4) | SHA no equivale a versión semántica ni a port completo de extensiones Pi |
| `j4500` conduce sync y uninstall contra un agy falso | [`antigravityCLIRuntimeFixture`](../../bench/journeys_antigravity_cli.go#L18-L67) | No prueba credenciales, modelos, carga real de reglas ni sandbox |

## Cómo leer la evidencia

Usamos cuatro categorías, siempre separadas:

1. **Implementación inspeccionada:** código o asset actual que expresa el comportamiento.
2. **Cobertura existente:** test leído que comprueba una condición bajo una fixture. No significa que se haya ejecutado en esta tarea.
3. **Observación recibida:** resultado concreto aportado por el verificador o fuente oficial recuperada por el padre, con versión/fecha y alcance.
4. **Pendiente/no demostrado:** falta de resultado actual, runtime autenticado o prueba que cubra la afirmación.

El verificador independiente informó **17 paquetes PASS, exit 0, el 2026-10-06**, sobre HEAD `628bc61ec2851040f2c95de150d8c9e62883b17d`; sin resultado por-test para contar skips. Driven, orgánica y gate upstream `--check` no se ejecutaron. Las salidas antiguas v3 y las expresiones «100 %», «production-ready» o «paridad completa» no son evidencia de esta rama. El [registro](09-verification.md) separa unitarias, mocks, driven y orgánicas; el [índice](10-source-index.md) proporciona trazabilidad.

## Límites que conviene conocer antes de usar

- Usar alcance global como ruta documentada. El alcance workspace tiene una discrepancia entre componentes y despliegue/import global: [análisis de rutas](01-foundation.md#alcance-workspace-no-es-un-camino-garantizado).
- No guardar personalizaciones dentro de `agents/` o `skills/` del plugin: el reconciliador poda directorios ajenos al bundle: [`DeployPluginTree`](../../internal/agents/antigravitycli/adapter.go#L589-L625).
- La revisión conserva `HOME` para resolver configuración/credenciales y no desactiva explícitamente toda discovery de plugins o herramientas: [frontera de aislamiento](05-review-transport.md#aislamiento-solicitado-no-cleanroom-demostrado).
- Un exit 0 de headless no garantiza éxito de todas las herramientas: la documentación oficial describe denegaciones blandas. Véase [fuentes oficiales](10-source-index.md#fuentes-oficiales-y-observaciones-locales).
- La separación CLI/IDE tiene un canario limitado, no una demostración universal de rutas disjuntas: [comparación IDE](../antigravity-ide-vs-cli.md).

**Siguiente paso:** [leer qué cambia la configuración global](01-foundation.md), antes de autorizar escrituras.
