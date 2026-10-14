// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// send.go — Envíos de las tarjetas nuevas (imagen+botones, carrusel, lista,
// encuesta, pin, evento, respuestas enriquecidas). Todos pasan por la cola
// anti-ban y todos caen a texto plano si WhatsApp no los acepta.
package wa

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"

	"github.com/riokuroxi-svg/Shin-Go/internal/ui"
)

// trySend intenta el envío rico con sus nodos; si falla, manda el texto.
func (c *Client) trySend(ctx context.Context, chat types.JID, msg *waE2E.Message, nodes []binary.Node, fallback string) error {
	var extra []whatsmeow.SendRequestExtra
	if len(nodes) > 0 {
		n := nodes
		extra = []whatsmeow.SendRequestExtra{{AdditionalNodes: &n}}
	}

	if _, err := c.WA.SendMessage(ctx, chat, msg, extra...); err == nil {
		return nil
	} else if fallback == "" {
		return err
	} else {
		c.Log.Warn("envío rico falló (" + err.Error() + "), cayendo a texto plano")
	}

	_, err := c.WA.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(fallback)})
	return err
}

// ─── Tarjetas con botones ────────────────────────────────────────────────────

// SendButtons: tarjeta con botones (el comportamiento de siempre).
func (c *Client) SendButtons(ctx context.Context, chat types.JID, title, body, footer string, buttons ...ui.Button) error {
	if err := c.Queue.Wait(ctx, len([]rune(body)), true); err != nil && err != ErrWarmupLimit {
		return err
	}
	msg := ui.BuildButtons(title, body, footer, buttons...)
	return c.trySend(ctx, chat, msg, ui.BizNodes(chat, true), ui.Plain(body, footer, buttons...))
}

// SendButtonsImage: tarjeta con IMAGEN de cabecera + botones. Es la que más
// luce en un menú: foto arriba, texto y botones. Si la subida falla, degrada
// a la tarjeta de texto (nunca se queda sin responder).
func (c *Client) SendButtonsImage(ctx context.Context, chat types.JID, image []byte, title, body, footer string, buttons ...ui.Button) error {
	if err := c.Queue.Wait(ctx, len([]rune(body)), true); err != nil && err != ErrWarmupLimit {
		return err
	}
	imgMsg, err := c.uploadImage(ctx, image)
	if err != nil {
		return c.SendButtons(ctx, chat, title, body, footer, buttons...)
	}
	msg := ui.BuildButtonsImage(imgMsg, title, body, footer, buttons...)
	return c.trySend(ctx, chat, msg, ui.BizNodes(chat, true), ui.Plain(body, footer, buttons...))
}

// SendCarousel: carrusel de tarjetas deslizables (con el envoltorio carousel).
func (c *Client) SendCarousel(ctx context.Context, chat types.JID, body string, cards ...ui.Card) error {
	if err := c.Queue.Wait(ctx, len([]rune(body)), true); err != nil && err != ErrWarmupLimit {
		return err
	}
	return c.trySend(ctx, chat, ui.BuildCarousel(cards...), ui.BizNodesCarousel(chat, true), ui.PlainCarousel(body, cards...))
}

// SendList: desplegable clásico. No lleva nodos <biz>: el propio servidor le
// añade sus atributos (v=2, type=single_select).
func (c *Client) SendList(ctx context.Context, chat types.JID, title, body, buttonText, footer string, sections ...ui.ListSection) error {
	if err := c.Queue.Wait(ctx, len([]rune(body)), true); err != nil && err != ErrWarmupLimit {
		return err
	}
	return c.trySend(ctx, chat, ui.BuildList(title, body, buttonText, footer, sections...), nil,
		ui.PlainList(title, body, buttonText, footer, sections...))
}

// ─── Encuestas y extras del proto ────────────────────────────────────────────

// SendPoll usa el helper OFICIAL de whatsmeow (BuildPollCreation). En Baileys
// esto se arma a mano: el PollCreationMessage lo pone el usuario.
func (c *Client) SendPoll(ctx context.Context, chat types.JID, question string, options []string, selectable int) error {
	if err := c.Queue.Wait(ctx, len([]rune(question)), false); err != nil && err != ErrWarmupLimit {
		return err
	}
	_, err := c.WA.SendMessage(ctx, chat, c.WA.BuildPollCreation(question, options, selectable))
	return err
}

// SendPollAddOption: añade una opción a una encuesta EN VIVO (sin helper
// oficial en ninguna lib: el proto está, aquí lo armamos).
func (c *Client) SendPollAddOption(ctx context.Context, chat types.JID, pollKey *waCommon.MessageKey, option string) error {
	if err := c.Queue.Wait(ctx, len([]rune(option)), false); err != nil && err != ErrWarmupLimit {
		return err
	}
	_, err := c.WA.SendMessage(ctx, chat, ui.BuildPollAddOption(pollKey, option))
	return err
}

// PinMessage: fija/desfija un mensaje en el chat.
func (c *Client) PinMessage(ctx context.Context, chat types.JID, key *waCommon.MessageKey, pin bool) error {
	_, err := c.WA.SendMessage(ctx, chat, ui.BuildPin(key, pin))
	return err
}

// SendEvent: evento de calendario (lo avisa WhatsApp, no el bot).
func (c *Client) SendEvent(ctx context.Context, chat types.JID, name, desc string, start, end time.Time, extraGuests bool, reminder time.Duration) error {
	if err := c.Queue.Wait(ctx, len([]rune(name)), false); err != nil && err != ErrWarmupLimit {
		return err
	}
	_, err := c.WA.SendMessage(ctx, chat, ui.BuildEvent(name, desc, start, end, extraGuests, reminder))
	return err
}

// SendRich: respuesta enriquecida estilo IA (markdown/código/tabla).
// Gateada por RICH_AI=1 y SIEMPRE con fallback: falta verificar en teléfono
// que la app la pinte desde una cuenta normal (por eso no va por defecto).
func (c *Client) SendRich(ctx context.Context, chat types.JID, msg *waE2E.Message, fallback string) error {
	if err := c.Queue.Wait(ctx, len([]rune(fallback)), false); err != nil && err != ErrWarmupLimit {
		return err
	}
	if !c.cfg.RichAI {
		_, err := c.WA.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(fallback)})
		return err
	}
	return c.trySend(ctx, chat, msg, ui.BizNodes(chat, true), fallback)
}

// uploadImage sube una imagen y devuelve el ImageMessage listo para encabezar
// una tarjeta interactiva.
func (c *Client) uploadImage(ctx context.Context, data []byte) (*waE2E.ImageMessage, error) {
	up, err := c.WA.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return nil, err
	}
	return &waE2E.ImageMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		Mimetype:      proto.String("image/jpeg"),
	}, nil
}
