package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitStatementsSeparatesOnSemicolon(t *testing.T) {
	script := "CREATE TABLE a (id int);\n\nINSERT INTO a VALUES (1);\n"
	statements := SplitStatements(script)
	assert.Equal(t, []string{"CREATE TABLE a (id int)", "INSERT INTO a VALUES (1)"}, statements)
}

func TestSplitStatementsKeepsDollarQuotedBodyWhole(t *testing.T) {
	script := "DO $$ BEGIN PERFORM 1; PERFORM 2; END $$;\nSELECT 1;"
	statements := SplitStatements(script)
	assert.Equal(t, []string{"DO $$ BEGIN PERFORM 1; PERFORM 2; END $$", "SELECT 1"}, statements)
}

func TestSplitStatementsIgnoresSemicolonInsideStrings(t *testing.T) {
	statements := SplitStatements("INSERT INTO a VALUES ('x;y');SELECT 1")
	assert.Equal(t, []string{"INSERT INTO a VALUES ('x;y')", "SELECT 1"}, statements)
}

func TestWithDatabaseSwapsThePath(t *testing.T) {
	rewritten := WithDatabase("postgres://user:pass@host:5432/strata?sslmode=disable", "postgres")
	assert.Equal(t, "postgres://user:pass@host:5432/postgres?sslmode=disable", rewritten)
}

func TestDatabaseNameReadsThePath(t *testing.T) {
	assert.Equal(t, "strata", DatabaseName("postgres://user:pass@host:5432/strata?sslmode=disable"))
}
