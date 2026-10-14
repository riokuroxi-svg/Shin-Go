// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
package ui

import (
	"encoding/json"
	"testing"
	"time"

	"go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waAICommonDeprecated"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// ── Lista desplegable ────────────────────────────────────────────────────────

func TestBuildList(t *testing.T) {
	msg := BuildList("SHIN-MD", "Elige", "Ver opciones", "pie",
		ListSection{Title: "General", Rows: []ListRow{
			{ID: "ping", Title: "Ping", Desc: "latencia"},
			{ID: "menu:economia", Title: "Economía", Desc: "monedas"},
		}},
	)
	lm := msg.GetListMessage()
	if lm == nil {
		t.Fatal("no es ListMessage")
	}
	if lm.GetListType() != waE2E.ListMessage_SINGLE_SELECT {
		t.Fatalf("listType = %v", lm.GetListType())
	}
	if len(lm.GetSections()) != 1 || len(lm.GetSections()[0].GetRows()) != 2 {
		t.Fatal("secciones/filas mal armadas")
	}
	if lm.GetSections()[0].GetRows()[1].GetRowID() != "menu:economia" {
		t.Fatal("rowID incorrecto")
	}
	if lm.GetButtonText() != "Ver opciones" {
		t.Fatal("buttonText incorrecto")
	}
}

// ── Carrusel ─────────────────────────────────────────────────────────────────

func TestBuildCarousel(t *testing.T) {
	msg := BuildCarousel(
		Card{Title: "Carta 1", Body: "uno", Buttons: []Button{QuickReply("Ver", "ping")}},
		Card{Title: "Carta 2", Body: "dos", Buttons: []Button{QuickReply("Ver", "menu")}},
	)
	im := msg.GetInteractiveMessage()
	if im == nil {
		t.Fatal("no es InteractiveMessage")
	}
	car := im.GetCarouselMessage()
	if car == nil {
		t.Fatal("sin carouselMessage")
	}
	if car.GetCarouselCardType() != waE2E.InteractiveMessage_CarouselMessage_HSCROLL_CARDS {
		t.Fatalf("tipo de carrusel = %v", car.GetCarouselCardType())
	}
	if len(car.GetCards()) != 2 {
		t.Fatalf("%d cartas", len(car.GetCards()))
	}
	nf := car.GetCards()[0].GetNativeFlowMessage()
	if nf == nil || len(nf.GetButtons()) != 1 || nf.GetButtons()[0].GetName() != "quick_reply" {
		t.Fatal("la carta no lleva sus botones")
	}
}

// El carrusel lleva name=carousel en el nodo native_flow.
func TestBizNodesCarousel(t *testing.T) {
	nodes := BizNodesCarousel(privJID(t), true)
	kids, ok := nodes[0].Content.([]binary.Node)
	if !ok {
		t.Fatal("contenido del nodo biz no es []binary.Node")
	}
	grandkids, ok := kids[0].Content.([]binary.Node)
	if !ok || grandkids[0].Attrs["name"] != "carousel" {
		t.Fatal("el nodo no dice carousel")
	}
}

// ── Botones nuevos ───────────────────────────────────────────────────────────

func TestNewButtons(t *testing.T) {
	cases := []struct {
		btn  Button
		name string
		want map[string]string
	}{
		{CTACall("Llamar", "+52155"), "cta_call", map[string]string{"phone_number": "+52155"}},
		{SendLocation("Mi ubicación"), "send_location", map[string]string{"display_text": "Mi ubicación"}},
		{OpenWebview("Web", "https://x.dev"), "open_webview", map[string]string{"url": "https://x.dev"}},
		{FlowButton("123", "tok", "Empezar"), "flow", map[string]string{"flow_id": "123", "flow_cta": "Empezar"}},
	}
	for _, c := range cases {
		if c.btn.Name != c.name {
			t.Fatalf("%s: nombre %q", c.name, c.btn.Name)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(c.btn.Params), &got); err != nil {
			t.Fatalf("%s: JSON inválido %v", c.name, err)
		}
		for k, v := range c.want {
			if got[k] != v {
				t.Fatalf("%s: %s = %v (esperado %v)", c.name, k, got[k], v)
			}
		}
	}
}

func TestSingleSelect(t *testing.T) {
	b := SingleSelect("Categorías",
		SelectSection{Title: "Menú", Rows: []SelectRow{{ID: "menu:economia", Title: "Economía", Desc: "monedas"}}},
	)
	if b.Name != "single_select" {
		t.Fatalf("nombre = %s", b.Name)
	}
	var got struct {
		Title    string `json:"title"`
		Sections []struct {
			Title string `json:"title"`
			Rows  []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"rows"`
		} `json:"sections"`
	}
	if err := json.Unmarshal([]byte(b.Params), &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Categorías" || got.Sections[0].Rows[0].ID != "menu:economia" {
		t.Fatalf("params mal: %+v", got)
	}
}

// ── Rich (IA), evento, pin, encuesta ─────────────────────────────────────────

func TestRichMessages(t *testing.T) {
	text := RichText("# Hola **mundo**")
	rm := text.GetRichResponseMessage()
	if rm == nil || len(rm.GetSubmessages()) != 1 {
		t.Fatal("rich text mal armado")
	}
	if rm.GetSubmessages()[0].GetMessageType() != waAICommonDeprecated.AIRichResponseSubMessageType_AI_RICH_RESPONSE_TEXT {
		t.Fatal("tipo de submensaje incorrecto")
	}
	if rm.GetSubmessages()[0].GetMessageText() != "# Hola **mundo**" {
		t.Fatal("texto perdido")
	}

	code := RichCode("go", "fmt.Println(1)")
	if code.GetRichResponseMessage().GetSubmessages()[0].GetCodeMetadata().GetCodeLanguage() != "go" {
		t.Fatal("rich code mal armado")
	}

	tbl := RichTable("Niveles", []string{"user", "xp"}, [][]string{{"ana", "10"}})
	rows := tbl.GetRichResponseMessage().GetSubmessages()[0].GetTableMetadata().GetRows()
	if len(rows) != 2 || !rows[0].GetIsHeading() || rows[1].GetItems()[1] != "10" {
		t.Fatal("rich table mal armada")
	}

	if RichTextCard("txt", "go", "x").GetRichResponseMessage() == nil {
		t.Fatal("rich combinado")
	}
}

func TestBuildEventAndPin(t *testing.T) {
	start := time.Now()
	msg := BuildEvent("Reunión", "grupo", start, start.Add(time.Hour), true, 10*time.Minute)
	ev := msg.GetEventMessage()
	if ev == nil || ev.GetName() != "Reunión" || !ev.GetHasReminder() {
		t.Fatal("evento mal armado")
	}
	if ev.GetReminderOffsetSec() != 600 {
		t.Fatalf("recordatorio = %d s", ev.GetReminderOffsetSec())
	}

	pin := BuildPin(Key(chatJID(t), false, "ABC123", senderJID(t)), true)
	pm := pin.GetPinInChatMessage()
	if pm == nil || pm.GetType() != waE2E.PinInChatMessage_PIN_FOR_ALL || pm.GetKey().GetID() != "ABC123" {
		t.Fatal("pin mal armado")
	}
	unpin := BuildPin(Key(chatJID(t), false, "ABC123", senderJID(t)), false)
	if unpin.GetPinInChatMessage().GetType() != waE2E.PinInChatMessage_UNPIN_FOR_ALL {
		t.Fatal("unpin mal armado")
	}
}

func TestBuildPollAddOption(t *testing.T) {
	msg := BuildPollAddOption(Key(chatJID(t), true, "POLL1", senderJID(t)), "opción nueva")
	pa := msg.GetPollAddOptionMessage()
	if pa == nil || pa.GetAddOption().GetOptionName() != "opción nueva" {
		t.Fatal("poll add option mal armado")
	}
	if pa.GetPollCreationMessageKey().GetID() != "POLL1" {
		t.Fatal("clave de encuesta perdida")
	}
}

// ── helpers de prueba ────────────────────────────────────────────────────────

func privJID(t *testing.T) types.JID {
	t.Helper()
	j, err := types.ParseJID("5215555555555@s.whatsapp.net")
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func chatJID(t *testing.T) types.JID   { return privJID(t) }
func senderJID(t *testing.T) types.JID { return privJID(t) }
