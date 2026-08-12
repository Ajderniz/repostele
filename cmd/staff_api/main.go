package main

import (
	"log"

	"github.com/ajderniz/repostele/internal/maininit"
)

func main() {
	err := maininit.InitMain("Repostele staff API", maininit.SERVER_BINARY_STAFF)
	if err != nil { log.Fatal("FATAL: "+err.Error()) }
}