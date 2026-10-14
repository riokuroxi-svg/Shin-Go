// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// buttons.go — Catálogo COMPLETO de botones y tarjetas de WhatsApp, tipado.
//
// Puerto de src/lib/wa-nodes.js + src/commands/interactive.js + lib/native-params.js,
// más lo que WhatsApp añadió después (single_select, cta_call, webview, flow) y
// lo que sí trae whatsmeow hoy y Baileys 6.7.24 no (carrusel del emisor, lista
// legacy, encuestas con opción añadida, evento de calendario, pin, rich AI).
//
// Regla que no cambia: los botones NO son un flag de la librería. Son un
// proto InteractiveMessage → nativeFlowMessage(v9) + los nodos <biz>. Si faltan
// los nodos, el servidor acepta y el teléfono lo tira sin pintar y sin error.
package ui

import (
	"encoding/json"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// ─── Botones nativos (name + params JSON) ────────────────────────────────────

type Button struct {
	Name   string
	Params string
}

func jsonButton(name string, params any) Button {
	raw, err := json.Marshal(params)
	if err != nil {
		raw = []byte("{}")
	}
	return Button{Name: name, Params: string(raw)}
}

func QuickReply(label, id string) Button {
	return jsonButton("quick_reply", map[string]any{"display_text": label, "id": id})
}

func CTAURL(label, url string) Button {
	return jsonButton("cta_url", map[string]any{"display_text": label, "url": url, "merchant_url": url})
}

func CTACopy(label, code string) Button {
	return jsonButton("cta_copy", map[string]any{"display_text": label, "copy_code": code})
}

// CTACall: botón de llamada telefónica.
func CTACall(label, phone string) Button {
	return jsonButton("cta_call", map[string]any{"display_text": label, "phone_number": phone})
}

// OpenWebview: abre una web dentro de WhatsApp (webview embebido).
func OpenWebview(title, url string) Button {
	return jsonButton("open_webview", map[string]any{"title": title, "url": url})
}

// SendLocation: botón "enviar ubicación" (sin parámetros).
func SendLocation(label string) Button {
	if label != "" {
		return jsonButton("send_location", map[string]any{"display_text": label})
	}
	return jsonButton("send_location", map[string]any{})
}

// MPM: mensaje multi-producto (catálogo de negocio).
func MPM(businessJID string) Button {
	return jsonButton("mpm", map[string]any{"business_phone_number": businessJID})
}

// FlowButton: abre un WhatsApp Flow publicado (requiere cuenta business).
func FlowButton(flowID, token, cta string) Button {
	return jsonButton("flow", map[string]any{
		"flow_message_version": "3",
		"flow_token":           token,
		"flow_id":              flowID,
		"flow_cta":             cta,
		"flow_action":          "navigate",
		"flow_action_payload":  map[string]any{"screen": "WELCOME"},
	})
}

// SelectRow: una fila del desplegable nativo (single_select).
type SelectRow struct {
	ID    string
	Title string
	Desc  string
}

type SelectSection struct {
	Title string
	Rows  []SelectRow
}

// SingleSelect: el "lista desplegable" moderno (native flow, sin tocar
// ListMessage). Es el mismo que usa .menucat en el bot de JS.
func SingleSelect(title string, sections ...SelectSection) Button {
	type rowJSON struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description,omitempty"`
	}
	type sectionJSON struct {
		Title string    `json:"title"`
		Rows  []rowJSON `json:"rows"`
	}
	out := make([]sectionJSON, 0, len(sections))
	for _, s := range sections {
		sec := sectionJSON{Title: s.Title}
		for _, r := range s.Rows {
			sec.Rows = append(sec.Rows, rowJSON{ID: r.ID, Title: r.Title, Description: r.Desc})
		}
		out = append(out, sec)
	}
	return jsonButton("single_select", map[string]any{"title": title, "sections": out})
}

// ─── Construcción de tarjetas ────────────────────────────────────────────────

func nativeFlow(buttons []Button) *waE2E.InteractiveMessage_NativeFlowMessage {
	nf := &waE2E.InteractiveMessage_NativeFlowMessage{
		MessageVersion: proto.Int32(9), // v9 = mixed (quick_reply + cta_*)
	}
	for _, b := range buttons {
		nf.Buttons = append(nf.Buttons, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			Name:             proto.String(b.Name),
			ButtonParamsJSON: proto.String(b.Params),
		})
	}
	return nf
}

type headerOpts struct {
	image *waE2E.ImageMessage
	video *waE2E.VideoMessage
	title string
}

