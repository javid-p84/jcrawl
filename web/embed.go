// Package web embeds the static single-page UI so it ships inside the
// compiled binary — no separate static file directory to deploy alongside it.
package web

import "embed"

//go:embed index.html
var Files embed.FS
