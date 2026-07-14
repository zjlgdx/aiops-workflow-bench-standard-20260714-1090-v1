package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type commandResult struct {
	stdout string
	stderr string
	err    error
}

func buildTodo(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "todo")
	command := exec.Command("go", "build", "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build todo: %v\n%s", err, output)
	}

	return binary
}

func runTodo(t *testing.T, binary, database string, args ...string) commandResult {
	t.Helper()

	command := exec.Command(binary, args...)
	command.Env = append(os.Environ(), "TODO_DB="+database)
	stdout, err := command.Output()
	result := commandResult{stdout: string(stdout), err: err}
	if exitError, ok := err.(*exec.ExitError); ok {
		result.stderr = string(exitError.Stderr)
	}
	return result
}

func TestAddPersistsAndListShowsTodosInIDOrder(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	first := runTodo(t, binary, database, "add", "Buy milk")
	if first.err != nil {
		t.Fatalf("first add: %v, stderr %q", first.err, first.stderr)
	}
	if first.stdout != "added 1\n" {
		t.Fatalf("first add stdout = %q, want %q", first.stdout, "added 1\n")
	}

	second := runTodo(t, binary, database, "add", "Call Mom")
	if second.err != nil {
		t.Fatalf("second add: %v, stderr %q", second.err, second.stderr)
	}
	if second.stdout != "added 2\n" {
		t.Fatalf("second add stdout = %q, want %q", second.stdout, "added 2\n")
	}

	list := runTodo(t, binary, database, "list")
	if list.err != nil {
		t.Fatalf("list: %v, stderr %q", list.err, list.stderr)
	}
	want := "1\tactive\tBuy milk\n2\tactive\tCall Mom\n"
	if list.stdout != want {
		t.Fatalf("list stdout = %q, want %q", list.stdout, want)
	}
}

func TestAddTrimsTitle(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	add := runTodo(t, binary, database, "add", "  Buy milk\t")
	if add.err != nil {
		t.Fatalf("add: %v, stderr %q", add.err, add.stderr)
	}

	list := runTodo(t, binary, database, "list")
	if list.err != nil {
		t.Fatalf("list: %v, stderr %q", list.err, list.stderr)
	}
	want := "1\tactive\tBuy milk\n"
	if list.stdout != want {
		t.Fatalf("list stdout = %q, want %q", list.stdout, want)
	}
}

func TestAddRejectsEmptyTitleWithoutChangingDatabase(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	seed := runTodo(t, binary, database, "add", "Keep me")
	if seed.err != nil {
		t.Fatalf("seed add: %v, stderr %q", seed.err, seed.stderr)
	}
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database before rejected add: %v", err)
	}

	rejected := runTodo(t, binary, database, "add", " \t\n ")
	if rejected.err == nil {
		t.Fatal("whitespace-only add succeeded, want non-zero exit")
	}
	if rejected.stdout != "" {
		t.Fatalf("rejected add stdout = %q, want empty", rejected.stdout)
	}
	if rejected.stderr != "title must not be empty\n" {
		t.Fatalf("rejected add stderr = %q, want %q", rejected.stderr, "title must not be empty\n")
	}

	after, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database after rejected add: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("database changed after rejected add:\nbefore: %s\nafter:  %s", before, after)
	}

	second := runTodo(t, binary, database, "add", "Still here")
	if second.err != nil {
		t.Fatalf("second valid add: %v, stderr %q", second.err, second.stderr)
	}
	if second.stdout != "added 2\n" {
		t.Fatalf("second valid add stdout = %q, want %q", second.stdout, "added 2\n")
	}
}
