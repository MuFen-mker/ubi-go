package utils

import (
	"alexanderthegreat96/ubi-go/internal/constants"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var consoleLogger *log.Logger

// console logger initialization
func InitLogger() {
	consoleLogger = log.New(os.Stdout, "", 0)
}

// print message to console
func LogMessage(message, level string) {
	color := getColorForLevel(level)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("%s[Schedulr][%s][%s]%s - ", color, levelUpper(level), timestamp, constants.ColorReset)

	consoleLogger.SetPrefix(header)
	consoleLogger.Println(message)
}

func getColorForLevel(level string) string {
	switch level {
	case "info":
		return constants.ColorGreen
	case "success":
		return constants.ColorOpenGreen
	case "warn", "warning":
		return constants.ColorYellow
	case "error":
		return constants.ColorRed
	case "debug":
		return constants.ColorBlue
	default:
		return constants.ColorReset
	}
}

func levelUpper(level string) string {
	switch level {
	case "info":
		return "INFO"
	case "success":
		return "SUCCESS"
	case "warn":
		return "WARN"
	case "warning":
		return "WARNING"
	case "error":
		return "ERROR"
	case "debug":
		return "DEBUG"
	default:
		return strings.ToUpper(level)
	}
}
