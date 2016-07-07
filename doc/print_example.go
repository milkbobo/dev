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
		"func": func(a string, b int) (bool, error) {
			return true, nil
		},
	}

	dev.Dump(value)
}
