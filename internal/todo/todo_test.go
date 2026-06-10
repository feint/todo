package todo

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddAppendsChecklistItem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")

	if err := Add(path, "here is a todo item"); err != nil {
		t.Fatal(err)
	}

	assertFile(t, path, "- [ ] here is a todo item\n")
}

func TestAddSeparatesExistingContentWithoutTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte("# Todos"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Add(path, "new item"); err != nil {
		t.Fatal(err)
	}

	assertFile(t, path, "# Todos\n- [ ] new item\n")
}

func TestListPrintsNumberedChecklistItems(t *testing.T) {
	path := writeTodo(t, "# Project\n- [ ] first\nnotes\n- [x] second\n")

	var out bytes.Buffer
	if err := List(path, &out); err != nil {
		t.Fatal(err)
	}

	want := "1. [ ] first\n2. [x] second\n"
	if out.String() != want {
		t.Fatalf("got %q, want %q", out.String(), want)
	}
}

func TestToggleChangesDoneStateAndPreservesOtherMarkdown(t *testing.T) {
	path := writeTodo(t, "# Project\n- [ ] first\nnotes\n- [x] second\n")

	if err := Toggle(path, 1); err != nil {
		t.Fatal(err)
	}
	if err := Toggle(path, 2); err != nil {
		t.Fatal(err)
	}

	assertFile(t, path, "# Project\n- [x] first\nnotes\n- [ ] second\n")
}

func TestDeleteRemovesChecklistItemByNumber(t *testing.T) {
	path := writeTodo(t, "# Project\n- [ ] first\nnotes\n- [x] second\n")

	if err := Delete(path, 1); err != nil {
		t.Fatal(err)
	}

	assertFile(t, path, "# Project\nnotes\n- [x] second\n")
}

func TestInvalidItemNumberReturnsError(t *testing.T) {
	path := writeTodo(t, "- [ ] first\n")

	err := Toggle(path, 2)
	if err == nil || !strings.Contains(err.Error(), "item number 2 does not exist") {
		t.Fatalf("got error %v", err)
	}
}

func TestMissingTodoReturnsNoTodosForListDoneAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")

	var out bytes.Buffer
	for name, fn := range map[string]func() error{
		"list":   func() error { return List(path, &out) },
		"toggle": func() error { return Toggle(path, 1) },
		"delete": func() error { return Delete(path, 1) },
	} {
		if err := fn(); err != errNoTodos {
			t.Fatalf("%s got %v, want %v", name, err, errNoTodos)
		}
	}
}

func TestResolvePathFallsBackToCurrentDirectoryOutsideGit(t *testing.T) {
	dir := t.TempDir()

	path, err := ResolvePath(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(dir, "todo.md")
	if path != want {
		t.Fatalf("got %q, want %q", path, want)
	}
}

func TestResolvePathUsesGitRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runGit(t, root, "init")

	subdir := filepath.Join(root, "nested")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	path, err := ResolvePath(subdir)
	if err != nil {
		t.Fatal(err)
	}

	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "todo.md")
	if path != want {
		t.Fatalf("got %q, want %q", path, want)
	}
}

func TestRunParsesCommands(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	var out bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"from", "run"}, &out, &stderr, "0.1.0")
	if code != 0 {
		t.Fatalf("add exited %d, stderr %q", code, stderr.String())
	}

	out.Reset()
	stderr.Reset()
	code = Run([]string{"--list"}, &out, &stderr, "0.1.0")
	if code != 0 {
		t.Fatalf("list exited %d, stderr %q", code, stderr.String())
	}
	if out.String() != "1. [ ] from run\n" {
		t.Fatalf("got %q", out.String())
	}
}

func writeTodo(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertFile(t *testing.T, path string, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("got %q, want %q", string(got), want)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
