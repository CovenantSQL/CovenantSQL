/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package resolver

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// FieldMap defines the field map for future column level acl enforce.
type FieldMap map[string]bool

// Merge combine two field maps.
func (m FieldMap) Merge(f FieldMap) {
	for k := range f {
		m[k] = true
	}
}

var opMap = map[string]string{
	// comparison op
	"$eq":  "=",
	"$gt":  ">",
	"$gte": ">=",
	"$lt":  "<",
	"$lte": "<=",
	"$ne":  "<>",
	"$in":  "IN",
	"$nin": "NOT IN",

	// top-level logic op
	"$and": "AND",
	"$or":  "OR",
	"$nor": "AND",

	// update op
	"$inc": `"{field}" = "{field}" + ?`,
	"$max": `"{field}" = max("{field}", ?)`,
	"$min": `"{field}" = min("{field}", ?)`,
	"$mul": `"{field}" = "{field}" * ?`,
	"$set": `"{field}" = ?`,
}

// ResolveProjection resolves projection object and returns project sql statement and dependent fields.
func ResolveProjection(p map[string]interface{}, availFields FieldMap) (fm FieldMap, statement string, err error) {
	fm = FieldMap{}
	var subStatements []string

	for k, v := range p {
		if !availFields[k] {
			err = errors.Errorf("unknown field: %s", k)
			return
		}

		if asBool(v) {
			fm[k] = true
			subStatements = append(subStatements, fmt.Sprintf(`"%s"`, k))
		}
	}

	if len(subStatements) == 0 {
		for k := range availFields {
			fm[k] = true
			subStatements = append(subStatements, fmt.Sprintf(`"%s"`, k))
		}
	}

	statement = strings.Join(subStatements, ",")

	return
}

func logicRelation(childQueries []map[string]interface{}, availFields FieldMap, op string) (
	fields FieldMap, statement string, args []interface{}, err error) {
	fields = FieldMap{}

	if len(childQueries) == 0 {
		err = errors.New("$and/$or operator needs non-empty array")
		return
	}

	var subStatements []string

	for _, cq := range childQueries {
		var (
			childField     FieldMap
			childStatement string
			childArgs      []interface{}
		)
		childField, childStatement, childArgs, err = ResolveFilter(cq, availFields)
		if err != nil {
			return
		}

		if childStatement == "" {
			continue
		}

		fields.Merge(childField)
		subStatements = append(subStatements, childStatement)
		args = append(args, childArgs...)
	}

	if len(subStatements) == 0 {
		return
	}

	statement = "(" + strings.Join(subStatements, ") "+op+" (") + ")"
	return
}

// ResolveFieldCondition resolves field condition object and returns as where condition statement.
func ResolveFieldCondition(field string, v interface{}) (statement string, args []interface{}, err error) {
	if rv, ok := v.(map[string]interface{}); ok {
		var subStatements []string

		for k, v := range rv {
			switch k {
			case "$eq", "$gt", "$gte", "$lt", "$lte", "$ne":
				if !isLiteral(v) {
					err = errors.Errorf("argument of operator %s is not a literal", k)
					return
				}

				subStatements = append(subStatements, fmt.Sprintf(`"%s" %s ?`, field, opMap[k]))
				args = append(args, v)
			case "$in", "$nin":
				var lv []interface{}

				if lv, ok = v.([]interface{}); !ok {
					err = errors.New("$in/$nin operator needs array")
					return
				}

				for _, lve := range lv {
					if !isLiteral(lve) {
						err = errors.Errorf("can not nest non-literal under %s operator", k)
						return
					}
				}

				if len(lv) == 0 {
					if k == "$in" {
						subStatements = append(subStatements, "1=0")
					} else {
						subStatements = append(subStatements, "1=1")
					}
				} else {
					subStatements = append(subStatements, fmt.Sprintf(`"%s" %s (%s?)`,
						field, opMap[k], strings.Repeat("?,", len(lv)-1)))
					args = append(args, lv...)
				}
			case "$not":
				if rv, ok := v.(map[string]interface{}); !ok || len(rv) == 0 {
					err = errors.New("$not operator needs non-empty object")
					return
				}

				var (
					notSubStatement string
					notSubArgs      []interface{}
				)
				notSubStatement, notSubArgs, err = ResolveFieldCondition(field, v)
				if err != nil {
					return
				}

				subStatements = append(subStatements, "NOT ("+notSubStatement+")")
				args = append(args, notSubArgs...)
			case "$comment":
				// ignore
			default:
				err = errors.Errorf("unknown operator %s", k)
				return
			}
		}

		statement = "(" + strings.Join(subStatements, ") AND (") + ")"
	} else if isLiteral(v) {
		statement = fmt.Sprintf(`"%s" = ?`, field)
		args = append(args, v)
	} else {
		// invalid
		err = errors.New("does not support $eq to non-literal type")
	}

	return
}

