// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// client.go — Envoltura sobre whatsmeow: sesión SQLite, emparejamiento
// (QR y código de 8 dígitos), reconexión y envíos con fallback.
//
// Equivalencias con el Shin-MD de JS:
//
//	src/core/socket.js   →  este archivo (whatsmeow ya reconecta solo)
//	src/core/auth.js     →  store/sqlstore (SQLite con migraciones propias)
//	scripts/patch-baileys-pairing.cjs → NO HACE FALTA (no hay forks ni parches)
package wa

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"shin-go/internal/config"
	"shin-go/internal/xlog"
)

type Client struct {
	WA        *whatsmeow.Client
	Queue     *Queue
	Log       *xlog.Logger
	container *sqlstore.Container
	cfg       config.Config

	onMessage func(ctx context.Context, evt *events.Message)
}

// New abre la base de sesión y construye el cliente. No se conecta todavía.
func New(ctx context.Context, cfg config.Config, log *xlog.Logger) (*Client, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.AuthFile), 0o700); err != nil {
		return nil, fmt.Errorf("crear carpeta de sesión: %w", err)
	}

	// modernc.org/sqlite es SQLite en Go PURO: sin cgo, sin compilar nada,
	// compila igual en Windows, en un VPS y en Termux (arm64).
	dsn := "file:" + cfg.AuthFile + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
	container, err := sqlstore.New(ctx, "sqlite", dsn, waLog.Stdout("wa-db", "ERROR", true))
	if err != nil {
		return nil, fmt.Errorf("abrir sesión SQLite: %w", err)
	}

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("leer dispositivo: %w", err)
	}

	cli := whatsmeow.NewClient(device, waLog.Stdout("whatsmeow", "WARN", true))
	cli.EnableAutoReconnect = true // clave: nada de borrar la sesión por cortes de red

	c := &Client{
		WA:        cli,
		Queue:     NewQueue(cfg),
		Log:       log,
		container: container,
		cfg:       cfg,
	}

	// Igual que en Shin-MD: nunca se borra una sesión registrada por un corte
	// de red. whatsmeow reconecta; aquí solo se avisa y se deja registro.
	cli.AddEventHandler(func(raw any) {
		switch evt := raw.(type) {
		case *events.Connected:
			log.Ok("Conectado a WhatsApp")
		case *events.Disconnected:
			log.Warn("Desconectado — reconexión automática en curso (sesión intacta)")
		case *events.ConnectFailure:
			log.Warn(fmt.Sprintf("Fallo de conexión: %s (%s)", evt.Reason, evt.Message))
		case *events.LoggedOut:
			log.Err("Sesión cerrada desde el teléfono (LoggedOut) — hace falta vincular otra vez")
		case *events.Message:
			if c.onMessage != nil {
				c.onMessage(context.Background(), evt)
			}
		}
	})

	return c, nil
}

func (c *Client) OnMessage(fn func(ctx context.Context, evt *events.Message)) { c.onMessage = fn }

func (c *Client) LoggedIn() bool { return c.WA.Store.ID != nil }

func (c *Client) Close() {
	c.WA.Disconnect()
	if c.container != nil {
		_ = c.container.Close()
	}
}

// Login: si ya hay sesión, conecta; si se pasa teléfono usa código de 8
// dígitos; si no, muestra el QR en la terminal.
func (c *Client) Login(ctx context.Context, phone string) error {
	if c.LoggedIn() {
		return c.WA.Connect()
	}

	if phone != "" {
		phone = strings.TrimPrefix(strings.TrimSpace(phone), "+")
		code, err := c.WA.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Shin-MD (Go)")
		if err != nil {
			return fmt.Errorf("pedir código de emparejamiento: %w", err)
		}
		c.Log.Info("Código de emparejamiento: " + code)
		c.Log.Info("WhatsApp → Dispositivos vinculados → Vincular con número de teléfono")
		return c.WA.Connect()
	}

	qrChan, err := c.WA.GetQRChannel(ctx)
	if err != nil {
		return err
	}
	if err = c.WA.Connect(); err != nil {
		return err
	}

	for item := range qrChan {
		switch item.Event {
		case "code":
			fmt.Println()
			qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, os.Stdout)
			c.Log.Info("Escanea el QR: WhatsApp → Dispositivos vinculados")
		case "success":
			c.Log.Ok("Vinculado ✔")
			return nil
		case "timeout":
			return fmt.Errorf("el QR caducó, vuelve a ejecutar")
		default:
			c.Log.Warn("QR: " + item.Event)
		}
	}
	return nil
}

// ─── Envíos ────────────────────────────────────────────────────────────────

// SendText respeta la cola anti-ban. priority salta el cupo de calentamiento.
func (c *Client) SendText(ctx context.Context, chat types.JID, text string, priority bool) error {
	if err := c.Queue.Wait(ctx, len([]rune(text)), priority); err != nil && err != ErrWarmupLimit {
		return err
	}
	_, err := c.WA.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(text)})
	return err
}

// SendImage sube y manda una imagen con pie de foto.
func (c *Client) SendImage(ctx context.Context, chat types.JID, data []byte, caption string) error {
	if err := c.Queue.Wait(ctx, len([]rune(caption)), false); err != nil && err != ErrWarmupLimit {
		return err
	}
	up, err := c.WA.Upload(ctx, data, whatsmeow.MediaImage)
	if err != nil {
		return err
	}
	msg := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
		Caption:       proto.String(caption),
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		Mimetype:      proto.String("image/jpeg"),
	}}
	_, err = c.WA.SendMessage(ctx, chat, msg)
	return err
}

// Presence: "available" / "unavailable" / "composing" (como el JS).
func (c *Client) SetTyping(ctx context.Context, chat types.JID) {
	_ = c.WA.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
}

// Prefixes / FirstPrefix: el router y los comandos leen los prefijos del .env.
func (c *Client) Prefixes() []string { return c.cfg.Prefixes }

func (c *Client) FirstPrefix() string {
	if len(c.cfg.Prefixes) == 0 {
		return "."
	}
	return c.cfg.Prefixes[0]
}
