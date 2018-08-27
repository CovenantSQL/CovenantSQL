/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package logictestparser implements sqllogictest parser,
// 	for more see: https://sqlite.org/sqllogictest/doc/trunk/about.wiki
package logictestparser

import (
	"io/ioutil"

	"regexp"

	"gitlab.com/thunderdb/ThunderDB/utils/log"
)

// RecordType is Statement or Query
type RecordType int

// MatchList is list of map[groupName]RegexMatchedString
type MatchList []map[string]string

const (
	// Statement is an SQL command that is to be evaluated but from which
	// 	we do not expect to get results (other than success or failure).
	// 	A statement might be a CREATE TABLE or an INSERT or an UPDATE or a DROP INDEX.
	Statement RecordType = iota
	// Query is an SQL command from which we expect to receive results.
	// 	The result set might be empty.
	Query
	// Halt can be inserted after a query that is giving an anomalous result,
	// 	causing the database to be left in the state where it gives the unexpected answer.
	// 	After sqllogictest exist, manually debugging can then proceed.
	Halt
)

// QueryRecord is query record:
// 	query <type-string> <sort-mode> <label>
//
// 	The <type-string> argument to the query statement is a short string that specifies the number
// 	of result columns and the expected datatype of each result column. There is one character in
// 	the <type-string> for each result column. The characters codes are "T" for a text result,
// 	"I" for an integer result, and "R" for a floating-point result.
//
// 	The <sort-mode> argument is optional. If included, it must be one of "nosort", "rowsort", or
// 	"valuesort". The default is "nosort". In nosort mode, the results appear in exactly the order
// 	in which they were received from the database engine. The nosort mode should only be used on
// 	queries that have an ORDER BY clause or which only have a single row of result, since otherwise
// 	the order of results is undefined and might vary from one database engine to another. The
// 	"rowsort" mode gathers all output from the database engine then sorts it by rows on the
// 	client side. Sort comparisons use strcmp() on the rendered ASCII text representation of the
// 	values. Hence, "9" sorts after "10", not before. The "valuesort" mode works like rowsort
// 	except that it does not honor row groupings. Each individual result value is sorted on its
// 	own.
//
// 	The <label> argument is also optional. If included, sqllogictest stores a hash of the results
// 	of this query under the given label. If the label is reused, then sqllogictest verifies that
// 	the results are the same. This can be used to verify that two or more queries in the same
// 	test script that are logically equivalent always generate the same output.
type QueryRecord struct {
	TypeString string
	SortMode   string
	Label      string
	SQL        string
	Result     string
}

// StatementRecord begins with one of the following two lines:
// 	statement ok
//	statement error
type StatementRecord struct {
	OK  bool
	SQL string
}

// Record is StatementRecord or QueryRecord
type Record struct {
	Type      RecordType
	Query     *QueryRecord
	Statement *StatementRecord
	SkipIf    string // eg. SkipIf sqlite # empty RHS
	OnlyIf    string // eg. onlyif sqlite # empty RHS
}

// SQLLogicTestSuite is SQL logic test records list
type SQLLogicTestSuite struct {
	ml      MatchList
	Records []*Record
}

// Parse parses the test file and returns a MatchList
func (slt *SQLLogicTestSuite) Parse(file string) (ml MatchList, err error) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		log.Errorf("read test file %s failed: %v", file, err)
		return
	}
	pattern := regexp.MustCompile(
		`(?ms)(?:(?:(?P<if>skipif|onlyif)\s+(?P<condition>.+?)\s*(?:#.*?){0,1}\n){0,1})` +
			`((?P<halt>halt)|(?:(?P<type>statement|query)\s+(?P<rconf>.*?)\n(?P<record>.*?))\n\n)`)
	matches := pattern.FindAllStringSubmatch(string(c), -1)
	if len(matches) > 0 {
		ml = make(MatchList, 0, len(matches))
		slt.ml = ml
	}

	for _, m := range matches {
		gm := make(map[string]string)
		for i, g := range pattern.SubexpNames() {
			if len(g) > 0 {
				gm[g] = m[i]
			}
		}
		ml = append(ml, gm)
	}
	log.Debugf("matched: %#v", ml)

	if log.GetLevel() >= log.DebugLevel {
		log.Debugf("matched: %d", len(ml))
		for _, m := range ml {
			log.Debugf("matched: %#v", m)
		}
	}
	return
}

func (slt *SQLLogicTestSuite) ParseTestCases(file string) (records []*Record, err error) {
	_, err = slt.Parse(file)
	if err != nil {
		log.Errorf("parse test file %s failed: %v", file, err)
	}

	for _, m := range slt.ml {
		r := new(Record)
		if len(m["halt"]) != 0 {
			// Halt is just to abort
			r.Type = Halt
		} else if m["type"] == "statement" {
			// statement can be:
			// 	statement ok
			//	statement error
			r.Type = Statement
			r.Statement = new(StatementRecord)
			switch m["rconf"] {
			case "ok":
				r.Statement.OK = true
			case "error":
				r.Statement.OK = false
			}
		} else if m["type"] == "query" {
			// query <type-string> <sort-mode> <label>
			r.Type = Query
			//strings.SplitN(m["rconf"])
		}
	}
	return
}
