package master

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/Galang17061/strata-api/internal/domain"
)

func (s *Service) TemplateWorkbook() ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	sheet := "Template"
	book.SetSheetName("Sheet1", sheet)
	if err := writeHeading(book, sheet, "Master Component Import Template"); err != nil {
		return nil, err
	}
	_ = book.SetCellValue(sheet, "A3", "Air Compressor")
	_ = book.SetCellValue(sheet, "B3", "KNORR")
	_ = book.SetCellValue(sheet, "C3", 0.0001)
	_ = book.SetCellValue(sheet, "D3", "1000000")
	fitColumns(book, sheet)
	return bookBytes(book)
}

func (s *Service) ExportWorkbook(ctx context.Context) ([]byte, error) {
	rows, err := s.store.ExportRows(ctx)
	if err != nil {
		return nil, err
	}
	book := excelize.NewFile()
	defer book.Close()
	sheet := "Master Component"
	book.SetSheetName("Sheet1", sheet)
	if err := writeHeading(book, sheet, "Master Component Data"); err != nil {
		return nil, err
	}
	bordered, _ := book.NewStyle(&excelize.Style{Border: thinBorders()})
	for index, row := range rows {
		line := index + 3
		_ = book.SetCellValue(sheet, fmt.Sprintf("A%d", line), row.ComponentName)
		_ = book.SetCellValue(sheet, fmt.Sprintf("B%d", line), row.ManufacturerName.String)
		if row.FailureRate != nil {
			_ = book.SetCellValue(sheet, fmt.Sprintf("C%d", line), row.FailureRate.InexactFloat64())
		}
		if row.Cost != nil {
			_ = book.SetCellValue(sheet, fmt.Sprintf("D%d", line), *row.Cost)
		}
	}
	if len(rows) > 0 {
		_ = book.SetCellStyle(sheet, "A3", fmt.Sprintf("D%d", len(rows)+2), bordered)
	}
	fitColumns(book, sheet)
	return bookBytes(book)
}

func (s *Service) ImportWorkbook(ctx context.Context, reader io.Reader, currentUser string) (*domain.ImportResult, error) {
	book, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, err
	}
	defer book.Close()
	result := &domain.ImportResult{FailedRows: []string{}}
	manufacturers, err := s.store.ListManufacturers(ctx, "")
	if err != nil {
		return nil, err
	}
	components, err := s.store.ListComponents(ctx)
	if err != nil {
		return nil, err
	}
	sheets := book.GetSheetList()
	if len(sheets) == 0 {
		return nil, domain.Argument("Worksheet position out of range.")
	}
	cells, err := book.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	for row := 3; row <= len(cells); row++ {
		line := cells[row-1]
		componentName := strings.TrimSpace(cell(line, 0))
		manufacturerName := strings.TrimSpace(cell(line, 1))
		failureRateText := strings.TrimSpace(cell(line, 2))
		cost := strings.TrimSpace(cell(line, 3))
		if componentName == "" {
			result.FailedRows = append(result.FailedRows, fmt.Sprintf("Row %d: Component Name is required.", row))
			continue
		}
		if manufacturerName == "" {
			result.FailedRows = append(result.FailedRows, fmt.Sprintf("Row %d: Vendor Name is required.", row))
			continue
		}
		var failureRate *domain.Number
		if failureRateText != "" {
			parsed, err := decimal.NewFromString(strings.ReplaceAll(failureRateText, ",", ""))
			if err != nil {
				result.FailedRows = append(result.FailedRows, fmt.Sprintf("Row %d: Invalid Failure Rate value '%s'.", row, failureRateText))
				continue
			}
			failureRate = domain.NumberPtr(domain.NewNumber(parsed))
		}
		if err := s.importRow(ctx, row, componentName, manufacturerName, failureRate, cost, currentUser, &manufacturers, &components, result); err != nil {
			result.FailedRows = append(result.FailedRows, fmt.Sprintf("Row %d: %s", row, err.Error()))
		}
	}
	result.Success = true
	return result, nil
}

