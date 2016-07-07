# Developer Tools Package

[![build](https://travis-ci.org/jumale-go/dev.svg)](https://travis-ci.org/jumale-go/dev)

Installation `go get github.com/jumale-go/dev`

## Dump Tool
It's used to print well-formatted and colorized variables
#### Example:
```go
package main

import "github.com/jumale-go/dev"

type Task struct {
	open bool
}

func main() {
	value := map[string]interface{}{
		"foo": "test",
		"bar": 2.25,
		"baz": [1]*Task{
			&Task{true},
		},
		"func": func (a string, b int) (bool, error) {
			return true, nil
		},
	}

	dev.Dump(value)
}
```
the result should be<br>
![Dump Result](doc/print_result.png)

#### Configuration
By default the result is printed out to os.Stdout, but you can change the destination by
```go
dev.Config.Writer = ... // io.Writer interface
```
By default each number is printed out with its own type as a prefix. You can disable the types by
```go
dev.Config.NumTypes = false
```
By default the Dump function also prints a location (file:line) where it's called. To disable location
```go
dev.Config.Location = false
```
By default tabulation in nested trees is two spaces. You can change it by
```go
dev.Config.Tab = "+" // now one tab will be the "+" character
```
Also you can change colors:
```go
// note: there is a list of color-constants, like dev.FgWhite, dev.FgGreen etc
dev.Config.Color.String = dev.FgGreen // strings
dev.Config.Color.Number = ...         // all ints, uints and floats
dev.Config.Color.Bool = ...           // booleans
dev.Config.Color.Punctuation = ...    // colons, commas, ampersands
dev.Config.Color.Braces = ...         // round- curly- and square-braces
dev.Config.Color.Type = ...           // type prefixes before structs, arrays, slices, maps and numbers
dev.Config.Color.Func = ...           // function types
```
#### Formatters
If you want implement a custom output for some specific type, you can set a formatter for it.</br>
Formatter is the function of type `func (val interface{}) string` - it receives a value of a specific type
and returns a string, which represents this value.</br>
Inside of the function you need to cast the value to its own type and then you can use it to create the result</br>
**Note:** the formatter will be called for both pointers and non-pointers, so that you need to define to which
type you need to cast - to the value's type or to pointer of this type.</br>
Also note that `nil` values do not go to formatters

Example:
```go
// Let's set a formatter for files, which returns their names
Config.Formatters["os.File"] = func(val interface{}) string {
    // we expect to work with pointers, because method "Name" expects receiver as a pointer
    var item *os.File
    // trying to cast the value as non-pointer type
    if casted, ok := val.(os.File); ok {
        item = &casted // if it's casted without errors, then get a pointer of the result
    } else { // otherwise cast it to pointer
        item = val.(*os.File)
    }
    return item.Name() // now you can call the method on the pointer value
}
```
