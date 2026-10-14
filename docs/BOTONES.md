# Botones: "iguales y mejores". Tenías razón — aquí está el catálogo completo

_Verificado el 13-oct-2026 contra el código fuente real, no de memoria._

## Lo primero: quién tiene qué (con evidencia)

Ninguna librería "trae botones": las dos mandan **protobuf**. Lo que cambia es
**(a)** qué campos del proto existen, **(b)** si hay helper o lo armas a mano y
**(c)** si el cliente de WhatsApp lo pinta en una cuenta normal.

| | Shin-MD hoy<br>`baileys@6.7.24` (tag **legacy**) | Baileys más nuevo<br>`7.0.0-rc14` (**RC**, no estable) | Shin-Go<br>whatsmeow `20261013` |
|---|---|---|---|
| `interactiveMessage` + `nativeFlowMessage` | sí (proto) | sí (proto) | sí (proto **+ builders tipados**) |
| `quick_reply` / `cta_url` / `cta_copy` | sí | sí | sí + **tests** |
| `cta_call`, `send_location`, `open_webview` | sí (string, a mano) | sí | sí + helpers `CTACall()` etc. |
| `single_select` (desplegable nativo) | a mano | a mano | helper `SingleSelect()` |
| `flow` (WhatsApp Flows) | a mano | a mano | helper `FlowButton()` |
| `ListMessage` (desplegable clásico) | proto | proto | helper `BuildList()` |
| **Carrusel** (`carouselMessage`) | proto | proto | helper `BuildCarousel()` |
| Encuesta | a mano (`PollCreationMessage`) | a mano | **helper oficial** `BuildPollCreation` |
| **Añadir opción a encuesta** (`PollAddOptionMessage`) | **no está el proto** | **no está** | proto presente + `BuildPollAddOption()` |
| Fijar mensaje (`PinInChatMessage`) | a mano | a mano | helper `BuildPin()` |
| Evento de calendario (`EventMessage`) | proto | proto | helper `BuildEvent()` |
| Respuesta enriquecida IA (`richResponseMessage`) | proto | proto | **lectura tipada** (`types/richresponse`, 2026) + builder |
| Comentarios (canales) | proto | proto | helpers `DecryptComment`/`EncryptComment` |
| Canales/newsletters | `newsletter.ts` | `newsletter.ts` | API completa (crear, seguir, mandar, reaccionar) |
| Bot list / perfiles de bot | no | no | `GetBotListV2`, `GetBotProfiles` |
| Passkeys, FB Messenger | no | no | `SendPasskeyResponse`, `SendFBMessage` |

Evidencia (comandos exactos, por si quieres repetirlo):

```bash
curl -s https://registry.npmjs.org/baileys | python3 -c "import json,sys;print(json.load(sys.stdin)['dist-tags'])"
# {'latest': '7.0.0-rc14', 'legacy': '6.7.24'}   ← tu bot usa la "legacy"

# proto de Baileys 6.7.24 vs proto de whatsmeow de hoy
grep -c pollAddOptionMessage baileys-6.7.24/WAProto/WAProto.proto   # 0
grep -c pollAddOptionMessage whatsmeow/proto/waE2E/*.proto          # 2
```

**Conclusión honesta:** sí, hay botones "más y mejores" — pero no porque una
librería los traiga de fábrica, sino porque **el proto de WhatsApp creció**. La
diferencia real entre tu JS y Go es que aquí eso queda **tipado, con helper y
con prueba**, y en JS toca copiar el JSON a mano desde un foro (y rezar porque
el fork de turno lo haya traído).

---

## El catálogo completo, ya en el código

### Botones nativos (`internal/ui/buttons.go`)

| Helper | name | Para qué |
|---|---|---|
| `QuickReply(texto, id)` | `quick_reply` | devuelve un id → el router lo convierte en comando |
| `CTAURL(texto, url)` | `cta_url` | abre un enlace |
| `CTACopy(texto, código)` | `cta_copy` | copia al portapapeles |
| `CTACall(texto, teléfono)` | `cta_call` | abrir llamada |
| `SendLocation(texto)` | `send_location` | compartir ubicación |
| `OpenWebview(título, url)` | `open_webview` | web embebida |
| `MPM(jidNegocio)` | `mpm` | catálogo multi‑producto |
| `FlowButton(flowID, token, cta)` | `flow` | WhatsApp Flow publicado |
| `SingleSelect(título, secciones…)` | `single_select` | **la lista desplegable moderna** |

### Tarjetas
- `BuildButtons(título, cuerpo, pie, botones…)` — clásica.
- **`BuildButtonsImage(imagen, …)`** — con **foto de cabecera** (se sube con `cli.Upload`).
- **`BuildCarousel(cartas…)`** — carrusel `HSCROLL_CARDS`, cada carta con su foto y sus botones.
- **`BuildList(…)`** — desplegable clásico (`ListMessage`), sin nodos.

### Extras que el JS no tiene en el proto
- `BuildPollAddOption(claveEncuesta, "opción")` — añadir opción a una encuesta viva.
- `BuildPin(clave, true|false)` — fijar/desfijar mensaje.
- `BuildEvent(nombre, desc, inicio, fin, invitados, recordatorio)` — evento de calendario.
- `RichText` / `RichCode` / `RichTable` / `RichTextCard` — respuesta enriquecida estilo IA.

### Envío (`internal/wa/send.go`)
Todos pasan por la cola anti-ban y **todos caen a texto plano** si fallan:
`SendButtons` · `SendButtonsImage` · `SendCarousel` · `SendList` · `SendPoll` ·
`SendPollAddOption` · `PinMessage` · `SendEvent` · `SendRich`.

---

## Cómo probarlo en tu teléfono (comandos nuevos)

| Comando | Manda |
|---|---|
| `.botones` | tarjeta con 4 botones (quick_reply, url, copy, call) |
| `.lista` | desplegable clásico con 2 secciones |
| `.carrusel` | 3 cartas deslizables |
| `.encuesta ¿Café?; sí; no` | encuesta |
| `.evento` | evento de calendario con recordatorio |
| `.pin` (respondiendo a un mensaje) | fija ese mensaje |
| `.rich` | respuesta enriquecida (experimental) |

## Estado de verificación (sin vender humo)

| Cosa | Estado |
|---|---|
| Estructura del proto (v9, carrusel, lista, pin, evento) | ✅ **verificado aquí**, con pruebas (`go test ./internal/ui/`) |
| Nodos `<biz>` (+`bot` solo en privados, `name=carousel` en carrusel) | ✅ verificado en código |
| Que WhatsApp **pinte** botones/tarjetas | ya lo sabes por tu bot de JS: si van los nodos, se pintan |
| **Rich AI** (markdown/tabla/código) desde cuenta normal | ⚠️ **pendiente en teléfono**; va detrás de `RICH_AI=1` y con fallback |
| `flow`, `mpm`, `cta_call` | proto ✅ · render depende de la cuenta (business para flow/mpm) |
| `PollAddOption`, `Pin` | proto ✅ · botón/acción del cliente: verificar en tu teléfono |

> El plan sensato: mandas `.botones`, `.lista`, `.carrusel` desde un número
> secundario, ves qué pinta tu teléfono y me dices; de ahí sale la tabla real
> por dispositivo (Android/iOS/Web), que es lo único que manda.
