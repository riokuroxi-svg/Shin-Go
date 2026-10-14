// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
package ui

import (
	"encoding/json"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

// El proto debe salir con version 9 y los params JSON exactos que espera
// WhatsApp: si esto cambia, los botones salen sin pintar (y sin error).
func TestBuildButtons(t *testing.T) {
	msg := BuildButtons("SHIN-MD", "Hola", "pie",
		QuickReply("Ping", "ping"),
		CTAURL("Repo", "https://example.com"),
		CTACopy("Copiar", ".menu"),
	)

	im := msg.GetInteractiveMessage()
	if im == nil {
		t.Fatal("no es InteractiveMessage")
	}
	nf := im.GetNativeFlowMessage()
	if nf == nil {
		t.Fatal("sin nativeFlowMessage")
	}
	if nf.GetMessageVersion() != 9 {
		t.Fatalf("messageVersion = %d, se esperaba 9", nf.GetMessageVersion())
	}
	if len(nf.GetButtons()) != 3 {
		t.Fatalf("%d botones, se esperaban 3", len(nf.GetButtons()))
	}

	var p struct {
		DisplayText string `json:"display_text"`
		ID          string `json:"id"`
	}
	if err := json.Unmarshal([]byte(nf.GetButtons()[0].GetButtonParamsJSON()), &p); err != nil {
		t.Fatalf("params JSON inválidos: %v", err)
	}
	if nf.GetButtons()[0].GetName() != "quick_reply" || p.ID != "ping" || p.DisplayText != "Ping" {
		t.Fatalf("botón 0 mal armado: %+v (%s)", p, nf.GetButtons()[0].GetButtonParamsJSON())
	}
	if im.GetBody().GetText() != "Hola" || im.GetFooter().GetText() != "pie" {
		t.Fatal("body o footer mal puestos")
	}
}

// Los nodos biz son los que hacen que WhatsApp pinte la tarjeta.
func TestBizNodes(t *testing.T) {
	priv, _ := types.ParseJID("5215555555555@s.whatsapp.net")
	group, _ := types.ParseJID("1234-5678@g.us")

	nodes := BizNodes(priv, true)
	if len(nodes) != 2 || nodes[0].Tag != "biz" || nodes[1].Tag != "bot" {
		t.Fatalf("privado: se esperaban biz+bot, hay %d nodos", len(nodes))
	}
	kids, ok := nodes[0].Content.([]binary.Node)
	if !ok || len(kids) != 2 {
		t.Fatalf("el nodo biz debe llevar 2 hijos, hay %T", nodes[0].Content)
	}
	if kids[0].Attrs["type"] != "native_flow" || kids[1].Tag != "quality_control" {
		t.Fatal("estructura del nodo biz incorrecta")
	}
	grandkids, ok := kids[0].Content.([]binary.Node)
	if !ok || len(grandkids) != 1 || grandkids[0].Attrs["name"] != "mixed" {
		t.Fatal("native_flow interno mal armado")
	}

	if g := BizNodes(group, true); len(g) != 1 {
		t.Fatalf("en grupos no debe ir el nodo bot (nodos=%d)", len(g))
	}
}

func TestPlainFallback(t *testing.T) {
	txt := Plain("cuerpo", "pie", QuickReply("Ping", "ping"), CTAURL("Repo", "https://x.dev"))
	for _, want := range []string{"cuerpo", "Ping", ".ping", "https://x.dev", "pie"} {
		if !strings.Contains(txt, want) {
			t.Fatalf("el fallback no contiene %q:\n%s", want, txt)
		}
	}
}
