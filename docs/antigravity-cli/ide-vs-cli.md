# El Ecosistema Antigravity: IDE, App (2.0) y CLI frente a la Integración Dedicada

El ecosistema de Google Antigravity abarca tres superficies de trabajo complementarias:

1. **Antigravity IDE**: La vista de editor (*Editor view*) de Google Antigravity. Ofrece autocompletado por tabulación (*tab autocompletion*), comandos de código en lenguaje natural (*natural language code commands*) y asistencia de edición con un agente configurable y consciente del contexto.
2. **Antigravity App (2.0)**: La plataforma dedicada (*dedicated platform*) de Google Antigravity para trabajar con agentes (anteriormente conocida como Antigravity 2.0). Permite orquestar múltiples agentes autónomos trabajando en paralelo a través de proyectos independientes.
3. **Antigravity CLI (`agy`)**: La superficie de terminal (*terminal-first surface*) de Google Antigravity para interactuar con agentes directamente en la consola. Te permite mantenerte en tu flujo de trabajo sin cambios de contexto (*context switching*).

Esta guía explica la diferencia entre el enfoque genérico heredado de upstream y nuestro adaptador dedicado, aclarando compatibilidades y por qué **no se deben utilizar ambos adaptadores a la vez**.

---

## 1. El Enfoque de Upstream frente a Nuestra Integración

### El Adaptador Genérico de Upstream (`--agent antigravity`)
En Gentle AI upstream se diseñó un único adaptador denominado `antigravity` que intentó dar soporte simultáneo a las tres superficies (IDE, App y CLI) mediante un enfoque plano:
- Concatena todas las directivas en un único archivo de texto plano (`.gemini/GEMINI.md`).
- Escribe skills y configuraciones dispersas en `~/.gemini/antigravity-cli/`.
- **Limitaciones graves en terminal**: Al tratar a la terminal como si fuera un editor tradicional, **no incluye subagentes**, no aprovecha el sistema modular de plugins de `agy`, no implementa hooks de seguridad ni ciclo de vida, carece de barra de estado nativa y **no soporta revisiones formales bajo RDD** (`transport: nil`).

### Nuestro Adaptador Dedicado (`--agent antigravity-cli`)
Diseñamos un adaptador desacoplado y especializado para `agy`:
- Despliega un **plugin nativo y completo** (`plugin.json`, `rules/`, `agents/`, `skills/`, `hooks.json`).
- Provee **10 subagentes nativos** en formato Markdown listos para delegación en terminal.
- Implementa **5 hooks de ciclo de vida** con protecciones en memoria contra lectura de secretos (`.env`) y advertencias ante comandos destructivos.
- Ofrece una **StatusLine en Go** de alto rendimiento con métricas de contexto, cuotas y memoria Engram.
- Habilita revisiones formales de código mediante **Receipt-Driven Development (RDD)** en subprocesos sandboxed.

> [!NOTE]
> **Compatibilidad con Antigravity IDE y Antigravity App (2.0):** Nuestra integración está diseñada, optimizada y probada a fondo para **Antigravity CLI (`agy`)**. Sin embargo, debido a que Antigravity IDE y Antigravity App (2.0) comparten gran parte de la infraestructura de configuración de Gemini/Antigravity, nuestra integración es **posiblemente compatible con Antigravity IDE y Antigravity App (2.0)**, aunque esto **no ha sido probado ni certificado oficialmente aún**.

---

## 2. Tabla Comparativa de Capacidades

| Característica | Upstream Genérico (`antigravity`) | Adaptador Dedicado (`antigravity-cli`) |
|---|---|---|
| **Público objetivo** | Intentó cubrir IDE, App (2.0) y CLI en un solo bloque | Especializado en CLI (`agy`) / Posiblemente compatible con IDE y App (2.0) |
| **Mecanismo de carga** | Concatena texto plano en `.gemini/GEMINI.md` | Plugin formal (`plugin.json`, `rules/`, `agents/`, `skills/`) |
| **Subagentes nativos** | ❌ Ninguno (`SubAgentsDir: ""` vacío) | ✅ 10 subagentes en Markdown listos para delegar |
| **Seguridad y Hooks** | ❌ Sin control en tiempo de ejecución | ✅ 5 hooks en Go (bloqueo `.env`, circuit breaker, `force_ask`) |
| **Revisiones RDD** | ❌ No soportado | ✅ Revisiones automáticas 4R en subprocesos `--sandbox` |
| **Barra de estado** | ❌ No disponible | ✅ StatusLine nativa en Go con cuotas y contexto |
| **Aislamiento en disco** | ⚠️ Mezcla rutas en `~/.gemini/antigravity-cli/` | ✅ Aislamiento estricto: `--scope global` vs `--scope workspace` |

---

## 3. Por qué NO se pueden usar ambas integraciones al mismo tiempo

Un error frecuente es suponer que se pueden instalar ambos adaptadores en la misma máquina. **Esto no es posible y generará conflictos**:

1. **Colisión en `agy`:** El adaptador de upstream escribe archivos en `~/.gemini/antigravity-cli/skills/` y `~/.gemini/antigravity-cli/settings.json`, mientras que nuestro adaptador despliega el plugin en `~/.gemini/config/plugins/gentle-ai/`.
2. **Duplicación de habilidades:** Al iniciar `agy`, la terminal descubriría skills duplicadas con el mismo nombre y directivas superpuestas.
3. **Incompatibilidad de configuraciones:** Ambas integraciones intentarían gobernar las opciones de terminal de Antigravity simultáneamente.

### Cómo elegir y migrar limpiamente
**Debés elegir uno u otro.** Si deseás contar con subagentes, revisiones RDD, hooks de seguridad y StatusLine en la terminal, debés desinstalar el adaptador genérico e instalar el dedicado:

```bash
# 1. Desinstalar el adaptador genérico de upstream:
gentle-ai uninstall --agent antigravity

# 2. Instalar el adaptador dedicado de agy (`antigravity-cli`):
gentle-ai install --agent antigravity-cli --scope global
```
