// Package static embeds the templates and public files.
package static

import "embed"

const (
  HTMDIR = "html"
  HXDIR  = "htmx"
  PUBLIC = "public"
  DYNDIR = "dyn"
  IMGDIR = DYNDIR+"/img"
)

//go:embed *
var FS embed.FS
