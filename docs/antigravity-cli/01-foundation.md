# 1. Fundamentos, rutas y puesta en marcha

**Esta integración es completamente independiente del adaptador para Antigravity IDE: se enfoca exclusivamente en el ejecutable de terminal `agy` (Antigravity CLI).** Configura `agy` mediante su arquitectura nativa de plugins, hooks y MCP, operando sobre un ejecutable ya instalado sin requerir extensiones gráficas.

[Índice](README.md) · [Arquitectura](SUMMARY.md) · [Fuentes](10-source-index.md)

## Identidad y detección del agente

El adaptador [`antigravitycli`](../../internal/agents/antigravitycli/adapter.go) implementa el soporte para Google Antigravity CLI bajo el identificador `antigravity-cli` (`model.AgentAntigravityCLI`):

- **Detección del binario:** Busca `agy` en las rutas del sistema mediante `exec.LookPath("agy")`.
- **Detección de configuración:** Comprueba la existencia del directorio de configuración global en `~/.gemini/antigravity-cli/`.
- **Nivel de soporte:** Declara `model.TierFull`, habilitando el despliegue del conjunto completo de componentes (reglas, subagentes nativos, MCP, skills y revisión RDD).
- **Pruebas unitarias:** Los escenarios de detección y resolución de rutas están cubiertos en [`adapter_test.go`](../../internal/agents/antigravitycli/adapter_test.go).

## Tres clases de rutas

### 1. Configuración de aplicación

```text
~/.gemini/antigravity-cli/
├── settings.json                    # overlay de permisos
└── gentle-ai/
    └── review-models.json           # asignaciones de revisión del usuario
```

