package dev

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

const (
	FgDefault   = 39
	FgBlack     = 30
	FgRed       = 31
	FgGreen     = 32
	FgYellow    = 33
	FgBlue      = 34
	FgMagenta   = 35
	FgCyan      = 36
	FgGray      = 37
	FgLoGray    = 90
	FgHiRed     = 91
	FgHiGreen   = 92
	FgHiYellow  = 93
	FgHiBlue    = 94
	FgHiMagenta = 95
	FgHiCyan    = 96
	FgWhite     = 97
)

/*
Main configuration:
	Color - specifies corresponding colors for printed parts
		String      - strings
		Number      - all integers and floats
		Bool        - booleans
		Punctuation - colons, commas, ampersands
		Braces      - round- curly- and square-braces
		Type        - type prefixes before struct items and numbers (if this option is enabled)
		Func        - function types
	Tab - one tabulation character(s)
	NumTypes - if true, then each number is prefixed with it's own type
	Location - if true, then it prints file name and line number, where the Dump function is called
*/
var Config config = config{
	Color: colorConfig{
		Location:    FgCyan,
		String:      FgGreen,
		Number:      FgBlue,
		Bool:        FgRed,
		Punctuation: FgYellow,
		Braces:      FgGray,
		Type:        FgWhite,
		Func:        FgMagenta,
	},
	Tab:      "  ",
	NumTypes: true,
	Location: true,
}

type colorConfig struct {
	Location    uint
	String      uint
	Number      uint
	Bool        uint
	Punctuation uint
	Braces      uint
	Type        uint
	Func        uint
}

type config struct {
	Color    colorConfig
	Tab      string
	NumTypes bool
	Location bool
}

var writer io.Writer = os.Stdout

// Prints well-formatted and colorized variables
func Dump(args ...interface{}) {
	_, _, fileLine, _ := runtime.Caller(1)

	if Config.Location {
		writeCF("%s:%d", Config.Color.Location, "/path/to/target/file.go", fileLine)
		write("\n")
	}

	for _, arg := range args {
		dumpValue(arg, 1)
		write("\n")
	}
	writeF("\033[%dm", FgDefault)
}

// By default, result goes to os.Stdout,
// but you can change the destination by this setter
func SetWriter(w io.Writer) {
	writer = w
}

// Dump a single value
func dumpValue(val interface{}, level int) {
	var refVal reflect.Value
	valType := reflect.TypeOf(val)

	// if value is already a reflection (in case of recursive call)
	// then cast itself and redefine the type
	// otherwise - fetch a reflection
	if valType.String() == "reflect.Value" {
		refVal = val.(reflect.Value)
		valType = refVal.Type()
	} else {
		refVal = reflect.ValueOf(val)
	}
	kind := valType.Kind()

	// if the value is a pointer, then dereference it
	// and mark result with ampersand
	if kind == reflect.Ptr {
		refVal = reflect.Indirect(refVal)
		kind = refVal.Kind()
		writeCF("%s", Config.Color.Punctuation, "&")
	}

	// Tabulations
	startTab := strings.Repeat(Config.Tab, level)
	endTab := strings.Repeat(Config.Tab, level-1)

	// Fetch real value based on its type and write it with a corresponding format
	switch kind {

	case reflect.String:
		writeCF("\"%s\"", Config.Color.String, refVal.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if Config.NumTypes {
			writeType(refVal)
		}
		writeCF("%d", Config.Color.Number, refVal.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if Config.NumTypes {
			writeType(refVal)
		}
		writeCF("%d", Config.Color.Number, refVal.Uint())

	case reflect.Float32, reflect.Float64:
		if Config.NumTypes {
			writeType(refVal)
		}
		writeCF("%v", Config.Color.Number, refVal.Float())

	case reflect.Bool:
		writeCF("%v", Config.Color.Bool, refVal.Bool())

	case reflect.Array, reflect.Slice:
		writeType(refVal)
		len := refVal.Len()

		if 0 == len {
			writeCF("%s", Config.Color.Braces, "[]")
			break
		}

		writeCF("%s", Config.Color.Braces, "[\n"+startTab)

		for i := 0; i < len; i++ {
			dumpValue(refVal.Index(i), level+1)

			end := ",\n"
			if i < len-1 {
				end += startTab
			}
			writeCF("%s", Config.Color.Punctuation, end)
		}

		writeCF("%s", Config.Color.Braces, endTab+"]")

	case reflect.Map:
		writeType(refVal)
		if 0 == len(refVal.MapKeys()) {
			writeCF("%s", Config.Color.Braces, "{}")
			break
		}

		writeCF("%s", Config.Color.Braces, "{\n"+startTab)

		len := refVal.Len()
		for _, key := range refVal.MapKeys() {
			dumpValue(key, level+1)
			writeCF("%s", Config.Color.Punctuation, ": ")
			dumpValue(refVal.MapIndex(key), level+1)

			end := ",\n"
			if len--; len > 0 {
				end += startTab
			}
			writeCF("%s", Config.Color.Punctuation, end)
		}

		writeCF("%s", Config.Color.Braces, endTab+"}")

	case reflect.Struct:
		writeCF("%s", Config.Color.Type, pureType(valType.String()))
		if 0 == refVal.NumField() {
			writeCF("%s", Config.Color.Braces, "{}")
			break
		}

		writeCF("%s", Config.Color.Braces, "{\n"+startTab)

		for i := 0; i < refVal.NumField(); i++ {
			dumpValue(refVal.Type().Field(i).Name, level+1)
			writeCF("%s", Config.Color.Punctuation, ": ")
			dumpValue(refVal.Field(i), level+1)

			end := ",\n"
			if i < refVal.NumField()-1 {
				end += startTab
			}
			writeCF("%s", Config.Color.Punctuation, end)
		}

		writeCF("%s", Config.Color.Braces, endTab+"}")

	case reflect.Func:
		writeCF("%s", Config.Color.Func, valType.String())

	case reflect.Interface:
		dumpValue(refVal.Elem(), level)

	default:
		writeCF("%v", FgDefault, refVal)
	}

}

// Leaves only last word in the type name
// E.g. "package_name.Foo" => "Foo"
// or "* Bar" => "Bar"
func pureType(typeName string) string {
	reg, _ := regexp.Compile(`\*?(\w+\.)+`)
	return reg.ReplaceAllString(typeName, "")
}

func write(value string) {
	writer.Write([]byte(value))
}

func writeF(format string, val ...interface{}) {
	write(fmt.Sprintf(format, val...))
}

func writeCF(format string, color uint, val ...interface{}) {
	str := fmt.Sprintf(format, val...)
	writeF("\033[%dm%s", color, str)
}

func writeType(v reflect.Value) {
	writeCF("%s", Config.Color.Type, "<"+pureType(v.Type().String())+">")
}
