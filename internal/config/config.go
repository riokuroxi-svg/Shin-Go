// Shin-Go — esqueleto de migración de Shin-MD (JS) a Go.
// Copyright (C) 2026 riokuroxi-svg · SPDX-License-Identifier: AGPL-3.0-only
//
// config.go — configuración por variables de entorno, con los mismos
// nombres que ya usa Shin-MD para que el .env se reutilice tal cual.
package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	StateDir string // carpeta de datos (equivalente a ./data)
	DBFile   string // economía/perfil/settings
	AuthFile string // sesión de WhatsApp (antes auth.db / creds.json)

	Prefixes []string // BOT_PREFIX=".,/,!,+" (mismo .env)

	LogLevel string
	RichAI   bool // RICH_AI=1: respuestas enriquecidas estilo IA (experimental)

	// Anti-ban (mismos parámetros que el motor JS)
	BaseDelay   time.Duration // 1200ms por defecto
	JitterSigma float64       // 0.25
	PerCharMS   int           // +25ms por carácter
	MaxDelay    time.Duration // tope del jitter (5s)
	WarmupStart int           // 20 mensajes/día
	WarmupEnd   int           // 500 mensajes/día a los 7 días
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// Load lee el entorno. No hay dependencias: en Go no hace falta dotenv,
// pero puedes exportar el .env con `set -a; . ./.env; set +a` antes de arrancar.
func Load() Config {
	stateDir := env("STATE_DIR", "data")
	prefixes := []string{".", "/", "#", "!"}
	if raw := strings.TrimSpace(os.Getenv("BOT_PREFIX")); raw != "" {
		prefixes = nil
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				prefixes = append(prefixes, p)
			}
		}
	}

	return Config{
		StateDir: stateDir,
		DBFile:   env("SHIN_DB_PATH", filepath.Join(stateDir, "shin.db")),
		AuthFile: env("SHIN_AUTH_DB", filepath.Join(stateDir, "auth.db")),

		Prefixes: prefixes,
		LogLevel: env("LOG_LEVEL", "info"),
		RichAI:   env("RICH_AI", "0") == "1",

		BaseDelay:   time.Duration(envInt("ANTIBAN_BASE_MS", 1200)) * time.Millisecond,
		JitterSigma: envFloat("ANTIBAN_SIGMA", 0.25),
		PerCharMS:   envInt("ANTIBAN_PERCHAR_MS", 25),
		MaxDelay:    5 * time.Second,
		WarmupStart: envInt("WARMUP_START", 20),
		WarmupEnd:   envInt("WARMUP_END", 500),
	}
}
