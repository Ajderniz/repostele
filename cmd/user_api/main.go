package main

import (
	"log"

	"github.com/ajderniz/repostele/internal/common"
)

func main() {
	err := common.InitMain("Repostele staff API", common.SERVER_BINARY_USER)
	if err != nil { log.Fatal("FATAL: "+err.Error()) }
}