// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// parse.go — Leer un mensaje entrante: texto normal, respuesta de botón
// nativo, o selección de lista. Puerto de parseButtonResponse() del JS.
package wa

import (
	"encoding/json"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// Parsed es el resultado de mirar un mensaje entrante.
type Parsed struct {
	Text      string // texto plano (vacío si es botón)
	ButtonID  string // id del botón tocado (formato "comando:args")
	IsButton  bool
	IsCommand bool // el texto empieza con prefijo
	Command   string
	Args      []string
	Prefix    string
}

// ParseMessage extrae lo relevante. Orden: botones → listas → texto.
func ParseMessage(msg *waE2E.Message, prefixes []string) Parsed {
	var out Parsed
	if msg == nil {
		return out
	}

	switch {
	case msg.GetConversation() != "":
		out.Text = msg.GetConversation()
	case msg.GetExtendedTextMessage() != nil:
		out.Text = msg.GetExtendedTextMessage().GetText()
	case msg.GetButtonsResponseMessage() != nil:
		out.ButtonID = msg.GetButtonsResponseMessage().GetSelectedButtonID()
		out.IsButton = true
		return out
	case msg.GetTemplateButtonReplyMessage() != nil:
		out.ButtonID = msg.GetTemplateButtonReplyMessage().GetSelectedID()
		out.IsButton = true
		return out
	case msg.GetListResponseMessage() != nil:
		out.ButtonID = msg.GetListResponseMessage().GetSingleSelectReply().GetSelectedRowID()
		out.IsButton = true
		return out
	case msg.GetInteractiveResponseMessage() != nil:
		// native_flow: el cliente devuelve {"id":"...","display_text":"..."} en paramsJSON
		if nf := msg.GetInteractiveResponseMessage().GetNativeFlowResponseMessage(); nf != nil {
			var p struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal([]byte(nf.GetParamsJSON()), &p)
			out.ButtonID = p.ID
			out.IsButton = true
		}
		return out
	}

	out.Text = strings.TrimSpace(out.Text)
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(out.Text, p) {
			rest := strings.TrimSpace(strings.TrimPrefix(out.Text, p))
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				break
			}
			out.IsCommand = true
			out.Prefix = p
			out.Command = strings.ToLower(fields[0])
			out.Args = fields[1:]
			break
		}
	}
	return out
}

// CommandFromButton traduce un id de botón a comando + args.
// Convención (idéntica al JS): "ping", "menu:categorias", "play:nombre canción".
func CommandFromButton(id string) (cmd string, args []string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil
	}
	parts := strings.SplitN(id, ":", 2)
	cmd = strings.ToLower(strings.TrimSpace(parts[0]))
	if len(parts) == 2 {
		args = strings.Fields(parts[1])
		if len(args) == 0 && strings.TrimSpace(parts[1]) != "" {
			args = []string{strings.TrimSpace(parts[1])}
		}
	}
	return cmd, args
}
