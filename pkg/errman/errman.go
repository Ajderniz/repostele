package errman

import (
	"log"
	"os"
)

func CheckFatal(err error) {
	if err != nil { log.Fatal("FATAL: " + err.Error()) }
}

var _ErrLogger = log.New(os.Stderr, "\033[31mERROR\033[0m: ", log.LstdFlags | log.Lmsgprefix)

func PrintError(err error) {
	_ErrLogger.Println(err.Error())
}
