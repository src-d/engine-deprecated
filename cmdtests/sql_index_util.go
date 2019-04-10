// +build integration regression

package cmdtests

import (
	"strings"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/icmd"
)

// building/loading index takes some time, wait maximum 15s
func IndexIsVisible(s commandSuite, table, name string) bool {
	for i := 0; i < 15; i++ {
		if hasIndex(s, table, name) {
			return true
		}
		time.Sleep(time.Second)
	}

	return hasIndex(s, table, name)
}

func hasIndex(s commandSuite, table, name string) bool {
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

		if strings.TrimSpace(cols[14]) == "YES" {
			return true
		}
		return false
	}

	return false
}

type commandSuite interface {
	RunCommand(cmd string, args ...string) *icmd.Result
	Require() *require.Assertions
}
