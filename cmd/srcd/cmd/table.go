package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Table struct {
	formats []string
	header  []string
	rows    [][]interface{}
}

func (t *Table) Header(header ...string) error {
	if len(header) != len(t.formats) {
		return fmt.Errorf("number of header provided '%d', required '%d'",
			len(header), len(t.formats))
	}

	t.header = header
	return nil
}

func (t *Table) Row(row ...interface{}) error {
	if len(row) != len(t.formats) {
		return fmt.Errorf("number of items provided '%d', required '%d'",
			len(row), len(t.formats))
	}

	t.rows = append(t.rows, row)
	return nil
}

func (t *Table) Print(output io.Writer) error {
	tw := tabwriter.NewWriter(output, 0, 0, 4, ' ', 0)
	if len(t.header) > 0 {
		sHeader := strings.Join(t.header, "\t") + "\n"
		fmt.Fprintf(tw, sHeader)
	}

	sFormats := strings.Join(t.formats, "\t") + "\n"
	for _, row := range t.rows {
		fmt.Fprintf(tw, sFormats, row...)
	}

	return tw.Flush()
}

func NewTable(formats ...string) *Table {
	return &Table{formats: formats}
}
