<div align="center">

# 反魂 · Shin-MD-Go

**El núcleo de Shin-MD, migrado a Go sobre whatsmeow.**

[![Estado](https://img.shields.io/badge/estado-🚧_en_desarrollo-EE3D86?style=flat-square&labelColor=2F1A24)](https://github.com/riokuroxi-svg/Shin-MD-Go)
[![CI](https://img.shields.io/github/actions/workflow/status/riokuroxi-svg/Shin-MD-Go/test.yml?style=flat-square&label=CI&labelColor=2F1A24&color=EE3D86)](https://github.com/riokuroxi-svg/Shin-MD-Go/actions)
![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&labelColor=2F1A24)
![Tests](https://img.shields.io/badge/pruebas-verdes_✅-3FB950?style=flat-square&labelColor=2F1A24)
![Licencia](https://img.shields.io/badge/licencia-AGPL--3.0--only-EE3D86?style=flat-square&labelColor=2F1A24)
![Termux](https://img.shields.io/badge/Termux-arm64_estático-000000?style=flat-square&labelColor=2F1A24)

<img alt="Aviso: en desarrollo" src="https://img.shields.io/badge/⚠️_ESTO_NO_ES_EL_BOT_COMPLETO_AÚN-faltan_los_bloques_B1--B8-EE3D86?style=for-the-badge&labelColor=2F1A24">

</div>

---

> **🚧 Proyecto en desarrollo.**
> Este repositorio es la **migración del núcleo** de
> [Shin-MD](https://github.com/riokuroxi-svg/Shin-MD) (JavaScript/Baileys) a **Go**, sobre
> [whatsmeow](https://github.com/tulir/whatsmeow).
> Lo que está aquí **compila, pasa pruebas y arranca de verdad**, pero todavía no tiene
> los 211 comandos: el bot de JS sigue siendo el bot de producción hasta cerrar los
> bloques B1–B8. Las versiones se publican cuando el bloque está verificado, no antes.

## ¿Por qué una versión en Go?

No es un capricho de lenguaje: es quitar de en medio la parte que se rompe.

| En el bot de JS | En Shin-MD-Go |
|---|---|
| `baileys` (fork de terceros) + `patch-baileys-pairing.cjs` parcheando el `postinstall` | Biblioteca de protocolo declarada (whatsmeow). **No hay nada que parchear** |
| `node_modules` + `sharp` nativo + Node ≥ 22.5 | **1 binario** (17 MB estático para Termux) |
| 15 reintentos → `clearSession()` que borraba la sesión en un corte de red | `EnableAutoReconnect`: la sesión no se borra por red, nunca |
| Cola de envío con `async` frágil (se congelaba si un envío fallaba) | Goroutine con `context`, timeout y `recover()` por handler |
| Pruebas que necesitan medio bot levantado | `go test ./...` prueba botones, parseo y SQLite **sin abrir WhatsApp** |

El motor anti-ban está portado **1:1** (misma huella: σ 0.25 · base 1200 ms · +25 ms/car ·
warm-up 20→500 en 7 días), así que sigue siendo una obra derivada de Shin-MD según el
[NOTICE](NOTICE).

## Estado actual — bloque B0 ✅

| Bloque | Qué incluye | Estado |
|---|---|---|
| **B0 · Núcleo** | cliente, sesión SQLite, pairing QR/código, cola anti-ban, logger, botones, router, economía, pruebas, binario arm64 | ✅ **hecho** |
| B1 · Router real | hooks `before` (antilink, antistatus, afk), self-mode, admins por LID y por teléfono | 🚧 siguiente |
| B2 · Menús y meta | `.menu`, `.menucat`, `.fax`, `.info`, `.owner` con botones | ⬜ |
| B3 · Economía y perfil | daily, work, banco, ranking, inventario, gacha (con migración de datos) | ⬜ |
| B4 · Grupos y admin | antilink, warns, kick/promote, mute, horario | ⬜ |
| B5 · Stickers y media | sticker/toimg/tovid con `ffmpeg`, EXIF de sticker en Go | ⬜ |
| B6 · Descargas | play/yt/tiktok/ig/fb/x (decidir: Go puro o microservicio Node) | ⬜ |
| B7 · Panel web + operación | `/health`, rotación de logs, unidad `systemd` | ⬜ |
| B8 · Paridad de pruebas | portar las 161 pruebas del JS y sumar las nuevas | ⬜ |

Detalle completo del plan: **[docs/MIGRACION.md](docs/MIGRACION.md)**.

## Lo que ya funciona (probado, no prometido)

**Comandos cargados hoy: 12 únicos · 35 nombres.**

| Comando | Qué hace |
|---|---|
| `.menu` / `.menucat` | menú con botones nativos y categorías |
| `.ping` | latencia y estado de conexión (con botones) |
| `.daily` · `.perfil` | economía y perfil (SQLite, migraciones versionadas) |
| `.botones` | banco de pruebas: quick_reply, cta_url, cta_copy, cta_call |
| `.lista` | desplegable clásico (`ListMessage`) |
| `.carrusel` | tarjetas deslizables (`carouselMessage`) |
| `.encuesta ¿Café?; sí; no` | encuesta |
| `.evento` | evento de calendario con recordatorio |
| `.pin` | fija el mensaje al que respondes |
| `.rich` | respuesta enriquecida estilo IA (experimental, `RICH_AI=1`) |

### Botones: catálogo completo

`quick_reply` · `cta_url` · `cta_copy` · `cta_call` · `send_location` · `open_webview` ·
`mpm` · `flow` · `single_select` · tarjetas con **imagen de cabecera** · **carrusel** ·
lista clásica · **añadir opción a encuesta** · **pin** · **evento** · **rich AI**.
Todos con **fallback a texto plano** si WhatsApp no los pinta.

Catálogo, comparativa con Baileys y estado de verificación: **[docs/BOTONES.md](docs/BOTONES.md)**.

## Instalación

```bash
git clone https://github.com/riokuroxi-svg/Shin-MD-Go
cd Shin-MD-Go

make build        # dist/shin
make run          # arranque directo
make termux       # binario estático arm64 para Android/Termux
make test         # pruebas
```

Requisitos: **Go 1.26 o superior**. Nada más — ni Node, ni Python, ni compiladores.

### Arrancar

```bash
./dist/shin                 # muestra el QR en la terminal
./dist/shin --code 52155…   # vincular con código de 8 dígitos
```

### Variables de entorno

Mismos nombres que el bot de JS, para reutilizar tu `.env`:

```env
BOT_PREFIX=".,/,!,+"
OWNERS="5215512345678"
STATE_DIR=data
SHIN_DB_PATH=data/shin.db
SHIN_AUTH_DB=data/auth.db
LOG_LEVEL=info
ANTIBAN_BASE_MS=1200
ANTIBAN_SIGMA=0.25
ANTIBAN_PERCHAR_MS=25
WARMUP_START=20
WARMUP_END=500
RICH_AI=0        # 1 = respuestas enriquecidas estilo IA (experimental)
```

## Arquitectura

```
Shin-MD-Go/
├── cmd/shin/main.go          arranque (equivale a index.js + boot/index.js)
├── cmds/                     1 archivo .go = 1 comando (equivalen a cmds/**/*.js)
├── internal/
│   ├── config/               .env
│   ├── xlog/                 logger con los iconos ◐ ◑ ✓ ✕
│   ├── ui/                   botones, tarjetas, carrusel, rich, pin, evento
│   ├── wa/                   cliente, cola anti-ban, parseo, envíos
│   ├── bot/                  registro de comandos + router
│   └── store/                SQLite puro Go (economía/perfil/settings)
├── docs/                     MIGRACION.md (plan) · BOTONES.md (catálogo)
└── Makefile
```

Mapa de equivalencias archivo por archivo con el bot de JS: [docs/MIGRACION.md](docs/MIGRACION.md).

## Pruebas

```bash
go test ./...    # botones, carrusel, lista, pin, evento, encuesta, parseo, cola, SQLite
go vet ./...     # limpio
gofmt -l .       # limpio
```

CI en cada push: [.github/workflows/test.yml](.github/workflows/test.yml).

## Aviso

Usa **un número secundario**. Ningún bot es 100% inmune a un ban, ni este ni el de JS.

## Licencia

AGPL-3.0-only · ver [LICENSE](LICENSE) y [NOTICE](NOTICE).
Basado en [Shin-MD](https://github.com/riokuroxi-svg/Shin-MD) por riokuroxi-svg.
