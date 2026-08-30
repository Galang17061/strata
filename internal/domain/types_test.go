package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateTimeTextFollowsZoneKind(t *testing.T) {
	wall := WallClock(time.Date(2025, 11, 19, 10, 0, 0, 333000000, time.UTC))
	assert.Equal(t, "2025-11-19T10:00:00.333", wall.Text())
	utc := DateTime{time.Date(2025, 11, 19, 10, 0, 0, 0, time.UTC)}
	assert.Equal(t, "2025-11-19T10:00:00Z", utc.Text())
	local := DateTime{time.Date(2025, 11, 19, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))}
	assert.Equal(t, "2025-11-19T10:00:00+07:00", local.Text())
}

func TestDateTimeParsesDateOnlyAndOffsets(t *testing.T) {
	var parsed DateTime
	require.NoError(t, json.Unmarshal([]byte(`"2025-01-15"`), &parsed))
	assert.Equal(t, "2025-01-15T00:00:00", parsed.Text())
	require.NoError(t, json.Unmarshal([]byte(`"2025-01-15T08:30:00Z"`), &parsed))
	assert.Equal(t, "2025-01-15T08:30:00Z", parsed.Text())
	assert.Error(t, json.Unmarshal([]byte(`"not a date"`), &parsed))
}

func TestNumberKeepsScaleWhenPrinted(t *testing.T) {
	n, err := NumberFromString("0.950000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "0.950000000000000000", n.Text())
	encoded, err := json.Marshal(struct {
		Value  Number  `json:"value"`
		Absent *Number `json:"absent"`
	}{Value: NewNumber(decimal.RequireFromString("0.90483741803596"))})
	require.NoError(t, err)
	assert.Equal(t, `{"value":0.90483741803596,"absent":null}`, string(encoded))
}

func TestGuidRoundTrip(t *testing.T) {
	parsed, err := ParseGuid("3F2504E0-4F89-11D3-9A0C-0305E82C3301")
	require.NoError(t, err)
	assert.Equal(t, "3f2504e0-4f89-11d3-9a0c-0305e82c3301", parsed.String())
	value, err := parsed.Value()
	require.NoError(t, err)
	var scanned Guid
	require.NoError(t, scanned.Scan(value))
	assert.Equal(t, parsed, scanned)
	_, err = ParseGuid("nope")
	assert.EqualError(t, err, "The value 'nope' is not valid.")
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", EmptyGuid.String())
}