// ResolveFilter resolves filter object and returns as full where statement.
func ResolveFilter(q map[string]interface{}, availFields FieldMap) (
	fields FieldMap, statement string, args []interface{}, err error) {
	if q == nil {
		return
	}

	fields = FieldMap{}
	var subStatements []string

	if len(q) == 0 {
		return
	}

	for k, v := range q {
		switch {
		case k == "$and" || k == "$or" || k == "$nor":
			var (
				childFields    FieldMap
				childStatement string
				childArgs      []interface{}
				childQuery     []map[string]interface{}
			)
			childQuery, err = getNonEmptyArrayOfObjects(v)
			if err != nil {
				err = errors.Wrapf(err, "%s operator", k)
				return
			}
			childFields, childStatement, childArgs, err = logicRelation(childQuery, availFields, opMap[k])
			if err != nil {
				return
			}
			if k == "$nor" {
				// $nor empty result should be false
				if childStatement == "" {
					childStatement = "TRUE"
				}
				childStatement = "NOT (" + childStatement + ")"
			} else {
				if childStatement == "" {
					continue
				}
			}
			fields.Merge(childFields)
			subStatements = append(subStatements, childStatement)
			args = append(args, childArgs...)
		case k == "$comment":
			// ignore
		case strings.HasPrefix(k, "$"):
			err = errors.Errorf("invalid operator %s", k)
			return
		default:
			// treat key as column
			if !availFields[k] {
				err = errors.Errorf("unknown field: %s", k)
				return
			}

			var (
				childStatement string
				childArgs      []interface{}
			)
			childStatement, childArgs, err = ResolveFieldCondition(k, v)
			if err != nil {
				return
			}

			fields[k] = true
			subStatements = append(subStatements, childStatement)
			args = append(args, childArgs...)
		}
	}

	if len(subStatements) == 0 {
		return
	}

	statement = "(" + strings.Join(subStatements, ") AND (") + ")"

	return
}

// ResolveUpdate resolves update object as sql update set statement.
func ResolveUpdate(q map[string]interface{}, availFields FieldMap) (
	fields FieldMap, statement string, args []interface{}, err error) {
	fields = FieldMap{}
	var (
		subStatements []string
		useDollarOp   bool
		useNormalSet  bool
	)

	if len(q) == 0 {
		err = errors.New("update to empty object is not supported")
		return
	}

	for k, v := range q {
		switch {
		case k == "$currentDate" || k == "$inc" || k == "$max" || k == "$min" || k == "$mul" || k == "$set":
			useDollarOp = true

			if useNormalSet {
				err = errors.New("could not use both normal field update and $ prefixed ops")
				return
			}

			// value must be an object
			var (
				ov map[string]interface{}
				ok bool
			)

			if ov, ok = v.(map[string]interface{}); !ok {
				err = errors.Errorf("$ operator needs object argument")
				return
			}

			for field, argument := range ov {
				if !availFields[field] {
					err = errors.Errorf("unknown field: %s", field)
					return
				}

				fields[field] = true

				if k == "$currentDate" {
					if isLiteral(argument) && asBool(argument) {
						subStatements = append(subStatements, fmt.Sprintf(`"%s" = ?`, field))
						args = append(args, time.Now().UTC())
					} else if oa, ok := argument.(map[string]interface{}); ok && isString(oa["$type"]) {
						switch oa["$type"] {
						case "date":
							subStatements = append(subStatements, fmt.Sprintf(`"%s" = ?`, field))
							args = append(args, time.Now().UTC().Format("2006-01-02"))
						case "timestamp":
							subStatements = append(subStatements, fmt.Sprintf(`"%s" = ?`, field))
							args = append(args, time.Now().Unix())
						case "datetime":
							// mongodb does not support $currentDate with datetime
							// this my special treatment for stupid users
							subStatements = append(subStatements, fmt.Sprintf(`"%s" = ?`, field))
							args = append(args, time.Now().UTC().Format("2006-01-02 15:04:05"))
						default:
							err = errors.Errorf("invalid $currentDate type %s", oa["$type"])
							return
						}
					} else {
						// invalid
						err = errors.Errorf("$currentDate operator requires true or valid type config")
						return
					}
				} else {
					if !isLiteral(argument) {
						err = errors.Errorf("%s operator requires literal value as argument", k)
						return
					}

					subStatements = append(subStatements, strings.Replace(opMap[k], "{field}", field, -1))
					args = append(args, argument)
				}
			}
		case k == "$comment":
			// ignore
		case strings.HasPrefix(k, "$"):
			// error
			err = errors.Errorf("invalid operator %s", k)
			return
		default:
			// treat as column
			useNormalSet = true

			if useDollarOp {
				err = errors.New("could not use both normal field set and $ prefixed ops")
				return
			}

			if !isLiteral(v) {
				err = errors.Errorf("%s operator requires literal value as argument", k)
				return
			}

			if !availFields[k] {
				err = errors.Errorf("unknown field: %s", k)
				return
			}

			fields[k] = true

			subStatements = append(subStatements, fmt.Sprintf(`"%s" = ?`, k))
			args = append(args, v)
		}
	}

	statement = "SET " + strings.Join(subStatements, ", SET ")

	return
}

