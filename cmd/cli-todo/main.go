package main

import (
	"os"

	"github.com/feint/cli-todo/internal/todo"
)

var version = "0.2.0"

func main() {
	os.Exit(todo.Run(os.Args[1:], os.Stdout, os.Stderr, version))
}
