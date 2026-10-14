// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// menu.go — Puerto de cmds/main/menu.js + el .menucat con botones.
// Es el comando que mejor demuestra las dos cosas que te importan:
// botones nativos (con nodos <biz>) y arranque rápido (.menu con prioridad).
package cmds

import (
	"fmt"
	"sort"
	"strings"

	"github.com/riokuroxi-svg/Shin-Go/internal/bot"
	"github.com/riokuroxi-svg/Shin-Go/internal/ui"
)

func init() {
	bot.Register(bot.Command{
		Name:     "menu",
		Aliases:  []string{"ayuda", "help", "h"},
		Category: bot.CatMain,
		Desc:     "Muestra el menú con botones",
		Priority: true, // no espera el calentamiento: es el comando que más se usa
		Run: func(c *bot.Context) error {
			unique, aliases := bot.Count()

			if len(c.Args) > 0 {
				return menuCategory(c, strings.ToLower(c.Args[0]))
			}

			body := fmt.Sprintf(
				"╭─「 *SHIN-MD* 」\n"+
					"│ 反魂 · núcleo en Go\n"+
					"│ %d comandos · %d nombres\n"+
					"╰─ prefijos: %s",
				unique, aliases, strings.Join(c.Bot.Prefixes(), " "))

			return c.ReplyButtons(
				"SHIN-MD",
				body,
				"Toca un botón o escribe .menu <categoría>",
				bot.QuickReply("📋 Categorías", "menucat"),
				bot.QuickReply("🏓 Ping", "ping"),
				bot.CTACopy("📮 Copiar prefijo", c.Bot.FirstPrefix()),
				bot.CTAURL("🌐 Repositorio", "https://github.com/riokuroxi-svg/Shin-MD"),
			)
		},
	})

	bot.Register(bot.Command{
		Name:     "menucat",
		Aliases:  []string{"categorias", "menucat"},
		Category: bot.CatMain,
		Desc:     "Lista de categorías con sus comandos",
		Priority: true,
		Run: func(c *bot.Context) error {
			if len(c.Args) > 0 {
				return menuCategory(c, strings.ToLower(c.Args[0]))
			}

			cats := bot.ByCategory()
			names := make([]string, 0, len(cats))
			for cat := range cats {
				names = append(names, string(cat))
			}
			sort.Strings(names)

			var sb strings.Builder
			sb.WriteString("╭─「 *CATEGORÍAS* 」\n")
			for _, name := range names {
				fmt.Fprintf(&sb, "│ ▸ *%s* (%d)\n", name, len(cats[bot.Category(name)]))
			}
			sb.WriteString("╰─ .menu <categoría> para ver los comandos")

			// Un botón por categoría: WhatsApp acepta hasta 10 en native_flow.
			var btns []ui.Button
			for i, name := range names {
				if i >= 8 {
					break
				}
				btns = append(btns, bot.QuickReply("📁 "+name, "menu:"+name))
			}
			return c.ReplyButtons("SHIN-MD", sb.String(), "Elige una categoría", btns...)
		},
	})
}

func menuCategory(c *bot.Context, cat string) error {
	cats := bot.ByCategory()
	cmds, ok := cats[bot.Category(cat)]
	if !ok {
		return c.Reply("✕ No existe la categoría *" + cat + "*.\nUsa .menucat")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "╭─「 *%s* 」\n", strings.ToUpper(cat))
	for _, cmd := range cmds {
		fmt.Fprintf(&sb, "│ %s%s — %s\n",
			c.Bot.FirstPrefix(), cmd.Name, cmd.Desc)
	}
	sb.WriteString("╰─")

	return c.ReplyPriority(sb.String())
}
