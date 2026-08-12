package web

import "embed"

const (
  HTMDIR = "html"
  HXDIR  = "htmx"
  PUBLIC = "public"
)

//go:embed *
var FS embed.FS