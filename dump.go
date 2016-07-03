package dev

import (
	"fmt"
	"io"
	"os"
	r "reflect"
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
	_, fileName, fileLine, _ := runtime.Caller(1)

	if Config.Location {
		writeCF("%s:%d", Config.Color.Location, fileName, fileLine)
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
	var refVal r.Value
	valType := r.TypeOf(val)

	// if value is already a reflection (in case of recursive call)
	// then cast itself and redefine the type
	// otherwise - fetch a reflection
	if valType.String() == "reflect.Value" {
		refVal = val.(r.Value)
		valType = refVal.Type()
	} else {
		refVal = r.ValueOf(val)
	}
	kind := valType.Kind()

	// if the value is a pointer, then dereference it
	// and mark result with ampersand
	if kind == r.Ptr {
		refVal = r.Indirect(refVal)
		kind = refVal.Kind()
		writeCF("%s", Config.Color.Punctuation, "&")
	}

	// Tabulations
	startTab := strings.Repeat(Config.Tab, level)
	endTab := strings.Repeat(Config.Tab, level-1)

	// Print number type if needed
	if Config.NumTypes && isNumber(kind) {
		writeType(refVal)
	}

	// Fetch real value based on its type and write it with a corresponding format
	switch kind {

	case r.String:
		writeCF("\"%s\"", Config.Color.String, refVal.String())

	case r.Int, r.Int8, r.Int16, r.Int32, r.Int64:
		writeCF("%d", Config.Color.Number, refVal.Int())

	case r.Uint, r.Uint8, r.Uint16, r.Uint32, r.Uint64:
		writeCF("%d", Config.Color.Number, refVal.Uint())

	case r.Float32, r.Float64:
		writeCF("%v", Config.Color.Number, refVal.Float())

	case r.Bool:
		writeCF("%v", Config.Color.Bool, refVal.Bool())

	case r.Array, r.Slice:
		writeType(refVal)

		// Just show empty brace for empty value
		len := refVal.Len()
		if 0 == len {
			writeCF("%s", Config.Color.Braces, "[]")
			break
		}

		// Open brace
		writeCF("%s", Config.Color.Braces, "[\n"+startTab)

		// Print nested values
		for i := 0; i < len; i++ {
			dumpValue(refVal.Index(i), level+1)

			end := ",\n"
			if i < len-1 {
				end += startTab
			}
			writeCF("%s", Config.Color.Punctuation, end)
		}

		// Close brace
		writeCF("%s", Config.Color.Braces, endTab+"]")

	case r.Map:
		writeType(refVal)

		// Just show empty brace for empty value
		if 0 == len(refVal.MapKeys()) {
			writeCF("%s", Config.Color.Braces, "{}")
			break
		}

		// Open brace
		writeCF("%s", Config.Color.Braces, "{\n"+startTab)

		// Print nested key:value's
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

		// Close brace
		writeCF("%s", Config.Color.Braces, endTab+"}")

	case r.Struct:
		writeCF("%s", Config.Color.Type, pureType(valType.String()))

		// Just show empty brace for empty value
		if 0 == refVal.NumField() {
			writeCF("%s", Config.Color.Braces, "{}")
			break
		}

		// Open brance
		writeCF("%s", Config.Color.Braces, "{\n"+startTab)

		// Print nested fieldName:Value's
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

		// Close brace
		writeCF("%s", Config.Color.Braces, endTab+"}")

	case r.Func:
		writeCF("%s", Config.Color.Func, valType.String())

	case r.Interface:
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

func writeType(v r.Value) {
	writeCF("%s", Config.Color.Type, "<"+pureType(v.Type().String())+">")
}

var numbers = [12]r.Kind{
	r.Int, r.Int8, r.Int16, r.Int32, r.Int64,
	r.Uint, r.Uint8, r.Uint16, r.Uint32, r.Uint64,
	r.Float32, r.Float64,
}

func isNumber(kind r.Kind) bool {
	for i := 0; i < len(numbers); i++ {
		if kind == numbers[i] {
			return true
		}
	}
	return false
}
