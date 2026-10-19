# 6. Propiedad, import, deriva y retirada

**El plugin `gentle-ai` es un destino administrado, no una carpeta segura para personalizaciones.** La procedencia del bundle y la conservación de JSON ajeno son contratos específicos; no prometen preservar todo archivo ni mantener automáticamente cada copia interna del cliente.

[Índice](README.md) · [Transpilador](02-asset-transpiler.md) · [Decisiones](08-decisions.md)

## Mapa de propiedad actual

| Superficie | Owner de contenido | Cómo se escribe | Protección/límite |
|---|---|---|---|
| `internal/assets/antigravitycli/` | Fuente Gentle Shell más transformaciones Gentle AI | Transpilador; luego embed | Pin y `--check` contra committed HEAD |
| Plugin agents/skills | Bundle compilado | `DeployPluginTree` | Podado de directorios fuera del set actual |
| Satélites/support/chain/plugin.json | Bundle compilado | `BundleAssets`, escritura por archivo | Assets propios reemplazados; no editar in situ |
| Regla orchestrator final | Bundle + contrato proveedor Go | Finalización CLI | Frontmatter y marker/contrato, test idempotencia/tamaño |
| Routing | Componente agentguidance | Archivo único protocol-only | Marker propio para stale-part cleanup |
| AGENTS.md | Componentes/secciones administradas | Estrategia append/merge | Distinto de asset orchestrator literal |
| MCP | Componentes Engram/Context7/CodeGraph | Merge/reconcile de entradas | Base malformed puede ser reemplazada |
| settings.json | Usuario más overlay de permisos | Merge de JSON | Conservación semántica de base válida, no todos los bytes |
| review-models.json | Usuario | Resolver read-only; picker/save aparte | Sin repo fallback; invalid fail-closed |
| import_manifest.json | Cliente agy más preservación del producto | Snapshot, comando externo, merge/restore | Conservar root/imports ajenos del snapshot |

