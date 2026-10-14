// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// db.go — Migraciones versionadas simples (mismo espíritu que
// src/storage/migrations.js) y consultas de la economía.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // driver "sqlite" en Go puro, sin cgo
)

type Store struct{ db *sql.DB }

// migraciones: cada una es un escalón. Nunca se edita una ya publicada,
// se añade otra al final (igual que en Shin-MD JS).
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS users (
		jid        TEXT PRIMARY KEY,
		coins      INTEGER NOT NULL DEFAULT 0,
		bank       INTEGER NOT NULL DEFAULT 0,
		xp         INTEGER NOT NULL DEFAULT 0,
		level      INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS settings (
		jid  TEXT PRIMARY KEY,
		json TEXT NOT NULL DEFAULT '{}'
	);`,
	`CREATE TABLE IF NOT EXISTS warnings (
		jid TEXT NOT NULL,
		group_jid TEXT NOT NULL,
		count INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (jid, group_jid)
	);`,
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite escribe de uno en uno: evita "database is locked"
	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	// La DB guarda economía de usuarios: permisos 600, como en B5.1 del JS.
	_ = os.Chmod(path, 0o600)
	return s, nil
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	var current int
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_version`).Scan(&current)

	for i := current; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, migrations[i]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migración %d: %w", i+1, err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_version(version, applied_at) VALUES(?,?)`, i+1, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Get devuelve la cuenta, creándola si no existe (como el upsert perezoso del JS).
func (s *Store) Get(ctx context.Context, jid string) (*Account, error) {
	acc := &Account{JID: jid}
	err := s.db.QueryRowContext(ctx, `SELECT coins, bank, xp, level, created_at FROM users WHERE jid = ?`, jid).
		Scan(&acc.Coins, &acc.Bank, &acc.XP, &acc.Level, &acc.CreatedAt)
	if err == nil {
		return acc, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Bug que cazó la prueba: en JS el default era level 1, en Go el valor
	// cero del struct es 0. Se fija a mano al crear la cuenta.
	if acc.Level == 0 {
		acc.Level = 1
	}
	acc.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users(jid, coins, bank, xp, level, created_at) VALUES(?,?,?,?,?,?)`,
		jid, acc.Coins, acc.Bank, acc.XP, acc.Level, acc.CreatedAt)
	return acc, err
}

// AddCoins suma (o resta, con delta negativo) en una sola sentencia.
func (s *Store) AddCoins(ctx context.Context, jid string, delta int64) (int64, error) {
	if _, err := s.Get(ctx, jid); err != nil {
		return 0, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET coins = coins + ? WHERE jid = ?`, delta, jid); err != nil {
		return 0, err
	}
	var coins int64
	err := s.db.QueryRowContext(ctx, `SELECT coins FROM users WHERE jid = ?`, jid).Scan(&coins)
	return coins, err
}

// AddXP sube experiencia y calcula nivel (curva sencilla: nivel*100 por tramo).
func (s *Store) AddXP(ctx context.Context, jid string, delta int64) (*Account, error) {
	acc, err := s.Get(ctx, jid)
	if err != nil {
		return nil, err
	}
	acc.XP += delta
	level := 1
	for xp := acc.XP; xp >= int64(level*100); xp -= int64(level * 100) {
		level++
	}
	if level != acc.Level {
		acc.Level = level
	}
	_, err = s.db.ExecContext(ctx, `UPDATE users SET xp = ?, level = ? WHERE jid = ?`, acc.XP, acc.Level, jid)
	return acc, err
}

// Setting guarda config por JID (self mode, antilink, etc.) igual que el JS.
func (s *Store) Setting(ctx context.Context, jid, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings(jid, json) VALUES(?, json_set(COALESCE((SELECT json FROM settings WHERE jid=?), '{}'), '$.' || ?, ?))
		ON CONFLICT(jid) DO UPDATE SET json = json_set(settings.json, '$.' || ?, ?)`,
		jid, jid, key, value, key, value)
	return err
}

// CountUsers: útil para /health del panel web.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}
