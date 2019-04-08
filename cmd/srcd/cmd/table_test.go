// +build !integration

package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TableTestSuite struct {
	suite.Suite
}

func TestTableTestSuite(t *testing.T) {
	suite.Run(t, new(TableTestSuite))
}

func (s *TableTestSuite) TestPrintWrongHeaderSize() {
	require := s.Require()

	var out bytes.Buffer
	t := NewTable("%s", "%d", "%b")
	t.Header("col1")
	t.Row("f1", "f2", "f3")
	require.EqualError(t.Print(&out), fmt.Sprintf("number of header provided '%d', required '%d'", 1, 3))
}

func (s *TableTestSuite) TestPrintWrongRowSize() {
	require := s.Require()

	var out bytes.Buffer
	t := NewTable("%s", "%d", "%b")
	t.Header("col1", "col2", "col3")
	t.Row("f1")
	require.EqualError(t.Print(&out), fmt.Sprintf("number of items in row provided '%d', required '%d'", 1, 3))
}

func (s *TableTestSuite) sampleFixture() ([]string, []string, [][]interface{}) {
	formats := []string{"%s", "%d", "%b"}
	header := []string{"col1", "col2", "col3"}
	rows := [][]interface{}{
		[]interface{}{"s1", 1, 1},
		[]interface{}{"s2", 2, 2},
		[]interface{}{"s3", 3, 3},
		[]interface{}{"s4", 4, 4},
		[]interface{}{"s5", 5, 5},
	}

	return formats, header, rows
}

func (s *TableTestSuite) TestPrint() {
	require := s.Require()

	var out bytes.Buffer
	formats, header, rows := s.sampleFixture()

	t := NewTable(formats...)
	t.Header(header...)
	for _, r := range rows {
		t.Row(r...)
	}

	require.NoError(t.Print(&out))
	expected := `col1    col2    col3
s1      1       1
s2      2       10
s3      3       11
s4      4       100
s5      5       101
`
	require.Equal(expected, out.String())
}

func (s *TableTestSuite) TestPrintNoHeader() {
	require := s.Require()

	var out bytes.Buffer
	formats, _, rows := s.sampleFixture()

	t := NewTable(formats...)
	for _, r := range rows {
		t.Row(r...)
	}

	require.NoError(t.Print(&out))
	expected := `s1    1    1
s2    2    10
s3    3    11
s4    4    100
s5    5    101
`
	require.Equal(expected, out.String())
}
