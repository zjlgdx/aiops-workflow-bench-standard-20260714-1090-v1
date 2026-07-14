package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type todo struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}

type database struct {
	NextID int    `json:"next_id"`
	Todos  []todo `json:"todos"`
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Getenv("TODO_DB"), os.Stdout, os.Stderr))
}

func runCLI(args []string, databasePath string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintln(stdout, "todo-bench seed")
		return 0
	}

	switch {
	case len(args) == 2 && args[0] == "add":
		if databasePath == "" {
			fmt.Fprintln(stderr, "TODO_DB must be set")
			return 1
		}
		title := strings.TrimSpace(args[1])
		if title == "" {
			fmt.Fprintln(stderr, "title must not be empty")
			return 1
		}
		if strings.ContainsAny(title, "\r\n") {
			fmt.Fprintln(stderr, "title must not contain newlines")
			return 1
		}

		state, err := loadDatabase(databasePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}

		item := todo{ID: state.NextID, Status: "active", Title: title}
		state.Todos = append(state.Todos, item)
		state.NextID++
		if err := saveDatabase(databasePath, state); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "added %d\n", item.ID)
		return 0

	case len(args) == 1 && args[0] == "list":
		if databasePath == "" {
			fmt.Fprintln(stderr, "TODO_DB must be set")
			return 1
		}
		state, err := loadDatabase(databasePath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		sort.Slice(state.Todos, func(i, j int) bool {
			return state.Todos[i].ID < state.Todos[j].ID
		})
		for _, item := range state.Todos {
			fmt.Fprintf(stdout, "%d\t%s\t%s\n", item.ID, item.Status, item.Title)
		}
		return 0

	default:
		fmt.Fprintln(stderr, "usage: todo <add|list|done>")
		return 2
	}
}

func loadDatabase(path string) (database, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return database{NextID: 1}, nil
	}
	if err != nil {
		return database{}, fmt.Errorf("read database: %w", err)
	}

	var state database
	if err := json.Unmarshal(data, &state); err != nil {
		return database{}, fmt.Errorf("decode database: %w", err)
	}
	return state, nil
}

func saveDatabase(path string, state database) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode database: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write database: %w", err)
	}
	return nil
}