`GlobalConfigDir` y `SettingsPath`: [adapter.go, líneas 112–148](../../internal/agents/antigravitycli/adapter.go#L112-L148). La ruta de modelos se resuelve mediante `os.UserHomeDir`, no el repositorio: [`AntigravityReviewModelsPath`](../../internal/reviewerprovider/antigravitycli_routing.go#L75-L81).

### 2. Árbol canónico escrito por Gentle AI

```text
~/.gemini/config/
├── import_manifest.json
└── plugins/gentle-ai/
    ├── plugin.json
    ├── mcp_config.json
    ├── hooks.json
    ├── hooks/{hook.sh,hook.cmd}
    ├── rules/
    │   ├── AGENTS.md
    │   ├── gentle-ai-routing.md
    │   └── gentle-ai-orchestrator.md
    ├── agents/<nombre>/agent.md
    ├── skills/<directorio>/...
    ├── orchestrator-{delegation,memory,skills,prompts,tracking,verification,writer}.md
    ├── support/{strict-tdd.md,strict-tdd-verify.md}
    └── chains/4r-review.chain.md
```

Las rutas y mappings pertenecen a [`PluginDir`, `BundleAssets`, `SubAgentsDir`](../../internal/agents/antigravitycli/adapter.go#L116-L216). `AGENTS.md`, routing y MCP reciben contenido por componentes separados; no todo el árbol es una copia literal del asset: [finalización de reglas](03-orchestration-and-hooks.md#reglas-y-carga-de-contexto).

### 3. Rutas de plugins según la especificación de agy

Antigravity CLI reconoce dos ubicaciones oficiales para plugins:
- **Global:** `~/.gemini/config/plugins/<nombre>` (gestionado con registro en `~/.gemini/config/import_manifest.json`).
- **Por proyecto (Workspace):** `<workspace>/.agents/plugins/<nombre>` (descubierto automáticamente por `agy` al ejecutarse en el proyecto).

## Puesta en marcha (Global y Workspace)

Gentle AI soporta instalación y sincronización en ambos alcances:
- **Global (`--scope global`)**: Despliega el plugin en `~/.gemini/config/plugins/gentle-ai` y lo registra en el manifiesto global para que esté disponible en cualquier sesión de `agy`.
- **Workspace (`--scope workspace`)**: Despliega el plugin en `<workspace>/.agents/plugins/gentle-ai/`, aislado al repositorio actual. `agy` lo descubre automáticamente sin tocar la configuración de usuario ni mutar `HOME`.

### Antes de autorizar escrituras

1. Confirmar que el binario `gentle-ai` corresponde al módulo v4: [go.mod](../../go.mod#L1-L3).
2. Confirmar binario y disponibilidad de `agy` según [Headless](https://antigravity.google/docs/cli/headless/). La documentación oficial exige credenciales interactivas previas; esta guía no automatiza login.
3. Decidir el alcance deseado (`--scope global` para todas las carpetas o `--scope workspace` para aislarlo al proyecto actual).
4. Guardar personalizaciones propias fuera del árbol del plugin administrado, ya que el sincronizador poda directorios ajenos al bundle.

### Comandos de instalación y sincronización

Con ambos binarios ya disponibles:

```sh
# Instalación global (afecta a todas las sesiones de agy del usuario)
gentle-ai install --agent antigravity-cli --scope global
gentle-ai sync --agents antigravity-cli --scope global

# Instalación en workspace (aislada a este repositorio)
gentle-ai install --agent antigravity-cli --scope workspace
gentle-ai sync --agents antigravity-cli --scope workspace
```

Las opciones `--agent`/`--agents` y `--scope` están registradas por [`ParseInstallFlags`](../../internal/cli/install.go#L53-L70) y [`ParseSyncFlags`](../../internal/cli/sync.go#L171-L193). La resolución de alcances está definida en [`ResolveInstallScope`](../../internal/cli/scope.go#L24-L44). Los comandos configuran el entorno para el agente, pero no instalan el binario `agy`: el adaptador no cuenta con autoinstalación de ejecutables externos.

En alcance global, la secuencia del paso final es `DeployPluginTree → ensureAntigravityOrchestratorRule → EnsurePluginImported`: [`antigravityCLIPluginImportStep.Run`](../../internal/cli/run.go#L921-L940). En alcance workspace, se despliega el árbol en `<workspace>/.agents/plugins/gentle-ai/` y se inicializa la regla de orquestador localmente, prescindiendo del registro en el manifiesto global.

### Estructura desplegada en cada alcance

Ambos alcances despliegan el bundle completo de assets:

```text
# En --scope global:
~/.gemini/config/
├── import_manifest.json
└── plugins/gentle-ai/
    ├── plugin.json
    ├── mcp_config.json
    ├── hooks.json
    ├── hooks/{hook.sh,hook.cmd}
    ├── rules/
    │   ├── AGENTS.md
    │   ├── gentle-ai-routing.md
    │   └── gentle-ai-orchestrator.md
    ├── agents/<nombre>/agent.md
    ├── skills/<directorio>/...
    ├── orchestrator-{delegation,memory,skills,prompts,tracking,verification,writer}.md
    ├── support/{strict-tdd.md,strict-tdd-verify.md}
    └── chains/4r-review.chain.md

# En --scope workspace:
<workspace>/.agents/plugins/gentle-ai/
├── plugin.json
├── mcp_config.json
├── hooks.json
├── hooks/{hook.sh,hook.cmd}
├── rules/
│   ├── AGENTS.md
│   ├── gentle-ai-routing.md
│   └── gentle-ai-orchestrator.md
├── agents/<nombre>/agent.md
├── skills/<directorio>/...
├── orchestrator-{delegation,memory,skills,prompts,tracking,verification,writer}.md
├── support/{strict-tdd.md,strict-tdd-verify.md}
└── chains/4r-review.chain.md
```

### Particularidades por alcance:
1. **Descubrimiento automático en workspace:** `agy` descubre nativamente los plugins situados en `.agents/plugins/<nombre>/`, por lo que el instalador no modifica `~/.gemini/config/import_manifest.json`.
2. **Aislamiento de `HOME`:** En `--scope workspace`, el directorio de usuario (`HOME`) permanece completamente inalterado.
3. **Precedencia:** Cuando existen ambos, las reglas y configuraciones del workspace tienen prioridad sobre las globales.

### Verificación automatizada tras la instalación o sincronización

No hace falta inspeccionar carpetas a mano. Podés validar la instalación directamente con las herramientas de línea de comandos:

1. **Verificar que `agy` cargó el plugin y sus componentes:**
   ```bash
   agy plugins list
   ```
   Debe responder confirmando el import de `gentle-ai` con sus 4 componentes activos:
   ```json
   {
     "imports": [
       {
         "name": "gentle-ai",
         "source": "antigravity",
         "components": [
           "skills",
           "agents",
           "mcpServers",
           "hooks"
         ]
       }
     ]
   }
   ```

2. **Verificar la salud del entorno y servicios MCP:**
   ```bash
   gentle-ai doctor
   ```
   Comprueba que el agente `antigravity-cli` esté registrado, el archivo de estado sea consistente y el servidor MCP de memoria (Engram) responda en su canal stdio.

## Permisos: overlay de seguridad en settings.json

Para evitar comandos destructivos accidentales o fuga de secretos en sesiones automáticas, Gentle AI inyecta en `settings.json` el overlay completo [`antigravityCLIOverlayJSON`](../../internal/components/permissions/inject.go#L132-L159):

```json
{
  "permissions": {
    "deny": [
      "command(sudo rm -rf /)",
      "command(rm -rf ~)",
      "command(sudo rm -rf ~)",
      "command(git reset --hard)",
      "command(git clean -fd)",
      "command(git push --force)",
      "command(chmod -R 777)",
      "command(chown -R)",
      "read_file(.env)",
      "write_file(.env)",
      "read_file(.ssh)",
      "write_file(.ssh)",
      "read_file(.credentials)",
      "write_file(.credentials)",
      "read_file(.aws)",
      "write_file(.aws)"
    ],
    "allow": [
      "command(gentle-ai codegraph init)",
      "command(gentle-ai review assess)",
      "command(gentle-ai review status)"
    ]
  }
}
```

- **Bloqueos preventivos (`deny`):** 8 comandos de terminal destructivos o irreversibles y 8 operaciones de lectura/escritura sobre archivos sensibles (`.env`, `.ssh`, `.credentials`, `.aws`).
- **Comandos permitidos (`allow`):** Comandos internos indispensables para la navegación de código y la revisión RDD.
- **Fusión inteligente:** Gentle AI combina estas listas con tu configuración previa sin pisar preferencias de usuario.
- **Backup previo:** Antes de cualquier escritura en `settings.json`, se genera un respaldo automático recuperable.

## CLI frente a IDE

Gentle AI ofrece dos integraciones distintas para el ecosistema de Antigravity:
- **Antigravity CLI (`agy`):** Despliega el plugin completo con reglas, subagentes nativos, MCP servers y revisión RDD vía subproceso Go.
- **Antigravity IDE:** Configura directivas a través de archivos de contexto como `GEMINI.md` en el workspace.

Para una comparación detallada entre ambos modelos, consultá [Antigravity IDE frente a CLI](../antigravity-ide-vs-cli.md).

---

**Siguiente paso:** [Cómo Gentle Shell se convierte en assets agy](02-asset-transpiler.md).
