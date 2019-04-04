// +build !integration

package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TableTestSuite struct {
	suite.Suite
}

func TestTableTestSuite(t *testing.T) {
	suite.Run(t, new(TableTestSuite))
}

func (s *TableTestSuite) TestPrintWrongSize() {
	require := s.Require()

	t := NewTable("%s", "%d", "%b")
	require.Error(t.Header("col1"))
	require.Error(t.Row("f1", "f2"))
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
	require.NoError(t.Header(header...))
	for _, r := range rows {
		require.NoError(t.Row(r...))
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
		require.NoError(t.Row(r...))
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
