package transfer

import (
	"log"
	"os"
)

var (
	Logger *log.Logger
)

func init() {
	Logger = log.New(os.Stderr, "trans", log.LstdFlags|log.Lshortfile)
}

func SetLogger(l *log.Logger) {
	if l != nil {
		Logger = l
	}
}