// BuildButtons: tarjeta con botones y cabecera de texto.
func BuildButtons(title, body, footer string, buttons ...Button) *waE2E.Message {
	return BuildButtonsMedia(headerOpts{title: title}, body, footer, buttons...)
}

// BuildButtonsImage: tarjeta con IMAGEN de cabecera + botones.
// Es el formato "rico" que más vende en los menús (el header acepta
// ImageMessage ya subida con cli.Upload).
func BuildButtonsImage(image *waE2E.ImageMessage, title, body, footer string, buttons ...Button) *waE2E.Message {
	return BuildButtonsMedia(headerOpts{image: image, title: title}, body, footer, buttons...)
}

func BuildButtonsMedia(h headerOpts, body, footer string, buttons ...Button) *waE2E.Message {
	interactive := &waE2E.InteractiveMessage{
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: nativeFlow(buttons),
		},
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String(body)},
	}

	if h.image != nil || h.video != nil || h.title != "" {
		header := &waE2E.InteractiveMessage_Header{Title: proto.String(h.title)}
		switch {
		case h.image != nil:
			header.Media = &waE2E.InteractiveMessage_Header_ImageMessage{ImageMessage: h.image}
			header.HasMediaAttachment = proto.Bool(true)
		case h.video != nil:
			header.Media = &waE2E.InteractiveMessage_Header_VideoMessage{VideoMessage: h.video}
			header.HasMediaAttachment = proto.Bool(true)
		default:
			header.HasMediaAttachment = proto.Bool(false)
		}
		interactive.Header = header
	}
	if footer != "" {
		interactive.Footer = &waE2E.InteractiveMessage_Footer{
			Text:               proto.String(footer),
			HasMediaAttachment: proto.Bool(false),
		}
	}
	return &waE2E.Message{InteractiveMessage: interactive}
}

// Card: una tarjeta del carrusel (cada una con su foto y sus propios botones).
type Card struct {
	Title   string
	Body    string
	Image   *waE2E.ImageMessage
	Buttons []Button
}

// BuildCarousel: carrusel horizontal (HSCROLL_CARDS). Es el "menú con tarjetas
// deslizables" que los forks de Baileys venden como novedad; aquí es una
// función de 30 líneas porque InteractiveMessage_CarouselMessage ya existe en
// el proto.
func BuildCarousel(cards ...Card) *waE2E.Message {
	protoCards := make([]*waE2E.InteractiveMessage, 0, len(cards))
	for _, c := range cards {
		card := &waE2E.InteractiveMessage{
			InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
				NativeFlowMessage: nativeFlow(c.Buttons),
			},
			Body: &waE2E.InteractiveMessage_Body{Text: proto.String(c.Body)},
		}
		if c.Title != "" || c.Image != nil {
			h := &waE2E.InteractiveMessage_Header{Title: proto.String(c.Title)}
			if c.Image != nil {
				h.Media = &waE2E.InteractiveMessage_Header_ImageMessage{ImageMessage: c.Image}
				h.HasMediaAttachment = proto.Bool(true)
			}
			card.Header = h
		}
		protoCards = append(protoCards, card)
	}

	return &waE2E.Message{
		InteractiveMessage: &waE2E.InteractiveMessage{
			InteractiveMessage: &waE2E.InteractiveMessage_CarouselMessage_{
				CarouselMessage: &waE2E.InteractiveMessage_CarouselMessage{
					Cards:            protoCards,
					MessageVersion:   proto.Int32(1),
					CarouselCardType: waE2E.InteractiveMessage_CarouselMessage_HSCROLL_CARDS.Enum(),
				},
			},
			Body: &waE2E.InteractiveMessage_Body{Text: proto.String("")},
		},
	}
}

// ─── Lista clásica (desplegable legacy) ──────────────────────────────────────

type ListRow struct {
	ID    string
	Title string
	Desc  string
}

type ListSection struct {
	Title string
	Rows  []ListRow
}

// BuildList: el ListMessage de siempre. WhatsApp lo sigue pintando como
// desplegable en la mayoría de clientes; el server le añade sus atributos
// (v=2, type=single_select) por lo que NO hay que mandar nodos <biz>.
func BuildList(title, body, buttonText, footer string, sections ...ListSection) *waE2E.Message {
	protoSections := make([]*waE2E.ListMessage_Section, 0, len(sections))
	for _, s := range sections {
		sec := &waE2E.ListMessage_Section{Title: proto.String(s.Title)}
		for _, r := range s.Rows {
			sec.Rows = append(sec.Rows, &waE2E.ListMessage_Row{
				Title:       proto.String(r.Title),
				Description: proto.String(r.Desc),
				RowID:       proto.String(r.ID),
			})
		}
		protoSections = append(protoSections, sec)
	}

	return &waE2E.Message{
		ListMessage: &waE2E.ListMessage{
			Title:       proto.String(title),
			Description: proto.String(body),
			ButtonText:  proto.String(buttonText),
			FooterText:  proto.String(footer),
			ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    protoSections,
		},
	}
}

