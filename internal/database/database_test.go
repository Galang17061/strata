package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitBatchesSeparatesOnGo(t *testing.T) {
	script := "CREATE TABLE a (id int)\nGO\n\nINSERT INTO a VALUES (1)\ngo\n"
	batches := SplitBatches(script)
	assert.Equal(t, []string{"CREATE TABLE a (id int)", "INSERT INTO a VALUES (1)"}, batches)
}

func TestSplitBatchesKeepsScriptWithoutGo(t *testing.T) {
	batches := SplitBatches("SELECT 1")
	assert.Equal(t, []string{"SELECT 1"}, batches)
}

func TestWithDatabaseRewritesUrlForm(t *testing.T) {
	rewritten := WithDatabase("sqlserver://user:pass@host:1433?database=strata&encrypt=disable", "master")
	assert.Contains(t, rewritten, "database=master")
	assert.NotContains(t, rewritten, "database=strata")
}

func TestWithDatabaseRewritesKeyValueForm(t *testing.T) {
	rewritten := WithDatabase("server=host,1433;database=strata;user id=u;password=p", "master")
	assert.Equal(t, "server=host,1433;user id=u;password=p;database=master", rewritten)
}
