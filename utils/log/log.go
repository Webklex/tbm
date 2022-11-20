package log

import (
	"fmt"
	"github.com/fatih/color"
	"os"
)

const (
	LogAll       = 0
	LogSuccess   = 1
	LogWarning   = 2
	LogStatistic = 3
	LogError     = 4
)

var Mode = LogAll

//
// Info
// @Description: Print an info line
// @param format string
// @param args ...interface{}
func Info(format string, args ...interface{}) {
	if Mode <= LogAll {
		Log(color.FgCyan, "info", format, args...)
	}
}

//
// Error
// @Description: Print an error or message as error
// @param format interface{}
// @param args ...interface{}
func Error(format interface{}, args ...interface{}) {
	if Mode <= LogAll || Mode == LogError {
		switch format.(type) {
		case error:
			Log(color.FgRed, "error", format.(error).Error(), args...)
		case string:
			Log(color.FgRed, "error", format.(string), args...)
		default:
			Log(color.FgRed, "error", fmt.Sprintf("%v", format), args...)
		}
	}
}

//
// Fatal
// @Description: Print a fatal error
// @param format interface{}
// @param args ...interface{}
func Fatal(format interface{}, args ...interface{}) {
	Error(format, args...)
	os.Exit(2)
}

//
// Warning
// @Description: Print a warning line
// @param format string
// @param args ...interface{}
func Warning(format string, args ...interface{}) {
	if Mode <= LogAll || Mode == LogWarning {
		Log(color.FgYellow, "warning", format, args...)
	}
}

//
// Success
// @Description: Print a success line
// @param format string
// @param args ...interface{}
func Success(format string, args ...interface{}) {
	if Mode <= LogAll || Mode == LogSuccess {
		Log(color.FgGreen, "success", format, args...)
	}
}

func Statistic(format string, args ...interface{}) {
	if Mode <= LogAll || Mode == LogStatistic {
		Log(color.FgMagenta, "statistic", format, args...)
	}
}

//
// Log
// @Description: Print a styled log line
// @param c color.Attribute
// @param kind string
// @param format string
// @param args ...interface{}
func Log(c color.Attribute, kind, format string, args ...interface{}) {
	Printf(c, 18, fmt.Sprintf("[%s]", kind), format, args...)
}

func Printf(c color.Attribute, space int, kind, format string, args ...interface{}) {
	kind = color.New(c).Sprintf("%s", kind)
	fmt.Printf(fmt.Sprintf("%%-%ds %%s\n", space), kind, fmt.Sprintf(format, args...))
}

func BoolString(b bool) string {
	c := color.FgRed
	if b {
		c = color.FgGreen
	}
	return color.New(c).Sprintf("%t", b)
}