func (s *Service) importRow(ctx context.Context, row int, componentName, manufacturerName string, failureRate *domain.Number, cost, currentUser string, manufacturers *[]domain.MasterManufacturer, components *[]domain.MasterComponent, result *domain.ImportResult) error {
	var manufacturer *domain.MasterManufacturer
	for index := range *manufacturers {
		if (*manufacturers)[index].ManufacturerName == manufacturerName {
			manufacturer = &(*manufacturers)[index]
			break
		}
	}
	if manufacturer == nil {
		lastId, err := s.store.LastManufacturerId(ctx)
		if err != nil {
			return err
		}
		newId, err := nextPrefixedId(lastId, "VD-")
		if err != nil {
			return err
		}
		now := domain.Now()
		created := domain.MasterManufacturer{VendorId: newId, ManufacturerName: manufacturerName, CreatedAt: now, UpdatedAt: now, CreatedBy: domain.StringPtr(currentUser), UpdatedBy: domain.StringPtr(currentUser)}
		if err := s.store.InsertManufacturer(ctx, created); err != nil {
			return err
		}
		*manufacturers = append(*manufacturers, created)
		manufacturer = &(*manufacturers)[len(*manufacturers)-1]
	}
	vendorId := manufacturer.VendorId
	for index := range *components {
		existing := &(*components)[index]
		if existing.ComponentName == componentName && existing.VendorId == vendorId {
			existing.FailureRate = failureRate
			existing.Cost = domain.StringPtr(cost)
			existing.UpdatedAt = domain.Now()
			existing.UpdatedBy = domain.StringPtr(currentUser)
			if err := s.store.UpdateComponent(ctx, *existing); err != nil {
				return err
			}
			result.UpdatedCount++
			return nil
		}
	}
	lastId, err := s.store.LastComponentId(ctx)
	if err != nil {
		return err
	}
	newId, err := nextPrefixedId(lastId, "CM-")
	if err != nil {
		return err
	}
	now := domain.Now()
	created := domain.MasterComponent{ComponentId: newId, ComponentName: componentName, VendorId: vendorId, FailureRate: failureRate, Cost: domain.StringPtr(cost), CreatedAt: now, UpdatedAt: now, CreatedBy: domain.StringPtr(currentUser), UpdatedBy: domain.StringPtr(currentUser)}
	if err := s.store.InsertComponent(ctx, created); err != nil {
		return err
	}
	*components = append(*components, created)
	result.SuccessCount++
	return nil
}

func cell(line []string, index int) string {
	if index < len(line) {
		return line[index]
	}
	return ""
}

func writeHeading(book *excelize.File, sheet, title string) error {
	_ = book.SetCellValue(sheet, "A1", title)
	_ = book.MergeCell(sheet, "A1", "D1")
	titleStyle, err := book.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	if err != nil {
		return err
	}
	_ = book.SetCellStyle(sheet, "A1", "A1", titleStyle)
	headers := []string{"Component Name", "Vendor", "Failure Rate", "Cost"}
	for index, header := range headers {
		_ = book.SetCellValue(sheet, fmt.Sprintf("%c2", 'A'+index), header)
	}
	headerStyle, err := book.NewStyle(&excelize.Style{
		Font:   &excelize.Font{Bold: true},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D3D3D3"}},
		Border: thinBorders(),
	})
	if err != nil {
		return err
	}
	_ = book.SetCellStyle(sheet, "A2", "D2", headerStyle)
	return nil
}

func thinBorders() []excelize.Border {
	return []excelize.Border{
		{Type: "top", Style: 1, Color: "000000"},
		{Type: "bottom", Style: 1, Color: "000000"},
		{Type: "left", Style: 1, Color: "000000"},
		{Type: "right", Style: 1, Color: "000000"},
	}
}

func fitColumns(book *excelize.File, sheet string) {
	rows, err := book.GetRows(sheet)
	if err != nil {
		return
	}
	widths := map[int]float64{}
	for _, row := range rows {
		for index, value := range row {
			if width := float64(len(value)) + 2; width > widths[index] {
				widths[index] = width
			}
		}
	}
	for index, width := range widths {
		column, _ := excelize.ColumnNumberToName(index + 1)
		_ = book.SetColWidth(sheet, column, column, width)
	}
}

func bookBytes(book *excelize.File) ([]byte, error) {
	buffer := new(bytes.Buffer)
	if err := book.Write(buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
