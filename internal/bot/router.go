// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// router.go — Puerto de src/commands/router.js:
// mensaje → ¿botón? → serializar → prefijo → cooldown → permisos → handler.
// Cooldown por usuario (10 req/min como en el JS) y recuperación de pánico:
// un handler que reviente NO tumba el bot (equivalente al anti-crash).
package bot

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/riokuroxi-svg/Shin-Go/internal/config"
	"github.com/riokuroxi-svg/Shin-Go/internal/store"
	"github.com/riokuroxi-svg/Shin-Go/internal/wa"
	"github.com/riokuroxi-svg/Shin-Go/internal/xlog"
)

type Router struct {
	cfg    config.Config
	log    *xlog.Logger
	client *wa.Client
	db     *store.Store
	owner  []string // números de owner en formato "521…"

	mu       sync.Mutex
	cooldown map[string][]time.Time // jid → instantes de uso (ventana 1 min)
}

func NewRouter(cfg config.Config, log *xlog.Logger, client *wa.Client, db *store.Store, owners []string) *Router {
	return &Router{
		cfg:      cfg,
		log:      log,
		client:   client,
		db:       db,
		owner:    owners,
		cooldown: map[string][]time.Time{},
	}
}

// Handle es el punto de entrada (se registra con client.OnMessage).
func (r *Router) Handle(_ context.Context, evt *events.Message) {
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Err(fmt.Sprintf("pánico en handler: %v\n%s", rec, debug.Stack()))
		}
	}()

	chat := evt.Info.Chat
	sender := evt.Info.Sender
	isGroup := chat.Server == "g.us"

	parsed := wa.ParseMessage(evt.Message, r.cfg.Prefixes)

	var cmdName string
	var args []string
	fromButton := false
	fullText := parsed.Text

	if parsed.IsButton && parsed.ButtonID != "" {
		cmdName, args = wa.CommandFromButton(parsed.ButtonID)
		fromButton = true
	} else if parsed.IsCommand {
		cmdName, args = parsed.Command, parsed.Args
		fullText = parsed.Text
	} else {
		return // no es comando: aquí van los hooks (antilink, afk, xp…)
	}

	cmd, ok := Lookup(cmdName)
	if !ok {
		return
	}

	// ── permisos ──────────────────────────────────────────────────────────
	isOwner := r.isOwner(sender)
	if cmd.OwnerOnly && !isOwner {
		_ = r.client.SendText(context.Background(), chat, "✕ Solo el owner puede usar ese comando.", true)
		return
	}
	if cmd.GroupOnly && !isGroup {
		_ = r.client.SendText(context.Background(), chat, "✕ Ese comando es solo para grupos.", true)
		return
	}

	// ── cooldown por usuario (10 req/min) ────────────────────────────────
	if !isOwner {
		if wait, limited := r.rateLimited(sender.String()); limited {
			r.log.Warn(fmt.Sprintf("rate-limit: %s (esperar %s)", sender, wait.Round(time.Second)))
			return
		}
	}

	ctx := &Context{
		Ctx:        context.Background(),
		Bot:        r.client,
		DB:         r.db,
		Log:        r.log,
		Event:      evt,
		Chat:       chat,
		Sender:     sender,
		IsGroup:    isGroup,
		IsOwner:    isOwner,
		PushName:   evt.Info.PushName,
		Command:    cmd.Name,
		Args:       args,
		Text:       fullText,
		Raw:        parsed.Text,
		fromButton: fromButton,
	}

	start := time.Now()
	if err := cmd.Run(ctx); err != nil {
		r.log.Err(fmt.Sprintf(".%s falló: %v", cmd.Name, err))
		return
	}
	r.log.Info(fmt.Sprintf(".%s ok en %dms", cmd.Name, time.Since(start).Milliseconds()))
}

// rateLimited: 10 peticiones por minuto y por usuario, como cooldown.js.
func (r *Router) rateLimited(key string) (time.Duration, bool) {
	const window = time.Minute
	const max = 10

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	kept := r.cooldown[key][:0]
	for _, t := range r.cooldown[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(kept) >= max {
		r.cooldown[key] = kept
		return window - now.Sub(kept[0]), true
	}
	r.cooldown[key] = append(kept, now)
	return 0, false
}

// isOwner compara solo el número (el JID puede venir como @s.whatsapp.net
// o como @lid, el bug clásico del JS: aquí se normaliza a User).
func (r *Router) isOwner(jid types.JID) bool {
	for _, o := range r.owner {
		if o == jid.User {
			return true
		}
	}
	return false
}
