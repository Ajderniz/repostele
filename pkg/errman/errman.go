package errman

import (
	"log"
	"os"
)

func CheckFatal(err error) {
	if err != nil { log.Fatal("FATAL: " + err.Error()) }
}

var _ErrLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags | log.Lmsgprefix)

func PrintError(err error) {
	_ErrLogger.Println(err.Error())
}
