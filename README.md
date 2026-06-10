# todo

A tiny project-local todo CLI.

`todo` writes Markdown checklist items to `todo.md`. Inside a git repository it writes to the repository root. Outside git it writes to the current directory.

```sh
todo here is a todo item
todo --list
todo --done 1
todo --delete 1
```

## Installation

### Homebrew

After the Homebrew tap is published:

```sh
brew install feint/todo/todo
```

### Go

```sh
go install github.com/feint/todo/cmd/todo@latest
```

## Usage

```sh
todo <item text>        Add an item to todo.md
todo --list            List checklist items
todo --done <number>   Toggle an item between done and not done
todo --delete <number> Delete an item
todo --version         Print the version
todo --help            Print help
```

Items are stored as Markdown checkboxes:

```md
- [ ] write the first release
- [x] make coffee
```

Only Markdown checklist lines are numbered by `--list`, `--done`, and `--delete`; other Markdown in `todo.md` is preserved.

## Homebrew Packaging

The source repository is `github.com/feint/todo`. The Homebrew tap repository is `github.com/feint/homebrew-todo`.

Release checklist:

1. Tag the source release:

   ```sh
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. Create a GitHub release for the tag.

3. Calculate the source archive checksum:

   ```sh
   curl -L https://github.com/feint/todo/archive/refs/tags/v0.1.0.tar.gz | shasum -a 256
   ```

4. Add or update `Formula/todo.rb` in `github.com/feint/homebrew-todo`:

   ```ruby
   class Todo < Formula
     desc "Tiny project-local todo CLI"
     homepage "https://github.com/feint/todo"
     url "https://github.com/feint/todo/archive/refs/tags/v0.1.0.tar.gz"
     sha256 "PASTE_SHA256_HERE"
     license "MIT"

     depends_on "go" => :build

     def install
       ldflags = "-s -w -X main.version=#{version}"
       system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/todo"
     end

     test do
       assert_match "todo #{version}", shell_output("#{bin}/todo --version")
     end
   end
   ```

5. Test the formula:

   ```sh
   brew install --build-from-source feint/todo/todo
   brew test feint/todo/todo
   ```

## Development

```sh
go test ./...
go run ./cmd/todo --help
```

## License

MIT
