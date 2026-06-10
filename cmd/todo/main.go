package main

import (
	"os"

	"github.com/feint/todo/internal/todo"
)

var version = "0.1.0"

func main() {
	os.Exit(todo.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
