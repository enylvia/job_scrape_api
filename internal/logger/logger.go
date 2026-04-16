package logger

import (
	"log"
	"os"
)

func New() *log.Logger {
	return log.New(os.Stdout, "[job-aggregator] ", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
}