Fuentes: [`DeployPluginTree`](../../internal/agents/antigravitycli/adapter.go#L474-L638), [`injectAntigravityCLIRule`](../../internal/components/agentguidance/inject.go#L320-L382), [`ensureAntigravityOrchestratorRule`](../../internal/cli/antigravity_orchestrator_rule.go#L28-L52), [`ResolveAntigravityCLIReviewRouting`](../../internal/reviewerprovider/antigravitycli_routing.go#L364-L425), [merge de import](../../internal/agents/antigravitycli/adapter.go#L352-L375).

El componente del catálogo de skills **omite agy**. No existe en esta entrega una política agy de «añadir `--skills` para completar skills ausentes del bundle»: [`DeliversFor`](../../internal/components/skills/inject.go#L28-L46). Las exclusiones de presets del catálogo de otros agentes no describen qué llega a este plugin.

## Qué se poda y por qué importa

`DeployPluginTree` calcula el set desde `assets.FS`. Elimina directorios `agents/` y `skills/` que no pertenecen al bundle, incluso si fueron personalizados o añadidos manualmente. Limpia también paths legacy concretos, el ledger Pi y routing duplicado en AGENTS.md: [líneas 589–636](../../internal/agents/antigravitycli/adapter.go#L589-L636).

| Ubicación | Cambio de sync | Recomendación |
|---|---|---|
| `agents/custom/` del plugin | Puede desaparecer | Usar superficie propia del cliente, fuera del plugin administrado |
| `skills/go-testing/` no incluido en bundle | Se poda | No asumir que selección de catálogo lo restaura para agy |
| `skills/_shared/review-ledger-contract.md` viejo | Se elimina | No volver a copiar ledger específico Pi |
| `rules/gentle-ai-routing-5.md` con marker | Se elimina como stale | Mantener notas humanas fuera del namespace administrado |
| `rules/gentle-ai-routing-notes.md` sin marker | Writer de routing lo conserva | No confundir conservación de este archivo con protección de todo el árbol |
| Asset administrado de mismo nombre | Se reemplaza | Cambiar canon/transpilador, no installed bytes |

Tests existentes: [`TestDeployPluginTree`](../../internal/agents/antigravitycli/adapter_test.go#L612-L725), [`TestDeployPluginTreePrunesStaleSkills`](../../internal/agents/antigravitycli/adapter_test.go#L727-L805), [`StalePartCleanup`](../../internal/components/agentguidance/inject_test.go#L1086-L1140).

Algunas eliminaciones legacy son best-effort e ignoran errores. No describir el retorno nil del deploy como prueba de que todo archivo obsoleto fue eliminado en cualquier filesystem. Tampoco es un directory swap: la atomicidad es por archivo.

## Primer import y registro

La operación `EnsurePluginImported` está separada del despliegue:

```text
snapshot import_manifest.json
  ├─ malformed/symlink/no regular → error antes de comando externo
  ├─ registro gentle-ai ya existe → no ejecutar import
  └─ registro falta
       → verificar árbol regular/no symlink
       → copiar árbol a source temporal distinto
       → agy plugin install <staging>, HOME/USERPROFILE = home elegido
       → exigir exactamente un nuevo registro gentle-ai
       → merge con snapshot original si existía
       → restaurar snapshot en fallo, reportando fallo de restauración si ocurre
```

[`readPluginImportManifest`](../../internal/agents/antigravitycli/adapter.go#L296-L350), [`EnsurePluginImported`](../../internal/agents/antigravitycli/adapter.go#L643-L732), [`copyPluginTree`](../../internal/agents/antigravitycli/adapter.go#L749-L793).

### Preservación frente a rewrite del cliente

El producto conserva fields root y entries ajenos como raw JSON del snapshot. El registro `gentle-ai` creado por agy se añade a ese original. La semántica ajena y el modo original se preservan en el merge; formato/whitespace global puede normalizarse. En un import ya existente no se reescribe el manifest por esa función.

`TestEnsurePluginImportedMergesDestructiveManifest` simula que el cliente destruye root/imports y comprueba reparación y modo: [test, líneas 268–343](../../internal/agents/antigravitycli/adapter_test.go#L268-L343). `ManifestGate` comprueba no-command/no-mutation cuando ya existe registro o la base es inválida: [líneas 345–380](../../internal/agents/antigravitycli/adapter_test.go#L345-L380).

No afirmar «registra siempre aunque el comando salga 0»: falta de registro o más de uno se rechazan. No afirmar «rollback total de todos los efectos agy»: la restauración mostrada cubre el manifest; el cliente externo puede tener otros efectos no modelados aquí.

### Tiempo y entorno del import

- Timeout predeterminado **20 s**; WaitDelay **1 s**.
- Usa `CombinedOutput` para detalle del comando.
- Entorno del import parte de `os.Environ`, sustituye HOME y USERPROFILE. **No es la allowlist mínima del reviewer.**
- La ayuda de error dice instalar agy 1.2.9 o posterior; no ejecuta un gate semver previo al import.

Fuentes: [constantes](../../internal/agents/antigravitycli/adapter.go#L38-L43), [env/comando](../../internal/agents/antigravitycli/adapter.go#L670-L698), [`runPluginInstall`](../../internal/agents/antigravitycli/adapter.go#L734-L739).

`TestEnsurePluginImportedFailures` cubre binario, árbol, comando fallido, ausencia de registro y timeout con runner falso: [líneas 382–427](../../internal/agents/antigravitycli/adapter_test.go#L382-L427). Esto no mide los tiempos del agy local.

## Qué no debe prometernos el import

1. **No prueba activación en cada versión.** El registro es una señal del cliente/contrato, no inspección del contexto del modelo.
2. **No prueba recarga de cambios.** Se evita reimport cuando existe registro; debe comprobarse cómo el cliente usa las copias/rutas en la versión instalada.
3. **No prueba salud MCP.** El manifest de import y la lista MCP global tienen poblaciones diferentes.
4. **No es universalmente pasiva la consulta `plugin list`.** La documentación histórica anterior advertía rewrite del manifest por agy 1.2.9. Aquí se conserva como precaución histórica, no como fallo confirmado de 1.3.0.
5. **No garantiza rutas workspace.** El paso final usa HOME real en install/sync; [análisis](01-foundation.md#alcance-workspace-no-es-un-camino-garantizado).

El mock de [j4500](../../bench/journeys_antigravity_cli.go#L27-L67) reproduce el rewrite destructivo atribuido a 1.2.9. La fecha/timestamp en ese script es dato fixture, no evidencia de un import realizado en esta tarea.

## Deriva reproducible y alcance de `--check`

`runCheckGate`:

1. Extrae el árbol **committed en HEAD** y su procedencia.
2. Acepta `--source-dir` solo si su HEAD coincide con el pin; si no, usa clon remoto con checkout del pin.
3. Regenera en temporal y escribe procedencia determinista.
4. Compara set de archivos y bytes contra el árbol committed.
5. Falla con diferencias si no coinciden.

[main.go, líneas 1571–1667](../../scripts/transpile-antigravitycli/main.go#L1571-L1667).

| Pregunta | ¿La responde `--check`? |
|---|---|
| ¿HEAD reproduce los bytes del pin con este transpilador? | Sí, es su contrato |
| ¿El bundle dirty del worktree tiene todos sus cambios revisados? | No: compara HEAD, no esa copia |
| ¿El installed plugin coincide con el binary actual? | No |
| ¿El cliente carga rules/MCP/agentes? | No |
| ¿La versión upstream más nueva está incorporada? | No; reproducibilidad y freshness son diferentes |
| ¿Los modelos respetan cada contrato? | No |

No correr el gate como un comando read-only puro: puede clonar/generar temporales. En esta tarea no se ejecutó ni se regeneraron assets. La página [verificación](09-verification.md) conserva esta distinción.

## Retirada y limpieza: ownership antes que «borrar todo»

El servicio de uninstall agrega operaciones retenidas del plugin cuando se retiran todos los componentes del agente: [service.go, líneas 553–558](../../internal/components/uninstall/service.go#L553-L558). `retainedAntigravityCLIPluginOperations` enumera bundle, reglas administradas, agentes embedded, hooks, entrada CodeGraph y registro de import: [líneas 1408–1460](../../internal/components/uninstall/service.go#L1408-L1460).

No es un `RemoveAll(pluginDir)` universal. Se eliminan directorios si quedan vacíos y se preservan registros/imports ajenos. Componentes parciales no implican automáticamente retirada total. Datos del cliente fuera de las rutas enumeradas no quedan cubiertos por ese plan.

El journey utiliza `uninstall --agents antigravity-cli --yes` en fixture; la guía no ejecutó uninstall sobre HOME real. No ofrecer `--components sdd` como receta v4 ni afirmar que borra conversaciones, caches o runtime MCP agy.

## Ruta de mantenimiento responsable

- Cambio de doctrina compartida → cambiar canon Gentle Shell y revisar adaptación/pin.
- Cambio de mecanismo agy → cambiar transpilador o runtime Go correspondiente, con tests de esa frontera.
- Cambio de contrato RDD → owner Go; no editar ledger Pi ni manual chain para obtener autoridad.
- Cambio de configuración personal → fuera del bundle o vía picker de modelos que corresponde.
- Actualización de pin → reproducibilidad, revisión de diff semántico y runtime observado; no solo regenerar sin mirar.

[Decisiones y alternativas](08-decisions.md) distingue motivos históricos documentados de beneficios inferidos.
