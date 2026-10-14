// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// main.go — Arranque. Equivale a index.js → boot/index.js en el JS.
//
//	shin                 → QR si no hay sesión
//	shin --code 521…     → código de 8 dígitos
//	shin --version
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "shin-go/cmds" // sus init() registran los comandos
	"shin-go/internal/bot"
	"shin-go/internal/config"
	"shin-go/internal/store"
	"shin-go/internal/wa"
	"shin-go/internal/xlog"
)

var version = "0.1.0-skeleton"

func main() {
	var (
		phone   = flag.String("code", "", "vincular con número (código de 8 dígitos)")
		showVer = flag.Bool("version", false, "mostrar versión")
	)
	flag.Parse()

	if *showVer {
		fmt.Println("Shin-MD (Go)", version)
		return
	}

	cfg := config.Load()
	log := xlog.New(cfg.LogLevel, os.Stdout)

	// ctx se cancela con Ctrl+C → cierre limpio (adeós PM2 y sus reinicios sucios).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(cfg.DBFile)
	if err != nil {
		log.Err("no se pudo abrir la base de datos: " + err.Error())
		os.Exit(1)
	}
	defer db.Close()

	client, err := wa.New(ctx, cfg, log)
	if err != nil {
		log.Err("no se pudo crear el cliente: " + err.Error())
		os.Exit(1)
	}
	defer client.Close()

	owners := strings.FieldsFunc(os.Getenv("OWNERS"), func(r rune) bool { return r == ',' || r == ' ' })

	uniq, aliases := bot.Count()
	log.Info(fmt.Sprintf("Comandos cargados: %d únicos · %d nombres", uniq, aliases))

	router := bot.NewRouter(cfg, log, client, db, owners)
	client.OnMessage(router.Handle)

	if err := client.Login(ctx, *phone); err != nil {
		log.Err("login: " + err.Error())
		os.Exit(1)
	}

	log.Ok("Shin-MD en marcha. Ctrl+C para salir.")
	<-ctx.Done()
	log.Info("Cerrando limpio…")
}
