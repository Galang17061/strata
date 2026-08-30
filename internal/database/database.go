package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
)

const databaseScriptPrefix = "0000_"

func Open(ctx context.Context, connectionString string) (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlserver", connectionString)
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
	master, err := Open(ctx, WithDatabase(connectionString, "master"))
	if err != nil {
		return err
	}
	defer master.Close()
	names, err := scriptNames(dir)
	if err != nil {
		return err
	}
	for _, name := range names {
		if strings.HasPrefix(name, databaseScriptPrefix) {
			if err := runScript(ctx, master, dir, name); err != nil {
				return err
			}
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
		if strings.HasPrefix(name, databaseScriptPrefix) {
			continue
		}
		if err := runScript(ctx, db, dir, name); err != nil {
			return err
		}
	}
	return nil
}

func WithDatabase(connectionString, database string) string {
	if parsed, err := url.Parse(connectionString); err == nil && parsed.Scheme != "" {
		query := parsed.Query()
		query.Set("database", database)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	parts := strings.Split(connectionString, ";")
	kept := parts[:0]
	for _, part := range parts {
		key := strings.ToLower(strings.TrimSpace(strings.SplitN(part, "=", 2)[0]))
		if key == "database" || key == "initial catalog" {
			continue
		}
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(append(kept, "database="+database), ";")
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
	for index, batch := range SplitBatches(string(script)) {
		if _, err := db.ExecContext(ctx, batch); err != nil {
			return fmt.Errorf("%s batch %d: %w", name, index+1, err)
		}
	}
	return nil
}

func SplitBatches(script string) []string {
	batches := []string{}
	current := []string{}
	flush := func() {
		text := strings.TrimSpace(strings.Join(current, "\n"))
		if text != "" {
			batches = append(batches, text)
		}
		current = current[:0]
	}
	for _, line := range strings.Split(script, "\n") {
		if strings.EqualFold(strings.TrimSpace(line), "GO") {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return batches
}
