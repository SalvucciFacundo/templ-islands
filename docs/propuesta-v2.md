# Nivel 3: Islands híbridas — render server-side O client-side bajo Go (v2)

> **Documento de especulación técnica / propuesta.** Estado: idea refinada (2026-08-08).
> Competencia verificada con búsqueda en GitHub + awesome-templ + issues de a-h/templ el 2026-08-08.

## La visión en una frase (v2)

**El mismo componente templ marca el contrato (`@island`); la página se renderiza server-side; las islas calientes se hidratan en el cliente con un renderer JS liviano + optimistic UI; el servidor responde datos, no HTML.**

## El problema real (reframe)

El objetivo: templ + htmx/Datastar para proyectos de alto flujo (ej: red social). El modelo server-driven paga un costo por CADA interacción:

- round-trip al servidor
- re-render del fragmento en el server (CPU)
- latencia de red para el usuario

**Lo que NO se ahorra moviendo el render al cliente** (para no perseguir la solución equivocada):

| Costo | ¿Se resuelve con render client? |
|---|---|
| Escrituras (like/comment → DB) | NO — van al server sí o sí |
| Fan-out (el like de Juan en feeds de 500 seguidores) | NO — es propagación de datos |
| Tiempo real de OTROS usuarios | NO — es push (SSE/WS), no render |
| Reads redundantes + re-render de fragmentos | **SÍ** — esto es el hueco |

El techo honesto de la idea: **eliminar los round-trips + re-render de las interacciones.** Valioso, pero es una parte — no toda — de la pelea de una red social.

## Investigación de competencia (verificada 2026-08-08)

El claim del doc v1 ("no existe una solución madura así en Go") es **VERDADERO con matices**:

