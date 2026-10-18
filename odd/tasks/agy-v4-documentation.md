# Documentación verificable de agy para Gentle AI v4

## Objetivo y motivo
Documentar en español la integración de Antigravity CLI (`agy`) en la rama `antigravity-cli-v4`, su base en Gentle Shell, su funcionamiento y decisiones, con referencias a código y pruebas y una comparación por capacidad con los agentes de primera clase.

## Alcance y restricciones
- Documentación técnica extensa dentro de `docs/antigravity-cli/`; actualizar enlaces y afirmaciones de entrada si hace falta.
- Investigar código, tests, procedencia de assets y fuentes oficiales. El texto anterior no es prueba de funcionamiento.
- No usar memoria para responder sobre código o funcionamiento; solamente para decisiones históricas. El espejo de esta lista es seguimiento, no evidencia técnica.
- Separar implementación, cobertura existente, pruebas ejecutadas, observación del runtime y expectativas no demostradas.
- No modificar comportamiento, instalar plugins, ejecutar revisiones con autoridad, publicar ni crear commits sin autorización explícita.
- Un solo escritor. Exploración y verificación delegadas en modo de solo lectura.
- No corresponde RED/GREEN: documentación pasiva sin cambio de comportamiento. Sí corresponde verificar referencias, coherencia y tests aplicables al funcionamiento descrito.

## Tareas
- [x] T1 — Investigar y clasificar evidencia de integración, comparación y procedencia. Mapa primario observado; verificación de tests separada en T3.
- [x] T2 — Redactar la documentación detallada en español con decisiones, límites y referencias verificables. Doce documentos (aprox. 15.900 palabras), entradas relacionadas y comandos literales incorporados; `git diff --check` PASS.
- [x] T3 — Auditar la trazabilidad del texto contra código y tests y comprobar enlaces. Revalidación independiente: 521 enlaces locales, cero incidencias; nueve URLs únicas HTTP 200; 30 correcciones aceptadas.
- [x] T4 — Incorporar los resultados, revisar navegación, restaurar perfil temporal y entregar el índice con verificaciones pendientes explícitas. Auditoría final registrada en `09-verification.md`, índice revisado y perfil original restaurado.

## Criterios de aceptación
- Cada afirmación importante sobre funcionamiento tiene una referencia concreta a implementación y, cuando existe, un test relevante.
- La matriz comparativa distingue capacidades declaradas, implementadas y demostradas, sin promesas genéricas de paridad.
- Las decisiones distinguen evidencia histórica de razonamiento inferido.
- Hay instrucciones reproducibles, resultados observados y controles negativos cuando los tests los cubren.
- Las fuentes externas incluyen URL y versión o fecha observada; una fuente no accesible se declara no verificada.
- Las limitaciones de pruebas unitarias frente a ejecución real de agy están explícitas.

