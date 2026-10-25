# Guía de Instalación: Antigravity CLI (`agy`) con Gentle AI

Esta guía detalla el proceso completo de instalación, configuración y verificación de la integración de Antigravity CLI (`agy`) en Gentle AI.

---

## 1. Prerrequisitos

Antes de comenzar, verificá que tu entorno cumpla con los siguientes requisitos:

1. **Antigravity CLI (`agy`)**: Instalado y con sesión activa (`agy --version` >= 1.4.0).
2. **Go**: Versión 1.25 o superior instalada en tu sistema (`go version`).
3. **Git**: Instalado y disponible en el path del sistema.

---

## 2. Obtención e Instalación del Binario `gentle-ai` (Desde este Fork)

Para contar con el soporte nativo de **Antigravity CLI (`agy`)**, debés instalar el binario correspondiente a este fork (ya que upstream aún no incorpora este adaptador):

### Opción A: Compilación local desde el repositorio (Recomendado)
Es el método estándar y directo para instalar forks en el ecosistema Go. Clona la rama con la integración y compila el ejecutable en tu máquina:

```bash
# 1. Clonar el repositorio en la rama de integración de AGY:
git clone -b agy/ai-v4.2.0_shell-v4.3.0 https://github.com/ariasbruno/gentle-ai.git
cd gentle-ai

# 2. Compilar e instalar:
# Vía compilación directa (genera el binario local './gentle-ai'):
go build -o gentle-ai ./cmd/gentle-ai
./gentle-ai --version
mv gentle-ai ~/.local/bin/   # O sudo mv gentle-ai /usr/local/bin/

# O vía go install local (lo instala directo en tu $GOPATH/bin):
go install ./cmd/gentle-ai
```

### Opción B: Descarga de binarios precompilados
Si no disponés del compilador de Go en tu máquina, podés descargar el binario compilado para tu plataforma desde las [Releases del fork](https://github.com/ariasbruno/gentle-ai/releases):
```bash
chmod +x gentle-ai
mv gentle-ai ~/.local/bin/   # O sudo mv gentle-ai /usr/local/bin/
```

---

## 3. Alcance de la Instalación: Global vs Workspace

Gentle AI ofrece dos formas de instalación perfectamente aisladas:

| Dimensión | Alcance Global (`--scope global`) | Alcance Workspace (`--scope workspace`) |
|---|---|---|
| **Destino en disco** | `~/.gemini/config/plugins/gentle-ai/` | `<repo>/.agents/plugins/gentle-ai/` |
| **Afecta a** | Todos tus repositorios y proyectos | Exclusivamente al repositorio actual |
| **Descubrimiento** | Registrado en `~/.gemini/config/import_manifest.json` | Automático por `agy` al abrir la terminal en el proyecto |
| **Impacto en HOME** | Escribe en configuración global | Cero mutación en `$HOME` |
| **Uso recomendado** | Tu máquina de desarrollo personal | Entornos compartidos, CI o configuraciones por equipo |

### Comando de Instalación Global
Para instalar en toda tu máquina:

```bash
gentle-ai install --agent antigravity-cli --scope global
```

*(Opcional: Si querés que además configure la StatusLine y el título dinámico en `settings.json`, sumá la bandera opcional `--include-theme` al comando de instalación).*

*(Nota: El comando `gentle-ai sync` no hace falta al instalar; solo se utiliza después, en el día a día, si actualizás el repositorio o cambiás reglas y querés refrescar el plugin).*

### Comando de Instalación en Workspace (Aislado)
Para instalar únicamente dentro de este proyecto:

```bash
gentle-ai install --agent antigravity-cli --scope workspace
```

### ¿Venías usando la integración genérica de Antigravity? (Migración obligatoria)
En upstream existía un adaptador genérico llamado `antigravity` que intentaba configurar Antigravity escribiendo tanto en `.gemini/antigravity-cli/` como en `GEMINI.md`.

> [!WARNING]
> **No es posible usar ambas integraciones al mismo tiempo:** Ambos adaptadores configuran el entorno de Antigravity en tu máquina. Mantener los dos activos provocaría duplicación de skills, configuraciones redundantes y colisiones en la terminal. **Debés elegir uno u otro.**

- **Para migrar al adaptador de Antigravity CLI:**
  Desinstalá primero el adaptador genérico e instalá el adaptador dedicado:
  ```bash
  gentle-ai uninstall --agent antigravity
  gentle-ai install --agent antigravity-cli --scope global
  ```

> [!NOTE]
> **Compatibilidad con Antigravity IDE y Antigravity App (2.0):** Nuestra integración está centrada, optimizada y verificada en **Antigravity CLI (`agy`)**. Sin embargo, dado que Antigravity IDE y Antigravity App (2.0) comparten gran parte de la infraestructura de Gemini/Antigravity, nuestra integración es **posiblemente compatible con Antigravity IDE y Antigravity App (2.0)**, aunque esto **no ha sido probado ni certificado oficialmente aún**.

---

## 4. ¿Qué se instala en tu disco?

Gentle AI sigue un principio estricto de **transparencia total** e higiene del sistema.

### Estructura del Plugin Desplegado
Tanto en instalación global como en workspace, se despliega el siguiente árbol:

```text
gentle-ai/
├── plugin.json                  # Manifiesto del plugin para agy
├── mcp_config.json              # Configuración de servidores MCP (Engram, CodeGraph, Context7)
├── hooks.json                   # Definición de hooks de ciclo de vida
├── hooks/
│   ├── hook.sh                  # Script de puente para sistemas Unix / macOS
│   └── hook.cmd                 # Script de puente para Windows
├── rules/                       # Reglas operativas (AGENTS.md, orchestrator, ODD, strict-tdd)
├── agents/                      # 10 subagentes nativos Markdown
│   ├── gentle-ai-explore/agent.md
│   ├── gentle-ai-worker/agent.md
│   ├── gentle-ai-verify/agent.md
│   ├── jd-judge-a/agent.md
│   ├── jd-judge-b/agent.md
│   ├── jd-fix-agent/agent.md
│   ├── review-risk/agent.md
│   ├── review-readability/agent.md
│   ├── review-reliability/agent.md
│   └── review-resilience/agent.md
├── skills/                      # Catálogo completo de skills transpiladas
├── chains/                      # Flujos compuestos (4r-review.chain.md)
└── .native-rule-ownership.json  # Control de digests para evitar sobreescritura accidental
```

### Modificaciones en `settings.json` (Solo con `--include-theme`)
Si especificás `--include-theme`, se inyectan las siguientes líneas en `~/.gemini/antigravity-cli/settings.json`:

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
   Comprobá que el plugin cumpla al 100% con las especificaciones oficiales de `agy`:
   ```bash
   agy plugin validate --strict ~/.gemini/config/plugins/gentle-ai
   ```
   *(Verifica manifiestos, esquemas, los 10 subagentes, skills y hooks con 0 errores y 0 warnings).*

2. **Diagnóstico general de Gentle AI (Opcional):**
   ```bash
   gentle-ai doctor
   ```
   *(Comprueba herramientas del sistema en `$PATH`, espacio en disco y conectividad con la memoria persistente Engram).*

---

## 6. Desinstalación Limpia

Si deseás remover la integración:

```bash
# Desinstalación global:
gentle-ai uninstall --agent antigravity-cli --scope global

# Desinstalación en workspace:
gentle-ai uninstall --agent antigravity-cli --scope workspace
```
Esto removerá el árbol del plugin y limpiará los registros administrados en `import_manifest.json` y `settings.json`.
