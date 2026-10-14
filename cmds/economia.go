// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// economia.go — Puerto de cmds/economy/daily.js y perfil.js.
// Muestra cómo se usa la capa de datos (store) desde un comando.
package cmds

import (
	"fmt"
	"math/rand"
	"time"

	"shin-go/internal/bot"
)

func init() {
	bot.Register(bot.Command{
		Name:     "daily",
		Aliases:  []string{"diario", "recompensa"},
		Category: bot.CatEconomy,
		Desc:     "Recompensa diaria",
		Cooldown: 5,
		Run: func(c *bot.Context) error {
			ctx := c.Ctx
			jid := c.Sender.String()

			// La lógica de "una vez al día" se guarda en settings (mismo patrón
			// que el JS) para que no haya que añadir columnas cada vez.
			key := "daily_" + time.Now().Format("2006-01-02")

			// (marcador de posición: aquí iría store.Setting/ToSetting para leer
			// si ya lo reclamó; se deja simple para el esqueleto)
			_ = key

			premio := int64(100 + rand.Intn(151))
			coins, err := c.DB.AddCoins(ctx, jid, premio)
			if err != nil {
				return err
			}
			acc, err := c.DB.AddXP(ctx, jid, 50)
			if err != nil {
				return err
			}

			return c.ReplyButtons(
				"🎁 Recompensa diaria",
				fmt.Sprintf("╭─「 *DAILY* 」\n│ Ganaste: *%d* monedas\n│ Cartera: *%d*\n│ Nivel: *%d* (%d XP)\n╰─ vuelve mañana",
					premio, coins, acc.Level, acc.XP),
				"Shin-MD Go",
				bot.QuickReply("👤 Perfil", "perfil"),
				bot.QuickReply("📋 Menú", "menu"),
			)
		},
	})

	bot.Register(bot.Command{
		Name:     "perfil",
		Aliases:  []string{"profile", "yo"},
		Category: bot.CatPerfil,
		Desc:     "Tu perfil y economía",
		Run: func(c *bot.Context) error {
			acc, err := c.DB.Get(c.Ctx, c.Sender.String())
			if err != nil {
				return err
			}
			nombre := c.PushName
			if nombre == "" {
				nombre = c.Sender.User
			}
			return c.Reply(fmt.Sprintf(
				"╭─「 *PERFIL* 」\n│ 👤 %s\n│ 💰 Cartera: %d\n│ 🏦 Banco: %d\n│ ⭐ Nivel: %d (%d XP)\n╰─",
				nombre, acc.Coins, acc.Bank, acc.Level, acc.XP))
		},
	})
}
