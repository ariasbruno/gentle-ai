# Guía de Instalación: Antigravity CLI (`agy`) con Gentle AI

Esta guía detalla el proceso completo de instalación, configuración, migración desde upstream y verificación de la integración de Antigravity CLI (`agy`) en Gentle AI.

> [!IMPORTANT]
> **Versión fijada actual**: **v4.2.1** (basada en upstream Gentle AI v4.2.0, Gentle Shell v4.3.0 y Antigravity CLI v1.4.4).

---

## 1. Prerrequisitos

Antes de comenzar, verificá que tu entorno cumpla con los siguientes requisitos:

1. **Antigravity CLI (`agy`)**: Instalado y con sesión activa (`agy --version` v1.4.4 o superior).
2. **Go**: Versión 1.25 o superior instalada en tu sistema (`go version`).
3. **Git**: Instalado y disponible en el path del sistema.

---

## 2. Obtención e Instalación del Binario `gentle-ai` (Desde este Fork)

Debés compilar e instalar el binario correspondiente a este fork (ya que upstream aún no incorpora este adaptador dedicado):

```bash
# 1. Clonar el repositorio en la rama fijada de la integración:
git clone -b agy/ai-v4.2.0_shell-v4.3.0 https://github.com/ariasbruno/gentle-ai.git
cd gentle-ai

# 2. Compilar e instalar el binario en tu sistema:
go install ./cmd/gentle-ai

# 3. Comprobar que el binario instalado esté disponible:
gentle-ai --version
```

---

## 3. Instalación en Antigravity CLI: Migración vs Instalación Limpia

Antes de ejecutar la instalación, determiná cuál es tu situación:

### Caso A: Si ya usabas Gentle AI oficial / upstream (Migración obligatoria)
En upstream existe un adaptador genérico llamado `antigravity` que intentaba abarcar todo el ecosistema (IDE, 2.0 y CLI) escribiendo configuraciones tanto en `.gemini/antigravity-cli/` como en `GEMINI.md`.

> [!WARNING]
> **No es posible usar ambas integraciones al mismo tiempo:** Ambos adaptadores configuran el entorno de Antigravity en tu máquina. Mantener los dos activos provocaría duplicación de skills y configuraciones redundantes. **Debés desinstalar el adaptador genérico antes de instalar el nuevo.**

1. **Paso 1: Desinstalá el adaptador genérico de upstream:**
   ```bash
   gentle-ai uninstall --agent antigravity
   ```
   *(Este comando remueve las configuraciones y reglas genéricas viejas sin tocar el resto de tus herramientas).*

2. **Paso 2: Instalá el adaptador dedicado para Antigravity CLI:**
   ```bash
   gentle-ai install --agent antigravity-cli --scope global
   ```
   *(Opcional: Si querés que además configure la StatusLine y el título dinámico, sumá el flag `--include-theme` al comando anterior).*

---

### Caso B: Instalación limpia (Primera vez con Gentle AI)

Si nunca antes habías instalado Gentle AI en tu máquina, elegí el alcance según tu flujo de trabajo:

| Dimensión | Alcance Global (`--scope global`) | Alcance Workspace (`--scope workspace`) |
|---|---|---|
| **Destino en disco** | `~/.gemini/config/plugins/gentle-ai/` | `<repo>/.agents/plugins/gentle-ai/` |
| **Afecta a** | Todos tus repositorios y proyectos | Exclusivamente al repositorio actual |
| **Impacto en HOME** | Escribe en configuración global | Cero mutación en `$HOME` |
| **Uso recomendado** | Tu máquina de desarrollo personal | Entornos compartidos, CI o configuraciones por equipo |

- **Instalación Global (en toda tu máquina):**
  ```bash
  gentle-ai install --agent antigravity-cli --scope global
  ```
  *(Opcional: Si querés que además configure la StatusLine y el título dinámico en `~/.gemini/antigravity-cli/settings.json`, sumá el flag `--include-theme` al comando global).*

- **Instalación en Workspace (aislada solo a este repositorio):**
  ```bash
  gentle-ai install --agent antigravity-cli --scope workspace
  ```
  *(Nota: El flag `--include-theme` no aplica en workspace ya que la configuración de terminal de `agy` es global).*

