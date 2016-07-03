# Development Tools Package

Installation `go get github.com/jumale-go/dev`

## Dump Tool
It's used to print well-formatted and colorized variables
Example:
```go
import "github.com/jumale-go/dev"

type Task struct {
    open bool
}

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
```