// ResolveInsert resolves insert data as columns and values statement in sql.
func ResolveInsert(q map[string]interface{}, availFields FieldMap) (fields FieldMap, statement string, args []interface{}, err error) {
	fields = FieldMap{}

	if len(q) == 0 {
		err = errors.New("could not insert empty object")
		return
	}

	var colStatements []string

	for k, v := range q {
		if !availFields[k] {
			err = errors.Errorf("unknown field: %s", k)
			return
		}

		fields[k] = true

		colStatements = append(colStatements, fmt.Sprintf(`"%s"`, k))
		args = append(args, v)
	}

	statement = "(" + strings.Join(colStatements, ",") + ") VALUES(" +
		strings.Repeat("?,", len(colStatements)-1) + "?)"
	return
}

// ResolveOrderBy resolves order by statement.
func ResolveOrderBy(q map[string]interface{}, availFields FieldMap) (fields FieldMap, statement string, err error) {
	fields = FieldMap{}

	if len(q) == 0 {
		// no order by config
		return
	}

	var colStatements []string

	for k, v := range q {
		if !availFields[k] {
			err = errors.Errorf("unknown field: %s", k)
			return
		}

		if !isLiteral(v) {
			err = errors.Errorf("invalid sort direction: %v", v)
			return
		}

		var direction string

		if asInt(v) > 0 {
			direction = "ASC"
		} else if asInt(v) < 0 {
			direction = "DESC"
		} else {
			err = errors.Errorf("invalid sort direction: %v", v)
			return
		}

		fields[k] = true

		colStatements = append(colStatements, fmt.Sprintf(`"%s" %s`, k, direction))
	}

	statement = strings.Join(colStatements, ",")

	return
}

func asInt(v interface{}) int64 {
	switch d := v.(type) {
	case bool:
		if d {
			return 1
		}

		return 0
	case int:
		return int64(d)
	case int8:
		return int64(d)
	case int16:
		return int64(d)
	case int32:
		return int64(d)
	case int64:
		return d
	case uint:
		return int64(d)
	case uint8:
		return int64(d)
	case uint16:
		return int64(d)
	case uint32:
		return int64(d)
	case uint64:
		return int64(d)
	case float32:
		return int64(d)
	case float64:
		return int64(d)
	case string:
		i, _ := strconv.ParseInt(d, 10, 64)
		return i
	}

	return 0
}

func asBool(v interface{}) bool {
	switch d := v.(type) {
	case bool:
		return d
	case int:
		return d != 0
	case int8:
		return d != 0
	case int16:
		return d != 0
	case int32:
		return d != 0
	case int64:
		return d != 0
	case uint:
		return d != 0
	case uint8:
		return d != 0
	case uint16:
		return d != 0
	case uint32:
		return d != 0
	case uint64:
		return d != 0
	case float32:
		return math.Abs(float64(d)) > 0
	case float64:
		return math.Abs(d) > 0
	case string:
		return len(d) > 0
	}

	return false
}

func isString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

func isLiteral(v interface{}) bool {
	switch v.(type) {
	case string,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}

	return false
}

func getNonEmptyArrayOfObjects(v interface{}) (res []map[string]interface{}, err error) {
	arr, ok := v.([]interface{})
	if !ok {
		err = errors.New("requires array argument")
		return
	}
	if len(arr) == 0 {
		err = errors.New("requires non-empty array argument")
		return
	}

	for _, v := range arr {
		var (
			o  map[string]interface{}
			ok bool
		)

		if o, ok = v.(map[string]interface{}); !ok {
			err = errors.New("requires array full of objects")
			return
		}
		res = append(res, o)
	}

	return
}
