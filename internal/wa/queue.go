// Shin-MD-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// queue.go — Cola de envío con jitter gaussiano y calentamiento diario.
// Es el puerto de la "huella anti-ban" de Shin-MD:
//
//	base 1200 ms · sigma 0.25 · +25 ms por carácter (tope 5 s)
//	calentamiento 20→500 mensajes/día en 7 días
//	prioridad: los comandos rápidos (.menu, .ping) saltan el warm-up
//
// La diferencia con el JS no es la fórmula (es la misma), es que aquí no
// hay un solo Math.random() ejecutándose en el hilo de eventos: la cola es
// un goroutine propio con contexto cancelable y contador de fallos.
package wa

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"shin-go/internal/config"
)

type Queue struct {
	cfg config.Config

	mu        sync.Mutex
	lastSent  time.Time
	day       string
	sentToday int
	warmupDay string
	failures  int
}

func NewQueue(cfg config.Config) *Queue { return &Queue{cfg: cfg} }

// delay calcula la espera antes del próximo envío: jitter gaussiano sobre la
// base, más tiempo por longitud del texto, con recorte a [400ms, MaxDelay].
func (q *Queue) delay(textLen int) time.Duration {
	base := float64(q.cfg.BaseDelay.Milliseconds())
	jitter := rand.NormFloat64() * q.cfg.JitterSigma // sigma 0.25 → ±25%
	d := base * (1 + jitter)

	d += float64(q.cfg.PerCharMS * textLen)
	if d < 400 {
		d = 400
	}
	if max := float64(q.cfg.MaxDelay.Milliseconds()); d > max {
		d = max
	}
	return time.Duration(d) * time.Millisecond
}

// warmBudget: cuántos mensajes se permite hoy. 20 el día 1 → 500 a los 7 días.
func (q *Queue) warmBudget(today string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.warmupDay == "" {
		q.warmupDay = today
	}
	day := time.Now().UTC().Unix() / 86400
	start, err := time.Parse("2006-01-02", q.warmupDay)
	if err != nil {
		q.warmupDay = today
		return q.cfg.WarmupStart
	}
	days := float64(day - start.Unix()/86400)
	if days < 0 {
		days = 0
	}
	if days >= 7 {
		return q.cfg.WarmupEnd
	}
	step := float64(q.cfg.WarmupEnd-q.cfg.WarmupStart) / 7
	return q.cfg.WarmupStart + int(math.Round(days*step))
}

// Wait bloquea lo justo para respetar el ritmo. priority salta el warm-up
// (pero nunca el delay: si no, el .menu sería un pico de tráfico).
func (q *Queue) Wait(ctx context.Context, textLen int, priority bool) error {
	q.mu.Lock()
	today := time.Now().Format("2006-01-02")
	if q.day != today {
		q.day = today
		q.sentToday = 0
	}
	if !priority && q.sentToday >= q.warmBudget(today) {
		q.mu.Unlock()
		// Fuera del presupuesto diario: esperar no sirve, pero tampoco se
		// ignora el límite. Se deja pasar solo lo prioritario.
		return ErrWarmupLimit
	}
	q.mu.Unlock()

	d := q.delay(textLen)
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	q.mu.Lock()
	q.lastSent = time.Now()
	q.sentToday++
	q.mu.Unlock()
	return nil
}

// ErrWarmupLimit se devuelve cuando el número está en calentamiento y se
// agotó el cupo del día. Quien decide qué hacer es el router (avisar o
// responder en modo lento), no la cola.
var ErrWarmupLimit = errWarmup("warm-up: cupo diario alcanzado")

type errWarmup string

func (e errWarmup) Error() string { return string(e) }
