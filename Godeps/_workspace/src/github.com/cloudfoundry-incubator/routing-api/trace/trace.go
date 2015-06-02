package trace

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
)

const RTR_TRACE = "RTR_TRACE"

type Printer interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
}

type nullLogger struct{}

func (*nullLogger) Print(v ...interface{})                 {}
func (*nullLogger) Printf(format string, v ...interface{}) {}
func (*nullLogger) Println(v ...interface{})               {}

var stdOut io.Writer = os.Stdout
var Logger Printer

func init() {
	Logger = NewLogger("")
}

func SetStdout(s io.Writer) {
	stdOut = s
}

func NewLogger(rtr_trace string) Printer {

	if rtr_trace == "true" {
		Logger = newStdoutLogger()
	} else {
		Logger = new(nullLogger)
	}

	return Logger
}

func newStdoutLogger() Printer {
	return log.New(stdOut, "", 0)
}

func Sanitize(input string) (sanitized string) {
	var sanitizeJson = func(propertyName string, json string) string {
		regex := regexp.MustCompile(fmt.Sprintf(`"%s":\s*"[^"]*"`, propertyName))
		return regex.ReplaceAllString(json, fmt.Sprintf(`"%s":"%s"`, propertyName, PRIVATE_DATA_PLACEHOLDER()))
	}

	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized = re.ReplaceAllString(input, "Authorization: "+PRIVATE_DATA_PLACEHOLDER())
	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+PRIVATE_DATA_PLACEHOLDER()+"&")

	sanitized = sanitizeJson("access_token", sanitized)
	sanitized = sanitizeJson("refresh_token", sanitized)
	sanitized = sanitizeJson("token", sanitized)
	sanitized = sanitizeJson("password", sanitized)
	sanitized = sanitizeJson("oldPassword", sanitized)

	return
}

func PRIVATE_DATA_PLACEHOLDER() string {
	return "[PRIVATE DATA HIDDEN]"
}
