// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// store.go — Economía/perfil/settings. Equivale a src/storage/database.js
// (node:sqlite) pero con database/sql + modernc.org/sqlite (SQLite puro Go).
package store

type Account struct {
	JID       string
	Coins     int64
	Bank      int64
	XP        int64
	Level     int
	CreatedAt string
}

func (s *Store) Close() error { return s.db.Close() }
