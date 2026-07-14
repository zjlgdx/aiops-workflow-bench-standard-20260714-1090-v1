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

func TestInvalidInvocationsShowUsageWithoutDatabase(t *testing.T) {
	binary := buildTodo(t)

	for _, args := range [][]string{nil, {"nope"}} {
		result := runTodo(t, binary, "", args...)
		exitError, ok := result.err.(*exec.ExitError)
		if !ok {
			t.Fatalf("todo %v error = %v, want exit error", args, result.err)
		}
		if exitError.ExitCode() != 2 {
			t.Fatalf("todo %v exit code = %d, want 2", args, exitError.ExitCode())
		}
		if result.stdout != "" {
			t.Fatalf("todo %v stdout = %q, want empty", args, result.stdout)
		}
		if result.stderr != "usage: todo <add|list|done>\n" {
			t.Fatalf("todo %v stderr = %q, want usage", args, result.stderr)
		}
	}
}

func TestAddRejectsNewlineInTitleWithoutChangingDatabase(t *testing.T) {
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

	for _, title := range []string{"first\nsecond", "first\rsecond"} {
		rejected := runTodo(t, binary, database, "add", title)
		if rejected.err == nil {
			t.Fatalf("add %q succeeded, want non-zero exit", title)
		}
		if rejected.stdout != "" {
			t.Fatalf("add %q stdout = %q, want empty", title, rejected.stdout)
		}
		if rejected.stderr != "title must not contain newlines\n" {
			t.Fatalf("add %q stderr = %q, want newline error", title, rejected.stderr)
		}
	}

	after, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database after rejected adds: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("database changed after rejected adds:\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestDoneCompletesTodoAndIsIdempotent(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	add := runTodo(t, binary, database, "add", "Buy milk")
	if add.err != nil {
		t.Fatalf("add: %v, stderr %q", add.err, add.stderr)
	}

	completed := runTodo(t, binary, database, "done", "1")
	if completed.err != nil {
		t.Fatalf("first done: %v, stderr %q", completed.err, completed.stderr)
	}
	if completed.stdout != "completed 1\n" {
		t.Fatalf("first done stdout = %q, want %q", completed.stdout, "completed 1\n")
	}

	beforeSecond, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database before second done: %v", err)
	}
	completedAgain := runTodo(t, binary, database, "done", "1")
	if completedAgain.err != nil {
		t.Fatalf("second done: %v, stderr %q", completedAgain.err, completedAgain.stderr)
	}
	if completedAgain.stdout != "completed 1\n" {
		t.Fatalf("second done stdout = %q, want %q", completedAgain.stdout, "completed 1\n")
	}
	afterSecond, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database after second done: %v", err)
	}
	if !bytes.Equal(afterSecond, beforeSecond) {
		t.Fatalf("database changed after idempotent done:\nbefore: %s\nafter:  %s", beforeSecond, afterSecond)
	}

	list := runTodo(t, binary, database, "list")
	if list.err != nil {
		t.Fatalf("list: %v, stderr %q", list.err, list.stderr)
	}
	if list.stdout != "1\tdone\tBuy milk\n" {
		t.Fatalf("list stdout = %q, want %q", list.stdout, "1\tdone\tBuy milk\n")
	}
}

func TestDoneRejectsMissingMalformedAndUnknownIDsWithoutChangingDatabase(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	add := runTodo(t, binary, database, "add", "Keep active")
	if add.err != nil {
		t.Fatalf("add: %v, stderr %q", add.err, add.stderr)
	}
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database before rejected done: %v", err)
	}

	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{name: "missing argument", args: []string{"done"}, wantStderr: "todo ID is required\n"},
		{name: "non-numeric", args: []string{"done", "nope"}, wantStderr: "invalid todo ID \"nope\"\n"},
		{name: "non-positive", args: []string{"done", "0"}, wantStderr: "invalid todo ID \"0\"\n"},
		{name: "unknown", args: []string{"done", "999"}, wantStderr: "todo 999 not found\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := runTodo(t, binary, database, test.args...)
			if result.err == nil {
				t.Fatalf("todo %v succeeded, want non-zero exit", test.args)
			}
			if result.stdout != "" {
				t.Fatalf("todo %v stdout = %q, want empty", test.args, result.stdout)
			}
			if result.stderr != test.wantStderr {
				t.Fatalf("todo %v stderr = %q, want %q", test.args, result.stderr, test.wantStderr)
			}

			after, err := os.ReadFile(database)
			if err != nil {
				t.Fatalf("read database after todo %v: %v", test.args, err)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("database changed after todo %v:\nbefore: %s\nafter:  %s", test.args, before, after)
			}
		})
	}
}

func TestListFiltersByStatusInIDOrder(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	for _, title := range []string{"First", "Second", "Third"} {
		add := runTodo(t, binary, database, "add", title)
		if add.err != nil {
			t.Fatalf("add %q: %v, stderr %q", title, add.err, add.stderr)
		}
	}
	completed := runTodo(t, binary, database, "done", "2")
	if completed.err != nil {
		t.Fatalf("done: %v, stderr %q", completed.err, completed.stderr)
	}

	active := runTodo(t, binary, database, "list", "--status", "active")
	if active.err != nil {
		t.Fatalf("list active: %v, stderr %q", active.err, active.stderr)
	}
	if active.stdout != "1\tactive\tFirst\n3\tactive\tThird\n" {
		t.Fatalf("list active stdout = %q", active.stdout)
	}

	done := runTodo(t, binary, database, "list", "--status", "done")
	if done.err != nil {
		t.Fatalf("list done: %v, stderr %q", done.err, done.stderr)
	}
	if done.stdout != "2\tdone\tSecond\n" {
		t.Fatalf("list done stdout = %q", done.stdout)
	}
}

func TestListRejectsUnsupportedStatusWithoutChangingDatabase(t *testing.T) {
	binary := buildTodo(t)
	database := filepath.Join(t.TempDir(), "todos.json")

	add := runTodo(t, binary, database, "add", "Keep active")
	if add.err != nil {
		t.Fatalf("add: %v, stderr %q", add.err, add.stderr)
	}
	before, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database before rejected list: %v", err)
	}

	result := runTodo(t, binary, database, "list", "--status", "archived")
	if result.err == nil {
		t.Fatal("unsupported status succeeded, want non-zero exit")
	}
	if result.stdout != "" {
		t.Fatalf("unsupported status stdout = %q, want empty", result.stdout)
	}
	if result.stderr != "unsupported status \"archived\"; want active or done\n" {
		t.Fatalf("unsupported status stderr = %q", result.stderr)
	}

	after, err := os.ReadFile(database)
	if err != nil {
		t.Fatalf("read database after rejected list: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("database changed after rejected list:\nbefore: %s\nafter:  %s", before, after)
	}
}
