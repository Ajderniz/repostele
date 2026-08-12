package root

import "embed"

const (
	ROOT   = "web"
  HTMDIR = ROOT+"/html"
)

//go:embed web/*
var FS embed.FS