| Proyecto | Qué es | Estado | ¿Compite? |
|---|---|---|---|
| [romshark/demo-islands](https://github.com/romshark/demo-islands) | Demo de arquitectura islands con templ + HTMX + Shoelace + Lit + Alpine.js | **ARCHIVADA** por su autor (se pasó a Datastar) | No como librería. Las islas renderizan server-side con templ; la interactividad client la manejan Alpine/Lit — **el render client NO es templ** |
| [aforamitdev/go-templ-react](https://github.com/aforamitdev/go-templ-react) | "templ island architecture react" | Abandonada (2024, 1 commit) | No |
| [fjakobs/go-template-wasm](https://github.com/fjakobs/go-template-wasm) | POC: render de templates Go estándar en browser con WASM | Experimento (2025, 4 días) | No como producto. **Sí valida que render Go→WASM es viable** |
| Issues en `a-h/templ` (islands / client-side / wasm) | Discusión en el repo oficial | **Cero resultados** | Nadie lo pide en el ecosistema |
| [awesome-templ](https://github.com/templ-go/awesome-templ) (lista oficial) | Librerías del ecosistema | UI kits (templui), toasts (goaster), i18n, SEO... | **Ninguna librería de islands ni render client de templ** |
| go-app / vugu / gio (Go→browser) | Frameworks SPA Go/WASM | Maduros pero de nicho | No integran templ, no son islands |

**Conclusión:** el hueco literal ("mismo componente templ render server O client, mismo lenguaje en ambos lados") **no está ocupado**. Pero el dato más importante es el POR QUÉ: el único islands demo de templ (romshark) fue **archivada por su propio autor, que se fue al modelo server-driven (Datastar)** — señal de que islands client-side no encontró suficiente tracción en el ecosistema templ. El hueco existe; que valga la pena llenarlo es otra cosa (ver Plan de validación).

## La propuesta v2: `templ-islands`

**Nombre técnico (repo):** `templ-islands`
**Package Go:** `github.com/SalvucciFacundo/templ-islands/islands`

Proyecto de dos piezas como templ:
- 📚 **Librería Go**: `islands.Register(...)`, middleware de modo, endpoints de datos
- 🛠️ **CLI**: detecta `@island` en los `.templ`, genera manifiesto + registro del web component
- ⚙️ **Base**: Datastar/htmx se mantiene para el modo server y las islas no calientes (aditivo sobre Nivel 2)

### Qué cambió vs v1

| v1 | v2 |
|---|---|
| Decisor automático por tráfico en runtime (Prometheus, p95) | **Fuera.** El modo se decide en el código, por componente. Único "auto" = por capacidad del cliente (sin JS → server) |
| WASM Go para render client (2-10MB por componente) | **Fuera del MVP.** Renderer client = JS liviano contra el contrato `@island`, con test de paridad. WASM = evolución futura para lógica compleja |
| Hidratación sobre HTML inicial + shadow DOM | Light DOM: la isla hidrata sobre el HTML server (SEO ok) |
| Server devuelve HTML renderizado por interacción | Server devuelve **JSON** (`{likes: 43}`), no fragmentos |
| Estado no tratado | Estado en el browser en modo client: optimistic UI + sync en background; push separado (SSE/WS) |

### Agnóstico al hipermedia: core + adapters

La librería **no empaqueta ni htmx ni Datastar**: el núcleo es agnóstico y el usuario elige el adapter del modo server. El modo client (la parte interesante: optimistic UI + renderer JS) es **idéntico** sin importar el hipermedia.

```
templ-islands/
├── islands/                  # CORE — agnóstico
│   ├── register.go           # contrato @island (props, eventos)
│   ├── manifest.go           # islands.json
│   ├── middleware.go         # decisión de modo (client vs server)
│   └── render/client.go      # web component + renderer JS  ← no depende de nada
└── islands/adapters/
    ├── htmx/                 # render server con atributos hx-*
    └── datastar/             # render server con atributos data-* + SSE
```

El modo server es un **contrato**: "la isla en modo server emite estos atributos de interacción". Cada adapter implementa ese contrato:

```go
// El usuario elige el adapter al registrar:
islands.Register(LikeButton, "like-button",
    islands.WithServerAdapter(datastar.Adapter{}),  // o htmx.Adapter{}
    islands.WithMode("client"),
)
```

Ventaja: el usuario no abandona su hipermedia — la librería se adapta a lo que ya usa. Datastar (demo, nativo del ecosistema templ) y htmx (producción, madurez) quedan como opciones intercambiables.

### Cómo funciona el flujo

```
GET /feed
  └─ render server de la página (templ normal, cache/CDN)
       └─ isla <like-button post="42">
             ├─ cliente con JS  → se hidrata: runtime client + optimistic UI
             │                    POST /api/like → {likes: 43} (JSON)
             │                    el server NO re-renderiza nada
             └─ sin JS          → modo server: adapter (Datastar o htmx)
```

### Componentes de la solución

| Pieza | Descripción | Esfuerzo |
|---|---|---|
| **Contrato `@island`** | Directiva en `.templ` que marca el componente + define props, eventos y tipo de isla (mutación / re-render) | Bajo |
| **CLI `templ-islands`** | Detecta islas, genera `islands.json` + registro del web component + golden tests de paridad + **valida los selectores de `mutates` contra el HTML renderizado** (`go install ...@latest`) | Medio |
| **Runtime Go (core)** | `Register()`, middleware de modo, endpoints de datos — agnóstico al hipermedia | Medio |
| **Adapters** | `htmx` / `datastar`: render server del modo server según el hipermedia elegido | Bajo (cada uno) |
| **Runtime client JS** | Genérico (estimación honesta ~4-8KB minificado): store compartido por clave de dominio + mutaciones atómicas + optimistic UI con ciclo de vida. Cero renderer por componente | Medio |
| **Renderer client (solo islas de re-render)** | Reimplementación liviana del render + golden tests generados por el CLI | **Alto** (acotado al 10%) |
| **Protocolo de datos** | Endpoints JSON para las acciones de las islas | Bajo |

## Resolver el renderer en sync (la decisión de diseño clave)

El doble renderer en sync (render JS == render templ) era el costo más alto. Se resuelve **eliminando el renderer client donde no hace falta**, no haciéndolo más robusto:

**1. Dos tipos de islas (el 90% / el 10%).**

- **Isla de mutación (90% de una red social)**: like, follow, contador, toggle. NO re-renderiza nada — muta nodos específicos del HTML que ya mandó el server.
  ```go
  // @island(mutates: "#like-count", "#like-btn")
  templ LikeButton(postID int, liked bool, likes int) { ... }
  ```
  El CLI genera un **runtime client genérico** (igual para todas las islas): escucha el evento, muta los targets localmente (optimistic), llama al endpoint JSON, aplica la respuesta y revierte si falla. **No hay renderer client por componente → NO hay sync que mantener.** El HTML viene 100% de templ, siempre.

  Detalles que el runtime debe incluir desde el día 1:
  - **Validación en build**: el CLI verifica que cada selector de `mutates` exista en el HTML renderizado por templ. Selector que no exista = error de compilación, no bug en producción.
  - **Store compartido por clave de dominio** (`postID`), no por instancia DOM: si el mismo like aparece en el feed Y en el modal, un like actualiza ambas instancias.
  - **Ciclo de vida explícito de la mutación**: `pending → success` (aplica la respuesta del server, que es la **fuente de verdad** final — puede rechazar: ya estaba dado, rate limit, sesión expirada) o `pending → error` (rollback + mensaje). Emite eventos DOM (`like:start`, `like:success`, `like:error`) para que islas complejas puedan reaccionar.

- **Isla de re-render (10%)**: listas dinámicas, filtros, búsquedas. Estas SÍ necesitan render en el cliente. Para estas, y solo estas:
  - El CLI genera **golden tests de paridad automáticamente**: renderiza el `.templ` con fixtures de ejemplo y genera el HTML esperado para el caso JS. Si divergen, falla el CI.
  - El manifiesto (`islands.json`) tipa props y eventos (schema) — el esqueleto JS se **genera del manifiesto**, no a mano. La lógica manual queda acotada a lo no derivable.

**2. Contrato de props tipado en el manifiesto.** props y eventos se declaran una vez con tipos; los esqueletos (server y client) se generan de ahí. La divergencia estructural se vuelve imposible; la divergencia de lógica queda en los golden tests.

**3. Parity check en dev.** En modo desarrollo, el runtime client puede re-renderizar desde el endpoint de datos y comparar contra el HTML server, loggeando cualquier divergencia que escape a los tests.

**4. Evolución (fuera del MVP):** transpilador `.templ` → JS para el subset de templ que usan las islas de re-render — elimina el mantenimiento del 10% restante. Es un proyecto en sí; se justifica solo si la adopción lo pide.

## Los tres problemas distintos (no confundir)

En una app de alto flujo hay TRES problemas que suenan parecidos pero NO son lo mismo. Confundirlos lleva a diseñar una cosa que intenta resolver todo y no resuelve nada bien:

| Problema | Qué es | Quién lo resuelve |
|---|---|---|
| **1. Costo de render en el servidor** | El server re-renderiza fragmentos por cada interacción → CPU + round-trips | **Esta librería** (modo client + endpoints JSON) |
| **2. Latencia percibida** | El usuario espera a que el server responda para ver el cambio | **Optimistic UI** (se ve instantáneo, sync en background) |
| **3. Consistencia en tiempo real** | El like de Juan debe aparecer en la pantalla de María sin recargar | **Push separado** (SSE/WS) — componente aparte, NO la librería |

La librería resuelve 1 y 2. El 3 es otro componente (push), y el 95% del SSR estático ni siquiera entra en estos — va con cache/CDN.

## Riesgos y honestidad (v2)

| Riesgo | Severidad | Mitigación |
|---|---|---|
| Doble renderer en sincronía (JS vs templ) | Alta → **Mitigada por diseño** | Ver "Resolver el renderer en sync": el 90% de las islas NO tienen renderer client (mutaciones atómicas); el 10% usa golden tests generados + contrato tipado; el transpilador .templ→JS queda como evolución |
| El ecosistema no pide esto (demo-islands archivada) | Media | Es el riesgo de "impulsar un stack" — aceptado, PERO ver Plan de validación |
| Optimistic UI + reconciliación (like falla → revertir) | Media | Mitigado por diseño: ciclo de vida explícito `pending → success/error` en el runtime, rollback + eventos DOM; el server es la fuente de verdad final (puede rechazar) |
| SEO | Baja | Light DOM, nada de shadow DOM para contenido indexable |
| Datastar/htmx podrían absorber la parte server | Baja | Son base del proyecto, no competencia |

## Plan de validación (ANTES de escribir la librería)

El hueco es real, pero "es gasto de tiempo" depende de esto:

1. **Construir el problema con lo que YA existe**: la red social / ruta de MisCanarios con templ + Datastar/htmx + optimistic UI con JS vanilla (Alpine/Lit) + endpoints JSON para los componentes calientes. Sin inventar nada.
2. **Medir**: con tráfico simulado, cuánta CPU de render y round-trips se ahorran vs el modelo server-driven puro.
3. **Extraer el patrón** (no la librería): cómo marcar componentes calientes, cómo hacer optimistic UI con templ, cómo medir el ahorro.
4. **Recién entonces decidir**: si el ahorro es real y el patrón se repite → ahí la librería `templ-islands` tiene justificación.

La librería como producto de arranque = costo alto (doble renderer), adopción probablemente baja. La validación con stack existente = semanas, no meses, y resuelve el problema real igual.

## Resultado del demo de validación (2026-08-08) ✅

El paso 1 y 2 del plan se ejecutaron: `demo-social/` — feed SSR con templ + htmx + optimistic UI manual + endpoints JSON, SIN librería.

| Paso | Qué se construyó | Resultado |
|---|---|---|
| Baseline server-driven | `POST /like/{id}` re-renderiza el botón con templ | El problema confirmado: cada like = round-trip + render |
| Optimistic UI + JSON | `app.js` intercepta el click (capture phase), muta local, `POST /api/like/{id}` devuelve `{likes, liked}` | El server pasa de devolver HTML a devolver 4 bytes |
| Medición | Contadores atómicos en el server + `bench.sh` (300 likes por modo) | **Modo A: 300 renders (0.061 ms c/u). Modo B: 0 renders (0.010 ms todo).** Render del botón ~6x más caro que TODA la operación B — y el botón es el fragmento más chico posible |
| Fallback | `hx-post` se conserva en el botón | Nivel 1 verificado: sin `app.js` el modo server-driven sigue funcionando (htmx). Nivel 2 (cero JS) no se implementó: nicho (<1% navegadores), se agrega con un `<form>` si algún día hace falta |

**Conclusiones del demo:**
- El modo client elimina el **100% del render** por interacción; la proporción empeora con fragmentos reales (post completo > botón).
- El progressive enhancement por capas funciona: `hx-post` (fallback) + `app.js` (optimistic) conviven sin romperse.
- El runtime client genérico (~2.6 KB) con ciclo de vida `pending → success/error` es viable como pieza de la futura librería.
- Stack usado: Go 1.26.5, templ v0.3.1020, htmx 2.0.4. Lecciones de templ guardadas en memoria (imports automáticos, `templ.KV`, `if` directo, texto plano).

## MVP de la librería (solo si la validación confirma)

1. `@island` + CLI que detecta y genera `islands.json`.
2. Una isla de **mutación**: botón de like con optimistic UI (runtime client genérico), server responde JSON.
3. Fallback: sin JS → el mismo like anda con el adapter (Datastar o htmx).
4. Una isla de **re-render** con golden test de paridad generado por el CLI.
5. Demo con tráfico simulado: medir CPU de render ahorrada.

**NO en el MVP:** WASM, transpilador .templ→JS, decisor por tráfico.

## Cómo encaja con go-arch-cli

```
go-arch new --frontend templ-datastar   ← Nivel 2 (ya listo para hacer)
go-arch new --frontend templ-islands    ← Nivel 3 (después de validar)
```

Ambos comparten: `views/`, `static/`, handlers, main.go. El Nivel 3 solo agrega el CLI de islas + runtime + renderer JS.

## Decisión pendiente

**La validación (pasos 1-2 del plan) ya se ejecutó y confirmó el ahorro** (ver "Resultado del demo de validación"). Queda UNA decisión:

1. **¿Construir la librería `templ-islands` ahora?** (el hueco existe + la validación dio resultados — pero es apostar por adopción incierta; el costo real es el doble renderer acotado al 10% y el CLI)
2. **¿Llevar el patrón al proyecto real (MisCanarios) y extraer la librería si se repite?** (menos riesgo, más lento para "producto opensource")

Mi recomendación: **1, en versión acotada** — extraer del demo el runtime client genérico + el contrato `@island` como librería mínima, y validarlo en una ruta real antes de invertir en el CLI completo.

---

*Documento de propuesta v2. Fecha: 2026-08-08. Investigación verificada hoy: demo-islands archivada por su autor, go-template-wasm es POC, cero issues de islands en a-h/templ, awesome-templ sin librerías islands, Datastar v1.0.2 con SDK Go activo. Demo de validación ejecutado el 2026-08-08: `demo-social/` — resultados en la sección correspondiente.*
