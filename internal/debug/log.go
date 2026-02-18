package debug

import (
	"fmt"
	"os"
	"time"
)

var logFile *os.File

func Init(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	logFile = f
}

func Log(format string, args ...any) {
	if logFile == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
	logFile.Sync()
}
