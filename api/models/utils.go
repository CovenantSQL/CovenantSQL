package models

import (
	"regexp"
	"strings"
)

var (
	selectFromRegexp = regexp.MustCompile("(?is)select\\s+.+?\\s+from")
)

func buildCountSQL(querySQL string) string {
	return selectFromRegexp.ReplaceAllString(querySQL, "SELECT count(*) FROM")
}

func buildSQLWithConds(querySQL, countSQL string, conds []string) (newQuerySQL, newCountSQL string) {
	whereSQL := ""
	if len(conds) > 0 {
		whereSQL = " WHERE " + strings.Join(conds, " AND ")
	}
	return querySQL + whereSQL, countSQL + whereSQL
}
