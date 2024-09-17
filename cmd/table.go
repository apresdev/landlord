package cmd

import (
	"fmt"
)

// Table holds data for simple tables to be rendered in the CLI
type Table struct {
	Cols   int
	Header []string
	Data   [][]string
}

func NewTable(header []string) *Table {
	return &Table{Cols: len(header), Header: header, Data: make([][]string, 0)}
}

func (t *Table) getMaxWidths() []int {
	maxWidths := make([]int, t.Cols)
	for i, cell := range t.Header {
		if len(cell) > maxWidths[i] {
			maxWidths[i] = len(cell)
		}
	}
	for _, row := range t.Data {
		for i, cell := range row {
			if len(cell) > maxWidths[i] {
				maxWidths[i] = len(cell)
			}
		}
	}
	return maxWidths
}

func (t *Table) AddRow(row []string) error {
	if len(row) != t.Cols {
		return fmt.Errorf("row has %d columns, expected %d", len(row), t.Cols)
	}
	t.Data = append(t.Data, row)
	return nil
}

func (t *Table) String() string {
	maxWidths := t.getMaxWidths()
	result := ""
	for i, cell := range t.Header {
		result += fmt.Sprintf("%-*s ", maxWidths[i], cell)
	}
	result += "\n"
	for _, row := range t.Data {
		for i, cell := range row {
			result += fmt.Sprintf("%-*s ", maxWidths[i], cell)
		}
		result += "\n"
	}
	return result
}
