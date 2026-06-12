# cli-todo

A tiny project-local todo CLI.

`cli-todo` writes Markdown checklist items to `todo.md`. Inside a git repository it writes to the repository root. Outside git it writes to the current directory.

```sh
cli-todo here is a todo item
cli-todo --path
cli-todo --open
cli-todo --list
cli-todo --done 1
cli-todo --edit 1 updated todo text
cli-todo --delete 1
cli-todo --clean
cli-todo --master
```

## Installation

### Homebrew

```sh
brew install feint/todo/cli-todo
```

Homebrew also installs `todo` as an alias for `cli-todo`.

### Go

```sh
go install github.com/feint/cli-todo/cmd/cli-todo@latest
```

## Usage

```sh
cli-todo <item text>        Add an item to todo.md
cli-todo --path            Print the path to todo.md
cli-todo --open            Open todo.md in $EDITOR
cli-todo --list            List checklist items
cli-todo --done <number>   Toggle an item between done and not done
cli-todo --edit <number> <new text> Edit an item's text
cli-todo --delete <number> Delete an item
cli-todo --clean           Remove completed items
cli-todo --master          Build .todo.master.md from child todo.md files
cli-todo --version         Print the version
cli-todo --help            Print help
```

Items are stored as Markdown checkboxes:

```md
- [ ] write the first release
- [x] make coffee
```

Only Markdown checklist lines are numbered by `--list`, `--done`, `--edit`, `--delete`, and `--clean`; other Markdown in `todo.md` is preserved.

## Development

```sh
go test ./...
go run ./cmd/cli-todo --help
```

## License

MIT
