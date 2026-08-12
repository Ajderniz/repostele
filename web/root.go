package root

import "embed"

const (
  HTMDIR = "public/html"
  PUBLIC = "public"
)

//go:embed *
var FS embed.FS