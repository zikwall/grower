package log

import (
	"fmt"
	"log"
)

const (
	Reset  Color = "\033[0m"
	Green  Color = "\033[32m"
	Yellow Color = "\033[33m"
	Cyan   Color = "\033[36m"
)

type Color string

func Colored(content string, color Color) string {
	return string(color) + content + string(Reset)
}

func Info(message interface{}) {
	log.Printf("%s: %v \n", Colored("[GROWER INFO]", Cyan), message)
}

func Infof(message string, v ...interface{}) {
	Info(fmt.Sprintf(message, v...))
}

func Warning(message interface{}) {
	log.Printf("%s: %v \n", Colored("[GROWER WARNING]", Yellow), message)
}

func Warningf(message string, v ...interface{}) {
	Warning(fmt.Sprintf(message, v...))
}