// ─── Fallback a texto (nunca dejar al usuario sin respuesta) ─────────────────

func Plain(body, footer string, buttons ...Button) string {
	var sb strings.Builder
	sb.WriteString(body)
	for _, b := range buttons {
		var p struct {
			DisplayText string `json:"display_text"`
			Title       string `json:"title"`
			ID          string `json:"id"`
			URL         string `json:"url"`
		}
		_ = json.Unmarshal([]byte(b.Params), &p)
		sb.WriteString("\n▸ ")
		switch {
		case p.DisplayText != "":
			sb.WriteString(p.DisplayText)
		case p.Title != "":
			sb.WriteString(p.Title)
		default:
			sb.WriteString(b.Name)
		}
		if p.ID != "" {
			sb.WriteString("  ·  ." + strings.TrimPrefix(p.ID, "."))
		}
		if p.URL != "" {
			sb.WriteString("  ·  " + p.URL)
		}
	}
	if footer != "" {
		sb.WriteString("\n\n" + footer)
	}
	return sb.String()
}

func PlainList(title, body, buttonText, footer string, sections ...ListSection) string {
	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(body)
	for _, s := range sections {
		sb.WriteString("\n\n▪ " + s.Title)
		for _, r := range s.Rows {
			sb.WriteString("\n  ▸ " + r.Title + "  ·  ." + strings.TrimPrefix(r.ID, "."))
		}
	}
	if buttonText != "" {
		sb.WriteString("\n\n(" + buttonText + ")")
	}
	if footer != "" {
		sb.WriteString("\n" + footer)
	}
	return sb.String()
}

func PlainCarousel(body string, cards ...Card) string {
	var sb strings.Builder
	sb.WriteString(body)
	for _, c := range cards {
		sb.WriteString("\n\n━━ " + c.Title + " ━━\n" + c.Body)
		sb.WriteString(Plain("", "", c.Buttons...))
	}
	return sb.String()
}

// ─── Nodos binarios ──────────────────────────────────────────────────────────

func IsPrivate(jid types.JID) bool {
	return jid.Server == types.DefaultUserServer || jid.Server == types.HiddenUserServer
}

// BizNodes: los nodos que hacen que WhatsApp pinte el mensaje interactivo.
//
//	biz → interactive(type=native_flow, v=1) → native_flow(v=9, name=mixed)
//	+ quality_control
//	+ bot(biz_bot=1)  SOLO en chats privados (enciende la ✨ de IA)
func BizNodes(jid types.JID, ai bool) []binary.Node {
	biz := binary.Node{
		Tag: "biz",
		Attrs: binary.Attrs{
			"actual_actors":   "2",
			"host_storage":    "2",
			"privacy_mode_ts": time.Now().Unix(),
		},
		Content: []binary.Node{
			{
				Tag:   "interactive",
				Attrs: binary.Attrs{"type": "native_flow", "v": "1"},
				Content: []binary.Node{
					{Tag: "native_flow", Attrs: binary.Attrs{"v": "9", "name": "mixed"}},
				},
			},
			{Tag: "quality_control", Attrs: binary.Attrs{"source_type": "third_party"}},
		},
	}

	nodes := []binary.Node{biz}
	if ai && IsPrivate(jid) {
		nodes = append(nodes, binary.Node{Tag: "bot", Attrs: binary.Attrs{"biz_bot": "1"}})
	}
	return nodes
}

// BizNodesCarousel: el carrusel lleva su propio envoltorio (name=carousel).
func BizNodesCarousel(jid types.JID, ai bool) []binary.Node {
	nodes := BizNodes(jid, ai)
	if len(nodes) > 0 {
		if kids, ok := nodes[0].Content.([]binary.Node); ok && len(kids) > 0 {
			if grandkids, ok := kids[0].Content.([]binary.Node); ok && len(grandkids) > 0 {
				grandkids[0].Attrs["name"] = "carousel"
			}
		}
	}
	return nodes
}

// Buildea un MessageKey (para pin, edición, respuestas…).
func Key(chat types.JID, fromMe bool, id types.MessageID, sender types.JID) *waCommon.MessageKey {
	k := &waCommon.MessageKey{
		RemoteJID: proto.String(chat.String()),
		FromMe:    proto.Bool(fromMe),
		ID:        proto.String(id),
	}
	if !sender.IsEmpty() {
		k.Participant = proto.String(sender.String())
	}
	return k
}
