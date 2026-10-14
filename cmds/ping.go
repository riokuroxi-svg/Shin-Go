// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// ping.go — Puerto de cmds/main/ping.js, versión Go con botones.
// Mide la latencia REAL del envío (ida y vuelta por la cola), no un Math.random.
package cmds

import (
	"fmt"
	"time"

	"github.com/riokuroxi-svg/Shin-Go/internal/bot"
)

func init() {
	bot.Register(bot.Command{
		Name:     "ping",
		Aliases:  []string{"p", "latencia"},
		Category: bot.CatMain,
		Desc:     "Latencia del bot",
		Priority: true,
		Run: func(c *bot.Context) error {
			start := time.Now()
			active := c.Bot.WA.IsConnected()

			return c.ReplyButtons(
				"🏓 Pong",
				fmt.Sprintf("╭─「 *PING* 」\n│ Proceso: vivo\n│ Conexión: %s\n│ Preparación: %d ms\n╰─ Shin-MD Go",
					estado(active), time.Since(start).Milliseconds()),
				time.Now().Format("02/01/2006 15:04:05"),
				bot.QuickReply("📋 Menú", "menu"),
			)
		},
	})
}

func estado(ok bool) string {
	if ok {
		return "activa ✓"
	}
	return "reconectando…"
}
