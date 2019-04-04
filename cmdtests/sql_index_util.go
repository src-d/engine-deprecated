// +build integration regression

package cmdtests

import (
	"strings"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/icmd"
)

// building/loading index takes some time, wait maximum 15s
func IndexIsVisible(s commandSuite, table, name string) string {
	var visibleValue string
	for i := 0; i < 15; i++ {
		visibleValue = hasIndex(s, table, name)
		if visibleValue != "YES" {
			time.Sleep(time.Second)
		} else {
			break
		}
	}

	return visibleValue
}

func hasIndex(s commandSuite, table, name string) string {
	r := s.RunCommand("sql", "SHOW INDEX FROM "+table)
	s.Require().NoError(r.Error, r.Combined())

	// parse result and check that correct index was built and it is visiable
	lines := strings.Split(r.Stdout(), "\n")
	for _, line := range lines {
		cols := strings.Split(line, "|")
		if len(cols) < 15 {
			continue
		}
		if strings.TrimSpace(cols[1]) != table {
			continue
		}
		if strings.TrimSpace(cols[3]) != name {
			continue
		}

		return strings.TrimSpace(cols[14])
	}

	return "NO"
}

type commandSuite interface {
	RunCommand(cmd string, args ...string) *icmd.Result
	Require() *require.Assertions
}
