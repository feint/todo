package todo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const fileName = "todo.md"

var errNoTodos = errors.New("no todos found")

type item struct {
	lineIndex int
	done      bool
	text      string
}

// Run executes the todo CLI and returns a process exit code.
func Run(args []string, stdout io.Writer, stderr io.Writer, version string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "--version":
		fmt.Fprintf(stdout, "todo %s\n", version)
		return 0
	case "--list":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "todo: --list does not accept extra arguments")
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		if err := List(path, stdout); err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		return 0
	case "--done":
		n, ok := parseItemNumber(args, stderr, "--done")
		if !ok {
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		if err := Toggle(path, n); err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		return 0
	case "--delete":
		n, ok := parseItemNumber(args, stderr, "--delete")
		if !ok {
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		if err := Delete(path, n); err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "todo: unknown option %q\n", args[0])
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		if err := Add(path, strings.Join(args, " ")); err != nil {
			fmt.Fprintf(stderr, "todo: %v\n", err)
			return 1
		}
		return 0
	}
}

func ResolvePath(cwd string) (string, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err == nil {
		root := strings.TrimSpace(string(output))
		if root != "" {
			return filepath.Join(root, fileName), nil
		}
	}

	return filepath.Join(cwd, fileName), nil
}

func Add(path string, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("todo text cannot be empty")
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > 0 {
		if err := ensureTrailingNewline(path, file); err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(file, "- [ ] %s\n", text)
	return err
}

func List(path string, out io.Writer) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}

	items := checklistItems(lines)
	if len(items) == 0 {
		return errNoTodos
	}

	for i, item := range items {
		mark := " "
		if item.done {
			mark = "x"
		}
		fmt.Fprintf(out, "%d. [%s] %s\n", i+1, mark, item.text)
	}

	return nil
}

func Toggle(path string, number int) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}

	items := checklistItems(lines)
	if len(items) == 0 {
		return errNoTodos
	}
	if number < 1 || number > len(items) {
		return fmt.Errorf("item number %d does not exist", number)
	}

	target := items[number-1]
	if target.done {
		lines[target.lineIndex] = "- [ ] " + target.text
	} else {
		lines[target.lineIndex] = "- [x] " + target.text
	}

	return writeLines(path, lines)
}

func Delete(path string, number int) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}

	items := checklistItems(lines)
	if len(items) == 0 {
		return errNoTodos
	}
	if number < 1 || number > len(items) {
		return fmt.Errorf("item number %d does not exist", number)
	}

	target := items[number-1]
	lines = append(lines[:target.lineIndex], lines[target.lineIndex+1:]...)
	return writeLines(path, lines)
}

func parseItemNumber(args []string, stderr io.Writer, flag string) (int, bool) {
	if len(args) != 2 {
		fmt.Fprintf(stderr, "todo: %s requires an item number\n", flag)
		return 0, false
	}

	n, err := strconv.Atoi(args[1])
	if err != nil || n < 1 {
		fmt.Fprintf(stderr, "todo: %s requires a positive item number\n", flag)
		return 0, false
	}

	return n, true
}

func printHelp(out io.Writer) {
	fmt.Fprint(out, `Usage:
  todo <item text>
  todo --list
  todo --done <number>
  todo --delete <number>
  todo --version
  todo --help

Examples:
  todo here is a todo item
  todo --done 1
`)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errNoTodos
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func writeLines(path string, lines []string) error {
	content := strings.Join(lines, "\n")
	if len(lines) > 0 {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func checklistItems(lines []string) []item {
	var items []item
	for i, line := range lines {
		done, text, ok := parseChecklistLine(line)
		if !ok {
			continue
		}
		items = append(items, item{
			lineIndex: i,
			done:      done,
			text:      text,
		})
	}
	return items
}

func parseChecklistLine(line string) (bool, string, bool) {
	if strings.HasPrefix(line, "- [ ] ") {
		return false, strings.TrimPrefix(line, "- [ ] "), true
	}
	if strings.HasPrefix(line, "- [x] ") {
		return true, strings.TrimPrefix(line, "- [x] "), true
	}
	return false, "", false
}

func ensureTrailingNewline(path string, file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return nil
	}

	readFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer readFile.Close()

	if _, err := readFile.Seek(-1, io.SeekEnd); err != nil {
		return err
	}

	last := make([]byte, 1)
	if _, err := readFile.Read(last); err != nil {
		return err
	}
	if last[0] != '\n' {
		_, err = file.WriteString("\n")
		return err
	}
	return nil
}
