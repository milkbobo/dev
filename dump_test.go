package dev

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Mock writer as the struct which stores data to a local property
var wr = TestWriter{}

type TestWriter struct{ result string }

// implement io.Writer
func (s TestWriter) Write(p []byte) (n int, err error) {
	// works as closure because we can not ask for pointer inside of this method
	wr.result += string(p)
	return 0, nil
}

// Fixtures
type Foo struct {
	bar string
	bat int
}
type Bar struct {
}
type Lorem struct {
	foos []Foo
}

// Define list of cases for Dump function
// The number in braces like (1) - it's the number of color in the colors list (see the end of this file)
// this number will be replaced with proper color code
var DumpCases = []struct {
	input    interface{}
	expected string
}{
	{
		"test",
		"(0)\"test\"\n",
	},
	{
		1,
		"(1)1\n",
	},
	{
		uint64(25),
		"(1)25\n",
	},
	{
		7.123,
		"(1)7.123\n",
	},
	{
		false,
		"(2)false\n",
	},
	{
		&[2]int{7},
		`
(5)<*[2]int>(4)[
+(1)7(3),
+(1)0(3),
(4)]
`,
	},
	{
		[]string{"foo", "bar"},
		`
(5)<[]string>(4)[
+(0)"foo"(3),
+(0)"bar"(3),
(4)]
`,
	},
	{
		[]int{},
		`
(5)<[]int>(4)[]
`,
	},
	{
		map[string]int{"a": 1, "b": 2},
		`
(5)<map[string]int>(4){
+(0)"a"(3): (1)1(3),
+(0)"b"(3): (1)2(3),
(4)}
`,
	},
	{
		map[string]int{},
		`
(5)<map[string]int>(4){}
`,
	},
	{
		Foo{bar: "baz"},
		`
(5)Foo(4){
+(0)"bar"(3): (0)"baz"(3),
+(0)"bat"(3): (1)0(3),
(4)}
`,
	},
	{
		&Bar{},
		`
(3)&(5)Bar(4){}
`,
	},
	{
		Lorem{[]Foo{Foo{bar: "baz"}}},
		`
(5)Lorem(4){
+(0)"foos"(3): (5)<[]Foo>(4)[
++(5)Foo(4){
+++(0)"bar"(3): (0)"baz"(3),
+++(0)"bat"(3): (1)0(3),
(4)++}(3),
(4)+](3),
(4)}
`,
	},
	{
		func(val int) string { return "" },
		`
(6)func(int) string
`,
	},
	{
		map[string]interface{}{"a": [1]int{}, "b": "test"},
		`
(5)<map[string]interface {}>(4){
+(0)"a"(3): (5)<[1]int>(4)[
++(1)0(3),
(4)+](3),
+(0)"b"(3): (0)"test"(3),
(4)}
`,
	},
}

func beforeTest() {
	wr.result = ""
	Config.Writer = wr
	Config.Tab = "+"
	Config.NumTypes = false
}

func TestDump(t *testing.T) {
	beforeTest()

	for _, cs := range DumpCases {
		var err string
		Dump(cs.input)

		match, _ := regexp.MatchString(`/(\w+/)+\w+\.go:\d+`, wr.result)
		if !match {
			err = "Result does not contain file and line information"
		}

		expected := colorize(cs.expected)
		if !strings.Contains(wr.result, expected) {
			err = fmt.Sprintf("%v is not printed as %s", cs.input, expected)
		}

		if err != "" {
			t.Error(fmt.Sprintf("%s\nResult:\n%s", err, wr.result))
		}
	}
}

type Holder struct {
	Foo Formatted
	Bar *Formatted
	Baz Formatted  // tested as nil
	Bat *Formatted // tested as nil
}
type Formatted struct {
	name string
}

func (m *Formatted) Name() string {
	return m.name
}

func TestFormatters(t *testing.T) {
	beforeTest()

	Config.Formatters["dev.Formatted"] = func(val interface{}) string {
		var item *Formatted
		if casted, ok := val.(Formatted); ok {
			item = &casted
		} else {
			item = val.(*Formatted)
		}
		return item.Name()
	}

	input := Holder{
		Foo: Formatted{"foo"},
		Bar: &Formatted{"bar"},
	}
	Dump(input)

	var err string
	expected := colorize(`
(5)Holder(4){
+(0)"Foo"(3): (5)<Formatted>(0)"foo"(3),
+(0)"Bar"(3): (5)<*Formatted>(0)"bar"(3),
+(0)"Baz"(3): (5)<Formatted>(0)""(3),
+(0)"Bat"(3): (5)<*Formatted>(5)<nil>(3),
(4)}
`)

	if !strings.Contains(wr.result, expected) {
		err = fmt.Sprintf("%v is not printed as %s", input, expected)
	}

	if err != "" {
		err = fmt.Sprintf("%s\nResult:\n%s", err, wr.result)
		ioutil.WriteFile("log/failed.log", []byte(err), os.ModePerm)
		t.Error(err)
	}
}

func benchmarkDump(val interface{}, b *testing.B) {
	Config.Writer = wr
	for i := 0; i < b.N; i++ {
		Dump(val)
	}
}

func BenchmarkInteger(b *testing.B) { benchmarkDump(1, b) }
func BenchmarkString(b *testing.B)  { benchmarkDump("Some test string", b) }
func BenchmarkArray(b *testing.B)   { benchmarkDump([10]int{777, 12, 45}, b) }

// Colorize string
// Replaces color numbers with their real codes
// Also adds a default color code at the end and trims spaces for the resulting string
// Implemented to simplify expected values
// For example: (2) will be replaced as \033[%dm
// where %d is colors[2] value (which is Config.Color.Bool in this case)
var cnf = Config.Color
var colors = [7]uint{
	cnf.String,
	cnf.Number,
	cnf.Bool,
	cnf.Punctuation,
	cnf.Braces,
	cnf.Type,
	cnf.Func,
}

func colorize(str string) string {
	for i := 0; i < len(colors); i++ {
		id := fmt.Sprintf("(%d)", i)
		code := fmt.Sprintf("\033[%dm", colors[i])
		str = strings.Replace(str, id, code, -1)
	}
	str += fmt.Sprintf("\033[%dm", FgDefault)

	return strings.TrimSpace(str)
}
