package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func Open(ctx context.Context, connectionString string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func EnsureDatabase(ctx context.Context, connectionString, dir string) error {
	name := DatabaseName(connectionString)
	if name == "" {
		return fmt.Errorf("connection string has no database name")
	}
	maintenance, err := Open(ctx, WithDatabase(connectionString, "postgres"))
	if err != nil {
		return err
	}
	defer maintenance.Close()
	var exists bool
	if err := maintenance.GetContext(ctx, &exists, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name); err != nil {
		return err
	}
	if !exists {
		if _, err := maintenance.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %q`, name)); err != nil {
			return err
		}
	}
	return nil
}

func Migrate(ctx context.Context, db *sqlx.DB, dir string) error {
	names, err := scriptNames(dir)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := runScript(ctx, db, dir, name); err != nil {
			return err
		}
	}
	return nil
}

func DatabaseName(connectionString string) string {
	parsed, err := url.Parse(connectionString)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Path, "/")
}

func WithDatabase(connectionString, database string) string {
	parsed, err := url.Parse(connectionString)
	if err != nil {
		return connectionString
	}
	parsed.Path = "/" + database
	return parsed.String()
}

func scriptNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func runScript(ctx context.Context, db *sqlx.DB, dir, name string) error {
	script, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	for index, statement := range SplitStatements(string(script)) {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%s statement %d: %w", name, index+1, err)
		}
	}
	return nil
}

func SplitStatements(script string) []string {
	statements := []string{}
	var current strings.Builder
	inSingleQuote := false
	inDollarQuote := false
	flush := func() {
		text := strings.TrimSpace(current.String())
		if text != "" {
			statements = append(statements, text)
		}
		current.Reset()
	}
	runes := []rune(script)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if inSingleQuote {
			current.WriteRune(c)
			if c == '\'' {
				inSingleQuote = false
			}
			continue
		}
		if inDollarQuote {
			current.WriteRune(c)
			if c == '$' && i+1 < len(runes) && runes[i+1] == '$' {
				current.WriteRune(runes[i+1])
				i++
				inDollarQuote = false
			}
			continue
		}
		switch {
		case c == '\'':
			inSingleQuote = true
			current.WriteRune(c)
		case c == '$' && i+1 < len(runes) && runes[i+1] == '$':
			inDollarQuote = true
			current.WriteRune(c)
			current.WriteRune(runes[i+1])
			i++
		case c == ';':
			flush()
		default:
			current.WriteRune(c)
		}
	}
	flush()
	return statements
}
