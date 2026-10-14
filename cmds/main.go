// Shin-Go — Copyright (C) 2026 riokuroxi-svg · AGPL-3.0-only
//
// main.go (paquete cmds) — Aquí viven los comandos. Cada archivo llama a
// bot.Register() en su init(). Para migrar un comando del JS basta con:
//  1. leer el handler viejo (cmds/<categoria>/<comando>.js)
//  2. portar la lógica (casi siempre: un fetch y un formateo)
//  3. copiar la misma tabla de metadatos (nombre, alias, categoría)
package cmds
