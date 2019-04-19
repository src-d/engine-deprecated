// +build integration regression

package cmdtests

import (
	"strings"
	"time"

	"github.com/stretchr/testify/require"
)

// building/loading index takes some time, wait maximum 15s
func IndexIsVisible(s requirer, commander *Commander, table, name string) bool {
	for i := 0; i < 15; i++ {
		if hasIndex(s, commander, table, name) {
			return true
		}
		time.Sleep(time.Second)
	}

	return hasIndex(s, commander, table, name)
}

func hasIndex(s requirer, commander *Commander, table, name string) bool {
	r := commander.RunCommand("sql", "SHOW INDEX FROM "+table)
	s.Require().NoError(r.Error, r.Combined())

	// parse result and check that correct index was built and it is visiable
	// see example output here: https://dev.mysql.com/doc/refman/8.0/en/show-index.html
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

type requirer interface {
	Require() *require.Assertions
}