## Progreso y evidencia
- Estado inicial: rama `antigravity-cli-v4`, sin cambios informados por `git status --short`.
- Existe una colección previa en `docs/antigravity-cli/` que se investigará, no se tomará como autoridad.
- Exploración delegada: `muwuery7-1-8njs` terminó con `assistant reported an error`, sin mapa de evidencia utilizable. T1 permanece incompleta.
- Preferencia explícita: documentación en español; extensión y tiempo de investigación no son restricciones.
- Investigación externa: las delegaciones `muwui0dt-2-y4ol` y `muwv8p16-3-yc7z` fallaron sin evidencia. Diagnóstico del registro de la segunda: proveedor `nvidia`, modelo `deepseek-ai/deepseek-v4.1-flash`, error repetido `Request timed out.`, cero llamadas a herramientas. No es evidencia de un fallo en agy.
- Recuperación autorizada con «continuá» después de proponer el perfil alternativo: creado `.pi/subagents.json` exclusivamente para los roles explore/worker/verify del proyecto, con `openai-codex/gpt-6.1-sol` y esfuerzo alto; configuración global intacta. La precedencia proyecto/global está implementada en `lib/agents-config.ts` del paquete instalado, líneas 273 y 297. Pendiente comprobar evidencia de ejecución exitosa.
- `muwx0rgb-1-ty2y` se canceló al cerrar la sesión del padre; no se toma como trabajo recuperado. El reintento `muwx9iv3-1-bq1f` falló con 429/servicio sobrecargado, todavía en NVIDIA. `muwxaxtq-2-2n8s` se canceló para evitar duplicar trabajo con ese perfil.
- Diagnóstico confirmado: `.git/gentle-ai/profile-pin.json` fijaba `gpt-nomtron`; la implementación `withPinnedModelProfiles` reemplaza la selección local. Se apartó reversiblemente como `.git/gentle-ai/profile-pin.json.agy-docs-backup` para hacer efectivo el perfil local autorizado; no se modificó el almacén global. El pin afecta al clon y sus worktrees, no solamente a un archivo de configuración de este worktree.
- Reintento con pin apartado: mapa de código `muwxdia9-3-td2l` completado; referencias a implementación, tests y fuentes canónicas recibidas. El mapa distingue los cinco agentes TierFull, assets derivados de Gentle Shell con SHA, políticas de prompts frente a enforcement, hooks condicionalmente fail-closed y bench mock frente a runtime autenticado. Es evidencia de exploración, no de tests ejecutados.
- Plan documental: reescribir en español los ocho archivos existentes de `docs/antigravity-cli/`; ampliar con comparación, catálogo de pruebas/decisiones y referencias cuando ayude a navegación. No convertir el comportamiento ambiguo de workspace en instrucciones de instalación garantizadas. Corregir solamente afirmaciones de entrada relacionadas en README y comparación IDE/CLI. La comprobación oficial/local terminada fue `muwx1ef0-2-8g8n`; el rol explore no tiene web/bash, por lo que la comprobación HTTP se asignó al verificador.
- Perfil temporal restaurado al cierre: `.git/gentle-ai/profile-pin.json` volvió a `gpt-nomtron`, sin sobrescribir cambios ajenos; `.pi/subagents.json` retirado de discovery y conservado como backup `.git/gentle-ai/agy-docs-subagents-backup.json`. Se comprobó el contenido esperado antes de moverlo. Configuración global intacta.
- Acceso directo: las páginas oficiales de headless y agentes devolvieron contenido HTML; la ruta `/docs/cli/plugins/` redirige por HTML a `https://antigravity.google/docs/plugins?tab=cli`. Acceso no equivale a validación del contenido técnico.
- Verificación local recibida: `agy --version` informa `1.3.0`, binario `$HOME/.local/bin/agy`; `agy --help` enumera los flags de headless, agente, modelos, permisos y sandbox. Esto no prueba la integración en ejecución.
- Fuentes oficiales recuperadas mediante `fetch_content` en modo readable (respuesta `muwx602but8kzc`): headless describe `-p`, JSON/NDJSON, denegaciones blandas y permisos; agentes describe `agent.md`; reglas documenta 24.000 bytes por archivo tras includes y 20.000 tokens agregados; hooks documenta eventos sin `SessionStart`; plugins describe manifiesto y estructura. Se debe distinguir el tab y superficie CLI/2.0/IDE antes de usar rutas generales como prueba de CLI.
- El parser HTTP delegado no recuperó adecuadamente varias páginas; su falta de coincidencias no demuestra ausencia de contratos. La continuación externa `muwx4fnt-3-73ey` falló; no se usa como evidencia.
- Escritor documental único `muwxnuan-4-2swk`: autorizado para `docs/antigravity-cli/*.md`, comparación IDE/CLI y solamente la entrada agy del README raíz. Sin cambios de comportamiento ni commits.
- Verificador `muwxo7oc-5-e4i2`: ejecuta las cinco agrupaciones de paquetes Go identificadas y `(cd bench && go test ./...)`, en modo de solo lectura salvo caches y fixtures temporales propios de tests. Es comprobación de evidencia de T2; la auditoría del texto final corresponde a T3.
- Spot-check del padre: `internal/reviewerprovider/antigravitycli_adapter.go:34–71` confirma stdin, omisión intencional de `-p`, flag `--sandbox`, directorio temporal y bytes crudos. Pedir sandbox no prueba por sí solo su enforcement en un runtime autenticado.
- Tests orgánicos autenticados y journeys driven: no ejecutados en esta etapa; no se lanzan modelos remotos ni instalaciones.
- Verificación automática terminada (`muwxo7oc-5-e4i2`): siete comandos exit 0; 17 paquetes PASS y cero FAIL; sin resultados `(cached)`. Snapshot funcional HEAD `628bc61ec2851040f2c95de150d8c9e62883b17d`, Go `1.27.1 linux/amd64`, agy `1.3.0`.
- Comandos PASS: `go test ./internal/catalog ./internal/agents ./internal/agents/antigravitycli ./internal/agents/capabilitymanifest`; `go test ./scripts/transpile-antigravitycli ./internal/assets`; `go test ./internal/components/agentguidance ./internal/components/persona ./internal/components/skills ./internal/components/permissions ./internal/components/uninstall ./internal/components/communitytool`; `go test ./internal/reviewerprovider ./internal/cli ./internal/tui ./internal/tui/screens`; `(cd bench && go test ./...)`. `internal/cli` tardó 323,877 s; bench 4,787 s. Salida no verbose: no prueba conteos individuales ni tests omitidos.
- Control de mutaciones: verificador observó archivos documentales modificados durante la ejecución y no atribuyó su autoría. Padre comprobó que las siete rutas modificadas actuales pertenecen a la superficie del escritor activo; no hay cambios de código. Se conserva esta distinción, sin atribuir a tests cambios documentales.
- Escritor `muwxnuan-4-2swk` entregó 14 archivos documentales y estado parcial por faltar la transcripción de comandos. Padre incorporó mecánicamente los siete comandos literales en `09-verification.md` sin reejecutarlos ni inventar flags. Doce páginas de la colección suman 1.596 líneas y 15.899 palabras tras ese cambio.
- `git diff --check` del padre PASS. Assessment nativo read-only no pudo clasificar por nuevos archivos untracked no declarados; plan fail-closed exige verificación independiente, que se aplica a la auditoría documental T3. No hay cierre/autoridad de revisión inferido de este fallo.
- T3: auditoría `muwykmw4-6-y6o9` terminó; 14 archivos leídos, 521 enlaces locales comprobados, 24 externos no comprobados por red. Quince áreas contrastadas directamente contra fuente. Contenido/decisiones/comparación aceptables, pero 29 rangos de citas exceden sus archivos; una cita no alcanza el retorno allow y una frase debe distinguir unión de deny frente a allow/ask en permisos. Corrección `muwyra4y-7-6q9t` completada: 30 anchors y precisión de merge; whitespace PASS. Revalidación independiente `muwyx4yn-8-koni` completada: 14 archivos, 521 enlaces locales, 24 ocurrencias externas y cero incidencias; nueve URLs únicas todas HTTP 200, cero errores. Las 30 referencias y la precisión de merge fueron aceptadas; siete comandos literales preservados. Existencia/acceso no equivalen a validación semántica exhaustiva de todas las citas ni fragments remotos. Sin commits autorizados ni realizados.
- Preflight RDD inspect: seleccionadas las cuatro páginas nuevas, excluido el seguimiento. STOP `managed_assets_outdated`; no lineage creado ni START ejecutado. Continuación nativa ofrecida: `$HOME/.local/share/mise/installs/node/26.2.0/lib/node_modules/gentle-pi/.gentle-ai/v4.0.0/gentle-ai sync --agent pi`. No ejecutada: actualizar assets Pi excede el alcance documental y la configuración local autorizada.
- Assessment con outcome explícito `unavailable`: unassessable por declaración untracked; fallback exige auditoría independiente. Se cumple con el verificador documental, sin presentar su resultado como aprobación RDD nativa.
- Commits: no autorizados; no creados.

## Límite de rollback
Las unidades documentales se revierten quitando únicamente sus cambios en `docs/antigravity-cli/` y enlaces de documentación asociados; el seguimiento se limita a este archivo. Ningún cambio de comportamiento pertenece a estas unidades.

## Próximo paso
Documentación entregable desde `docs/antigravity-cli/README.md`; comparar capacidades en `07-comparison.md`, decisiones en `08-decisions.md` y pruebas/límites en `09-verification.md`. Cierre del padre: `git diff --check` PASS después del registro de auditoría y la restauración. Solo cambios documentales y seguimiento permanecen en Git; sin staging, commits ni publicación.

Pendientes explícitos fuera del alcance documental: pruebas orgánicas autenticadas, ejecución driven de j4500, gate upstream real `--check`, sandbox/discovery/activación y workspace en cliente real, y aprobación RDD nativa bloqueada por assets desactualizados. La accesibilidad externa no certifica fragments ni semántica completa. El usuario decide si autoriza esas verificaciones o prepara commits/PR.
