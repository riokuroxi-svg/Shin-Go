// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// command.go — Registro de comandos. En el Shin-MD de JS cada comando es un
// archivo que se carga con import() dinámico; aquí se registra en un mapa al
// arrancar: nada de leer el disco en caliente, nada de rutas roscadas.
package bot

import (
	"context"
	"errors"
	"sort"
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/riokuroxi-svg/Shin-Go/internal/store"
	"github.com/riokuroxi-svg/Shin-Go/internal/ui"
	"github.com/riokuroxi-svg/Shin-Go/internal/wa"
	"github.com/riokuroxi-svg/Shin-Go/internal/xlog"
)

type Category string

const (
	CatMain       Category = "main"
	CatEconomy    Category = "economia"
	CatDownload   Category = "descargas"
	CatGroup      Category = "grupo"
	CatStickers   Category = "stickers"
	CatGacha      Category = "gacha"
	CatOwner      Category = "owner"
	CatUtils      Category = "utilidades"
	CatNSFW       Category = "nsfw"
	CatGames      Category = "juegos"
	CatPerfil     Category = "perfil"
	CatAnime      Category = "anime"
	CatMedia      Category = "media"
	CatSocket     Category = "socket"
	CatEventsCode Category = "eventos"
)

// Context es lo que recibe un handler: equivale al objeto ctx/m del JS
// (msg, args, sock, isGroup, isOwner, reply…) pero tipado.
type Context struct {
	Ctx      context.Context
	Bot      *wa.Client
	Log      *xlog.Logger
	Event    *events.Message
	DB       *store.Store
	Chat     types.JID
	Sender   types.JID
	IsGroup  bool
	IsOwner  bool
	PushName string

	Command string
	Args    []string
	Text    string // texto completo sin el prefijo
	Raw     string // texto tal cual llegó

	fromButton bool
}

// Reply contesta en el mismo chat (pasando por la cola).
func (c *Context) Reply(text string) error { return c.Bot.SendText(c.Ctx, c.Chat, text, false) }

// ReplyPriority contesta saltando el cupo de calentamiento (.menu, .ping…).
func (c *Context) ReplyPriority(text string) error {
	return c.Bot.SendText(c.Ctx, c.Chat, text, true)
}

// ReplyButtons contesta con tarjeta y botones nativos.
func (c *Context) ReplyButtons(title, body, footer string, buttons ...Button) error {
	return c.Bot.SendButtons(c.Ctx, c.Chat, title, body, footer, buttons...)
}

// Button es alias para no obligar a importar internal/ui en cada comando.
type Button = ui.Button

// Atajos de botones para los handlers (mismo nombre que en JS: quickReply…).
var (
	QuickReply   = ui.QuickReply
	CTAURL       = ui.CTAURL
	CTACopy      = ui.CTACopy
	CTACall      = ui.CTACall
	OpenWebview  = ui.OpenWebview
	SendLocation = ui.SendLocation
	SingleSelect = ui.SingleSelect
)

// FromButton indica si la invocación vino de tocar un botón.
func (c *Context) FromButton() bool { return c.fromButton }

// Command es la definición de un comando.
type Command struct {
	Name     string
	Aliases  []string
	Category Category
	Desc     string

	OwnerOnly bool
	GroupOnly bool
	AdminOnly bool
	Priority  bool // salta el cupo diario del calentamiento
	Cooldown  int  // segundos; 0 = usar el del router

	Run func(c *Context) error
}

var (
	registry   = map[string]*Command{}
	categories = map[Category][]*Command{}
)

// Register añade un comando al registro global. Se llama desde los init()
// de la carpeta cmds/ — el equivalente a "el archivo existe, luego carga".
func Register(cmd Command) {
	if cmd.Run == nil || cmd.Name == "" {
		panic("bot.Register: comando sin nombre o sin Run")
	}
	if cmd.Cooldown == 0 {
		cmd.Cooldown = 3
	}
	c := &cmd
	registry[strings.ToLower(cmd.Name)] = c
	for _, a := range cmd.Aliases {
		registry[strings.ToLower(a)] = c
	}
	categories[cmd.Category] = append(categories[cmd.Category], c)
}

// All devuelve los comandos únicos (sin alias) ordenados por categoría.
func All() []*Command {
	seen := map[*Command]bool{}
	var out []*Command
	for _, c := range registry {
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return out[i].Name < out[j].Name
		}
		return out[i].Category < out[j].Category
	})
	return out
}

// ByCategory agrupa para el .menu.
func ByCategory() map[Category][]*Command {
	out := map[Category][]*Command{}
	for cat, cmds := range categories {
		seen := map[*Command]bool{}
		for _, c := range cmds {
			if !seen[c] {
				seen[c] = true
				out[cat] = append(out[cat], c)
			}
		}
		sort.Slice(out[cat], func(i, j int) bool { return out[cat][i].Name < out[cat][j].Name })
	}
	return out
}

func Lookup(name string) (*Command, bool) {
	c, ok := registry[strings.ToLower(name)]
	return c, ok
}

func Count() (unique int, withAliases int) {
	unique = len(All())
	withAliases = len(registry)
	return
}

var ErrNotAllowed = errors.New("no permitido")
