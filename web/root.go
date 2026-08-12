package root

import "embed"

const (
  HTMDIR = "html"
  PUBLIC = "public"
)

//go:embed *
var FS embed.FS