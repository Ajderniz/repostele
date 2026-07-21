package main

import (
	"github.com/ajderniz/repostele/internal/common"
	"github.com/ajderniz/repostele/pkg/errman"
)

func main() {
	errman.CheckFatal(common.InitMain("Repostele user API", common.SERVER_BINARY_USER))
}