*(Nota sobre sync: El comando `gentle-ai sync` no hace falta al instalar; solo se utiliza después, en el día a día, si actualizás el binario y querés refrescar el plugin).*

> [!NOTE]
> **Compatibilidad con Antigravity IDE y Antigravity App (2.0):** Esta integración está centrada, optimizada y verificada en **Antigravity CLI (`agy`)**. Sin embargo, debido a que Antigravity IDE y Antigravity App (2.0) comparten gran parte de la infraestructura base de Antigravity, esta integración es **posiblemente compatible con Antigravity IDE y Antigravity App (2.0)**, aunque esto **no ha sido probado oficialmente aún**.

---

## 4. ¿Qué se instala en tu disco?

Gentle AI sigue un principio estricto de **transparencia total** e higiene del sistema.

### Mapa Completo de Superficies Desplegadas

```text
Entorno del Sistema y Repositorio
│
├── 1. Plugin Gentle AI
│   (Global: ~/.gemini/config/plugins/gentle-ai/  |  Workspace: <repo>/.agents/plugins/gentle-ai/)
│   ├── plugin.json                                      # Manifiesto del plugin para agy
│   ├── mcp_config.json                                  # Servidores MCP (CodeGraph, Context7, Engram)
│   ├── hooks.json                                       # Definición de hooks de ciclo de vida
│   ├── hooks/                                           # Ejecutables puente (hook.sh / hook.cmd)
│   ├── rules/                                           # Reglas operativas (AGENTS.md, orchestrator, review RDD, routing ODD, strict-tdd)
│   ├── agents/                                          # 10 Subagentes nativos Markdown
│   │   ├── gentle-ai-explore/agent.md
│   │   ├── gentle-ai-worker/agent.md
│   │   ├── gentle-ai-verify/agent.md
│   │   ├── jd-judge-a/agent.md
│   │   ├── jd-judge-b/agent.md
│   │   ├── jd-fix-agent/agent.md
│   │   ├── review-risk/agent.md
│   │   ├── review-readability/agent.md
│   │   ├── review-reliability/agent.md
│   │   └── review-resilience/agent.md
│   ├── skills/                                          # Catálogo completo de skills transpiladas
│   ├── chains/                                          # Flujos compuestos (4r-review.chain.md)
│   ├── review-models.json                               # Configuración de modelos para revisión RDD (opcional)
│   └── .native-rule-ownership.json                      # Registro de propiedad y digests de reglas
│
├── 2. Integración de StatusLine y Título (~/.gemini/antigravity-cli/settings.json)
│   ├── "statusLine": {"type": "command", "command": "gentle-ai statusline"}
│   └── "title": {"type": "command", "command": "gentle-ai statusline --title"}
│       (Solo con --include-theme en instalación global; gentle-ai preserva settings.json).
│
├── 3. Registro Global de Plugins (~/.gemini/config/import_manifest.json)
│   └── Registra el plugin de Gentle AI para su descubrimiento y carga inmediata en el arranque de agy.
│       (Solo en instalación global; en workspace se omite para aislamiento total).
│
├── 4. Aislamiento Limpio del Repositorio Local (<repo>/.git/info/exclude)
│   └── Excluye automáticamente artefactos internos (.atl, carpetas scratch) sin modificar el .gitignore del proyecto.
│
└── 5. Configuración y Estado en Tiempo de Ejecución (~/.gentle-ai/)
    ├── statusline.json                                  # Configuración de usuario para statusline (densidad y visibilidad)
    ├── statusline-phase/<conv_hash>.json                # Detección de fase de trabajo inferida vía PreToolUse
    ├── statusline-engram-state.json                     # Estado dinámico de observaciones Engram capturadas
    ├── statusline-rdd-cache.json                        # Caché del estado de revisión RDD
    ├── statusline-changes-cache.json                    # Caché de diferencias Git
    └── antigravity-mutations/v1/<conv_hash>.jsonl       # Ledger inmutable de mutaciones de archivos por sesión
```

### Modificaciones en `settings.json` (Solo con `--include-theme`)
Si especificás `--include-theme` en una instalación global, se inyectan las siguientes líneas en `~/.gemini/antigravity-cli/settings.json`:

```json
{
  "statusLine": { "type": "command", "command": "gentle-ai statusline" },
  "title": { "type": "command", "command": "gentle-ai statusline --title" }
}
```
Si no usás `--include-theme`, tu archivo `settings.json` **no es modificado** en absoluto.

### Protección de tu Repositorio Git
Gentle AI nunca ensucia tu árbol de trabajo:
- La sincronización utiliza `.git/info/exclude` en lugar de alterar tu `.gitignore`.
- Esto garantiza que tus commits y tu historial permanezcan limpios y sin archivos temporales de sincronización.

---

## 5. Verificación de la Instalación

Podés validar el estado de la integración mediante dos verificaciones complementarias:

1. **Validación del Plugin de Antigravity CLI (Mandatoria):**
   Comprobá que el plugin cumpla al 100% las especificaciones oficiales de `agy`:

   - **Si instalaste de forma global:**
     ```bash
     agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai
     ```

   - **Si instalaste en un workspace:**
     ```bash
     agy plugin validate --strict .agents/plugins/gentle-ai
     ```
   *(Verifica manifiestos, esquemas, los 10 subagentes, skills y hooks con 0 errores y 0 warnings).*

2. **Diagnóstico general de Gentle AI (Opcional):**
   ```bash
   gentle-ai doctor
   ```
   *(Comprueba herramientas del sistema, espacio en disco y conectividad con la memoria persistente Engram).*

---

## 6. Primer Uso y Verificación en Vivo

Una vez hecha la instalación, comproba que el entorno responda correctamente:

1. **Abrir Antigravity CLI:**
   ```bash
   agy
   ```

2. **Comandos de verificación dentro de la sesión de `agy`:**
   Dentro del prompt interactivo de `agy`, podés ejecutar los siguientes comandos slash para confirmar que todo esté operativo:
   - `/plugins`: Verificá que `gentle-ai` figure en la lista de plugins cargados y habilitados (con sus 10 subagentes, reglas y skills).
   - `/statusline`: Si instalaste con `--include-theme` y por la configuración de tu terminal no ves la barra inferior de inmediato, ejecutá `/statusline` para forzar su refresco.

3. **Comportamiento en tiempo real:**
   - Si activaste `--include-theme`, la **StatusLine nativa** se renderiza en el margen inferior mostrando el modelo activo, gauges de contexto, estado de memoria Engram y contador de cambios Git.
   - Al enviar tu primera instrucción, Gentle AI cargará de inmediato las reglas de orquestación ODD (`rules/AGENTS.md`) y tendrás disponibles los 10 subagentes nativos Markdown.

4. **Guías para profundizar y personalizar:**
   - [**README Principal**](README.md): Arquitectura completa, matriz de 10 subagentes y flujo de trabajo con hooks y ciclo RDD.
   - [**StatusLine y Experiencia de Terminal (statusline.md)**](statusline.md): Personalización de densidad (`comfortable`, `compact`), visibilidad de segmentos y atajos de teclado.
   - [**Seguridad, Jerarquías y Sandbox (security-and-sandbox.md)**](security-and-sandbox.md): Confinamiento del sandbox, prevención de lectura de secretos (`.env`) y filtros en memoria.
   - [**Resolución de Problemas (troubleshooting.md)**](troubleshooting.md): Diagnóstico rápido con `gentle-ai doctor` y respuestas a preguntas frecuentes.

---

## 7. Desinstalación Limpia

Si en algún momento deseás remover la integración de Antigravity CLI de tu sistema:

```bash
gentle-ai uninstall --agent antigravity-cli
```
Este comando:
- Remueve el árbol del plugin desplegado (tanto en instalación global `~/.gemini/config/plugins/gentle-ai/` como en el workspace activo `.agents/plugins/gentle-ai/`).
- Limpia de forma segura el registro en `~/.gemini/config/import_manifest.json`.
- Restaura `~/.gemini/antigravity-cli/settings.json` eliminando la configuración de `statusLine` y `title` si se había configurado con `--include-theme`.
- Preserva intactas las configuraciones de otros agentes que tengas instalados en tu máquina.
