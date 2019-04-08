package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Table represents a printable table composed by rows and an optional header
type Table struct {
	formats []string
	header  []string
	rows    [][]interface{}
}

// Header sets the header of the table with the given strings
func (t *Table) Header(header ...string) {
	t.header = header
}

// Row adds a row to the table
func (t *Table) Row(row ...interface{}) {
	t.rows = append(t.rows, row)
}

// Print prints the table to the given writer
// It returns an error if there's a mismatch between the length of the formats
// and the length of the header, or between the length of the formats and the
// length of any row.
func (t *Table) Print(output io.Writer) error {
	tw := tabwriter.NewWriter(output, 0, 0, 4, ' ', 0)
	if len(t.header) > 0 {
		if len(t.header) != len(t.formats) {
			return fmt.Errorf("number of header provided '%d', required '%d'",
				len(t.header), len(t.formats))
		}

		sHeader := strings.Join(t.header, "\t") + "\n"
		fmt.Fprintf(tw, sHeader)
	}

	sFormats := strings.Join(t.formats, "\t") + "\n"
	for _, row := range t.rows {
		if len(row) != len(t.formats) {
			return fmt.Errorf("number of items in row provided '%d', required '%d'",
				len(row), len(t.formats))
		}

		fmt.Fprintf(tw, sFormats, row...)
	}

	return tw.Flush()
}

// NewTable creates a new `Table` with the provided formats
func NewTable(formats ...string) *Table {
	return &Table{formats: formats}
}
