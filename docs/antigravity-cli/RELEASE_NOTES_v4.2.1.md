# Gentle AI v4.2.1 — Dedicated Google Antigravity CLI Edition

Esta release introduce soporte nativo completo para **Google Antigravity CLI (`agy`)**, incorporando las capacidades del núcleo **Gentle AI v4.2.0** y los prompts/skills de **Gentle Shell v4.3.0**, validados bajo **Antigravity CLI v1.4.4**.

---

## 🚀 Instrucción Obligatoria de Actualización

> [!IMPORTANT]
> **Después de instalar o actualizar el binario, ejecutá siempre:**
> ```bash
> gentle-ai sync
> ```
> *Los assets de revisión y ejecución están vinculados a la versión del binario. El ciclo de vida de revisión falla cerrado hasta que `sync` repara cualquier desajuste de proveniencia.*

---

## 🌟 Qué incluye esta versión

1. **Soporte Nativo de Google Antigravity CLI (`agy`)**:
   - Integración mediante el sistema oficial de plugins de Antigravity.
   - Compatibilidad total con validación estricta (`agy plugin validate --strict` pasa con 0 advertencias y 0 errores).
   - Doble alcance: Instalación global (`--scope global` en `~/.gemini/config/plugins/gentle-ai`) y por proyecto (`--scope workspace` en `.agents/plugins/gentle-ai`).

2. **Metodología ODD (Organic Driven Development)**:
   - Orquestador con control de contexto y presupuesto de tokens adaptado a Gemini.
   - Flujo de trabajo en 7 pasos estructurados (autorizar, explorar, resolver incertidumbre, clasificar, trackear, implementar y cerrar).

3. **Revisión Adversarial RDD (Receipt-Driven Development)**:
   - Transporte en subprocesos aislados y sandboxed para revisión ciega de código.
   - Lentes 4R especializadas:
     - `review-risk`: Seguridad, exposición de datos y límites de privilegios.
     - `review-readability`: Claridad, intención, convenciones y mantenibilidad.
     - `review-reliability`: Pruebas behavior-first, cobertura y determinismo.
     - `review-resilience`: Manejo de fallos, retries, degradación y carga.
   - Generación de recibos criptográficos e inmutables para autorizar entregas.

4. **10 Subagentes Nativos Especializados**:
   - Agentes de ciclo de vida ODD (`gentle-ai-explore`, `gentle-ai-worker`, `gentle-ai-verify`).
   - Agentes de revisión dual adversarial (`jd-judge-a`, `jd-judge-b`, `jd-fix-agent`).
   - Agentes de cuadrantes 4R (`review-risk`, `review-readability`, `review-reliability`, `review-resilience`).

5. **Hooks de Ciclo de Vida**:
   - `PreToolUse`: Evaluación dinámica in-process de rutas sensibles (.env*, .ssh, claves), circuit breaker contra loops y confirmación forzada (force_ask) para comandos destructivos.
   - `PreInvocation`: Preparación de entorno y gestión de exclusiones locales sin ensuciar el árbol Git.
   - `PostToolUse`: Ledger inmutable para rastreo de mutaciones de archivos.
   - `Stop`: Enforzamiento de revisión obligatoria para cambios de alto riesgo antes de permitir el cierre.

6. **Integración de Servidores MCP**:
   - Configuración lista para usar con `CodeGraph` (navegación y mapa de código), `Engram` (memoria persistente entre sesiones) y `Context7` (resolución de documentación de librerías).

---

## 📦 Instalación y Puesta en Marcha

### 1. Descarga del binario
Descargá el ejecutable correspondiente a tu arquitectura (`linux_amd64`, `linux_arm64`, `darwin_amd64`, `darwin_arm64`) desde los assets adjuntos y agregalo a tu `$PATH`.

### 2. Instalación del plugin
```bash
# Instalación global (recomendada):
gentle-ai install --agent antigravity-cli --scope global
gentle-ai sync --agents antigravity-cli --scope global

# O instalación aislada en tu repositorio:
gentle-ai install --agent antigravity-cli --scope workspace
gentle-ai sync --agents antigravity-cli --scope workspace
```

### 3. Verificación
```bash
gentle-ai doctor
```

---

## 🔐 Verificación de Integridad y Firma (Minisign)

Para verificar la autenticidad del release antes de ejecutar los binarios:

```bash
# 1. Descargar checksums.txt, checksums.txt.minisig y el archivo tar.gz de la release

# 2. Verificar la firma criptográfica con Minisign:
minisign -VQm checksums.txt \
  -x checksums.txt.minisig \
  -P "$GENTLE_AI_MINISIGN_PUBLIC_KEY"

# Debe imprimir exactamente:
# repo=Gentleman-Programming/gentle-ai;tag=v4.2.1

# 3. Validar los hashes SHA-256:
sha256sum --check --strict --ignore-missing checksums.txt
```

---

## 📚 Documentación Completa
- [Guía de Arquitectura de agy (Antigravity CLI)](https://github.com/ariasbruno/gentle-ai/blob/agy/ai-v4.2.0_shell-v4.3.0/docs/antigravity-cli/README.md)
- [Mapa Técnico y Mecánica Interna](https://github.com/ariasbruno/gentle-ai/blob/agy/ai-v4.2.0_shell-v4.3.0/docs/antigravity-cli/SUMMARY.md)
- [Registro de Decisiones Arquitectónicas](https://github.com/ariasbruno/gentle-ai/blob/agy/ai-v4.2.0_shell-v4.3.0/docs/antigravity-cli/08-decisions.md)
