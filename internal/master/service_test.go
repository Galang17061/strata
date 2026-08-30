package master

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestNextPrefixedIdContinuesTheSequence(t *testing.T) {
	last := "VD-00012"
	next, err := nextPrefixedId(&last, "VD-")
	require.NoError(t, err)
	assert.Equal(t, "VD-00013", next)
	first, err := nextPrefixedId(nil, "CM-")
	require.NoError(t, err)
	assert.Equal(t, "CM-00001", first)
	odd := "LEGACY-1"
	fallback, err := nextPrefixedId(&odd, "PJ-")
	require.NoError(t, err)
	assert.Equal(t, "PJ-00001", fallback)
}

func TestTemplateWorkbookCarriesHeadingsAndSample(t *testing.T) {
	service := NewService(nil, t.TempDir())
	content, err := service.TemplateWorkbook()
	require.NoError(t, err)
	book, err := excelize.OpenReader(bytes.NewReader(content))
	require.NoError(t, err)
	defer book.Close()
	rows, err := book.GetRows("Template")
	require.NoError(t, err)
	assert.Equal(t, "Master Component Import Template", rows[0][0])
	assert.Equal(t, []string{"Component Name", "Vendor", "Failure Rate", "Cost"}, rows[1])
	assert.Equal(t, "Air Compressor", rows[2][0])
	assert.Equal(t, "0.0001", rows[2][2])
}

func TestManufacturerComparerFallsBackToName(t *testing.T) {
	rows := []domain.MasterManufacturer{{VendorId: "VD-2", ManufacturerName: "Zeta"}, {VendorId: "VD-1", ManufacturerName: "alpha"}}
	domain.OrderBy(rows, false, manufacturerComparer("unknown"))
	assert.Equal(t, "alpha", rows[0].ManufacturerName)
	domain.OrderBy(rows, true, manufacturerComparer("vendorid"))
	assert.Equal(t, "VD-2", rows[0].VendorId)
}
