package todo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const fileName = "todo.md"
const masterFileName = ".todo.master.md"

var errNoTodos = errors.New("no todos found")
var runEditor = defaultRunEditor

type item struct {
	lineIndex int
	done      bool
	text      string
}

// Run executes the cli-todo CLI and returns a process exit code.
func Run(args []string, stdout io.Writer, stderr io.Writer, version string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "--version":
		fmt.Fprintf(stdout, "cli-todo %s\n", version)
		return 0
	case "--path":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "cli-todo: --path does not accept extra arguments")
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, path)
		return 0
	case "--open":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "cli-todo: --open does not accept extra arguments")
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := Open(path); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		return 0
	case "--list":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "cli-todo: --list does not accept extra arguments")
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := List(path, stdout); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
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
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := Toggle(path, n); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		return 0
	case "--edit":
		n, text, ok := parseEditArgs(args, stderr)
		if !ok {
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := Edit(path, n, text); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
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
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := Delete(path, n); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		return 0
	case "--clean":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "cli-todo: --clean does not accept extra arguments")
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := Clean(path); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		return 0
	case "--master":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "cli-todo: --master does not accept extra arguments")
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := WriteMaster(filepath.Dir(path)); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		return 0
	default:
		if strings.HasPrefix(args[0], "-") {
			fmt.Fprintf(stderr, "cli-todo: unknown option %q\n", args[0])
			return 1
		}
		path, err := ResolvePath("")
		if err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
			return 1
		}
		if err := Add(path, strings.Join(args, " ")); err != nil {
			fmt.Fprintf(stderr, "cli-todo: %v\n", err)
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

func Open(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return errors.New("$EDITOR is not set")
	}
	return runEditor(editor, path)
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

func Edit(path string, number int, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("todo text cannot be empty")
	}

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
		lines[target.lineIndex] = "- [x] " + text
	} else {
		lines[target.lineIndex] = "- [ ] " + text
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

func Clean(path string) error {
	lines, err := readLines(path)
	if err != nil {
		return err
	}

	foundChecklist := false
	var kept []string
	for _, line := range lines {
		done, _, ok := parseChecklistLine(line)
		if ok {
			foundChecklist = true
		}
		if ok && done {
			continue
		}
		kept = append(kept, line)
	}
	if !foundChecklist {
		return errNoTodos
	}

	return writeLines(path, kept)
}

func WriteMaster(root string) error {
	entries, err := collectTodos(root)
	if err != nil {
		return err
	}

	lines := []string{"# Master Todos", ""}
	for _, entry := range entries {
		lines = append(lines, "## "+entry.relPath)
		for _, item := range entry.items {
			mark := " "
			if item.done {
				mark = "x"
			}
			lines = append(lines, fmt.Sprintf("- [%s] %s", mark, item.text))
		}
		lines = append(lines, "")
	}

	return writeLines(filepath.Join(root, masterFileName), lines)
}

func parseItemNumber(args []string, stderr io.Writer, flag string) (int, bool) {
	if len(args) != 2 {
		fmt.Fprintf(stderr, "cli-todo: %s requires an item number\n", flag)
		return 0, false
	}

	n, err := strconv.Atoi(args[1])
	if err != nil || n < 1 {
		fmt.Fprintf(stderr, "cli-todo: %s requires a positive item number\n", flag)
		return 0, false
	}

	return n, true
}

func parseEditArgs(args []string, stderr io.Writer) (int, string, bool) {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "cli-todo: --edit requires an item number")
		return 0, "", false
	}

	n, err := strconv.Atoi(args[1])
	if err != nil || n < 1 {
		fmt.Fprintln(stderr, "cli-todo: --edit requires a positive item number")
		return 0, "", false
	}

	if len(args) < 3 {
		fmt.Fprintln(stderr, "cli-todo: --edit requires todo text")
		return 0, "", false
	}

	text := strings.TrimSpace(strings.Join(args[2:], " "))
	if text == "" {
		fmt.Fprintln(stderr, "cli-todo: --edit requires todo text")
		return 0, "", false
	}

	return n, text, true
}

func printHelp(out io.Writer) {
	fmt.Fprint(out, `Usage:
  cli-todo <item text>
  cli-todo --path
  cli-todo --open
  cli-todo --list
  cli-todo --done <number>
  cli-todo --edit <number> <new text>
  cli-todo --delete <number>
  cli-todo --clean
  cli-todo --master
  cli-todo --version
  cli-todo --help

Examples:
  cli-todo here is a todo item
  cli-todo --done 1
  cli-todo --edit 1 updated todo text
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

type todoFile struct {
	relPath string
	items   []item
}

func collectTodos(root string) ([]todoFile, error) {
	var entries []todoFile
	err := filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			if path != root && skipDir(dirEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if dirEntry.Name() != fileName {
			return nil
		}

		lines, err := readLines(path)
		if err != nil {
			return err
		}
		items := checklistItems(lines)
		if len(items) == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		entries = append(entries, todoFile{
			relPath: filepath.ToSlash(rel),
			items:   items,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relPath < entries[j].relPath
	})
	return entries, nil
}

func skipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "vendor", "dist", "build", "target", "tmp":
		return true
	default:
		return false
	}
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

func defaultRunEditor(editor string, path string) error {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return errors.New("$EDITOR is not set")
	}

	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
