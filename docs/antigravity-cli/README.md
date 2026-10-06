# Antigravity CLI (`agy`) con Gentle AI v4

**Gentle AI configura `agy` mediante un plugin y adapta los contratos de Gentle Shell; no convierte agy en Pi ni asume paridad total entre runtimes.** Esta documentación detalla la instalación, arquitectura, decisiones de diseño, revisión RDD, cobertura de pruebas y límites del soporte. Su propósito es orientar a los desarrolladores sobre qué componentes se configuran y permitir a los mantenedores auditar cada integración.

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

## Requisitos previos y verificación de entorno

Antes de instalar o modificar la configuración, se recomienda realizar una comprobación pasiva del entorno (comandos de solo lectura que no requieren autenticación ni llamadas a modelos):

```sh
command -v agy
agy --version
agy --help
go version
```

### Entorno de referencia validado

Esta documentación y la suite de pruebas toman como base el siguiente entorno de referencia:

- **Sistema operativo:** Linux `amd64`.
- **Versión de Antigravity CLI:** `agy 1.3.0`.
- **Entorno Go:** El proyecto declara compatibilidad base con Go `1.25.10` ([go.mod, líneas 1–3](../../go.mod#L1-L3)) y fue validado con cadenas de herramientas Go 1.25+.

> [!NOTE]
> La comprobación pasiva anterior únicamente verifica la presencia de los binarios en tu `PATH`. No implica que el plugin de Gentle AI esté activo ni evalúa el aislamiento del sandbox. Para aplicar la configuración en el sistema, consulta los pasos en [Puesta en marcha global](01-foundation.md#puesta-en-marcha-global).

## Qué está implementado y qué no está demostrado

| Afirmación acotada | Base | Límite |
|---|---|---|
| `antigravity-cli` está registrado y su adaptador declara `TierFull` | [`catalog.allAgents`](../../internal/catalog/agents.go#L19-L34), [`Adapter.Tier`](../../internal/agents/antigravitycli/adapter.go#L76-L78) | El nivel no promete igualdad de capacidades |
| El bundle contiene diez definiciones de subagentes ODD/JD/4R | [`expectedSubagentNames`, `Invariant1`](../../internal/assets/antigravitycli_assets_test.go#L38-L96) | Archivos presentes no prueban ejecución ni enforcement del frontmatter por agy |
| La captura RDD ejecuta un subprocess nuevo con stdin y `--sandbox` | [`AntigravityCLIAdapter.Review`](../../internal/reviewerprovider/antigravitycli_adapter.go#L33-L78) | Solicitud de sandbox, no medición de confinamiento del SO |
| Hay cinco hooks y respuestas distintas por evento | [Definición](../../internal/agents/antigravitycli/adapter.go#L407-L470), [handler](../../internal/cli/antigravity_hook.go#L88-L148) | No son todos fail-open; `PreToolUse` puede denegar |
| El origen del bundle está fijado a un SHA de Gentle Shell | [`PROVENANCE.json`](../../internal/assets/antigravitycli/PROVENANCE.json#L1-L4) | SHA no equivale a versión semántica ni a port completo de extensiones Pi |
| `j4500` conduce sync y uninstall contra un agy falso | [`antigravityCLIRuntimeFixture`](../../bench/journeys_antigravity_cli.go#L18-L67) | No prueba credenciales, modelos, carga real de reglas ni sandbox |

## Criterios de evidencia y trazabilidad

Para documentar el estado técnico con rigor, se distinguen cuatro niveles de certeza:

1. **Implementación en código:** código fuente o assets que definen el comportamiento actual.
2. **Pruebas automatizadas:** tests unitarios y de integración con fixtures que validan contratos específicos.
3. **Validación en entorno de referencia:** ejecuciones comprobadas sobre entornos reales o controlados con versiones y fechas explícitas.
4. **Límites pendientes o no demostrados:** capacidades que carecen de pruebas en runtime activo o que dependen de comportamiento externo de `agy`.

### Estado de validación actual
En el commit de referencia de esta versión (`628bc61ec2851040f2c95de150d8c9e62883b17d`, validado el 2026-10-06):
- **Tests unitarios:** 17 paquetes completados con éxito (`PASS`, exit 0).
- **Pruebas integrales:** Las pruebas de flujo guiado (driven) y orgánicas en entornos de producción permanecen como seguimiento separado.
- Para el desglose detallado entre pruebas unitarias, mocks y tests integrales, consulta el [Registro de verificación](09-verification.md) y el [Índice de fuentes](10-source-index.md).

## Límites que conviene conocer antes de usar

- Usar alcance global como ruta documentada. El alcance workspace tiene una discrepancia entre componentes y despliegue/import global: [análisis de rutas](01-foundation.md#alcance-workspace-no-es-un-camino-garantizado).
- No guardar personalizaciones dentro de `agents/` o `skills/` del plugin: el reconciliador poda directorios ajenos al bundle: [`DeployPluginTree`](../../internal/agents/antigravitycli/adapter.go#L589-L625).
- La revisión conserva `HOME` para resolver configuración/credenciales y no desactiva explícitamente toda discovery de plugins o herramientas: [frontera de aislamiento](05-review-transport.md#aislamiento-solicitado-no-cleanroom-demostrado).
- Un exit 0 de headless no garantiza éxito de todas las herramientas: la documentación oficial describe denegaciones blandas. Véase [fuentes oficiales](10-source-index.md#fuentes-oficiales-y-observaciones-locales).
- La separación CLI/IDE tiene un canario limitado, no una demostración universal de rutas disjuntas: [comparación IDE](../antigravity-ide-vs-cli.md).

**Siguiente paso:** [leer qué cambia la configuración global](01-foundation.md), antes de autorizar escrituras.
