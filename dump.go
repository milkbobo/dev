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
	Writer     - destination writer, interface of io.Writer, by default os.Stdout
	Formatters - list of functions defined for specific types needed some custom format of output
	Tab        - one tabulation character(s)
	NumTypes   - if true, then each number is prefixed with it's own type
	Location   - if true, then it prints file name and line number, where the Dump function is called
	Color      - specifies corresponding colors for printed parts
		String      - strings
		Number      - all integers and floats
		Bool        - booleans
		Punctuation - colons, commas, ampersands
		Braces      - round- curly- and square-braces
		Type        - type prefixes before struct items and numbers (if this option is enabled)
		Func        - function types
*/
var Config config = config{
	Writer:     os.Stdout,
	Formatters: map[string]Formatter{},
	Tab:        "  ",
	NumTypes:   true,
	Location:   true,
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
}

type config struct {
	Color      colorConfig
	Tab        string
	NumTypes   bool
	Location   bool
	Writer     io.Writer
	Formatters map[string]Formatter
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

type Formatter func(val interface{}) string
type Any interface{}

// Prints well-formatted and colorized variables
func Dump(args ...Any) {
	_, fileName, fileLine, _ := runtime.Caller(1)
	fileName = strings.Replace(fileName, os.Getenv("GOPATH"), "", 1)

	if Config.Location {
		writeColor("%s:%d", Config.Color.Location, fileName, fileLine)
		write("\n")
	}

	for _, arg := range args {
		dumpValue(arg, 1)
		write("\n")
	}
	writeFormat("\033[%dm", FgDefault)
}

// Dump a single value
func dumpValue(val Any, level int) {
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
	typeStr := valType.String()
	isPointer := kind == r.Ptr

	// Print nil value
	switch kind {
	case r.Chan, r.Func, r.Interface, r.Map, r.Ptr, r.Slice:
		if refVal.IsNil() {
			writeType(refVal, isPointer)
			writeColor("%s", Config.Color.Type, "<nil>")
			return
		}
	}

	// Use formatter if is set for this type
	if formatter, ok := Config.Formatters[typeName(typeStr)]; ok {
		writeType(refVal, isPointer)
		dumpValue(formatter(refVal.Interface()), level)
		return
	}

	// if the value is a pointer, then dereference it
	// and mark result with ampersand
	if isPointer {
		refVal = r.Indirect(refVal)
		kind = refVal.Kind()
		writeColor("%s", Config.Color.Punctuation, "&")
	}

	// Calculate tabulations
	startTab := strings.Repeat(Config.Tab, level)
	endTab := strings.Repeat(Config.Tab, level-1)

	// Print number type if needed
	if Config.NumTypes && isNumber(kind) {
		writeType(refVal, isPointer)
	}

	// Fetch real value based on its type and write it with a corresponding format
	switch kind {

	case r.String:
		writeColor("\"%s\"", Config.Color.String, refVal.String())

	case r.Int, r.Int8, r.Int16, r.Int32, r.Int64:
		writeColor("%d", Config.Color.Number, refVal.Int())

	case r.Uint, r.Uint8, r.Uint16, r.Uint32, r.Uint64:
		writeColor("%d", Config.Color.Number, refVal.Uint())

	case r.Float32, r.Float64:
		writeColor("%v", Config.Color.Number, refVal.Float())

	case r.Bool:
		writeColor("%v", Config.Color.Bool, refVal.Bool())

	case r.Array, r.Slice:
		writeType(refVal, isPointer)

		// Just show empty brace for empty value
		len := refVal.Len()
		if 0 == len {
			writeColor("%s", Config.Color.Braces, "[]")
			break
		}

		// Open brace
		writeColor("%s", Config.Color.Braces, "[\n"+startTab)

		// Print nested values
		for i := 0; i < len; i++ {
			dumpValue(refVal.Index(i), level+1)

			end := ",\n"
			if i < len-1 {
				end += startTab
			}
			writeColor("%s", Config.Color.Punctuation, end)
		}

		// Close brace
		writeColor("%s", Config.Color.Braces, endTab+"]")

	case r.Map:
		writeType(refVal, isPointer)

		// Just show empty brace for empty value
		if 0 == len(refVal.MapKeys()) {
			writeColor("%s", Config.Color.Braces, "{}")
			break
		}

		// Open brace
		writeColor("%s", Config.Color.Braces, "{\n"+startTab)

		// Print nested key:value's
		len := refVal.Len()
		for _, key := range refVal.MapKeys() {
			dumpValue(key, level+1)
			writeColor("%s", Config.Color.Punctuation, ": ")
			dumpValue(refVal.MapIndex(key), level+1)

			end := ",\n"
			if len--; len > 0 {
				end += startTab
			}
			writeColor("%s", Config.Color.Punctuation, end)
		}

		// Close brace
		writeColor("%s", Config.Color.Braces, endTab+"}")

	case r.Struct:
		writeColor("%s", Config.Color.Type, pureType(typeStr))

		// Just show empty brace for empty value
		if 0 == refVal.NumField() {
			writeColor("%s", Config.Color.Braces, "{}")
			break
		}

		// Open brace
		writeColor("%s", Config.Color.Braces, "{\n"+startTab)

		// Print nested fieldName:Value's
		for i := 0; i < refVal.NumField(); i++ {
			dumpValue(refVal.Type().Field(i).Name, level+1)
			writeColor("%s", Config.Color.Punctuation, ": ")
			dumpValue(refVal.Field(i), level+1)

			end := ",\n"
			if i < refVal.NumField()-1 {
				end += startTab
			}
			writeColor("%s", Config.Color.Punctuation, end)
		}

		// Close brace
		writeColor("%s", Config.Color.Braces, endTab+"}")

	case r.Func:
		writeColor("%s", Config.Color.Func, typeStr)

	case r.Interface:
		dumpValue(refVal.Elem(), level)

	default:
		writeColor("%v", FgDefault, refVal)
	}

}

// Removes both namespaces and pointers
func pureType(typeString string) string {
	return typeName(withoutNamespace(typeString))
}

// Removes pointer mark in its own type
// It will be replaced later with ampersand
// Does not remove pointers in sub-types
func typeName(typeString string) string {
	reg, _ := regexp.Compile(`^\*`)
	return reg.ReplaceAllString(typeString, "")
}

// Removes namespaces and returns pure name
func withoutNamespace(typeString string) string {
	reg, _ := regexp.Compile(`(\w+\.)+`)
	return reg.ReplaceAllString(typeString, "")
}

func write(value string) {
	Config.Writer.Write([]byte(value))
}

func writeFormat(format string, val ...Any) {
	write(fmt.Sprintf(format, toInterface(val)...))
}

func writeColor(format string, color uint, val ...Any) {
	str := fmt.Sprintf(format, toInterface(val)...)
	writeFormat("\033[%dm%s", color, str)
}

func writeType(v r.Value, isPointer bool) {
	ptr := ""
	if isPointer {
		ptr = "*"
	}
	writeColor("%s", Config.Color.Type, "<"+ptr+pureType(v.Type().String())+">")
}

// Converts []Any to []interface{}
func toInterface(val []Any) []interface{} {
	result := make([]interface{}, len(val))

	for i := 0; i < len(val); i++ {
		result[i] = val[i]
	}
	return result
}

// Checks if the value is any kind of number
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
